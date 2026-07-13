package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gin-looklook/internal/model"
	"gin-looklook/internal/platform"
	"gin-looklook/internal/repository"

	"github.com/redis/go-redis/v9"
)

const (
	SeckillStream = "gin:looklook:{seckill}:v1:stream"
	SeckillGroup  = "gin-looklook-seckill"
	seckillPrefix = "gin:looklook:{seckill}:v1"
)

var reserveSeckillScript = redis.NewScript(`
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

var compensateSeckillScript = redis.NewScript(`
if redis.call('HGET', KEYS[1], 'status') == 'pending' then
  redis.call('INCR', KEYS[2])
  redis.call('HDEL', KEYS[3], ARGV[1])
  redis.call('HSET', KEYS[1], 'status', 'failed', 'error', ARGV[2])
  return 1
end
return 0
`)

var completeSeckillScript = redis.NewScript(`
if ARGV[1] == '1' and redis.call('HGET', KEYS[1], 'stockRestored') ~= '1' then
  redis.call('INCR', KEYS[2])
  redis.call('HSET', KEYS[1], 'stockRestored', '1')
end
redis.call('HSET', KEYS[1], 'status', 'success', 'orderSn', ARGV[2], 'error', '')
return 1
`)

type SeckillService struct {
	repo   *repository.Repository
	orders *OrderService
}

func NewSeckillService(repo *repository.Repository, orders *OrderService) *SeckillService {
	return &SeckillService{repo: repo, orders: orders}
}

func seckillActivityKey(id int64) string { return fmt.Sprintf("%s:activity:%d", seckillPrefix, id) }
func seckillStockKey(id int64) string    { return fmt.Sprintf("%s:stock:%d", seckillPrefix, id) }
func seckillUsersKey(id int64) string    { return fmt.Sprintf("%s:users:%d", seckillPrefix, id) }
func seckillResultKey(sn string) string  { return fmt.Sprintf("%s:result:%s", seckillPrefix, sn) }
func seckillSequenceKey() string         { return seckillPrefix + ":sequence" }
func makeSeckillReservationSN(unix, sequence int64) string {
	return fmt.Sprintf("SKR%014d%08x", unix, uint32(sequence))
}

func (s *SeckillService) Warmup(ctx context.Context) error {
	activities, err := s.repo.ActiveSeckillActivities(ctx)
	if err != nil {
		return err
	}
	for _, activity := range activities {
		remaining := activity.Stock - activity.SoldCount
		if remaining < 0 {
			remaining = 0
		}
		expireAt := activity.EndTime.Add(7 * 24 * time.Hour)
		pipe := s.repo.Redis.TxPipeline()
		pipe.HSet(ctx, seckillActivityKey(activity.ID), "startAt", activity.StartTime.Unix(), "endAt", activity.EndTime.Unix(), "status", activity.Status)
		pipe.SetNX(ctx, seckillStockKey(activity.ID), remaining, time.Until(expireAt))
		pipe.ExpireAt(ctx, seckillActivityKey(activity.ID), expireAt)
		pipe.ExpireAt(ctx, seckillStockKey(activity.ID), expireAt)
		pipe.ExpireAt(ctx, seckillUsersKey(activity.ID), expireAt)
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *SeckillService) Activities(ctx context.Context) ([]model.SeckillActivity, error) {
	items, err := s.repo.ActiveSeckillActivities(ctx)
	if err != nil {
		return nil, platform.E(platform.CodeDB, "查询秒杀活动失败", err)
	}
	for i := range items {
		remaining, err := s.repo.Redis.Get(ctx, seckillStockKey(items[i].ID)).Int64()
		if err != nil {
			remaining = items[i].Stock - items[i].SoldCount
		}
		if remaining < 0 {
			remaining = 0
		}
		items[i].Remaining = remaining
	}
	return items, nil
}

func (s *SeckillService) Reserve(ctx context.Context, userID, activityID, liveStart, liveEnd, people int64, remark string) (string, error) {
	if activityID <= 0 || people <= 0 || liveEnd <= liveStart || time.Unix(liveEnd, 0).Sub(time.Unix(liveStart, 0)) < 24*time.Hour {
		return "", platform.E(platform.CodeParam, "秒杀参数错误，入住时间至少一晚", nil)
	}
	sequence, err := s.repo.Redis.Incr(ctx, seckillSequenceKey()).Result()
	if err != nil {
		return "", platform.E(platform.CodeCommon, "生成秒杀预约号失败", err)
	}
	reservationSN := makeSeckillReservationSN(time.Now().Unix(), sequence)
	keys := []string{seckillActivityKey(activityID), seckillStockKey(activityID), seckillUsersKey(activityID), seckillResultKey(reservationSN), SeckillStream}
	result, err := reserveSeckillScript.Run(ctx, s.repo.Redis, keys, userID, reservationSN, activityID, liveStart, liveEnd, people, remark).Slice()
	if err != nil {
		return "", platform.E(platform.CodeCommon, "秒杀系统繁忙，请稍后重试", err)
	}
	code, _ := result[0].(int64)
	sn, _ := result[1].(string)
	switch code {
	case 0, 1:
		platform.ObserveSeckillReservation("accepted")
		return sn, nil
	case -1:
		platform.ObserveSeckillReservation("not_started")
		return "", platform.E(platform.CodeCommon, "秒杀尚未开始", nil)
	case -2:
		platform.ObserveSeckillReservation("ended")
		return "", platform.E(platform.CodeCommon, "秒杀已经结束", nil)
	case -3:
		platform.ObserveSeckillReservation("sold_out")
		return "", platform.E(platform.CodeCommon, "秒杀商品已售罄", nil)
	default:
		platform.ObserveSeckillReservation("not_found")
		return "", platform.E(platform.CodeCommon, "秒杀活动不存在", nil)
	}
}

func (s *SeckillService) Result(ctx context.Context, userID int64, reservationSN string) (*model.SeckillResult, error) {
	values, err := s.repo.Redis.HGetAll(ctx, seckillResultKey(reservationSN)).Result()
	if err != nil {
		return nil, platform.E(platform.CodeCommon, "查询秒杀结果失败", err)
	}
	if len(values) == 0 || values["userId"] != strconv.FormatInt(userID, 10) {
		return nil, platform.E(platform.CodeCommon, "秒杀记录不存在", nil)
	}
	return &model.SeckillResult{ReservationSN: reservationSN, Status: values["status"], OrderSN: values["orderSn"], Error: values["error"]}, nil
}

func (s *SeckillService) Process(ctx context.Context, reservation model.SeckillReservation) error {
	activity, err := s.repo.SeckillActivityByID(ctx, reservation.ActivityID)
	if err == sql.ErrNoRows {
		return repository.ErrSeckillSoldOut
	}
	if err != nil {
		platform.ObserveSeckillOrder("attempt_failed")
		return err
	}
	orderSN, restoreStock, err := s.orders.CreateSeckill(ctx, reservation, *activity)
	if err != nil {
		platform.ObserveSeckillOrder("attempt_failed")
		return err
	}
	platform.ObserveSeckillOrder("success")
	restore := "0"
	if restoreStock {
		restore = "1"
	}
	return completeSeckillScript.Run(ctx, s.repo.Redis, []string{seckillResultKey(reservation.ReservationSN), seckillStockKey(reservation.ActivityID)}, restore, orderSN).Err()
}

func (s *SeckillService) IncrementAttempts(ctx context.Context, reservationSN string) int64 {
	n, err := s.repo.Redis.HIncrBy(ctx, seckillResultKey(reservationSN), "attempts", 1).Result()
	if err != nil {
		return 1
	}
	return n
}

func (s *SeckillService) FailAndCompensate(ctx context.Context, reservation model.SeckillReservation, cause error) error {
	message := "创建秒杀订单失败"
	if errors.Is(cause, repository.ErrSeckillSoldOut) {
		message = "秒杀商品已售罄"
	}
	keys := []string{seckillResultKey(reservation.ReservationSN), seckillStockKey(reservation.ActivityID), seckillUsersKey(reservation.ActivityID)}
	return compensateSeckillScript.Run(ctx, s.repo.Redis, keys, reservation.UserID, message).Err()
}
