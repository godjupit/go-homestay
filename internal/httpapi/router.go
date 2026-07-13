package httpapi

import (
	"net/http"

	"gin-looklook/internal/app"
	"gin-looklook/internal/model"
	"gin-looklook/internal/platform"
	"gin-looklook/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type Handler struct{ app *app.App }

func NewRouter(a *app.App) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), Metrics(), otelgin.Middleware("gin-looklook"))
	h := &Handler{app: a}
	r.GET("/healthz", func(c *gin.Context) { OK(c, gin.H{"status": "ok"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	u := r.Group("/usercenter/v1")
	u.POST("/user/register", h.register)
	u.POST("/user/login", h.login)
	ua := u.Group("")
	ua.Use(JWT(a.Config.JWTSecret))
	ua.POST("/user/detail", h.userDetail)
	ua.POST("/user/wxMiniAuth", h.wxMiniAuth)
	t := r.Group("/travel/v1")
	t.POST("/homestay/homestayList", h.homestayList)
	t.POST("/homestay/businessList", h.businessHomestays)
	t.POST("/homestay/guessList", h.guessList)
	t.POST("/homestay/homestayDetail", h.homestayDetail)
	t.POST("/homestayBussiness/goodBoss", h.goodBoss)
	t.POST("/homestayBussiness/homestayBussinessList", h.businessList)
	t.POST("/homestayBussiness/homestayBussinessDetail", h.businessDetail)
	t.POST("/homestayComment/commentList", h.commentList)
	t.POST("/seckill/activityList", h.seckillActivityList)
	t.POST("/search/homestays", h.homestaySearch)
	o := r.Group("/order/v1", JWT(a.Config.JWTSecret))
	o.POST("/homestayOrder/createHomestayOrder", h.createOrder)
	o.POST("/homestayOrder/userHomestayOrderList", h.orderList)
	o.POST("/homestayOrder/userHomestayOrderDetail", h.orderDetail)
	o.POST("/seckill/reserve", h.seckillReserve)
	o.POST("/seckill/result", h.seckillResult)
	p := r.Group("/payment/v1")
	p.POST("/thirdPayment/thirdPaymentWxPayCallback", h.wxPayCallback)
	pa := p.Group("", JWT(a.Config.JWTSecret))
	pa.POST("/thirdPayment/thirdPaymentWxPay", h.wxPay)
	admin := r.Group("/admin/v1")
	admin.POST("/auth/login", h.adminLogin)
	protected := admin.Group("", AdminJWT(a.Config.AdminJWTSecret), AdminAudit(a.Admin))
	protected.POST("/user/list", RequirePermission(a.Admin, "admin:user:list"), h.adminUserList)
	protected.POST("/user/create", RequirePermission(a.Admin, "admin:user:create"), h.adminUserCreate)
	protected.POST("/user/update", RequirePermission(a.Admin, "admin:user:update"), h.adminUserUpdate)
	protected.POST("/user/assignRoles", RequirePermission(a.Admin, "admin:user:assign"), h.adminAssignRoles)
	protected.POST("/role/list", RequirePermission(a.Admin, "admin:role:list"), h.adminRoleList)
	protected.POST("/role/create", RequirePermission(a.Admin, "admin:role:create"), h.adminRoleCreate)
	protected.POST("/role/configure", RequirePermission(a.Admin, "admin:role:configure"), h.adminRoleConfigure)
	protected.POST("/permission/list", RequirePermission(a.Admin, "admin:permission:list"), h.adminPermissionList)
	protected.POST("/permission/create", RequirePermission(a.Admin, "admin:permission:create"), h.adminPermissionCreate)
	protected.POST("/audit/list", RequirePermission(a.Admin, "admin:audit:list"), h.adminAuditList)
	protected.POST("/homestay/list", RequirePermission(a.Admin, "travel:homestay:list"), h.adminHomestayList)
	protected.POST("/homestay/update", RequirePermission(a.Admin, "travel:homestay:update"), h.adminHomestayUpdate)
	protected.POST("/search/rebuild", RequirePermission(a.Admin, "search:index:rebuild"), h.adminSearchRebuild)
	return r
}

func (h *Handler) register(c *gin.Context) {
	var req RegisterReq
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Users.Register(c, req.Mobile, req.Password, "", model.UserAuthTypeSystem, req.Mobile)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, TokenResp{v.AccessToken, v.AccessExpire, v.RefreshAfter})
}
func (h *Handler) login(c *gin.Context) {
	var req LoginReq
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Users.Login(c, req.Mobile, req.Password)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, TokenResp{v.AccessToken, v.AccessExpire, v.RefreshAfter})
}
func (h *Handler) userDetail(c *gin.Context) {
	v, err := h.app.Users.User(c, UserID(c))
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"userInfo": UserView{v.ID, v.Mobile, v.Nickname, v.Sex, v.Avatar, v.Info}})
}
func (h *Handler) wxMiniAuth(c *gin.Context) {
	var req WXMiniAuthReq
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Users.WXMiniAuth(c, req.Code, req.EncryptedData, req.IV)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, TokenResp{v.AccessToken, v.AccessExpire, v.RefreshAfter})
}

