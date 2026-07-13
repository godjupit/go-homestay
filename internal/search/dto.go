package search

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

type Query struct {
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

type SearchResult struct {
	Total int64
	Items []HomestayDoc
}

type HomestayDoc struct {
	ID                  int64
	Version             int64
	Title               string
	SubTitle            string
	Banner              string
	Info                string
	City                string
	Tags                string
	Star                float64
	Latitude            float64
	Longitude           float64
	PeopleNum           int64
	HomestayBusinessID  int64
	UserID              int64
	RowState            int64
	RowType             int64
	FoodInfo            string
	FoodPrice           int64
	HomestayPrice       int64
	MarketHomestayPrice int64
}
