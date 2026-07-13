package seckill

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	t := r.Group("/travel/v1")
	t.POST("/seckill/activityList", h.ActivityList)
}

func RegisterOrderRoutes(r *gin.RouterGroup, h *Handler, jwtMW gin.HandlerFunc) {
	o := r.Group("/order/v1", jwtMW)
	o.POST("/seckill/reserve", h.Reserve)
	o.POST("/seckill/result", h.Result)
}
