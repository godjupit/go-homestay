package model

import (
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
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CreateTime time.Time `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateTime time.Time `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
	DeleteTime time.Time `gorm:"column:delete_time"                 json:"deleteTime"`
	DelState   int64     `gorm:"column:del_state"                   json:"delState"`
	Version    int64     `gorm:"column:version"                     json:"version"`
	Mobile     string    `gorm:"column:mobile"                      json:"mobile"`
	Password   string    `gorm:"column:password;<-:create"          json:"-"`
	Nickname   string    `gorm:"column:nickname"                    json:"nickname"`
	Sex        int64     `gorm:"column:sex"                         json:"sex"`
	Avatar     string    `gorm:"column:avatar"                      json:"avatar"`
	Info       string    `gorm:"column:info"                        json:"info"`
}

type UserAuth struct {
	ID       int64  `gorm:"column:id;primaryKey;autoIncrement"`
	UserID   int64  `gorm:"column:user_id"`
	AuthKey  string `gorm:"column:auth_key"`
	AuthType string `gorm:"column:auth_type"`
}

type Homestay struct {
	ID                  int64   `gorm:"column:id;primaryKey;autoIncrement"       json:"id"`
	Version             int64   `gorm:"column:version"                           json:"version"`
	Title               string  `gorm:"column:title"                             json:"title"`
	SubTitle            string  `gorm:"column:sub_title"                         json:"subTitle"`
	Banner              string  `gorm:"column:banner"                            json:"banner"`
	Info                string  `gorm:"column:info"                              json:"info"`
	City                string  `gorm:"column:city"                              json:"city"`
	Tags                string  `gorm:"column:tags"                              json:"tags"`
	Star                float64 `gorm:"column:star"                              json:"star"`
	Latitude            float64 `gorm:"column:latitude"                          json:"latitude"`
	Longitude           float64 `gorm:"column:longitude"                         json:"longitude"`
	PeopleNum           int64   `gorm:"column:people_num"                        json:"peopleNum"`
	HomestayBusinessID  int64   `gorm:"column:homestay_business_id"              json:"homestayBusinessId"`
	UserID              int64   `gorm:"column:user_id"                           json:"userId"`
	RowState            int64   `gorm:"column:row_state"                         json:"rowState"`
	RowType             int64   `gorm:"column:row_type"                          json:"rowType"`
	FoodInfo            string  `gorm:"column:food_info"                         json:"foodInfo"`
	FoodPrice           int64   `gorm:"column:food_price"                        json:"foodPrice"`
	HomestayPrice       int64   `gorm:"column:homestay_price"                    json:"homestayPrice"`
	MarketHomestayPrice int64   `gorm:"column:market_homestay_price"             json:"marketHomestayPrice"`
}

type HomestayBusiness struct {
	ID        int64   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Title     string  `gorm:"column:title"                       json:"title"`
	UserID    int64   `gorm:"column:user_id"                     json:"userId"`
	Info      string  `gorm:"column:info"                        json:"info"`
	BossInfo  string  `gorm:"column:boss_info"                   json:"bossInfo"`
	RowState  int64   `gorm:"column:row_state"                   json:"rowState"`
	Star      float64 `gorm:"column:star"                        json:"star"`
	Tags      string  `gorm:"column:tags"                        json:"tags"`
	Cover     string  `gorm:"column:cover"                       json:"cover"`
	HeaderImg string  `gorm:"column:header_img"                  json:"headerImg"`
}

type HomestayComment struct {
	ID         int64   `gorm:"column:id;primaryKey;autoIncrement"`
	HomestayID int64   `gorm:"column:homestay_id"`
	UserID     int64   `gorm:"column:user_id"`
	Content    string  `gorm:"column:content"`
	Star       float64 `gorm:"column:star"`
}

type HomestayOrder struct {
	ID                  int64     `gorm:"column:id;primaryKey;autoIncrement"`
	CreateTime          time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime          time.Time `gorm:"column:update_time;autoUpdateTime"`
	DeleteTime          time.Time `gorm:"column:delete_time"`
	DelState            int64     `gorm:"column:del_state"`
	Version             int64     `gorm:"column:version"`
	SN                  string    `gorm:"column:sn"`
	UserID              int64     `gorm:"column:user_id"`
	HomestayID          int64     `gorm:"column:homestay_id"`
	Title               string    `gorm:"column:title"`
	SubTitle            string    `gorm:"column:sub_title"`
	Cover               string    `gorm:"column:cover"`
	Info                string    `gorm:"column:info"`
	PeopleNum           int64     `gorm:"column:people_num"`
	RowType             int64     `gorm:"column:row_type"`
	NeedFood            int64     `gorm:"column:need_food"`
	FoodInfo            string    `gorm:"column:food_info"`
	FoodPrice           int64     `gorm:"column:food_price"`
	HomestayPrice       int64     `gorm:"column:homestay_price"`
	MarketHomestayPrice int64     `gorm:"column:market_homestay_price"`
	HomestayBusinessID  int64     `gorm:"column:homestay_business_id"`
	HomestayUserID      int64     `gorm:"column:homestay_user_id"`
	LiveStartDate       time.Time `gorm:"column:live_start_date"`
	LiveEndDate         time.Time `gorm:"column:live_end_date"`
	LivePeopleNum       int64     `gorm:"column:live_people_num"`
	TradeState          int64     `gorm:"column:trade_state"`
	TradeCode           string    `gorm:"column:trade_code"`
	Remark              string    `gorm:"column:remark"`
	OrderTotalPrice     int64     `gorm:"column:order_total_price"`
	FoodTotalPrice      int64     `gorm:"column:food_total_price"`
	HomestayTotalPrice  int64     `gorm:"column:homestay_total_price"`
}

