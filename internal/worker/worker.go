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

	"gin-looklook/internal/config"
	"gin-looklook/internal/model"
	"gin-looklook/internal/repository"
	"gin-looklook/internal/service"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	wechat "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	miniConfig "github.com/silenceper/wechat/v2/miniprogram/config"
	"github.com/silenceper/wechat/v2/miniprogram/subscribe"
)

const (
	payTemplate  = "QIJPmfxaNqYzSjOlXGk1T6Xfw94JwbSPuOd3u_hi3WE"
	liveTemplate = "kmm-maRr6v_9eMxEPpj-5clJ2YW_EFpd8-ngyYk63e4"
)

type Runtime struct {
	cfg       config.Config
	repo      *repository.Repository
	orders    *service.OrderService
	seckill   *service.SeckillService
	search    *service.SearchService
	server    *asynq.Server
	scheduler *asynq.Scheduler
	reader    *kafka.Reader
	writer    *kafka.Writer
}

func New(cfg config.Config, repo *repository.Repository, orders *service.OrderService, seckill *service.SeckillService, search *service.SearchService, writer *kafka.Writer) *Runtime {
	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword}
	return &Runtime{cfg: cfg, repo: repo, orders: orders, seckill: seckill, search: search, server: asynq.NewServer(redisOpt, asynq.Config{Concurrency: 10, Queues: map[string]int{"critical": 6, "default": 3, "low": 1}}), scheduler: asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{Location: time.Local}), reader: kafka.NewReader(kafka.ReaderConfig{Brokers: cfg.KafkaBrokers, GroupID: cfg.PaymentGroup, Topic: cfg.PaymentTopic, MinBytes: 1, MaxBytes: 10e6}), writer: writer}
}
func (r *Runtime) Start(ctx context.Context) error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(service.TaskCloseOrder, r.closeOrder)
	mux.HandleFunc(service.TaskPaySuccessNotify, r.notifyUser)
	mux.HandleFunc(service.TaskSettle, r.settle)
	if _, err := r.scheduler.Register("*/1 * * * *", asynq.NewTask(service.TaskSettle, nil)); err != nil {
		return err
	}
	if err := r.repo.Redis.XGroupCreateMkStream(ctx, service.SeckillStream, service.SeckillGroup, "0").Err(); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
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
			items, err := r.repo.PendingSearchOutbox(ctx, 100)
			if err != nil {
				slog.Error("query search outbox", "error", err)
				continue
			}
			for _, item := range items {
				if item.EventType == "delete" {
					err = r.search.DeleteHomestay(ctx, item.AggregateID)
				} else {
					var homestay *model.Homestay
					homestay, err = r.repo.HomestayForIndex(ctx, item.AggregateID)
					if errors.Is(err, repository.ErrNotFound) {
						err = r.search.DeleteHomestay(ctx, item.AggregateID)
					} else if err == nil {
						err = r.search.IndexHomestay(ctx, homestay)
					}
				}
				if err != nil {
					_ = r.repo.RetrySearchOutbox(ctx, item.ID, item.RetryCount, err)
					slog.Error("sync search document", "outboxId", item.ID, "aggregateId", item.AggregateID, "error", err)
					continue
				}
				if err = r.repo.MarkSearchOutboxPublished(ctx, item.ID); err != nil {
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
		streams, err := r.repo.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{Group: service.SeckillGroup, Consumer: consumer, Streams: []string{service.SeckillStream, ">"}, Count: 20, Block: 2 * time.Second}).Result()
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
			messages, _, claimErr := r.repo.Redis.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: service.SeckillStream, Group: service.SeckillGroup, Consumer: consumer, MinIdle: 30 * time.Second, Start: "0-0", Count: 20}).Result()
			if claimErr != nil && claimErr != redis.Nil {
				slog.Error("claim seckill stream", "error", claimErr)
			} else {
				r.processSeckillMessages(ctx, messages)
			}
		default:
		}
	}
}

func streamString(values map[string]interface{}, key string) string {
	if value, ok := values[key]; ok {
		return fmt.Sprint(value)
	}
	return ""
}

