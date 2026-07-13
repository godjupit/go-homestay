package httpapi

import "gin-looklook/internal/model"

type RegisterReq struct {
	Mobile   string `json:"mobile" binding:"required,len=11"`
	Password string `json:"password" binding:"required"`
}
type LoginReq = RegisterReq
type TokenResp struct {
	AccessToken  string `json:"accessToken"`
	AccessExpire int64  `json:"accessExpire"`
	RefreshAfter int64  `json:"refreshAfter"`
}
type WXMiniAuthReq struct {
	Code          string `json:"code" binding:"required"`
	IV            string `json:"iv"`
	EncryptedData string `json:"encryptedData"`
}
type UserView struct {
	ID       int64  `json:"id"`
	Mobile   string `json:"mobile"`
	Nickname string `json:"nickname"`
	Sex      int64  `json:"sex"`
	Avatar   string `json:"avatar"`
	Info     string `json:"info"`
}

type HomestayView struct {
	ID                  int64   `json:"id"`
	Version             int64   `json:"version"`
	Title               string  `json:"title"`
	SubTitle            string  `json:"subTitle"`
	Banner              string  `json:"banner"`
	Info                string  `json:"info"`
	City                string  `json:"city"`
	Tags                string  `json:"tags"`
	Star                float64 `json:"star"`
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
	PeopleNum           int64   `json:"peopleNum"`
	HomestayBusinessID  int64   `json:"homestayBusinessId"`
	UserID              int64   `json:"userId"`
	RowState            int64   `json:"rowState"`
	RowType             int64   `json:"rowType"`
	FoodInfo            string  `json:"foodInfo"`
	FoodPrice           float64 `json:"foodPrice"`
	HomestayPrice       float64 `json:"homestayPrice"`
	MarketHomestayPrice float64 `json:"marketHomestayPrice"`
}
type HomestayListReq struct {
	Page     int64 `json:"page"`
	PageSize int64 `json:"pageSize"`
}
type BusinessListReq struct {
	LastID             int64 `json:"lastId"`
	PageSize           int64 `json:"pageSize"`
	HomestayBusinessID int64 `json:"homestayBusinessId"`
}
type HomestayDetailReq struct {
	ID int64 `json:"id" binding:"required"`
}
type CursorReq struct {
	LastID   int64 `json:"lastId"`
	PageSize int64 `json:"pageSize"`
}
type HomestayBusinessView struct {
	ID            int64   `json:"id"`
	Title         string  `json:"title"`
	Info          string  `json:"info"`
	Tags          string  `json:"tags"`
	Cover         string  `json:"cover"`
	Star          float64 `json:"star"`
	IsFav         int64   `json:"isFav"`
	HeaderImg     string  `json:"headerImg"`
	SellMonth     int64   `json:"sellMonth,omitempty"`
	PersonConsume int64   `json:"personConsume,omitempty"`
}
type BossView struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"userId"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Info     string `json:"info"`
	Rank     int64  `json:"rank"`
}
type CommentView struct {
	ID         int64   `json:"id"`
	HomestayID int64   `json:"homestayId"`
	Content    string  `json:"content"`
	Star       float64 `json:"star"`
	UserID     int64   `json:"userId"`
	Nickname   string  `json:"nickname"`
	Avatar     string  `json:"avatar"`
}

