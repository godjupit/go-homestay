package travel

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