func views(items []model.Homestay) []HomestayView {
	out := make([]HomestayView, 0, len(items))
	for _, v := range items {
		out = append(out, homestayView(v))
	}
	return out
}
func (h *Handler) homestayList(c *gin.Context) {
	var req HomestayListReq
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Travel.HomestayList(c, req.Page, req.PageSize)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": views(v)})
}
func (h *Handler) businessHomestays(c *gin.Context) {
	var req BusinessListReq
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Travel.BusinessHomestays(c, req.HomestayBusinessID, req.LastID, req.PageSize)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": views(v)})
}
func (h *Handler) guessList(c *gin.Context) {
	var req map[string]any
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Travel.Guess(c)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": views(v)})
}
func (h *Handler) homestayDetail(c *gin.Context) {
	var req HomestayDetailReq
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Travel.Homestay(c, req.ID)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"homestay": homestayView(*v)})
}
func (h *Handler) businessList(c *gin.Context) {
	var req CursorReq
	if !Bind(c, &req) {
		return
	}
	items, err := h.app.Travel.Businesses(c, req.LastID, req.PageSize)
	if err != nil {
		Fail(c, err)
		return
	}
	out := make([]HomestayBusinessView, 0, len(items))
	for _, v := range items {
		out = append(out, HomestayBusinessView{ID: v.ID, Title: v.Title, Info: v.Info, Tags: v.Tags, Cover: v.Cover, Star: v.Star, HeaderImg: v.HeaderImg})
	}
	OK(c, gin.H{"list": out})
}
func (h *Handler) businessDetail(c *gin.Context) {
	var req HomestayDetailReq
	if !Bind(c, &req) {
		return
	}
	u, err := h.app.Travel.BusinessBoss(c, req.ID)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"boss": BossView{ID: u.ID, UserID: u.ID, Nickname: u.Nickname, Avatar: u.Avatar, Info: u.Info}})
}
func (h *Handler) goodBoss(c *gin.Context) {
	var req map[string]any
	if !Bind(c, &req) {
		return
	}
	items, err := h.app.Travel.GoodBosses(c)
	if err != nil {
		Fail(c, err)
		return
	}
	out := make([]BossView, 0, len(items))
	for i, u := range items {
		out = append(out, BossView{ID: u.ID, UserID: u.ID, Nickname: u.Nickname, Avatar: u.Avatar, Info: u.Info, Rank: int64(i + 1)})
	}
	OK(c, gin.H{"list": out})
}
func (h *Handler) commentList(c *gin.Context) {
	var req CursorReq
	if !Bind(c, &req) {
		return
	}
	items, err := h.app.Travel.Comments(c, req.LastID, req.PageSize)
	if err != nil {
		Fail(c, err)
		return
	}
	out := make([]CommentView, 0, len(items))
	for _, v := range items {
		item := CommentView{ID: v.ID, HomestayID: v.HomestayID, UserID: v.UserID, Content: v.Content, Star: service.ParseStar(v.Star)}
		u, e := h.app.Users.User(c, v.UserID)
		if e == nil {
			item.Nickname = u.Nickname
			item.Avatar = u.Avatar
		}
		out = append(out, item)
	}
	OK(c, gin.H{"list": out})
}

func (h *Handler) seckillActivityList(c *gin.Context) {
	var req map[string]any
	if !Bind(c, &req) {
		return
	}
	items, err := h.app.Seckill.Activities(c)
	if err != nil {
		Fail(c, err)
		return
	}
	out := make([]SeckillActivityView, 0, len(items))
	for _, item := range items {
		out = append(out, SeckillActivityView{ID: item.ID, HomestayID: item.HomestayID, Title: item.Title, Price: platform.FenToYuan(item.Price), Stock: item.Stock, Remaining: item.Remaining, StartTime: item.StartTime.Unix(), EndTime: item.EndTime.Unix()})
	}
	OK(c, gin.H{"list": out})
}

