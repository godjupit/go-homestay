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
	if endUnix <= startUnix {
		return nil, shared.E(shared.CodeCommon, "Stay at least one night", nil)
	}
	h, err := s.travel.Homestay(ctx, homestayID)
	if err != nil {
		return nil, err
	}
	v := buildOrder(*h, userID, h.HomestayPrice, isFood, startUnix, endUnix, people, remark)
	if err := s.repo.CreateOrder(ctx, v); err != nil {
		return nil, shared.E(shared.CodeDB, "Order Database Exception", err)
	}
	s.scheduleClose(ctx, v.SN)
	return v, nil
}

func buildOrder(h travel.Homestay, userID, nightlyPrice int64, isFood bool, startUnix, endUnix, people int64, remark string) *HomestayOrder {
	start, end := time.Unix(startUnix, 0), time.Unix(endUnix, 0)
	days := int64(end.Sub(start).Hours() / 24)
	cover := ""
	if h.Banner != "" {
		cover = strings.Split(h.Banner, ",")[0]
	}
	v := &HomestayOrder{SN: shared.GenSN("HSO"), UserID: userID, HomestayID: h.ID, Title: h.Title, SubTitle: h.SubTitle, Cover: cover, Info: h.Info, PeopleNum: h.PeopleNum, RowType: h.RowType, FoodInfo: h.FoodInfo, FoodPrice: h.FoodPrice, HomestayPrice: nightlyPrice, MarketHomestayPrice: h.MarketHomestayPrice, HomestayBusinessID: h.HomestayBusinessID, HomestayUserID: h.UserID, LiveStartDate: start, LiveEndDate: end, LivePeopleNum: people, TradeState: TradeStateWaitPay, TradeCode: shared.Random(8), Remark: remark, HomestayTotalPrice: nightlyPrice * days}
	if isFood {
		v.NeedFood = NeedFoodYes
		v.FoodTotalPrice = h.FoodPrice * people * days
	}
	v.OrderTotalPrice = v.HomestayTotalPrice + v.FoodTotalPrice
	return v
}

func (s *Service) scheduleClose(ctx context.Context, orderSN string) {
	payload, _ := json.Marshal(CloseOrderPayload{SN: orderSN})
	if _, err := s.asynq.Enqueue(asynq.NewTask(TaskCloseOrder, payload), asynq.ProcessIn(30*time.Minute), asynq.TaskID("close:"+orderSN)); err != nil {
		slog.ErrorContext(ctx, "enqueue close order task", "orderSn", orderSN, "error", err)
	}
}

func (s *Service) CreateSeckill(ctx context.Context, reservationSN string, activityID, userID int64, activity SeckillActivity, liveStart, liveEnd, people int64, remark string) (orderSN string, restoreStock bool, err error) {
	if liveEnd <= liveStart || time.Unix(liveEnd, 0).Sub(time.Unix(liveStart, 0)) < 24*time.Hour {
		return "", false, shared.E(shared.CodeParam, "秒杀入住时间至少一晚", nil)
	}
	h, err := s.travel.Homestay(ctx, activity.HomestayID)
	if err != nil {
		return "", false, err
	}
	v := buildOrder(*h, userID, activity.Price, false, liveStart, liveEnd, people, remark)
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
	switch newState {
	case TradeStateCancel, TradeStateWaitUse:
		return oldState == TradeStateWaitPay
	case TradeStateUsed, TradeStateRefund, TradeStateExpire:
		return oldState == TradeStateWaitUse
	default:
		return false
	}
}

func (s *Service) UpdateState(ctx context.Context, sn string, newState int64) (*HomestayOrder, error) {
	v, err := s.Detail(ctx, 0, sn)
	if err != nil {
		return nil, err
	}
	if v.TradeState == newState {
		return v, nil
	}
	if !verifyState(v.TradeState, newState) {
		return nil, shared.E(shared.CodeCommon, "Changing this status is not supported", nil)
	}
	if err = s.repo.UpdateOrderState(ctx, v.ID, v.Version, newState); err != nil {
		return nil, shared.E(shared.CodeCommon, "Failed to update homestay order status", err)
	}
	v.Version++
	v.TradeState = newState
	if newState == TradeStateWaitUse {
		payload, _ := json.Marshal(NotifyPayload{OrderSN: sn})
		_, _ = s.asynq.Enqueue(asynq.NewTask(TaskPaySuccessNotify, payload))
	}
	return v, nil
}
