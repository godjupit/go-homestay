package payment

type WxPayReq struct {
	OrderSN     string `json:"orderSn" binding:"required"`
	ServiceType string `json:"serviceType" binding:"required"`
}

type WxPayResp struct {
	Appid     string `json:"appid"`
	NonceStr  string `json:"nonceStr"`
	PaySign   string `json:"paySign"`
	Package   string `json:"package"`
	Timestamp string `json:"timestamp"`
	SignType  string `json:"signType"`
}

type PrepayResult struct{ AppID, NonceStr, PaySign, Package, Timestamp, SignType string }
