package seckill

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gin-looklook/internal/order"
	"gin-looklook/internal/shared"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	Stream = "gin:looklook:{seckill}:v1:stream"
	Group  = "gin-looklook-seckill"
	prefix = "gin:looklook:{seckill}:v1"
)

var reserveScript = redis.NewScript(`
-- TODO(practice-06): 实现活动时间校验、一人一单、预扣库存、结果初始化和 Stream 投递。
return {-4, ''}
`)

var compensateScript = redis.NewScript(`
-- TODO(practice-06): 只能对 pending 预约补偿一次，并恢复库存/用户占用。
return 0
`)

var completeScript = redis.NewScript(`
-- TODO(practice-06): 幂等标记创单成功，必要时只恢复一次 Redis 库存。
return 0
`)

func activityKey(id int64) string { return fmt.Sprintf("%s:activity:%d", prefix, id) }
func stockKey(id int64) string    { return fmt.Sprintf("%s:stock:%d", prefix, id) }
func usersKey(id int64) string    { return fmt.Sprintf("%s:users:%d", prefix, id) }
func resultKey(sn string) string  { return fmt.Sprintf("%s:result:%s", prefix, sn) }
func sequenceKey() string         { return prefix + ":sequence" }
func makeReservationSN(unix, seq int64) string {
	// TODO(practice-06): 生成长度固定、以 SKR 开头的可唯一预约号。
	return ""
}

type Service struct {
	repo   *order.Repository
	orders *order.Service
	redis  *redis.Client
}

func NewService(repo *order.Repository, orders *order.Service, rdb *redis.Client) *Service {
	return &Service{repo: repo, orders: orders, redis: rdb}
}

