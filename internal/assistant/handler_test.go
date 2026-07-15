package assistant

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestChatReturnsUnavailableWhenAgentIsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/agent/v1/chat", NewHandler(&Service{}).Chat)
	req := httptest.NewRequest(http.MethodPost, "/agent/v1/chat", strings.NewReader(`{"question":"我的订单呢"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d, body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}
}
