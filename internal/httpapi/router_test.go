package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gin-looklook/internal/app"
	"gin-looklook/internal/config"

	"github.com/gin-gonic/gin"
)

func TestRouterContainsAllCompatibleRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(&app.App{Config: config.Config{JWTSecret: "test", AdminJWTSecret: "admin-test"}})
	if got := len(router.Routes()); got != 37 { // original compatible routes plus search and RBAC admin APIs
		t.Fatalf("route count=%d, want 37", got)
	}
	routes := make(map[string]bool)
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	for _, expected := range []string{
		"POST /travel/v1/search/homestays",
		"POST /admin/v1/auth/login",
		"POST /admin/v1/user/list",
		"POST /admin/v1/role/configure",
		"POST /admin/v1/homestay/update",
		"POST /admin/v1/search/rebuild",
	} {
		if !routes[expected] {
			t.Fatalf("missing route %s", expected)
		}
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/order/v1/homestayOrder/userHomestayOrderList", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), `"code":100003`) {
		t.Fatalf("unexpected auth response: status=%d body=%s", w.Code, w.Body.String())
	}
}
