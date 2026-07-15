package order

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler, jwtMW gin.HandlerFunc) {
	o := r.Group("/order/v1", jwtMW)
	o.POST("/homestayOrder/createHomestayOrder", h.CreateOrder)
	o.POST("/homestayOrder/userHomestayOrderList", h.OrderList)
	o.POST("/homestayOrder/userHomestayOrderDetail", h.OrderDetail)
	o.POST("/homestayOrder/userHomestayOrderCancel", h.OrderCancel)
}
