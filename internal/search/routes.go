package search

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	t := r.Group("/travel/v1")
	t.POST("/search/homestays", h.HomestaySearch)
}
