package user

import (
	"gin-looklook/internal/shared"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		fail(c, shared.E(shared.CodeParam, "参数错误, "+err.Error(), err))
		return false
	}
	return true
}
func ok(c *gin.Context, data any) { c.JSON(200, gin.H{"code": uint32(200), "msg": "OK", "data": data}) }
func fail(c *gin.Context, err error) {
	code, msg := shared.Public(err)
	httpStatus := 400
	if code >= 100005 {
		httpStatus = 500
	}
	c.JSON(httpStatus, gin.H{"code": code, "msg": msg})
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterReq
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Register(c, req.Mobile, req.Password, req.Nickname, AuthTypeSystem, req.Mobile)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, TokenResp{v.AccessToken, v.AccessExpire, v.RefreshAfter})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Login(c, req.Mobile, req.Password)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, TokenResp{v.AccessToken, v.AccessExpire, v.RefreshAfter})
}

func (h *Handler) UserDetail(c *gin.Context) {
	v, err := h.svc.User(c, userID(c))
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"userInfo": UserView{v.ID, v.Mobile, v.Nickname, v.Sex, v.Avatar, v.Info}})
}

func (h *Handler) WXMiniAuth(c *gin.Context) {
	var req WXMiniAuthReq
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.WXMiniAuth(c, req.Code, req.EncryptedData, req.IV)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, TokenResp{v.AccessToken, v.AccessExpire, v.RefreshAfter})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileReq

	if !bind(c, &req) {
		return
	}

	err := h.svc.UpdateProfile(c, userID(c), req.Nickname, req.Sex, req.Avatar, req.Info)

	if err != nil {
		fail(c, err)
		return
	}

	ok(c, gin.H{"msg": "Profile updated successfully"})

}

func userID(c *gin.Context) int64 { v, _ := c.Get("userID"); id, _ := v.(int64); return id }