type CreateOrderReq struct {
	HomestayID    int64  `json:"homestayId" binding:"required"`
	IsFood        bool   `json:"isFood"`
	LiveStartTime int64  `json:"liveStartTime" binding:"required"`
	LiveEndTime   int64  `json:"liveEndTime" binding:"required"`
	LivePeopleNum int64  `json:"livePeopleNum" binding:"required"`
	Remark        string `json:"remark"`
}
type OrderListReq struct {
	LastID     int64 `json:"lastId"`
	PageSize   int64 `json:"pageSize"`
	TradeState int64 `json:"tradeState"`
}
type OrderDetailReq struct {
	SN string `json:"sn" binding:"required"`
}
type SeckillReserveReq struct {
	ActivityID    int64  `json:"activityId" binding:"required"`
	LiveStartTime int64  `json:"liveStartTime" binding:"required"`
	LiveEndTime   int64  `json:"liveEndTime" binding:"required"`
	LivePeopleNum int64  `json:"livePeopleNum" binding:"required"`
	Remark        string `json:"remark"`
}
type SeckillResultReq struct {
	ReservationSN string `json:"reservationSn" binding:"required"`
}
type SeckillActivityView struct {
	ID         int64   `json:"id"`
	HomestayID int64   `json:"homestayId"`
	Title      string  `json:"title"`
	Price      float64 `json:"price"`
	Stock      int64   `json:"stock"`
	Remaining  int64   `json:"remaining"`
	StartTime  int64   `json:"startTime"`
	EndTime    int64   `json:"endTime"`
}
type SeckillResultView struct {
	ReservationSN string `json:"reservationSn"`
	Status        string `json:"status"`
	OrderSN       string `json:"orderSn"`
	Error         string `json:"error"`
}
type OrderView struct {
	SN                  string  `json:"sn"`
	UserID              int64   `json:"userId,omitempty"`
	HomestayID          int64   `json:"homestayId"`
	Title               string  `json:"title"`
	SubTitle            string  `json:"subTitle"`
	Cover               string  `json:"cover"`
	Info                string  `json:"info,omitempty"`
	FoodInfo            string  `json:"foodInfo,omitempty"`
	FoodPrice           float64 `json:"foodPrice,omitempty"`
	HomestayPrice       float64 `json:"homestayPrice,omitempty"`
	MarketHomestayPrice float64 `json:"marketHomestayPrice,omitempty"`
	HomestayBusinessID  int64   `json:"homestayBusinessId,omitempty"`
	HomestayUserID      int64   `json:"homestayUserId,omitempty"`
	OrderTotalPrice     float64 `json:"orderTotalPrice"`
	CreateTime          int64   `json:"createTime"`
	TradeState          int64   `json:"tradeState"`
	LiveStartDate       int64   `json:"liveStartDate"`
	LiveEndDate         int64   `json:"liveEndDate"`
	TradeCode           string  `json:"tradeCode"`
	FoodTotalPrice      float64 `json:"foodTotalPrice,omitempty"`
	HomestayTotalPrice  float64 `json:"homestayTotalPrice,omitempty"`
	Remark              string  `json:"remark,omitempty"`
	LivePeopleNum       int64   `json:"livePeopleNum,omitempty"`
	NeedFood            int64   `json:"needFood,omitempty"`
	PayTime             int64   `json:"payTime"`
	PayType             string  `json:"payType"`
}

type OrderListView struct {
	SN              string  `json:"sn"`
	Title           string  `json:"title"`
	SubTitle        string  `json:"subTitle"`
	HomestayID      int64   `json:"homestayId"`
	Cover           string  `json:"cover"`
	OrderTotalPrice float64 `json:"orderTotalPrice"`
	CreateTime      int64   `json:"createTime"`
	TradeState      int64   `json:"tradeState"`
	LiveStartDate   int64   `json:"liveStartDate"`
	LiveEndDate     int64   `json:"liveEndDate"`
	TradeCode       string  `json:"tradeCode"`
}

type WxPayReq struct {
	OrderSN     string `json:"orderSn" binding:"required"`
	ServiceType string `json:"serviceType" binding:"required"`
}
type WxPayResp struct {
	Appid     string `json:"appid"`
	NonceStr  string `json:"nonceStr"`
	PaySign   string `json:"paySign"`
	Package   string `json:"package"`
	Timestamp string `json:"timestamp"`
	SignType  string `json:"signType"`
}

func homestayView(v model.Homestay) HomestayView {
	return HomestayView{ID: v.ID, Version: v.Version, Title: v.Title, SubTitle: v.SubTitle, Banner: v.Banner, Info: v.Info, City: v.City, Tags: v.Tags, Star: v.Star, Latitude: v.Latitude, Longitude: v.Longitude, PeopleNum: v.PeopleNum, HomestayBusinessID: v.HomestayBusinessID, UserID: v.UserID, RowState: v.RowState, RowType: v.RowType, FoodInfo: v.FoodInfo, FoodPrice: float64(v.FoodPrice) / 100, HomestayPrice: float64(v.HomestayPrice) / 100, MarketHomestayPrice: float64(v.MarketHomestayPrice) / 100}
}

type AdminLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type PageReq struct {
	Page     int64 `json:"page"`
	PageSize int64 `json:"pageSize"`
}

type AdminCreateUserReq struct {
	Username     string  `json:"username" binding:"required"`
	Password     string  `json:"password" binding:"required"`
	Nickname     string  `json:"nickname"`
	Status       int64   `json:"status"`
	BusinessID   int64   `json:"businessId"`
	LinkedUserID int64   `json:"linkedUserId"`
	RoleIDs      []int64 `json:"roleIds"`
}

type AdminUpdateUserReq struct {
	ID           int64  `json:"id" binding:"required"`
	Version      int64  `json:"version"`
	Nickname     string `json:"nickname"`
	Status       int64  `json:"status"`
	BusinessID   int64  `json:"businessId"`
	LinkedUserID int64  `json:"linkedUserId"`
	Password     string `json:"password"`
}

type AdminAssignRolesReq struct {
	AdminUserID int64   `json:"adminUserId" binding:"required"`
	RoleIDs     []int64 `json:"roleIds"`
}

type AdminRoleCreateReq struct {
	Code      string `json:"code" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Status    int64  `json:"status"`
	ScopeType int64  `json:"scopeType" binding:"required"`
}

type AdminRoleConfigureReq struct {
	ID            int64   `json:"id" binding:"required"`
	Name          string  `json:"name" binding:"required"`
	Status        int64   `json:"status"`
	ScopeType     int64   `json:"scopeType" binding:"required"`
	Version       int64   `json:"version"`
	PermissionIDs []int64 `json:"permissionIds"`
	BusinessIDs   []int64 `json:"businessIds"`
}

type AdminPermissionCreateReq struct {
	Code   string `json:"code" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Method string `json:"method" binding:"required"`
	Path   string `json:"path" binding:"required"`
}

type AdminAuditListReq struct {
	AdminUserID    int64  `json:"adminUserId"`
	PermissionCode string `json:"permissionCode"`
	StartTime      string `json:"startTime"`
	EndTime        string `json:"endTime"`
	Page           int64  `json:"page"`
	PageSize       int64  `json:"pageSize"`
}

type AdminHomestayUpdateReq struct {
	ID                  int64   `json:"id" binding:"required"`
	Version             int64   `json:"version"`
	Title               string  `json:"title" binding:"required"`
	SubTitle            string  `json:"subTitle"`
	Banner              string  `json:"banner"`
	Info                string  `json:"info"`
	City                string  `json:"city"`
	Tags                string  `json:"tags"`
	Star                float64 `json:"star"`
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
	PeopleNum           int64   `json:"peopleNum"`
	RowState            int64   `json:"rowState"`
	RowType             int64   `json:"rowType"`
	FoodInfo            string  `json:"foodInfo"`
	FoodPrice           float64 `json:"foodPrice"`
	HomestayPrice       float64 `json:"homestayPrice"`
	MarketHomestayPrice float64 `json:"marketHomestayPrice"`
}

type HomestaySearchReq struct {
	Keyword    string   `json:"keyword"`
	City       string   `json:"city"`
	MinPrice   float64  `json:"minPrice"`
	MaxPrice   float64  `json:"maxPrice"`
	Tags       []string `json:"tags"`
	MinStar    float64  `json:"minStar"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	DistanceKM float64  `json:"distanceKm"`
	SortBy     []string `json:"sortBy"`
	Page       int64    `json:"page"`
	PageSize   int64    `json:"pageSize"`
}

type AdminUserView struct {
	ID           int64   `json:"id"`
	Username     string  `json:"username"`
	Nickname     string  `json:"nickname"`
	Status       int64   `json:"status"`
	BusinessID   int64   `json:"businessId"`
	LinkedUserID int64   `json:"linkedUserId"`
	Version      int64   `json:"version"`
	RoleIDs      []int64 `json:"roleIds"`
	CreatedAt    int64   `json:"createdAt"`
	UpdatedAt    int64   `json:"updatedAt"`
}