func (r *Runtime) processSeckillMessages(ctx context.Context, messages []redis.XMessage) {
	for _, message := range messages {
		activityID, err1 := strconv.ParseInt(streamString(message.Values, "activityId"), 10, 64)
		userID, err2 := strconv.ParseInt(streamString(message.Values, "userId"), 10, 64)
		liveStart, err3 := strconv.ParseInt(streamString(message.Values, "liveStartTime"), 10, 64)
		liveEnd, err4 := strconv.ParseInt(streamString(message.Values, "liveEndTime"), 10, 64)
		people, err5 := strconv.ParseInt(streamString(message.Values, "livePeopleNum"), 10, 64)
		reservation := model.SeckillReservation{ReservationSN: streamString(message.Values, "reservationSn"), ActivityID: activityID, UserID: userID, LiveStartTime: liveStart, LiveEndTime: liveEnd, LivePeopleNum: people, Remark: streamString(message.Values, "remark")}
		if err := errors.Join(err1, err2, err3, err4, err5); err == nil && reservation.ReservationSN != "" {
			err = r.seckill.Process(ctx, reservation)
			if err == nil {
				_ = r.repo.Redis.XAck(ctx, service.SeckillStream, service.SeckillGroup, message.ID).Err()
				continue
			}
			attempts := r.seckill.IncrementAttempts(ctx, reservation.ReservationSN)
			if errors.Is(err, repository.ErrSeckillSoldOut) || attempts >= 5 {
				_ = r.seckill.FailAndCompensate(ctx, reservation, err)
				_ = r.repo.Redis.XAck(ctx, service.SeckillStream, service.SeckillGroup, message.ID).Err()
			}
			slog.Error("process seckill reservation", "reservationSn", reservation.ReservationSN, "attempts", attempts, "error", err)
			continue
		}
		slog.Error("invalid seckill stream message", "id", message.ID, "values", message.Values)
		_ = r.repo.Redis.XAck(ctx, service.SeckillStream, service.SeckillGroup, message.ID).Err()
	}
}

func (r *Runtime) publishOutbox(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			items, err := r.repo.PendingOutbox(ctx, 100)
			if err != nil {
				slog.Error("query outbox", "error", err)
				continue
			}
			for _, item := range items {
				err = r.writer.WriteMessages(ctx, kafka.Message{Topic: item.Topic, Key: []byte(item.MessageKey), Value: item.Payload})
				if err != nil {
					_ = r.repo.RetryOutbox(ctx, item.ID)
					slog.Error("publish outbox", "id", item.ID, "error", err)
					continue
				}
				if err = r.repo.MarkOutboxPublished(ctx, item.ID); err != nil {
					slog.Error("mark outbox published", "id", item.ID, "error", err)
				}
			}
		}
	}
}
func (r *Runtime) Stop() { r.scheduler.Shutdown(); r.server.Shutdown(); _ = r.reader.Close() }
func (r *Runtime) closeOrder(ctx context.Context, task *asynq.Task) error {
	var p service.CloseOrderPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}
	order, err := r.orders.Detail(ctx, 0, p.SN)
	if err != nil {
		return err
	}
	if order.TradeState == model.OrderTradeStateWaitPay {
		_, err = r.orders.UpdateState(ctx, p.SN, model.OrderTradeStateCancel)
	}
	return err
}
func (r *Runtime) settle(context.Context, *asynq.Task) error {
	slog.Info("schedule settlement demo executed")
	return nil
}
func (r *Runtime) notifyUser(ctx context.Context, task *asynq.Task) error {
	var p service.NotifyPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return err
	}
	order, err := r.orders.Detail(ctx, 0, p.OrderSN)
	if err != nil {
		return err
	}
	auth, err := r.repo.UserAuthByUser(ctx, order.UserID, model.UserAuthTypeSmallWX)
	if err != nil {
		return err
	}
	if r.cfg.WxAppID == "" || r.cfg.WxAppSecret == "" {
		slog.Info("skip WeChat notification: not configured", "orderSn", order.SN)
		return nil
	}
	mini := wechat.NewWechat().GetMiniProgram(&miniConfig.Config{AppID: r.cfg.WxAppID, AppSecret: r.cfg.WxAppSecret, Cache: cache.NewMemory()})
	messages := []*subscribe.Message{{ToUser: auth.AuthKey, TemplateID: payTemplate, Data: map[string]*subscribe.DataItem{"character_string6": {Value: order.SN}, "thing1": {Value: order.Title}, "amount2": {Value: fmt.Sprintf("%.2f", float64(order.OrderTotalPrice)/100)}, "time4": {Value: order.LiveStartDate.Format("2006-01-02")}, "time5": {Value: order.LiveEndDate.Format("2006-01-02")}}}, {ToUser: auth.AuthKey, TemplateID: liveTemplate, Data: map[string]*subscribe.DataItem{"date2": {Value: order.LiveStartDate.Format("2006-01-02")}, "date3": {Value: order.LiveEndDate.Format("2006-01-02")}, "character_string4": {Value: order.TradeCode}, "thing1": {Value: "请不要将验证码告知商家以外人员，以防上当"}}}}
	for _, msg := range messages {
		msg.MiniprogramState = "developer"
		var sendErr error
		for attempt := 0; attempt < 5; attempt++ {
			if sendErr = mini.GetSubscribe().Send(msg); sendErr == nil {
				break
			}
			time.Sleep(time.Second)
		}
		if sendErr != nil {
			return sendErr
		}
	}
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
		var event model.PaymentStatusEvent
		if err = json.Unmarshal(msg.Value, &event); err == nil {
			var state int64 = -99
			if event.PayStatus == model.PaymentStatusSuccess {
				state = model.OrderTradeStateWaitUse
			} else if event.PayStatus == model.PaymentStatusRefund {
				state = model.OrderTradeStateRefund
			}
			if state != -99 {
				_, err = r.orders.UpdateState(ctx, event.OrderSN, state)
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
