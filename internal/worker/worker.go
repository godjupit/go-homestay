package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"gin-looklook/internal/order"
	"gin-looklook/internal/payment"
	"gin-looklook/internal/search"
	"gin-looklook/internal/seckill"
	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

type Runtime struct {
	cfg         shared.Config
	orders      *order.Service
	seckill     *seckill.Service
	search      *search.Service
	searchRepo  *search.Repository
	paymentRepo *payment.Repository
	server      *asynq.Server
	scheduler   *asynq.Scheduler
	reader      *kafka.Reader
	writer      *kafka.Writer
	redis       *redis.Client
}

func New(cfg shared.Config, orders *order.Service, seckillSvc *seckill.Service, searchSvc *search.Service, searchRepo *search.Repository, paymentRepo *payment.Repository, writer *kafka.Writer, rdb *redis.Client) *Runtime {
	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword}
	return &Runtime{
		cfg: cfg, orders: orders, seckill: seckillSvc, search: searchSvc,
		searchRepo: searchRepo, paymentRepo: paymentRepo,
		server:    asynq.NewServer(redisOpt, asynq.Config{Concurrency: 10, Queues: map[string]int{"critical": 6, "default": 3, "low": 1}}),
		scheduler: asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{Location: time.Local}),
		reader:    kafka.NewReader(kafka.ReaderConfig{Brokers: cfg.KafkaBrokers, GroupID: cfg.PaymentGroup, Topic: cfg.PaymentTopic, MinBytes: 1, MaxBytes: 10e6}),
		writer:    writer,
		redis:     rdb,
	}
}

func (r *Runtime) Start(ctx context.Context) error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(order.TaskCloseOrder, r.closeOrder)
	mux.HandleFunc(order.TaskPaySuccessNotify, r.notifyUser)
	mux.HandleFunc(order.TaskSettle, r.settle)
	if _, err := r.scheduler.Register("*/1 * * * *", asynq.NewTask(order.TaskSettle, nil)); err != nil {
		return err
	}
	if err := r.redis.XGroupCreateMkStream(ctx, seckill.Stream, seckill.Group, "0").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	if err := r.scheduler.Start(); err != nil {
		return err
	}
	if err := r.server.Start(mux); err != nil {
		r.scheduler.Shutdown()
		return err
	}
	go r.consumePayments(ctx)
	go r.publishOutbox(ctx)
	go r.consumeSeckill(ctx)
	go r.publishSearchOutbox(ctx)
	return nil
}

func (r *Runtime) publishSearchOutbox(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			items, err := r.searchRepo.PendingSearchOutbox(ctx, 100)
			if err != nil {
				slog.Error("query search outbox", "error", err)
				continue
			}
			for _, item := range items {
				if item.EventType == "delete" {
					err = r.search.DeleteHomestay(ctx, item.AggregateID)
				} else {
					var homestay *travel.Homestay
					homestay, err = r.searchRepo.HomestayForIndex(ctx, item.AggregateID)
					if errors.Is(err, gorm.ErrRecordNotFound) {
						err = r.search.DeleteHomestay(ctx, item.AggregateID)
					} else if err == nil {
						err = r.search.IndexHomestay(ctx, homestay)
					}
				}
				if err != nil {
					_ = r.searchRepo.RetrySearchOutbox(ctx, item.ID, item.RetryCount, err)
					slog.Error("sync search document", "outboxId", item.ID, "aggregateId", item.AggregateID, "error", err)
					continue
				}
				if err = r.searchRepo.MarkSearchOutboxPublished(ctx, item.ID); err != nil {
					slog.Error("mark search outbox published", "outboxId", item.ID, "error", err)
				}
			}
		}
	}
}

func (r *Runtime) consumeSeckill(ctx context.Context) {
	hostname, _ := os.Hostname()
	consumer := fmt.Sprintf("%s-%d", hostname, os.Getpid())
	claimTicker := time.NewTicker(15 * time.Second)
	defer claimTicker.Stop()
	for {
		streams, err := r.redis.XReadGroup(ctx, &redis.XReadGroupArgs{Group: seckill.Group, Consumer: consumer, Streams: []string{seckill.Stream, ">"}, Count: 20, Block: 2 * time.Second}).Result()
		if err != nil && err != redis.Nil && ctx.Err() == nil {
			slog.Error("read seckill stream", "error", err)
		}
		for _, stream := range streams {
			r.processSeckillMessages(ctx, stream.Messages)
		}
		select {
		case <-ctx.Done():
			return
		case <-claimTicker.C:
			messages, _, claimErr := r.redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: seckill.Stream, Group: seckill.Group, Consumer: consumer, MinIdle: 30 * time.Second, Start: "0-0", Count: 20}).Result()
			if claimErr != nil && claimErr != redis.Nil {
				slog.Error("claim seckill stream", "error", claimErr)
			} else {
				r.processSeckillMessages(ctx, messages)
			}
		default:
		}
	}
}