type ThirdPayment struct {
	ID             int64      `gorm:"column:id;primaryKey;autoIncrement"`
	SN             string     `gorm:"column:sn"`
	CreateTime     time.Time  `gorm:"column:create_time;autoCreateTime"`
	UpdateTime     time.Time  `gorm:"column:update_time;autoUpdateTime"`
	DeleteTime     time.Time  `gorm:"column:delete_time"`
	DelState       int64      `gorm:"column:del_state"`
	Version        int64      `gorm:"column:version"`
	UserID         int64      `gorm:"column:user_id"`
	PayMode        string     `gorm:"column:pay_mode"`
	TradeType      string     `gorm:"column:trade_type"`
	TradeState     string     `gorm:"column:trade_state"`
	PayTotal       int64      `gorm:"column:pay_total"`
	TransactionID  string     `gorm:"column:transaction_id"`
	TradeStateDesc string     `gorm:"column:trade_state_desc"`
	OrderSN        string     `gorm:"column:order_sn"`
	ServiceType    string     `gorm:"column:service_type"`
	PayStatus      int64      `gorm:"column:pay_status"`
	PayTime        *time.Time `gorm:"column:pay_time"`
}

type PaymentStatusEvent struct {
	PaymentSN string `json:"paymentSn"`
	OrderSN   string `json:"orderSn"`
	PayStatus int64  `json:"payStatus"`
}

type OutboxEvent struct {
	ID         int64  `gorm:"column:id;primaryKey;autoIncrement"`
	EventKey   string `gorm:"column:event_key"`
	Topic      string `gorm:"column:topic"`
	MessageKey string `gorm:"column:message_key"`
	Payload    []byte `gorm:"column:payload"`
	RetryCount int64  `gorm:"column:retry_count"`
}

func (OutboxEvent) TableName() string { return "event_outbox" }

type SeckillActivity struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	HomestayID int64     `gorm:"column:homestay_id"`
	Title      string    `gorm:"column:title"`
	Price      int64     `gorm:"column:price"`
	Stock      int64     `gorm:"column:stock"`
	SoldCount  int64     `gorm:"column:sold_count"`
	StartTime  time.Time `gorm:"column:start_time"`
	EndTime    time.Time `gorm:"column:end_time"`
	Status     int64     `gorm:"column:status"`
	Remaining  int64     `gorm:"-"`
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
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Username     string    `gorm:"column:username"`
	PasswordHash string    `gorm:"column:password_hash;<-:create"`
	Nickname     string    `gorm:"column:nickname"`
	Status       int64     `gorm:"column:status"`
	BusinessID   int64     `gorm:"column:business_id"`
	LinkedUserID int64     `gorm:"column:linked_user_id"`
	Version      int64     `gorm:"column:version"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
	RoleIDs      []int64   `gorm:"-"`
}

type AdminRole struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code          string    `gorm:"column:code"                        json:"code"`
	Name          string    `gorm:"column:name"                        json:"name"`
	Status        int64     `gorm:"column:status"                      json:"status"`
	ScopeType     int64     `gorm:"column:scope_type"                  json:"scopeType"`
	Version       int64     `gorm:"column:version"                     json:"version"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"   json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"   json:"updatedAt"`
	PermissionIDs []int64   `gorm:"-"                                  json:"permissionIds"`
	BusinessIDs   []int64   `gorm:"-"                                  json:"businessIds"`
}

type AdminPermission struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"column:code"                        json:"code"`
	Name      string    `gorm:"column:name"                        json:"name"`
	Method    string    `gorm:"column:method"                      json:"method"`
	Path      string    `gorm:"column:path"                        json:"path"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"   json:"createdAt"`
}

type AdminAuthorization struct {
	Permissions  map[string]struct{}
	AllData      bool
	BusinessIDs  []int64
	LinkedUserID int64
}

type AdminAudit struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AdminUserID    int64     `gorm:"column:admin_user_id"              json:"adminUserId"`
	Username       string    `gorm:"column:username"                   json:"username"`
	PermissionCode string    `gorm:"column:permission_code"            json:"permissionCode"`
	Method         string    `gorm:"column:method"                     json:"method"`
	Path           string    `gorm:"column:path"                       json:"path"`
	RequestID      string    `gorm:"column:request_id"                 json:"requestId"`
	IP             string    `gorm:"column:ip"                         json:"ip"`
	HTTPStatus     int       `gorm:"column:http_status"                json:"httpStatus"`
	Success        bool      `gorm:"column:success"                    json:"success"`
	DurationMS     int64     `gorm:"column:duration_ms"                json:"durationMs"`
	RequestBody    string    `gorm:"column:request_body"               json:"requestBody"`
	ErrorMessage   string    `gorm:"column:error_message"              json:"errorMessage"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"  json:"createdAt"`
}

func (AdminAudit) TableName() string { return "admin_audit_log" }

type SearchOutboxEvent struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement"`
	EventKey    string `gorm:"column:event_key"`
	AggregateID int64  `gorm:"column:aggregate_id"`
	EventType   string `gorm:"column:event_type"`
	RetryCount  int64  `gorm:"column:retry_count"`
}

func (SearchOutboxEvent) TableName() string { return "search_event_outbox" }

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
