package seckill

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

func (h *Handler) ActivityList(c *gin.Context) {
	var req map[string]any
	if !bind(c, &req) {
		return
	}
	items, err := h.svc.Activities(c)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]ActivityView, 0, len(items))
	for _, item := range items {
		out = append(out, ActivityView{ID: item.ID, HomestayID: item.HomestayID, Title: item.Title, Price: shared.FenToYuan(item.Price), Stock: item.Stock, Remaining: item.Remaining, StartTime: item.StartTime, EndTime: item.EndTime})
	}
	ok(c, gin.H{"list": out})
}

func (h *Handler) Reserve(c *gin.Context) {
	var req ReserveReq
	if !bind(c, &req) {
		return
	}
	reservationSN, err := h.svc.Reserve(c, userID(c), req.ActivityID, req.LiveStartTime, req.LiveEndTime, req.LivePeopleNum, req.Remark)
	if err != nil {
		fail(c, err)
		return
	}
	result, err := h.svc.Result(c, userID(c), reservationSN)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, ResultView{ReservationSN: result.ReservationSN, Status: result.Status, OrderSN: result.OrderSN, Error: result.Error})
}

func (h *Handler) Result(c *gin.Context) {
	var req ResultReq
	if !bind(c, &req) {
		return
	}
	result, err := h.svc.Result(c, userID(c), req.ReservationSN)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, ResultView{ReservationSN: result.ReservationSN, Status: result.Status, OrderSN: result.OrderSN, Error: result.Error})
}
