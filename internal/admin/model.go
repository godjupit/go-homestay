package admin

import "time"

const (
	DataScopeAll      int64 = 1
	DataScopeBusiness int64 = 2
	DataScopeCustom   int64 = 3
	DataScopeSelf     int64 = 4
)

type AdminUser struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Username     string    `gorm:"column:username"`
	PasswordHash string    `gorm:"column:password_hash;<-:create"`
	Nickname     string    `gorm:"column:nickname"`
	Status       int64     `gorm:"column:status"`
	BusinessID   int64     `gorm:"column:business_id"`
	LinkedUserID int64     `gorm:"column:linked_user_id"`
	Version      int64     `gorm:"column:version"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
	RoleIDs      []int64   `gorm:"-"`
}

type AdminRole struct {
	ID            int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code          string    `gorm:"column:code"                        json:"code"`
	Name          string    `gorm:"column:name"                        json:"name"`
	Status        int64     `gorm:"column:status"                      json:"status"`
	ScopeType     int64     `gorm:"column:scope_type"                  json:"scopeType"`
	Version       int64     `gorm:"column:version"                     json:"version"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"   json:"createdAt"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"   json:"updatedAt"`
	PermissionIDs []int64   `gorm:"-"                                  json:"permissionIds"`
	BusinessIDs   []int64   `gorm:"-"                                  json:"businessIds"`
}

type AdminPermission struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"column:code"                        json:"code"`
	Name      string    `gorm:"column:name"                        json:"name"`
	Method    string    `gorm:"column:method"                      json:"method"`
	Path      string    `gorm:"column:path"                        json:"path"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"   json:"createdAt"`
}

type Authorization struct {
	Permissions  map[string]struct{}
	AllData      bool
	BusinessIDs  []int64
	LinkedUserID int64
}

type AdminAudit struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	AdminUserID    int64     `gorm:"column:admin_user_id"              json:"adminUserId"`
	Username       string    `gorm:"column:username"                   json:"username"`
	PermissionCode string    `gorm:"column:permission_code"            json:"permissionCode"`
	Method         string    `gorm:"column:method"                     json:"method"`
	Path           string    `gorm:"column:path"                       json:"path"`
	RequestID      string    `gorm:"column:request_id"                 json:"requestId"`
	IP             string    `gorm:"column:ip"                         json:"ip"`
	HTTPStatus     int       `gorm:"column:http_status"                json:"httpStatus"`
	Success        bool      `gorm:"column:success"                    json:"success"`
	DurationMS     int64     `gorm:"column:duration_ms"                json:"durationMs"`
	RequestBody    string    `gorm:"column:request_body"               json:"requestBody"`
	ErrorMessage   string    `gorm:"column:error_message"              json:"errorMessage"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"  json:"createdAt"`
}

func (AdminAudit) TableName() string { return "admin_audit_log" }
