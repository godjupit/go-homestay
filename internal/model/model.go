package model

import (
	"database/sql"
	"time"
)

const (
	DelStateNo  int64 = 0
	DelStateYes int64 = 1

	UserAuthTypeSystem  = "system"
	UserAuthTypeSmallWX = "wxMini"

	HomestayOrderNeedFoodNo  int64 = 0
	HomestayOrderNeedFoodYes int64 = 1

	OrderTradeStateCancel  int64 = -1
	OrderTradeStateWaitPay int64 = 0
	OrderTradeStateWaitUse int64 = 1
	OrderTradeStateUsed    int64 = 2
	OrderTradeStateRefund  int64 = 3
	OrderTradeStateExpire  int64 = 4

	PaymentStatusFail    int64 = -1
	PaymentStatusWait    int64 = 0
	PaymentStatusSuccess int64 = 1
	PaymentStatusRefund  int64 = 2

	PaymentServiceHomestay = "homestayOrder"
	PaymentModeWechat      = "WECHAT_PAY"
)

type User struct {
	ID         int64
	CreateTime time.Time
	UpdateTime time.Time
	DeleteTime time.Time
	DelState   int64
	Version    int64
	Mobile     string
	Password   string `json:"-"`
	Nickname   string
	Sex        int64
	Avatar     string
	Info       string
}

type UserAuth struct {
	ID       int64
	UserID   int64
	AuthKey  string
	AuthType string
}

type Homestay struct {
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
	FoodPrice           int64   `json:"foodPrice"`
	HomestayPrice       int64   `json:"homestayPrice"`
	MarketHomestayPrice int64   `json:"marketHomestayPrice"`
}

type HomestayBusiness struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	UserID    int64   `json:"userId"`
	Info      string  `json:"info"`
	BossInfo  string  `json:"bossInfo"`
	RowState  int64   `json:"rowState"`
	Star      float64 `json:"star"`
	Tags      string  `json:"tags"`
	Cover     string  `json:"cover"`
	HeaderImg string  `json:"headerImg"`
}

type HomestayComment struct {
	ID         int64
	HomestayID int64
	UserID     int64
	Content    string
	Star       []byte
}

type HomestayOrder struct {
	ID                  int64
	CreateTime          time.Time
	UpdateTime          time.Time
	DeleteTime          time.Time
	DelState            int64
	Version             int64
	SN                  string
	UserID              int64
	HomestayID          int64
	Title               string
	SubTitle            string
	Cover               string
	Info                string
	PeopleNum           int64
	RowType             int64
	NeedFood            int64
	FoodInfo            string
	FoodPrice           int64
	HomestayPrice       int64
	MarketHomestayPrice int64
	HomestayBusinessID  int64
	HomestayUserID      int64
	LiveStartDate       time.Time
	LiveEndDate         time.Time
	LivePeopleNum       int64
	TradeState          int64
	TradeCode           string
	Remark              string
	OrderTotalPrice     int64
	FoodTotalPrice      int64
	HomestayTotalPrice  int64
}

type ThirdPayment struct {
	ID             int64
	SN             string
	CreateTime     time.Time
	UpdateTime     time.Time
	DeleteTime     time.Time
	DelState       int64
	Version        int64
	UserID         int64
	PayMode        string
	TradeType      string
	TradeState     string
	PayTotal       int64
	TransactionID  string
	TradeStateDesc string
	OrderSN        string
	ServiceType    string
	PayStatus      int64
	PayTime        sql.NullTime
}

type PaymentStatusEvent struct {
	PaymentSN string `json:"paymentSn"`
	OrderSN   string `json:"orderSn"`
	PayStatus int64  `json:"payStatus"`
}

type OutboxEvent struct {
	ID         int64
	EventKey   string
	Topic      string
	MessageKey string
	Payload    []byte
	RetryCount int64
}

type SeckillActivity struct {
	ID         int64
	HomestayID int64
	Title      string
	Price      int64
	Stock      int64
	SoldCount  int64
	StartTime  time.Time
	EndTime    time.Time
	Status     int64
	Remaining  int64
}

type SeckillReservation struct {
	ReservationSN string
	ActivityID    int64
	UserID        int64
	LiveStartTime int64
	LiveEndTime   int64
	LivePeopleNum int64
	Remark        string
}

type SeckillResult struct {
	ReservationSN string
	Status        string
	OrderSN       string
	Error         string
}

const (
	DataScopeAll      int64 = 1
	DataScopeBusiness int64 = 2
	DataScopeCustom   int64 = 3
	DataScopeSelf     int64 = 4
)

type AdminUser struct {
	ID           int64
	Username     string
	PasswordHash string
	Nickname     string
	Status       int64
	BusinessID   int64
	LinkedUserID int64
	Version      int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RoleIDs      []int64
}

type AdminRole struct {
	ID            int64     `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Status        int64     `json:"status"`
	ScopeType     int64     `json:"scopeType"`
	Version       int64     `json:"version"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	PermissionIDs []int64   `json:"permissionIds"`
	BusinessIDs   []int64   `json:"businessIds"`
}

type AdminPermission struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"createdAt"`
}

type AdminAuthorization struct {
	Permissions  map[string]struct{}
	AllData      bool
	BusinessIDs  []int64
	LinkedUserID int64
}

type AdminAudit struct {
	ID             int64     `json:"id"`
	AdminUserID    int64     `json:"adminUserId"`
	Username       string    `json:"username"`
	PermissionCode string    `json:"permissionCode"`
	Method         string    `json:"method"`
	Path           string    `json:"path"`
	RequestID      string    `json:"requestId"`
	IP             string    `json:"ip"`
	HTTPStatus     int       `json:"httpStatus"`
	Success        bool      `json:"success"`
	DurationMS     int64     `json:"durationMs"`
	RequestBody    string    `json:"requestBody"`
	ErrorMessage   string    `json:"errorMessage"`
	CreatedAt      time.Time `json:"createdAt"`
}

type SearchOutboxEvent struct {
	ID          int64
	EventKey    string
	AggregateID int64
	EventType   string
	RetryCount  int64
}

type HomestaySearchQuery struct {
	Keyword    string
	City       string
	MinPrice   int64
	MaxPrice   int64
	Tags       []string
	MinStar    float64
	Latitude   float64
	Longitude  float64
	DistanceKM float64
	SortBy     []string
	Page       int64
	PageSize   int64
}

type HomestaySearchResult struct {
	Total int64
	Items []Homestay
}
