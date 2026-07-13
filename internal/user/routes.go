package user

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, h *Handler, jwtSecret string, jwtMW gin.HandlerFunc) {
	u := r.Group("/usercenter/v1")
	u.POST("/user/register", h.Register)
	u.POST("/user/login", h.Login)
	ua := u.Group("")
	ua.Use(jwtMW)
	ua.POST("/user/detail", h.UserDetail)
	ua.POST("/user/wxMiniAuth", h.WXMiniAuth)
	ua.POST("/user/profile", h.UpdateProfile)
}
