package travel

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	t := r.Group("/travel/v1")
	t.POST("/homestay/homestayList", h.HomestayList)
	t.POST("/homestay/businessList", h.BusinessHomestays)
	t.POST("/homestay/guessList", h.GuessList)
	t.POST("/homestay/homestayDetail", h.HomestayDetail)
	t.POST("/homestayBussiness/goodBoss", h.GoodBoss)
	t.POST("/homestayBussiness/homestayBussinessList", h.BusinessList)
	t.POST("/homestayBussiness/homestayBussinessDetail", h.BusinessDetail)
	t.POST("/homestayComment/commentList", h.CommentList)
}
