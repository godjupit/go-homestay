package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	adm := r.Group("/admin/v1")
	adm.POST("/auth/login", h.AdminLogin)
}