func orderView(v model.HomestayOrder) OrderView {
	return OrderView{SN: v.SN, UserID: v.UserID, HomestayID: v.HomestayID, Title: v.Title, SubTitle: v.SubTitle, Cover: v.Cover, Info: v.Info, FoodInfo: v.FoodInfo, FoodPrice: platform.FenToYuan(v.FoodPrice), HomestayPrice: platform.FenToYuan(v.HomestayPrice), MarketHomestayPrice: platform.FenToYuan(v.MarketHomestayPrice), HomestayBusinessID: v.HomestayBusinessID, HomestayUserID: v.HomestayUserID, OrderTotalPrice: platform.FenToYuan(v.OrderTotalPrice), CreateTime: v.CreateTime.Unix(), TradeState: v.TradeState, LiveStartDate: v.LiveStartDate.Unix(), LiveEndDate: v.LiveEndDate.Unix(), TradeCode: v.TradeCode, FoodTotalPrice: platform.FenToYuan(v.FoodTotalPrice), HomestayTotalPrice: platform.FenToYuan(v.HomestayTotalPrice), Remark: v.Remark, LivePeopleNum: v.LivePeopleNum, NeedFood: v.NeedFood}
}
func (h *Handler) createOrder(c *gin.Context) {
	var req CreateOrderReq
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Orders.Create(c, UserID(c), req.HomestayID, req.IsFood, req.LiveStartTime, req.LiveEndTime, req.LivePeopleNum, req.Remark)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"orderSn": v.SN})
}
func (h *Handler) orderList(c *gin.Context) {
	var req OrderListReq
	if !Bind(c, &req) {
		return
	}
	items, err := h.app.Orders.List(c, UserID(c), req.LastID, req.PageSize, req.TradeState)
	if err != nil {
		Fail(c, err)
		return
	}
	out := make([]OrderListView, 0, len(items))
	for _, v := range items {
		out = append(out, OrderListView{SN: v.SN, Title: v.Title, SubTitle: v.SubTitle, HomestayID: v.HomestayID, Cover: v.Cover, OrderTotalPrice: platform.FenToYuan(v.OrderTotalPrice), CreateTime: v.CreateTime.Unix(), TradeState: v.TradeState, LiveStartDate: v.LiveStartDate.Unix(), LiveEndDate: v.LiveEndDate.Unix(), TradeCode: v.TradeCode})
	}
	OK(c, gin.H{"list": out})
}
func (h *Handler) orderDetail(c *gin.Context) {
	var req OrderDetailReq
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Orders.Detail(c, UserID(c), req.SN)
	if err != nil {
		Fail(c, err)
		return
	}
	out := orderView(*v)
	if payment, e := h.app.Payments.ByOrder(c, v.SN); e == nil && payment != nil {
		out.PayType = payment.PayMode
		if payment.PayTime.Valid {
			out.PayTime = payment.PayTime.Time.Unix()
		}
	}
	OK(c, out)
}

func (h *Handler) seckillReserve(c *gin.Context) {
	var req SeckillReserveReq
	if !Bind(c, &req) {
		return
	}
	reservationSN, err := h.app.Seckill.Reserve(c, UserID(c), req.ActivityID, req.LiveStartTime, req.LiveEndTime, req.LivePeopleNum, req.Remark)
	if err != nil {
		Fail(c, err)
		return
	}
	result, err := h.app.Seckill.Result(c, UserID(c), reservationSN)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, SeckillResultView{ReservationSN: result.ReservationSN, Status: result.Status, OrderSN: result.OrderSN, Error: result.Error})
}

func (h *Handler) seckillResult(c *gin.Context) {
	var req SeckillResultReq
	if !Bind(c, &req) {
		return
	}
	result, err := h.app.Seckill.Result(c, UserID(c), req.ReservationSN)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, SeckillResultView{ReservationSN: result.ReservationSN, Status: result.Status, OrderSN: result.OrderSN, Error: result.Error})
}

func (h *Handler) wxPay(c *gin.Context) {
	var req WxPayReq
	if !Bind(c, &req) {
		return
	}
	v, err := h.app.Payments.Prepay(c, UserID(c), req.OrderSN, req.ServiceType)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, WxPayResp{v.AppID, v.NonceStr, v.PaySign, v.Package, v.Timestamp, v.SignType})
}
func (h *Handler) wxPayCallback(c *gin.Context) {
	if err := h.app.Payments.HandleNotify(c, c.Request); err != nil {
		c.String(http.StatusBadRequest, "FAIL")
		return
	}
	c.String(http.StatusOK, "SUCCESS")
}
