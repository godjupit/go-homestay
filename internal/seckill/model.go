package seckill

type Reservation struct {
	ReservationSN string
	ActivityID    int64
	UserID        int64
	LiveStartTime int64
	LiveEndTime   int64
	LivePeopleNum int64
	Remark        string
}

type Result struct {
	ReservationSN string
	Status        string
	OrderSN       string
	Error         string
}
