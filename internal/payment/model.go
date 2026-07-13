package payment

import "time"

const (
	StatusFail    int64 = -1
	StatusWait    int64 = 0
	StatusSuccess int64 = 1
	StatusRefund  int64 = 2

	ServiceHomestay = "homestayOrder"
	ModeWechat      = "WECHAT_PAY"
)

type ThirdPayment struct {
	ID             int64      `gorm:"column:id;primaryKey;autoIncrement"`
	SN             string     `gorm:"column:sn"`
	CreateTime     time.Time  `gorm:"column:create_time;autoCreateTime"`
	UpdateTime     time.Time  `gorm:"column:update_time;autoUpdateTime"`
	DeleteTime     *time.Time `gorm:"column:delete_time;default:CURRENT_TIMESTAMP"`
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

type StatusEvent struct {
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
