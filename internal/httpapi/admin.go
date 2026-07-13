package httpapi

import (
	"math"
	"time"

	"gin-looklook/internal/model"
	"gin-looklook/internal/platform"

	"github.com/gin-gonic/gin"
)

func (h *Handler) adminLogin(c *gin.Context) {
	var req AdminLoginReq
	if !Bind(c, &req) {
		return
	}
	token, err := h.app.Admin.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, TokenResp{AccessToken: token.AccessToken, AccessExpire: token.AccessExpire, RefreshAfter: token.RefreshAfter})
}

func (h *Handler) adminUserList(c *gin.Context) {
	var req PageReq
	if !Bind(c, &req) {
		return
	}
	items, total, err := h.app.Admin.Users(c, req.Page, req.PageSize)
	if err != nil {
		Fail(c, err)
		return
	}
	views := make([]AdminUserView, 0, len(items))
	for _, item := range items {
		views = append(views, AdminUserView{ID: item.ID, Username: item.Username, Nickname: item.Nickname, Status: item.Status, BusinessID: item.BusinessID, LinkedUserID: item.LinkedUserID, Version: item.Version, RoleIDs: item.RoleIDs, CreatedAt: item.CreatedAt.Unix(), UpdatedAt: item.UpdatedAt.Unix()})
	}
	OK(c, gin.H{"list": views, "total": total})
}

func (h *Handler) adminUserCreate(c *gin.Context) {
	var req AdminCreateUserReq
	if !Bind(c, &req) {
		return
	}
	id, err := h.app.Admin.CreateUser(c, &model.AdminUser{Username: req.Username, Nickname: req.Nickname, Status: req.Status, BusinessID: req.BusinessID, LinkedUserID: req.LinkedUserID}, req.Password, req.RoleIDs)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"id": id})
}

func (h *Handler) adminUserUpdate(c *gin.Context) {
	var req AdminUpdateUserReq
	if !Bind(c, &req) {
		return
	}
	err := h.app.Admin.UpdateUser(c, &model.AdminUser{ID: req.ID, Version: req.Version, Nickname: req.Nickname, Status: req.Status, BusinessID: req.BusinessID, LinkedUserID: req.LinkedUserID}, req.Password)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"updated": true})
}

func (h *Handler) adminAssignRoles(c *gin.Context) {
	var req AdminAssignRolesReq
	if !Bind(c, &req) {
		return
	}
	if err := h.app.Admin.AssignRoles(c, req.AdminUserID, req.RoleIDs); err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"assigned": true})
}

func (h *Handler) adminRoleList(c *gin.Context) {
	var req map[string]any
	if !Bind(c, &req) {
		return
	}
	items, err := h.app.Admin.Roles(c)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": items})
}

func (h *Handler) adminRoleCreate(c *gin.Context) {
	var req AdminRoleCreateReq
	if !Bind(c, &req) {
		return
	}
	id, err := h.app.Admin.CreateRole(c, &model.AdminRole{Code: req.Code, Name: req.Name, Status: req.Status, ScopeType: req.ScopeType})
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"id": id})
}

func (h *Handler) adminRoleConfigure(c *gin.Context) {
	var req AdminRoleConfigureReq
	if !Bind(c, &req) {
		return
	}
	err := h.app.Admin.ConfigureRole(c, &model.AdminRole{ID: req.ID, Name: req.Name, Status: req.Status, ScopeType: req.ScopeType, Version: req.Version, PermissionIDs: req.PermissionIDs, BusinessIDs: req.BusinessIDs})
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"configured": true})
}

func (h *Handler) adminPermissionList(c *gin.Context) {
	var req map[string]any
	if !Bind(c, &req) {
		return
	}
	items, err := h.app.Admin.Permissions(c)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": items})
}

func (h *Handler) adminPermissionCreate(c *gin.Context) {
	var req AdminPermissionCreateReq
	if !Bind(c, &req) {
		return
	}
	id, err := h.app.Admin.CreatePermission(c, &model.AdminPermission{Code: req.Code, Name: req.Name, Method: req.Method, Path: req.Path})
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"id": id})
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

func (h *Handler) adminAuditList(c *gin.Context) {
	var req AdminAuditListReq
	if !Bind(c, &req) {
		return
	}
	start, err := parseOptionalTime(req.StartTime)
	if err != nil {
		Fail(c, platform.E(platform.CodeParam, "startTime 必须为 RFC3339 格式", err))
		return
	}
	end, err := parseOptionalTime(req.EndTime)
	if err != nil {
		Fail(c, platform.E(platform.CodeParam, "endTime 必须为 RFC3339 格式", err))
		return
	}
	items, total, err := h.app.Admin.Audits(c, req.AdminUserID, req.PermissionCode, start, end, req.Page, req.PageSize)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": items, "total": total})
}

func (h *Handler) adminHomestayList(c *gin.Context) {
	var req PageReq
	if !Bind(c, &req) {
		return
	}
	items, total, err := h.app.Admin.Homestays(c, AdminID(c), req.Page, req.PageSize)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": views(items), "total": total})
}

func yuanToFen(value float64) int64 { return int64(math.Round(value * 100)) }

func (h *Handler) adminHomestayUpdate(c *gin.Context) {
	var req AdminHomestayUpdateReq
	if !Bind(c, &req) {
		return
	}
	if req.Star < 0 || req.Star > 5 || req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 || req.FoodPrice < 0 || req.HomestayPrice < 0 || req.MarketHomestayPrice < 0 {
		Fail(c, platform.E(platform.CodeParam, "评分、坐标或价格参数错误", nil))
		return
	}
	v := &model.Homestay{ID: req.ID, Version: req.Version, Title: req.Title, SubTitle: req.SubTitle, Banner: req.Banner, Info: req.Info, City: req.City, Tags: req.Tags, Star: req.Star, Latitude: req.Latitude, Longitude: req.Longitude, PeopleNum: req.PeopleNum, RowState: req.RowState, RowType: req.RowType, FoodInfo: req.FoodInfo, FoodPrice: yuanToFen(req.FoodPrice), HomestayPrice: yuanToFen(req.HomestayPrice), MarketHomestayPrice: yuanToFen(req.MarketHomestayPrice)}
	if err := h.app.Admin.UpdateHomestay(c, AdminID(c), v); err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"updated": true, "nextVersion": req.Version + 1})
}

func (h *Handler) adminSearchRebuild(c *gin.Context) {
	var req map[string]any
	if !Bind(c, &req) {
		return
	}
	count, err := h.app.Admin.RebuildSearch(c)
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"queued": count})
}

func (h *Handler) homestaySearch(c *gin.Context) {
	var req HomestaySearchReq
	if !Bind(c, &req) {
		return
	}
	if req.MinPrice < 0 || req.MaxPrice < 0 || (req.MaxPrice > 0 && req.MinPrice > req.MaxPrice) || req.MinStar < 0 || req.MinStar > 5 || req.DistanceKM < 0 {
		Fail(c, platform.E(platform.CodeParam, "搜索参数错误", nil))
		return
	}
	result, err := h.app.Search.Search(c, model.HomestaySearchQuery{Keyword: req.Keyword, City: req.City, MinPrice: yuanToFen(req.MinPrice), MaxPrice: yuanToFen(req.MaxPrice), Tags: req.Tags, MinStar: req.MinStar, Latitude: req.Latitude, Longitude: req.Longitude, DistanceKM: req.DistanceKM, SortBy: req.SortBy, Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		Fail(c, err)
		return
	}
	OK(c, gin.H{"list": views(result.Items), "total": result.Total})
}
