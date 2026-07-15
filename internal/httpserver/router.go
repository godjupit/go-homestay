package httpserver

import (
	"gin-looklook/internal/admin"
	"gin-looklook/internal/order"
	"gin-looklook/internal/payment"
	"gin-looklook/internal/search"
	"gin-looklook/internal/seckill"
	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"
	"gin-looklook/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type Handlers struct {
	User    *user.Handler
	Travel  *travel.Handler
	Order   *order.Handler
	Payment *payment.Handler
	Seckill *seckill.Handler
	Search  *search.Handler
	Admin   *admin.Handler
}

func NewRouter(h Handlers, cfg shared.Config, adminSvc *admin.Service) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), RequestID(), Metrics(), otelgin.Middleware("gin-looklook"))
	r.GET("/healthz", func(c *gin.Context) { OK(c, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	jwtMW := JWT(cfg.JWTSecret)
	adminJwtMW := AdminJWT(cfg.AdminJWTSecret)

	user.RegisterRoutes(&r.RouterGroup, h.User, cfg.JWTSecret, LoginRateLimit(), jwtMW)
	travel.RegisterRoutes(&r.RouterGroup, h.Travel)
	search.RegisterRoutes(&r.RouterGroup, h.Search)
	order.RegisterRoutes(&r.RouterGroup, h.Order, jwtMW)
	payment.RegisterRoutes(&r.RouterGroup, h.Payment, jwtMW)
	seckill.RegisterRoutes(&r.RouterGroup, h.Seckill)
	seckill.RegisterOrderRoutes(&r.RouterGroup, h.Seckill, jwtMW)

	// Admin routes
	admin.RegisterRoutes(&r.RouterGroup, h.Admin)
	admGroup := r.Group("/admin/v1", adminJwtMW, AdminAudit(adminSvc))
	admGroup.POST("/user/list", RequirePermission(adminSvc, "admin:user:list"), h.Admin.AdminUserList)
	admGroup.POST("/user/create", RequirePermission(adminSvc, "admin:user:create"), h.Admin.AdminUserCreate)
	admGroup.POST("/user/update", RequirePermission(adminSvc, "admin:user:update"), h.Admin.AdminUserUpdate)
	admGroup.POST("/user/assignRoles", RequirePermission(adminSvc, "admin:user:assign"), h.Admin.AdminAssignRoles)
	admGroup.POST("/role/list", RequirePermission(adminSvc, "admin:role:list"), h.Admin.AdminRoleList)
	admGroup.POST("/role/create", RequirePermission(adminSvc, "admin:role:create"), h.Admin.AdminRoleCreate)
	admGroup.POST("/role/configure", RequirePermission(adminSvc, "admin:role:configure"), h.Admin.AdminRoleConfigure)
	admGroup.POST("/permission/list", RequirePermission(adminSvc, "admin:permission:list"), h.Admin.AdminPermissionList)
	admGroup.POST("/permission/create", RequirePermission(adminSvc, "admin:permission:create"), h.Admin.AdminPermissionCreate)
	admGroup.POST("/audit/list", RequirePermission(adminSvc, "admin:audit:list"), h.Admin.AdminAuditList)
	admGroup.POST("/homestay/list", RequirePermission(adminSvc, "travel:homestay:list"), h.Admin.AdminHomestayList)
	admGroup.POST("/homestay/update", RequirePermission(adminSvc, "travel:homestay:update"), h.Admin.AdminHomestayUpdate)
	admGroup.POST("/search/rebuild", RequirePermission(adminSvc, "search:index:rebuild"), h.Admin.AdminSearchRebuild)

	return r
}
