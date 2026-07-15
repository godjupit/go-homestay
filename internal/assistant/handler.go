package assistant

import (
	"log/slog"
	"net/http"

	"gin-looklook/internal/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type chatRequest struct {
	Question string `json:"question" binding:"required,min=1,max=1000"`
}

func (h *Handler) Chat(c *gin.Context) {
	if h.service == nil || !h.service.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": shared.CodeCommon, "msg": "AI assistant is not configured"})
		return
	}
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": shared.CodeParam, "msg": "参数错误, " + err.Error()})
		return
	}
	value, _ := c.Get("userID")
	userID, _ := value.(int64)
	answer, err := h.service.Ask(c.Request.Context(), userID, req.Question)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "AI assistant request failed", "user_id", userID, "error", err)
		code, msg := shared.Public(err)
		c.JSON(http.StatusBadGateway, gin.H{"code": code, "msg": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": uint32(200), "msg": "OK", "data": gin.H{"answer": answer}})
}
