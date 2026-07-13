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
local now = tonumber(redis.call('TIME')[1])
local startAt = tonumber(redis.call('HGET', KEYS[1], 'startAt'))
local endAt = tonumber(redis.call('HGET', KEYS[1], 'endAt'))
local status = tonumber(redis.call('HGET', KEYS[1], 'status'))
if not startAt or not endAt or status ~= 1 then return {-4, ''} end
if now < startAt then return {-1, ''} end
if now > endAt then return {-2, ''} end
local previous = redis.call('HGET', KEYS[3], ARGV[1])
if previous then return {1, previous} end
local stock = tonumber(redis.call('GET', KEYS[2]) or '0')
if stock <= 0 then return {-3, ''} end
redis.call('DECR', KEYS[2])
redis.call('HSET', KEYS[3], ARGV[1], ARGV[2])
redis.call('HSET', KEYS[4],
  'status', 'pending', 'userId', ARGV[1], 'activityId', ARGV[3],
  'liveStartTime', ARGV[4], 'liveEndTime', ARGV[5],
  'livePeopleNum', ARGV[6], 'remark', ARGV[7], 'attempts', '0')
redis.call('EXPIREAT', KEYS[3], endAt + 604800)
redis.call('EXPIREAT', KEYS[4], endAt + 604800)
redis.call('XADD', KEYS[5], 'MAXLEN', '~', 100000, '*',
  'reservationSn', ARGV[2], 'userId', ARGV[1], 'activityId', ARGV[3],
  'liveStartTime', ARGV[4], 'liveEndTime', ARGV[5],
  'livePeopleNum', ARGV[6], 'remark', ARGV[7])
return {0, ARGV[2]}
`)

var compensateScript = redis.NewScript(`
if redis.call('HGET', KEYS[1], 'status') == 'pending' then
  redis.call('INCR', KEYS[2])
  redis.call('HDEL', KEYS[3], ARGV[1])
  redis.call('HSET', KEYS[1], 'status', 'failed', 'error', ARGV[2])
  return 1
end
return 0
`)

var completeScript = redis.NewScript(`
if ARGV[1] == '1' and redis.call('HGET', KEYS[1], 'stockRestored') ~= '1' then
  redis.call('INCR', KEYS[2])
  redis.call('HSET', KEYS[1], 'stockRestored', '1')
end
redis.call('HSET', KEYS[1], 'status', 'success', 'orderSn', ARGV[2], 'error', '')
return 1
`)

func activityKey(id int64) string              { return fmt.Sprintf("%s:activity:%d", prefix, id) }
func stockKey(id int64) string                 { return fmt.Sprintf("%s:stock:%d", prefix, id) }
func usersKey(id int64) string                 { return fmt.Sprintf("%s:users:%d", prefix, id) }
func resultKey(sn string) string               { return fmt.Sprintf("%s:result:%s", prefix, sn) }
func sequenceKey() string                      { return prefix + ":sequence" }
func makeReservationSN(unix, seq int64) string { return fmt.Sprintf("SKR%014d%08x", unix, uint32(seq)) }

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
