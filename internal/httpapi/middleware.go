package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gin-looklook/internal/model"
	"gin-looklook/internal/platform"
	"gin-looklook/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/prometheus/client_golang/prometheus"
)

type successResponse struct {
	Code uint32 `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}
type errorResponse struct {
	Code uint32 `json:"code"`
	Msg  string `json:"msg"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, successResponse{Code: 200, Msg: "OK", Data: data})
}
func Fail(c *gin.Context, err error) {
	code, msg := platform.Public(err)
	c.Set("auditError", msg)
	slog.ErrorContext(c.Request.Context(), "request failed", "path", c.Request.URL.Path, "error", err)
	c.JSON(http.StatusBadRequest, errorResponse{Code: code, Msg: msg})
}
func Bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		Fail(c, platform.E(platform.CodeParam, "参数错误, "+err.Error(), err))
		return false
	}
	return true
}

func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if len(raw) > 7 && raw[:7] == "Bearer " {
			raw = raw[7:]
		}
		token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Code: platform.CodeToken, Msg: "token失效，请重新登陆"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Code: platform.CodeToken, Msg: "token失效，请重新登陆"})
			return
		}
		id, ok := claims["jwtUserId"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Code: platform.CodeToken, Msg: "token失效，请重新登陆"})
			return
		}
		c.Set("userID", int64(id))
		c.Next()
	}
}
func UserID(c *gin.Context) int64 { v, _ := c.Get("userID"); id, _ := v.(int64); return id }

func AdminJWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		token, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		claims, ok := tokenClaims(token)
		id, idOK := claims["adminId"].(float64)
		username, _ := claims["username"].(string)
		tokenType, _ := claims["tokenType"].(string)
		if err != nil || !ok || !idOK || id <= 0 || tokenType != "admin" {
			c.Set("auditError", "管理员 token 失效")
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse{Code: platform.CodeToken, Msg: "token失效，请重新登陆"})
			return
		}
		c.Set("adminID", int64(id))
		c.Set("adminUsername", username)
		c.Next()
	}
}

func tokenClaims(token *jwt.Token) (jwt.MapClaims, bool) {
	if token == nil || !token.Valid {
		return nil, false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	return claims, ok
}

func AdminID(c *gin.Context) int64 {
	v, _ := c.Get("adminID")
	id, _ := v.(int64)
	return id
}

func RequirePermission(admin *service.AdminService, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("permissionCode", code)
		auth, err := admin.Authorization(c.Request.Context(), AdminID(c))
		if err != nil {
			Fail(c, err)
			c.Abort()
			return
		}
		if _, ok := auth.Permissions[code]; !ok {
			c.Set("auditError", "无此操作权限")
			c.AbortWithStatusJSON(http.StatusForbidden, errorResponse{Code: platform.CodeForbidden, Msg: "无此操作权限"})
			return
		}
		c.Next()
	}
}

func AdminAudit(admin *service.AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = platform.Random(16)
		}
		c.Header("X-Request-ID", requestID)
		body := readAuditBody(c)
		c.Next()
		permission, _ := c.Get("permissionCode")
		username, _ := c.Get("adminUsername")
		errorMessage, _ := c.Get("auditError")
		audit := &model.AdminAudit{AdminUserID: AdminID(c), Username: fmtString(username), PermissionCode: fmtString(permission), Method: c.Request.Method, Path: c.Request.URL.Path, RequestID: requestID, IP: c.ClientIP(), HTTPStatus: c.Writer.Status(), Success: c.Writer.Status() < 400 && errorMessage == nil, DurationMS: time.Since(start).Milliseconds(), RequestBody: body, ErrorMessage: fmtString(errorMessage)}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := admin.SaveAudit(ctx, audit); err != nil {
				slog.Error("save admin audit", "error", err)
			}
		}()
	}
}

func fmtString(v any) string {
	value, _ := v.(string)
	return value
}

func readAuditBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	data, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(data))
	auditData := data
	if len(auditData) > 16<<10 {
		auditData = auditData[:16<<10]
	}
	var value any
	if json.Unmarshal(auditData, &value) == nil {
		redactSecrets(value)
		if sanitized, err := json.Marshal(value); err == nil {
			return string(sanitized)
		}
	}
	return string(auditData)
}

func redactSecrets(value any) {
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
				current[key] = "***"
				continue
			}
			redactSecrets(child)
		}
	case []any:
		for _, child := range current {
			redactSecrets(child)
		}
	}
}

var requestCount = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "gin_looklook_http_requests_total", Help: "HTTP requests"}, []string{"method", "path", "status"})
var requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "gin_looklook_http_request_duration_seconds", Help: "HTTP latency", Buckets: prometheus.DefBuckets}, []string{"method", "path"})

func init() { prometheus.MustRegister(requestCount, requestDuration) }
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		requestCount.WithLabelValues(c.Request.Method, path, strconv.Itoa(c.Writer.Status())).Inc()
		requestDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
