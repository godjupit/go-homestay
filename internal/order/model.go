package order

import "time"

const (
	NeedFoodNo  int64 = 0
	NeedFoodYes int64 = 1

	TradeStateCancel  int64 = -1
	TradeStateWaitPay int64 = 0
	TradeStateWaitUse int64 = 1
	TradeStateUsed    int64 = 2
	TradeStateRefund  int64 = 3
	TradeStateExpire  int64 = 4
)

type HomestayOrder struct {
	ID                  int64      `gorm:"column:id;primaryKey;autoIncrement"`
	CreateTime          time.Time  `gorm:"column:create_time;autoCreateTime"`
	UpdateTime          time.Time  `gorm:"column:update_time;autoUpdateTime"`
	DeleteTime          *time.Time `gorm:"column:delete_time;default:CURRENT_TIMESTAMP"`
	DelState            int64      `gorm:"column:del_state"`
	Version             int64      `gorm:"column:version"`
	SN                  string     `gorm:"column:sn"`
	UserID              int64      `gorm:"column:user_id"`
	HomestayID          int64      `gorm:"column:homestay_id"`
	Title               string     `gorm:"column:title"`
	SubTitle            string     `gorm:"column:sub_title"`
	Cover               string     `gorm:"column:cover"`
	Info                string     `gorm:"column:info"`
	PeopleNum           int64      `gorm:"column:people_num"`
	RowType             int64      `gorm:"column:row_type"`
	NeedFood            int64      `gorm:"column:need_food"`
	FoodInfo            string     `gorm:"column:food_info"`
	FoodPrice           int64      `gorm:"column:food_price"`
	HomestayPrice       int64      `gorm:"column:homestay_price"`
	MarketHomestayPrice int64      `gorm:"column:market_homestay_price"`
	HomestayBusinessID  int64      `gorm:"column:homestay_business_id"`
	HomestayUserID      int64      `gorm:"column:homestay_user_id"`
	LiveStartDate       time.Time  `gorm:"column:live_start_date"`
	LiveEndDate         time.Time  `gorm:"column:live_end_date"`
	LivePeopleNum       int64      `gorm:"column:live_people_num"`
	TradeState          int64      `gorm:"column:trade_state"`
	TradeCode           string     `gorm:"column:trade_code"`
	Remark              string     `gorm:"column:remark"`
	OrderTotalPrice     int64      `gorm:"column:order_total_price"`
	FoodTotalPrice      int64      `gorm:"column:food_total_price"`
	HomestayTotalPrice  int64      `gorm:"column:homestay_total_price"`
}
