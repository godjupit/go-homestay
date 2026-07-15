package order

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

type OrderCancelReq struct {
	SN string `json:"sn" binding:"required"`
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

type CloseOrderPayload struct {
	SN string `json:"sn"`
}

type NotifyPayload struct {
	OrderSN string `json:"orderSn"`
}

const (
	TaskCloseOrder       = "defer:homestay_order:close"
	TaskPaySuccessNotify = "msg:pay_success:notify_user"
	TaskSettle           = "schedule:settle_record:settle"
)