func (r *Runtime) processSeckillMessages(ctx context.Context, messages []redis.XMessage) {
	for _, message := range messages {
		activityID, err1 := strconv.ParseInt(streamString(message.Values, "activityId"), 10, 64)
		userID, err2 := strconv.ParseInt(streamString(message.Values, "userId"), 10, 64)
		liveStart, err3 := strconv.ParseInt(streamString(message.Values, "liveStartTime"), 10, 64)
		liveEnd, err4 := strconv.ParseInt(streamString(message.Values, "liveEndTime"), 10, 64)
		people, err5 := strconv.ParseInt(streamString(message.Values, "livePeopleNum"), 10, 64)
		reservation := seckill.Reservation{ReservationSN: streamString(message.Values, "reservationSn"), ActivityID: activityID, UserID: userID, LiveStartTime: liveStart, LiveEndTime: liveEnd, LivePeopleNum: people, Remark: streamString(message.Values, "remark")}
		if err := errors.Join(err1, err2, err3, err4, err5); err == nil && reservation.ReservationSN != "" {
			err = r.seckill.Process(ctx, reservation)
			if err == nil {
				_ = r.redis.XAck(ctx, seckill.Stream, seckill.Group, message.ID).Err()
				continue
			}
			attempts := r.seckill.IncrementAttempts(ctx, reservation.ReservationSN)
			if errors.Is(err, order.ErrSeckillSoldOut) || attempts >= 5 {
				_ = r.seckill.FailAndCompensate(ctx, reservation, err)
				_ = r.redis.XAck(ctx, seckill.Stream, seckill.Group, message.ID).Err()
			}
			slog.Error("process seckill reservation", "reservationSn", reservation.ReservationSN, "attempts", attempts, "error", err)
			continue
		}
		slog.Error("invalid seckill stream message", "id", message.ID, "values", message.Values)
		_ = r.redis.XAck(ctx, seckill.Stream, seckill.Group, message.ID).Err()
	}
}

func streamString(values map[string]any, key string) string {
	if v, ok := values[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func (r *Runtime) publishOutbox(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			items, err := r.paymentRepo.PendingOutbox(ctx, 100)
			if err != nil {
				slog.Error("query outbox", "error", err)
				continue
			}
			for _, item := range items {
				err = r.writer.WriteMessages(ctx, kafka.Message{Topic: item.Topic, Key: []byte(item.MessageKey), Value: item.Payload})
				if err != nil {
					_ = r.paymentRepo.RetryOutbox(ctx, item.ID)
					slog.Error("publish outbox", "id", item.ID, "error", err)
					continue
				}
				if err = r.paymentRepo.MarkOutboxPublished(ctx, item.ID); err != nil {
					slog.Error("mark outbox published", "id", item.ID, "error", err)
				}
			}
		}
	}
}

func (r *Runtime) Stop() { r.scheduler.Shutdown(); r.server.Shutdown(); _ = r.reader.Close() }

func (r *Runtime) closeOrder(ctx context.Context, task *asynq.Task) error {
	var p order.CloseOrderPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}
	ord, err := r.orders.Detail(ctx, 0, p.SN)
	if err != nil {
		return err
	}
	if ord.TradeState == order.TradeStateWaitPay {
		_, err = r.orders.UpdateState(ctx, p.SN, order.TradeStateCancel, 0)
	}
	return err
}

func (r *Runtime) settle(context.Context, *asynq.Task) error {
	slog.Info("schedule settlement demo executed")
	return nil
}

func (r *Runtime) notifyUser(ctx context.Context, task *asynq.Task) error {
	var p order.NotifyPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}
	_, err := r.orders.Detail(ctx, 0, p.OrderSN)
	if err != nil {
		return err
	}
	slog.Info("notify user demo executed", "orderSn", p.OrderSN)
	return nil
}

func (r *Runtime) consumePayments(ctx context.Context) {
	for {
		msg, err := r.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("fetch payment event", "error", err)
			time.Sleep(time.Second)
			continue
		}
		var event payment.StatusEvent
		if err = json.Unmarshal(msg.Value, &event); err == nil {
			var state int64 = -99
			if event.PayStatus == payment.StatusSuccess {
				state = order.TradeStateWaitUse
			} else if event.PayStatus == payment.StatusRefund {
				state = order.TradeStateRefund
			}
			if state != -99 {
				_, err = r.orders.UpdateState(ctx, event.OrderSN, state, 0)
			}
		}
		if err != nil {
			slog.Error("consume payment event", "error", err, "value", string(msg.Value))
			continue
		}
		if err = r.reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("commit payment event", "error", err)
		}
	}
}
