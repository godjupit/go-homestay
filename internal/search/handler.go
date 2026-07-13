package search

import (
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

func (h *Handler) HomestaySearch(c *gin.Context) {
	var req HomestaySearchReq
	if !bind(c, &req) {
		return
	}
	if req.MinPrice < 0 || req.MaxPrice < 0 || (req.MaxPrice > 0 && req.MinPrice > req.MaxPrice) || req.MinStar < 0 || req.MinStar > 5 || req.DistanceKM < 0 {
		fail(c, shared.E(shared.CodeParam, "搜索参数错误", nil))
		return
	}
	result, err := h.svc.Search(c, Query{Keyword: req.Keyword, City: req.City, MinPrice: YuanToFen(req.MinPrice), MaxPrice: YuanToFen(req.MaxPrice), Tags: req.Tags, MinStar: req.MinStar, Latitude: req.Latitude, Longitude: req.Longitude, DistanceKM: req.DistanceKM, SortBy: req.SortBy, Page: req.Page, PageSize: req.PageSize})
	if err != nil {
		fail(c, err)
		return
	}
	out := make([]travel.HomestayView, 0, len(result.Items))
	for _, item := range result.Items {
		out = append(out, travel.HomestayView{ID: item.ID, Version: item.Version, Title: item.Title, SubTitle: item.SubTitle, Banner: item.Banner, Info: item.Info, City: item.City, Tags: item.Tags, Star: item.Star, Latitude: item.Latitude, Longitude: item.Longitude, PeopleNum: item.PeopleNum, HomestayBusinessID: item.HomestayBusinessID, UserID: item.UserID, RowState: item.RowState, RowType: item.RowType, FoodInfo: item.FoodInfo, FoodPrice: float64(item.FoodPrice) / 100, HomestayPrice: float64(item.HomestayPrice) / 100, MarketHomestayPrice: float64(item.MarketHomestayPrice) / 100})
	}
	ok(c, gin.H{"list": out, "total": result.Total})
}
