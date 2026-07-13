package payment

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler, jwtMW gin.HandlerFunc) {
	p := r.Group("/payment/v1")
	p.POST("/thirdPayment/thirdPaymentWxPayCallback", h.WxPayCallback)
	pa := p.Group("", jwtMW)
	pa.POST("/thirdPayment/thirdPaymentWxPay", h.WxPay)
}
