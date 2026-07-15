package order

import (
	"gin-looklook/internal/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func ok(c *gin.Context, data any) { c.JSON(200, gin.H{"code": uint32(200), "msg": "OK", "data": data}) }
func fail(c *gin.Context, err error) {
	code, msg := shared.Public(err)
	hs := 400
	if code >= 100005 {
		hs = 500
	}
	c.JSON(hs, gin.H{"code": code, "msg": msg})
}
func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		fail(c, shared.E(shared.CodeParam, "参数错误, "+err.Error(), err))
		return false
	}
	return true
}

func userID(c *gin.Context) int64 { v, _ := c.Get("userID"); id, _ := v.(int64); return id }

func orderView(v HomestayOrder) OrderView {
	return OrderView{SN: v.SN, UserID: v.UserID, HomestayID: v.HomestayID, Title: v.Title, SubTitle: v.SubTitle, Cover: v.Cover, Info: v.Info, FoodInfo: v.FoodInfo, FoodPrice: shared.FenToYuan(v.FoodPrice), HomestayPrice: shared.FenToYuan(v.HomestayPrice), MarketHomestayPrice: shared.FenToYuan(v.MarketHomestayPrice), HomestayBusinessID: v.HomestayBusinessID, HomestayUserID: v.HomestayUserID, OrderTotalPrice: shared.FenToYuan(v.OrderTotalPrice), CreateTime: v.CreateTime.Unix(), TradeState: v.TradeState, LiveStartDate: v.LiveStartDate.Unix(), LiveEndDate: v.LiveEndDate.Unix(), TradeCode: v.TradeCode, FoodTotalPrice: shared.FenToYuan(v.FoodTotalPrice), HomestayTotalPrice: shared.FenToYuan(v.HomestayTotalPrice), Remark: v.Remark, LivePeopleNum: v.LivePeopleNum, NeedFood: v.NeedFood}
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req CreateOrderReq
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Create(c, userID(c), req.HomestayID, req.IsFood, req.LiveStartTime, req.LiveEndTime, req.LivePeopleNum, req.Remark)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"orderSn": v.SN})
}

func (h *Handler) OrderList(c *gin.Context) {
	var req OrderListReq
	if !bind(c, &req) {
		return
	}
	items, err := h.svc.List(c, userID(c), req.LastID, req.PageSize, req.TradeState)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]OrderListView, 0, len(items))
	for _, v := range items {
		out = append(out, OrderListView{SN: v.SN, Title: v.Title, SubTitle: v.SubTitle, HomestayID: v.HomestayID, Cover: v.Cover, OrderTotalPrice: shared.FenToYuan(v.OrderTotalPrice), CreateTime: v.CreateTime.Unix(), TradeState: v.TradeState, LiveStartDate: v.LiveStartDate.Unix(), LiveEndDate: v.LiveEndDate.Unix(), TradeCode: v.TradeCode})
	}
	ok(c, gin.H{"list": out})
}

func (h *Handler) OrderDetail(c *gin.Context) {
	var req OrderDetailReq
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Detail(c, userID(c), req.SN)
	if err != nil {
		fail(c, err)
		return
	}
	out := orderView(*v)
	// Payment inline fetch kept from original handler
	ok(c, out)
}

func (h *Handler) OrderCancel(c *gin.Context) {
	var req OrderCancelReq
	if !bind(c, &req) {
		return
	}

	v, err := h.svc.OrderCancel(c, userID(c), req.SN)

	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"tradeState": v.TradeState})
}
