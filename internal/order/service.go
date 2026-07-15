package order

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

type Service struct {
	repo   *Repository
	travel *travel.Service
	asynq  *asynq.Client
}

func NewService(repo *Repository, trvl *travel.Service, client *asynq.Client) *Service {
	return &Service{repo: repo, travel: trvl, asynq: client}
}

func (s *Service) Create(ctx context.Context, userID, homestayID int64, isFood bool, startUnix, endUnix, people int64, remark string) (*HomestayOrder, error) {
	nights, err := stayNights(startUnix, endUnix)
	if err != nil {
		return nil, err
	}
	h, err := s.travel.Homestay(ctx, homestayID)
	if err != nil {
		return nil, err
	}
	v := buildOrder(*h, userID, h.HomestayPrice, isFood, startUnix, endUnix, nights, people, remark)
	if err := s.repo.CreateOrder(ctx, v); err != nil {
		return nil, shared.E(shared.CodeDB, "Order Database Exception", err)
	}
	s.scheduleClose(ctx, v.SN)
	return v, nil
}

func stayNights(startUnix, endUnix int64) (int64, error) {
	// TODO(practice-02): 校验至少住一晚，并返回可用于计费的晚数。
	return 0, shared.E(shared.CodeParam, "TODO(practice-02): implement stay validation", nil)
}

func buildOrder(h travel.Homestay, userID, nightlyPrice int64, isFood bool, startUnix, endUnix, nights, people int64, remark string) *HomestayOrder {
	// TODO(practice-02): 生成订单快照、订单号/核销码，并以“分”计算住宿、餐食和总价。
	return &HomestayOrder{}
}

func (s *Service) scheduleClose(ctx context.Context, orderSN string) {
	payload, _ := json.Marshal(CloseOrderPayload{SN: orderSN})
	if _, err := s.asynq.Enqueue(asynq.NewTask(TaskCloseOrder, payload), asynq.ProcessIn(30*time.Minute), asynq.TaskID("close:"+orderSN)); err != nil {
		slog.ErrorContext(ctx, "enqueue close order task", "orderSn", orderSN, "error", err)
	}
}

func (s *Service) CreateSeckill(ctx context.Context, reservationSN string, activityID, userID int64, activity SeckillActivity, liveStart, liveEnd, people int64, remark string) (orderSN string, restoreStock bool, err error) {
	nights, err := stayNights(liveStart, liveEnd)
	if err != nil {
		return "", false, err
	}
	h, err := s.travel.Homestay(ctx, activity.HomestayID)
	if err != nil {
		return "", false, err
	}
	v := buildOrder(*h, userID, activity.Price, false, liveStart, liveEnd, nights, people, remark)
	v.SN = seckillOrderSN(reservationSN, v.SN)
	orderSN, existed, restoreStock, err := s.repo.CreateSeckillOrder(ctx, reservationSN, activityID, userID, v)
	if err != nil {
		return "", false, shared.E(shared.CodeDB, "创建秒杀订单失败", err)
	}
	if !existed {
		s.scheduleClose(ctx, orderSN)
	}
	return orderSN, restoreStock, nil
}

func seckillOrderSN(reservationSN, fallback string) string {
	if len(reservationSN) == 25 && strings.HasPrefix(reservationSN, "SKR") {
		return "HSO" + reservationSN[3:]
	}
	return fallback
}

func (s *Service) Detail(ctx context.Context, userID int64, sn string) (*HomestayOrder, error) {
	v, err := s.repo.OrderBySN(ctx, sn)
	if err == gorm.ErrRecordNotFound {
		return nil, shared.E(shared.CodeCommon, "order no exists", nil)
	}
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	if userID > 0 && v.UserID != userID {
		return nil, shared.E(shared.CodeCommon, "order no exists", nil)
	}
	return v, nil
}

func (s *Service) List(ctx context.Context, userID, lastID, pageSize, state int64) ([]HomestayOrder, error) {
	v, err := s.repo.OrdersByUser(ctx, userID, lastID, pageSize, state)
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}

func verifyState(oldState, newState int64) bool {
	// TODO(practice-03): 实现订单状态机，只允许题目文档中定义的迁移。
	return false
}

func (s *Service) UpdateState(ctx context.Context, sn string, newState int64, userID int64) (*HomestayOrder, error) {
	// userID == 0 means admin operation, otherwise user operation

	v, err := s.Detail(ctx, userID, sn)
	if err != nil {
		return nil, err
	}
	if v.TradeState == newState {
		return v, nil
	}
	oldState := v.TradeState
	if !verifyState(v.TradeState, newState) {
		shared.ObserveOrderTransition(oldState, newState, "rejected")
		return nil, shared.E(shared.CodeCommon, "Changing this status is not supported", nil)
	}
	if err = s.repo.UpdateOrderState(ctx, v.ID, v.Version, newState); err != nil {
		shared.ObserveOrderTransition(oldState, newState, "failed")
		return nil, shared.E(shared.CodeCommon, "Failed to update homestay order status", err)
	}
	shared.ObserveOrderTransition(oldState, newState, "success")
	v.Version++
	v.TradeState = newState
	if newState == TradeStateWaitUse {
		payload, _ := json.Marshal(NotifyPayload{OrderSN: sn})
		_, _ = s.asynq.Enqueue(asynq.NewTask(TaskPaySuccessNotify, payload))
	}
	return v, nil
}

func (s *Service) OrderCancel(ctx context.Context, userID int64, sn string) (*HomestayOrder, error) {
	return s.UpdateState(ctx, sn, TradeStateCancel, userID)
}
