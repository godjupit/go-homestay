package travel

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

func homestayView(v Homestay) HomestayView {
	return HomestayView{ID: v.ID, Version: v.Version, Title: v.Title, SubTitle: v.SubTitle, Banner: v.Banner, Info: v.Info, City: v.City, Tags: v.Tags, Star: v.Star, Latitude: v.Latitude, Longitude: v.Longitude, PeopleNum: v.PeopleNum, HomestayBusinessID: v.HomestayBusinessID, UserID: v.UserID, RowState: v.RowState, RowType: v.RowType, FoodInfo: v.FoodInfo, FoodPrice: float64(v.FoodPrice) / 100, HomestayPrice: float64(v.HomestayPrice) / 100, MarketHomestayPrice: float64(v.MarketHomestayPrice) / 100}
}

func Views(items []Homestay) []HomestayView {
	out := make([]HomestayView, 0, len(items))
	for _, v := range items {
		out = append(out, homestayView(v))
	}
	return out
}
