package httpserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoginRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", LoginRateLimit(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"mobile":"13800138000"}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		want := http.StatusNoContent
		if i == 5 {
			want = http.StatusTooManyRequests
		}
		if w.Code != want {
			t.Fatalf("request %d status = %d, want %d", i+1, w.Code, want)
		}
	}
}

func TestLoginRateLimitSeparatesAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/login", LoginRateLimit(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	for i := 0; i < 6; i++ {
		body := strings.NewReader(fmt.Sprintf(`{"mobile":"1380013%04d"}`, i))
		req := httptest.NewRequest(http.MethodPost, "/login", body)
		req.RemoteAddr = "203.0.113.2:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("distinct account request %d status = %d", i+1, w.Code)
		}
	}
}
