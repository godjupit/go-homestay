package assistant

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.RouterGroup, handler *Handler, jwtMW, rateLimitMW gin.HandlerFunc) {
	group := r.Group("/agent/v1", jwtMW, rateLimitMW)
	group.POST("/chat", handler.Chat)
}
