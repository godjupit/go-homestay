package admin

import (
	"math"
	"time"

	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func ok(c *gin.Context, data any) { c.JSON(200, gin.H{"code": uint32(200), "msg": "OK", "data": data}) }
func fail(c *gin.Context, err error) {
	code, msg := shared.Public(err)
	hs := 400
	if code >= 100005 {
		hs = 500
	}
	c.JSON(hs, gin.H{"code": code, "msg": msg})
}
func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		fail(c, shared.E(shared.CodeParam, "参数错误, "+err.Error(), err))
		return false
	}
	return true
}

func adminID(c *gin.Context) int64 { v, _ := c.Get("adminID"); id, _ := v.(int64); return id }
func adminUsername(c *gin.Context) string {
	v, _ := c.Get("adminUsername")
	s, _ := v.(string)
	return s
}

func yuanToFen(v float64) int64 { return int64(math.Round(v * 100)) }

func (h *Handler) AdminLogin(c *gin.Context) {
	var req AdminLoginReq
	if !bind(c, &req) {
		return
	}
	token, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"accessToken": token.AccessToken, "accessExpire": token.AccessExpire, "refreshAfter": token.RefreshAfter})
}

func (h *Handler) AdminUserList(c *gin.Context) {
	var req PageReq
	if !bind(c, &req) {
		return
	}
	items, total, err := h.svc.Users(c, req.Page, req.PageSize)
	if err != nil {
		fail(c, err)
		return
	}
	views := make([]AdminUserView, 0, len(items))
	for _, item := range items {
		views = append(views, AdminUserView{ID: item.ID, Username: item.Username, Nickname: item.Nickname, Status: item.Status, BusinessID: item.BusinessID, LinkedUserID: item.LinkedUserID, Version: item.Version, RoleIDs: item.RoleIDs, CreatedAt: item.CreatedAt.Unix(), UpdatedAt: item.UpdatedAt.Unix()})
	}
	ok(c, gin.H{"list": views, "total": total})
}

func (h *Handler) AdminUserCreate(c *gin.Context) {
	var req AdminCreateUserReq
	if !bind(c, &req) {
		return
	}
	id, err := h.svc.CreateUser(c, &AdminUser{Username: req.Username, Nickname: req.Nickname, Status: req.Status, BusinessID: req.BusinessID, LinkedUserID: req.LinkedUserID}, req.Password, req.RoleIDs)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"id": id})
}

func (h *Handler) AdminUserUpdate(c *gin.Context) {
	var req AdminUpdateUserReq
	if !bind(c, &req) {
		return
	}
	err := h.svc.UpdateUser(c, &AdminUser{ID: req.ID, Version: req.Version, Nickname: req.Nickname, Status: req.Status, BusinessID: req.BusinessID, LinkedUserID: req.LinkedUserID}, req.Password)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"updated": true})
}

func (h *Handler) AdminAssignRoles(c *gin.Context) {
	var req AdminAssignRolesReq
	if !bind(c, &req) {
		return
	}
	if err := h.svc.AssignRoles(c, req.AdminUserID, req.RoleIDs); err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"assigned": true})
}

func (h *Handler) AdminRoleList(c *gin.Context) {
	var req map[string]any
	if !bind(c, &req) {
		return
	}
	items, err := h.svc.Roles(c)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"list": items})
}

func (h *Handler) AdminRoleCreate(c *gin.Context) {
	var req AdminRoleCreateReq
	if !bind(c, &req) {
		return
	}
	id, err := h.svc.CreateRole(c, &AdminRole{Code: req.Code, Name: req.Name, Status: req.Status, ScopeType: req.ScopeType})
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"id": id})
}

func (h *Handler) AdminRoleConfigure(c *gin.Context) {
	var req AdminRoleConfigureReq
	if !bind(c, &req) {
		return
	}
	err := h.svc.ConfigureRole(c, &AdminRole{ID: req.ID, Name: req.Name, Status: req.Status, ScopeType: req.ScopeType, Version: req.Version, PermissionIDs: req.PermissionIDs, BusinessIDs: req.BusinessIDs})
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"configured": true})
}

func (h *Handler) AdminPermissionList(c *gin.Context) {
	var req map[string]any
	if !bind(c, &req) {
		return
	}
	items, err := h.svc.Permissions(c)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"list": items})
}

func (h *Handler) AdminPermissionCreate(c *gin.Context) {
	var req AdminPermissionCreateReq
	if !bind(c, &req) {
		return
	}
	id, err := h.svc.CreatePermission(c, &AdminPermission{Code: req.Code, Name: req.Name, Method: req.Method, Path: req.Path})
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"id": id})
}

func parseOptionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (h *Handler) AdminAuditList(c *gin.Context) {
	var req AdminAuditListReq
	if !bind(c, &req) {
		return
	}
	start, err := parseOptionalTime(req.StartTime)
	if err != nil {
		fail(c, shared.E(shared.CodeParam, "startTime 必须为 RFC3339 格式", err))
		return
	}
	end, err := parseOptionalTime(req.EndTime)
	if err != nil {
		fail(c, shared.E(shared.CodeParam, "endTime 必须为 RFC3339 格式", err))
		return
	}
	items, total, err := h.svc.Audits(c, req.AdminUserID, req.PermissionCode, start, end, req.Page, req.PageSize)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"list": items, "total": total})
}

func (h *Handler) AdminHomestayList(c *gin.Context) {
	var req PageReq
	if !bind(c, &req) {
		return
	}
	items, total, err := h.svc.Homestays(c, adminID(c), req.Page, req.PageSize)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"list": travel.Views(items), "total": total})
}

func (h *Handler) AdminHomestayUpdate(c *gin.Context) {
	var req AdminHomestayUpdateReq
	if !bind(c, &req) {
		return
	}
	if req.Star < 0 || req.Star > 5 || req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 || req.FoodPrice < 0 || req.HomestayPrice < 0 || req.MarketHomestayPrice < 0 {
		fail(c, shared.E(shared.CodeParam, "评分、坐标或价格参数错误", nil))
		return
	}
	v := &travel.Homestay{ID: req.ID, Version: req.Version, Title: req.Title, SubTitle: req.SubTitle, Banner: req.Banner, Info: req.Info, City: req.City, Tags: req.Tags, Star: req.Star, Latitude: req.Latitude, Longitude: req.Longitude, PeopleNum: req.PeopleNum, RowState: req.RowState, RowType: req.RowType, FoodInfo: req.FoodInfo, FoodPrice: yuanToFen(req.FoodPrice), HomestayPrice: yuanToFen(req.HomestayPrice), MarketHomestayPrice: yuanToFen(req.MarketHomestayPrice)}
	if err := h.svc.UpdateHomestay(c, adminID(c), v); err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"updated": true, "nextVersion": req.Version + 1})
}

func (h *Handler) AdminSearchRebuild(c *gin.Context) {
	var req map[string]any
	if !bind(c, &req) {
		return
	}
	count, err := h.svc.RebuildSearch(c)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"queued": count})
}
