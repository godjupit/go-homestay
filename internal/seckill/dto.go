package seckill

type ReserveReq struct {
	ActivityID    int64  `json:"activityId" binding:"required"`
	LiveStartTime int64  `json:"liveStartTime" binding:"required"`
	LiveEndTime   int64  `json:"liveEndTime" binding:"required"`
	LivePeopleNum int64  `json:"livePeopleNum" binding:"required"`
	Remark        string `json:"remark"`
}

type ResultReq struct {
	ReservationSN string `json:"reservationSn" binding:"required"`
}

type ActivityView struct {
	ID         int64   `json:"id"`
	HomestayID int64   `json:"homestayId"`
	Title      string  `json:"title"`
	Price      float64 `json:"price"`
	Stock      int64   `json:"stock"`
	Remaining  int64   `json:"remaining"`
	StartTime  int64   `json:"startTime"`
	EndTime    int64   `json:"endTime"`
}

type ResultView struct {
	ReservationSN string `json:"reservationSn"`
	Status        string `json:"status"`
	OrderSN       string `json:"orderSn"`
	Error         string `json:"error"`
}

// Activity mirrors order.SeckillActivity for the handler layer
type Activity struct {
	ID         int64
	HomestayID int64
	Title      string
	Price      int64
	Stock      int64
	SoldCount  int64
	Remaining  int64
	StartTime  int64
	EndTime    int64
	Status     int64
}
