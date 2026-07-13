package payment

import (
	"net/http"

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

func userID(c *gin.Context) int64 { v, _ := c.Get("userID"); id, _ := v.(int64); return id }

func (h *Handler) WxPay(c *gin.Context) {
	var req WxPayReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, shared.E(shared.CodeParam, "参数错误, "+err.Error(), err))
		return
	}
	v, err := h.svc.Prepay(c, userID(c), req.OrderSN, req.ServiceType)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, WxPayResp{v.AppID, v.NonceStr, v.PaySign, v.Package, v.Timestamp, v.SignType})
}

func (h *Handler) WxPayCallback(c *gin.Context) {
	if err := h.svc.HandleNotify(c, c.Request); err != nil {
		c.String(http.StatusBadRequest, "FAIL")
		return
	}
	c.String(http.StatusOK, "SUCCESS")
}