func (s *Service) Warmup(ctx context.Context) error {
	activities, err := s.repo.ActiveSeckillActivities(ctx)
	if err != nil {
		return err
	}
	for _, a := range activities {
		remaining := a.Stock - a.SoldCount
		if remaining < 0 {
			remaining = 0
		}
		expireAt := a.EndTime.Add(7 * 24 * time.Hour)
		pipe := s.redis.TxPipeline()
		pipe.HSet(ctx, activityKey(a.ID), "startAt", a.StartTime.Unix(), "endAt", a.EndTime.Unix(), "status", a.Status)
		pipe.SetNX(ctx, stockKey(a.ID), remaining, time.Until(expireAt))
		pipe.ExpireAt(ctx, activityKey(a.ID), expireAt)
		pipe.ExpireAt(ctx, stockKey(a.ID), expireAt)
		pipe.ExpireAt(ctx, usersKey(a.ID), expireAt)
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func toActivity(a order.SeckillActivity) Activity {
	return Activity{ID: a.ID, HomestayID: a.HomestayID, Title: a.Title, Price: a.Price, Stock: a.Stock, SoldCount: a.SoldCount, StartTime: a.StartTime.Unix(), EndTime: a.EndTime.Unix(), Status: a.Status}
}

func (s *Service) Activities(ctx context.Context) ([]Activity, error) {
	items, err := s.repo.ActiveSeckillActivities(ctx)
	if err != nil {
		return nil, shared.E(shared.CodeDB, "查询秒杀活动失败", err)
	}
	out := make([]Activity, 0, len(items))
	for i := range items {
		a := toActivity(items[i])
		remaining, err := s.redis.Get(ctx, stockKey(items[i].ID)).Int64()
		if err != nil {
			remaining = items[i].Stock - items[i].SoldCount
		}
		if remaining < 0 {
			remaining = 0
		}
		a.Remaining = remaining
		out = append(out, a)
	}
	return out, nil
}

func (s *Service) Reserve(ctx context.Context, userID, activityID, liveStart, liveEnd, people int64, remark string) (string, error) {
	if activityID <= 0 || people <= 0 || liveEnd <= liveStart || time.Unix(liveEnd, 0).Sub(time.Unix(liveStart, 0)) < 24*time.Hour {
		return "", shared.E(shared.CodeParam, "秒杀参数错误，入住时间至少一晚", nil)
	}
	sequence, err := s.redis.Incr(ctx, sequenceKey()).Result()
	if err != nil {
		return "", shared.E(shared.CodeCommon, "生成秒杀预约号失败", err)
	}
	reservationSN := makeReservationSN(time.Now().Unix(), sequence)
	keys := []string{activityKey(activityID), stockKey(activityID), usersKey(activityID), resultKey(reservationSN), Stream}
	result, err := reserveScript.Run(ctx, s.redis, keys, userID, reservationSN, activityID, liveStart, liveEnd, people, remark).Slice()
	if err != nil {
		return "", shared.E(shared.CodeCommon, "秒杀系统繁忙，请稍后重试", err)
	}
	code, _ := result[0].(int64)
	sn, _ := result[1].(string)
	switch code {
	case 0, 1:
		shared.ObserveSeckillReservation("accepted")
		return sn, nil
	case -1:
		shared.ObserveSeckillReservation("not_started")
		return "", shared.E(shared.CodeCommon, "秒杀尚未开始", nil)
	case -2:
		shared.ObserveSeckillReservation("ended")
		return "", shared.E(shared.CodeCommon, "秒杀已经结束", nil)
	case -3:
		shared.ObserveSeckillReservation("sold_out")
		return "", shared.E(shared.CodeCommon, "秒杀商品已售罄", nil)
	default:
		shared.ObserveSeckillReservation("not_found")
		return "", shared.E(shared.CodeCommon, "秒杀活动不存在", nil)
	}
}

func (s *Service) Result(ctx context.Context, userID int64, reservationSN string) (*Result, error) {
	values, err := s.redis.HGetAll(ctx, resultKey(reservationSN)).Result()
	if err != nil {
		return nil, shared.E(shared.CodeCommon, "查询秒杀结果失败", err)
	}
	if len(values) == 0 || values["userId"] != strconv.FormatInt(userID, 10) {
		return nil, shared.E(shared.CodeCommon, "秒杀记录不存在", nil)
	}
	return &Result{ReservationSN: reservationSN, Status: values["status"], OrderSN: values["orderSn"], Error: values["error"]}, nil
}

func (s *Service) Process(ctx context.Context, reservation Reservation) error {
	activity, err := s.repo.SeckillActivityByID(ctx, reservation.ActivityID)
	if err == gorm.ErrRecordNotFound {
		return order.ErrSeckillSoldOut
	}
	if err != nil {
		shared.ObserveSeckillOrder("attempt_failed")
		return err
	}
	orderSN, restoreStock, err := s.orders.CreateSeckill(ctx, reservation.ReservationSN, activity.ID, reservation.UserID, *activity, reservation.LiveStartTime, reservation.LiveEndTime, reservation.LivePeopleNum, reservation.Remark)
	if err != nil {
		shared.ObserveSeckillOrder("attempt_failed")
		return err
	}
	shared.ObserveSeckillOrder("success")
	restore := "0"
	if restoreStock {
		restore = "1"
	}
	return completeScript.Run(ctx, s.redis, []string{resultKey(reservation.ReservationSN), stockKey(reservation.ActivityID)}, restore, orderSN).Err()
}

func (s *Service) IncrementAttempts(ctx context.Context, reservationSN string) int64 {
	n, err := s.redis.HIncrBy(ctx, resultKey(reservationSN), "attempts", 1).Result()
	if err != nil {
		return 1
	}
	return n
}

func (s *Service) FailAndCompensate(ctx context.Context, reservation Reservation, cause error) error {
	message := "创建秒杀订单失败"
	if errors.Is(cause, order.ErrSeckillSoldOut) {
		message = "秒杀商品已售罄"
	}
	keys := []string{resultKey(reservation.ReservationSN), stockKey(reservation.ActivityID), usersKey(reservation.ActivityID)}
	return compensateScript.Run(ctx, s.redis, keys, reservation.UserID, message).Err()
}
