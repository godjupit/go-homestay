package user

import "time"

type User struct {
	ID         int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CreateTime time.Time  `gorm:"column:create_time;autoCreateTime" json:"createTime"`
	UpdateTime time.Time  `gorm:"column:update_time;autoUpdateTime" json:"updateTime"`
	DeleteTime *time.Time `gorm:"column:delete_time;default:CURRENT_TIMESTAMP" json:"deleteTime"`
	DelState   int64      `gorm:"column:del_state"                   json:"delState"`
	Version    int64      `gorm:"column:version"                     json:"version"`
	Mobile     string     `gorm:"column:mobile"                      json:"mobile"`
	Password   string     `gorm:"column:password;<-:create"          json:"-"`
	Nickname   string     `gorm:"column:nickname"                    json:"nickname"`
	Sex        int64      `gorm:"column:sex"                         json:"sex"`
	Avatar     string     `gorm:"column:avatar"                      json:"avatar"`
	Info       string     `gorm:"column:info"                        json:"info"`
}

type UserAuth struct {
	ID       int64  `gorm:"column:id;primaryKey;autoIncrement"`
	UserID   int64  `gorm:"column:user_id"`
	AuthKey  string `gorm:"column:auth_key"`
	AuthType string `gorm:"column:auth_type"`
}

const (
	DelStateNo  int64 = 0
	DelStateYes int64 = 1

	AuthTypeSystem  = "system"
	AuthTypeSmallWX = "wxMini"
)
