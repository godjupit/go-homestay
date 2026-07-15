package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAgentRateLimitPerAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/agent", func(c *gin.Context) {
		c.Set("userID", int64(42))
		c.Next()
	}, AgentRateLimit(), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for i := 0; i < 4; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/agent", nil))
		want := http.StatusNoContent
		if i == 3 {
			want = http.StatusTooManyRequests
		}
		if w.Code != want {
			t.Fatalf("request %d status = %d, want %d", i+1, w.Code, want)
		}
	}
}

func TestAgentRateLimitSeparatesUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/agent", func(c *gin.Context) {
		userID := int64(1)
		if c.Query("user") == "2" {
			userID = 2
		}
		c.Set("userID", userID)
		c.Next()
	}, AgentRateLimit(), func(c *gin.Context) { c.Status(http.StatusNoContent) })

	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/agent?user=1", nil))
		if w.Code != http.StatusNoContent {
			t.Fatalf("user 1 request %d status = %d", i+1, w.Code)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/agent?user=2", nil))
	if w.Code != http.StatusNoContent {
		t.Fatalf("independent user status = %d, want %d", w.Code, http.StatusNoContent)
	}
}
