package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"gin-looklook/internal/model"
	"gin-looklook/internal/platform"
	"gin-looklook/internal/repository"

	"github.com/hibiken/asynq"
)

const (
	TaskCloseOrder       = "defer:homestay_order:close"
	TaskPaySuccessNotify = "msg:pay_success:notify_user"
	TaskSettle           = "schedule:settle_record:settle"
)

type CloseOrderPayload struct {
	SN string `json:"sn"`
}
type NotifyPayload struct {
	OrderSN string `json:"orderSn"`
}
type OrderService struct {
	repo   *repository.Repository
	travel *TravelService
	asynq  *asynq.Client
}

func NewOrderService(repo *repository.Repository, travel *TravelService, client *asynq.Client) *OrderService {
	return &OrderService{repo: repo, travel: travel, asynq: client}
}
func (s *OrderService) Create(ctx context.Context, userID, homestayID int64, isFood bool, startUnix, endUnix, people int64, remark string) (*model.HomestayOrder, error) {
	if endUnix <= startUnix {
		return nil, platform.E(platform.CodeCommon, "Stay at least one night", nil)
	}
	h, err := s.travel.Homestay(ctx, homestayID)
	if err != nil {
		return nil, err
	}
	v := buildOrder(*h, userID, h.HomestayPrice, isFood, startUnix, endUnix, people, remark)
	if err := s.repo.CreateOrder(ctx, v); err != nil {
		return nil, platform.E(platform.CodeDB, "Order Database Exception", err)
	}
	s.scheduleClose(ctx, v.SN)
	return v, nil
}

func buildOrder(h model.Homestay, userID, nightlyPrice int64, isFood bool, startUnix, endUnix, people int64, remark string) *model.HomestayOrder {
	start, end := time.Unix(startUnix, 0), time.Unix(endUnix, 0)
	days := int64(end.Sub(start).Hours() / 24)
	cover := ""
	if h.Banner != "" {
		cover = strings.Split(h.Banner, ",")[0]
	}
	v := &model.HomestayOrder{SN: platform.GenSN("HSO"), UserID: userID, HomestayID: h.ID, Title: h.Title, SubTitle: h.SubTitle, Cover: cover, Info: h.Info, PeopleNum: h.PeopleNum, RowType: h.RowType, FoodInfo: h.FoodInfo, FoodPrice: h.FoodPrice, HomestayPrice: nightlyPrice, MarketHomestayPrice: h.MarketHomestayPrice, HomestayBusinessID: h.HomestayBusinessID, HomestayUserID: h.UserID, LiveStartDate: start, LiveEndDate: end, LivePeopleNum: people, TradeState: model.OrderTradeStateWaitPay, TradeCode: platform.Random(8), Remark: remark, HomestayTotalPrice: nightlyPrice * days}
	if isFood {
		v.NeedFood = model.HomestayOrderNeedFoodYes
		v.FoodTotalPrice = h.FoodPrice * people * days
	}
	v.OrderTotalPrice = v.HomestayTotalPrice + v.FoodTotalPrice
	return v
}

func (s *OrderService) scheduleClose(ctx context.Context, orderSN string) {
	payload, _ := json.Marshal(CloseOrderPayload{SN: orderSN})
	if _, err := s.asynq.Enqueue(asynq.NewTask(TaskCloseOrder, payload), asynq.ProcessIn(30*time.Minute), asynq.TaskID("close:"+orderSN)); err != nil {
		slog.ErrorContext(ctx, "enqueue close order task", "orderSn", orderSN, "error", err)
	}
}

func (s *OrderService) CreateSeckill(ctx context.Context, reservation model.SeckillReservation, activity model.SeckillActivity) (string, bool, error) {
	if reservation.LiveEndTime <= reservation.LiveStartTime || time.Unix(reservation.LiveEndTime, 0).Sub(time.Unix(reservation.LiveStartTime, 0)) < 24*time.Hour {
		return "", false, platform.E(platform.CodeParam, "秒杀入住时间至少一晚", nil)
	}
	h, err := s.travel.Homestay(ctx, activity.HomestayID)
	if err != nil {
		return "", false, err
	}
	v := buildOrder(*h, reservation.UserID, activity.Price, false, reservation.LiveStartTime, reservation.LiveEndTime, reservation.LivePeopleNum, reservation.Remark)
	v.SN = seckillOrderSN(reservation.ReservationSN, v.SN)
	orderSN, existed, restoreStock, err := s.repo.CreateSeckillOrder(ctx, reservation, v)
	if err != nil {
		return "", false, platform.E(platform.CodeDB, "创建秒杀订单失败", err)
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
func (s *OrderService) Detail(ctx context.Context, userID int64, sn string) (*model.HomestayOrder, error) {
	v, err := s.repo.OrderBySN(ctx, sn)
	if err == sql.ErrNoRows {
		return nil, platform.E(platform.CodeCommon, "order no exists", nil)
	}
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	if userID > 0 && v.UserID != userID {
		return nil, platform.E(platform.CodeCommon, "order no exists", nil)
	}
	return v, nil
}
func (s *OrderService) List(ctx context.Context, userID, lastID, pageSize, state int64) ([]model.HomestayOrder, error) {
	v, err := s.repo.OrdersByUser(ctx, userID, lastID, pageSize, state)
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}
func verifyState(oldState, newState int64) bool {
	switch newState {
	case model.OrderTradeStateCancel, model.OrderTradeStateWaitUse:
		return oldState == model.OrderTradeStateWaitPay
	case model.OrderTradeStateUsed, model.OrderTradeStateRefund, model.OrderTradeStateExpire:
		return oldState == model.OrderTradeStateWaitUse
	default:
		return false
	}
}
func (s *OrderService) UpdateState(ctx context.Context, sn string, newState int64) (*model.HomestayOrder, error) {
	v, err := s.Detail(ctx, 0, sn)
	if err != nil {
		return nil, err
	}
	if v.TradeState == newState {
		return v, nil
	}
	if !verifyState(v.TradeState, newState) {
		return nil, platform.E(platform.CodeCommon, "Changing this status is not supported", nil)
	}
	if err = s.repo.UpdateOrderState(ctx, v.ID, v.Version, newState); err != nil {
		return nil, platform.E(platform.CodeCommon, "Failed to update homestay order status", err)
	}
	v.Version++
	v.TradeState = newState
	if newState == model.OrderTradeStateWaitUse {
		payload, _ := json.Marshal(NotifyPayload{OrderSN: sn})
		_, _ = s.asynq.Enqueue(asynq.NewTask(TaskPaySuccessNotify, payload))
	}
	return v, nil
}
