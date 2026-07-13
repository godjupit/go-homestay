package user

type RegisterReq struct {
	Mobile   string `json:"mobile" binding:"required,len=11"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname" binding:"required,min=1,max=15"`
}

type LoginReq struct {
	Mobile   string `json:"mobile" binding:"required,len=11"`
	Password string `json:"password" binding:"required"`
}

type TokenResp struct {
	AccessToken  string `json:"accessToken"`
	AccessExpire int64  `json:"accessExpire"`
	RefreshAfter int64  `json:"refreshAfter"`
}

type WXMiniAuthReq struct {
	Code          string `json:"code" binding:"required"`
	IV            string `json:"iv"`
	EncryptedData string `json:"encryptedData"`
}

type UserView struct {
	ID       int64  `json:"id"`
	Mobile   string `json:"mobile"`
	Nickname string `json:"nickname"`
	Sex      int64  `json:"sex"`
	Avatar   string `json:"avatar"`
	Info     string `json:"info"`
}

type UpdateProfileReq struct {
	Nickname *string `json:"nickname" binding:"omitempty,min=1,max=15"`
	Sex      *int64  `json:"sex" binding:"omitempty,oneof=0 1 2"`
	Avatar   *string `json:"avatar" binding:"omitempty,max=500"`
	Info     *string `json:"info" binding:"omitempty,max=200"`
}
