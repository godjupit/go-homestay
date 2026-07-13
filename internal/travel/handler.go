package travel

import (
	"gin-looklook/internal/shared"
	"gin-looklook/internal/user"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc     *Service
	userSvc *user.Service
}

func NewHandler(svc *Service, userSvc *user.Service) *Handler {
	return &Handler{svc: svc, userSvc: userSvc}
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
func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		fail(c, shared.E(shared.CodeParam, "参数错误, "+err.Error(), err))
		return false
	}
	return true
}

func (h *Handler) HomestayList(c *gin.Context) {
	var req HomestayListReq
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.HomestayList(c, req.Page, req.PageSize)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"list": Views(v)})
}

func (h *Handler) BusinessHomestays(c *gin.Context) {
	var req BusinessListReq
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.BusinessHomestays(c, req.HomestayBusinessID, req.LastID, req.PageSize)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"list": Views(v)})
}

func (h *Handler) GuessList(c *gin.Context) {
	var req map[string]any
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Guess(c)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"list": Views(v)})
}

func (h *Handler) HomestayDetail(c *gin.Context) {
	var req HomestayDetailReq
	if !bind(c, &req) {
		return
	}
	v, err := h.svc.Homestay(c, req.ID)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"homestay": homestayView(*v)})
}

func (h *Handler) BusinessList(c *gin.Context) {
	var req CursorReq
	if !bind(c, &req) {
		return
	}
	items, err := h.svc.Businesses(c, req.LastID, req.PageSize)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]HomestayBusinessView, 0, len(items))
	for _, v := range items {
		out = append(out, HomestayBusinessView{ID: v.ID, Title: v.Title, Info: v.Info, Tags: v.Tags, Cover: v.Cover, Star: v.Star, HeaderImg: v.HeaderImg})
	}
	ok(c, gin.H{"list": out})
}

func (h *Handler) BusinessDetail(c *gin.Context) {
	var req HomestayDetailReq
	if !bind(c, &req) {
		return
	}
	u, err := h.svc.BusinessBoss(c, req.ID)
	if err != nil {
		fail(c, err)
		return
	}
	ok(c, gin.H{"boss": BossView{ID: u.ID, UserID: u.ID, Nickname: u.Nickname, Avatar: u.Avatar, Info: u.Info}})
}

func (h *Handler) GoodBoss(c *gin.Context) {
	var req map[string]any
	if !bind(c, &req) {
		return
	}
	items, err := h.svc.GoodBosses(c)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]BossView, 0, len(items))
	for i, u := range items {
		out = append(out, BossView{ID: u.ID, UserID: u.ID, Nickname: u.Nickname, Avatar: u.Avatar, Info: u.Info, Rank: int64(i + 1)})
	}
	ok(c, gin.H{"list": out})
}

func (h *Handler) CommentList(c *gin.Context) {
	var req CursorReq
	if !bind(c, &req) {
		return
	}
	items, err := h.svc.Comments(c, req.LastID, req.PageSize)
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]CommentView, 0, len(items))
	for _, v := range items {
		item := CommentView{ID: v.ID, HomestayID: v.HomestayID, UserID: v.UserID, Content: v.Content, Star: ParseStar(v.Star)}
		u, e := h.userSvc.User(c, v.UserID)
		if e == nil {
			item.Nickname = u.Nickname
			item.Avatar = u.Avatar
		}
		out = append(out, item)
	}
	ok(c, gin.H{"list": out})
}
