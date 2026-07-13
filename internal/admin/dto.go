package admin

type AdminLoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type PageReq struct {
	Page     int64 `json:"page"`
	PageSize int64 `json:"pageSize"`
}

type AdminCreateUserReq struct {
	Username     string  `json:"username" binding:"required"`
	Password     string  `json:"password" binding:"required"`
	Nickname     string  `json:"nickname"`
	Status       int64   `json:"status"`
	BusinessID   int64   `json:"businessId"`
	LinkedUserID int64   `json:"linkedUserId"`
	RoleIDs      []int64 `json:"roleIds"`
}

type AdminUpdateUserReq struct {
	ID           int64  `json:"id" binding:"required"`
	Version      int64  `json:"version"`
	Nickname     string `json:"nickname"`
	Status       int64  `json:"status"`
	BusinessID   int64  `json:"businessId"`
	LinkedUserID int64  `json:"linkedUserId"`
	Password     string `json:"password"`
}

type AdminAssignRolesReq struct {
	AdminUserID int64   `json:"adminUserId" binding:"required"`
	RoleIDs     []int64 `json:"roleIds"`
}

type AdminRoleCreateReq struct {
	Code      string `json:"code" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Status    int64  `json:"status"`
	ScopeType int64  `json:"scopeType" binding:"required"`
}

type AdminRoleConfigureReq struct {
	ID            int64   `json:"id" binding:"required"`
	Name          string  `json:"name" binding:"required"`
	Status        int64   `json:"status"`
	ScopeType     int64   `json:"scopeType" binding:"required"`
	Version       int64   `json:"version"`
	PermissionIDs []int64 `json:"permissionIds"`
	BusinessIDs   []int64 `json:"businessIds"`
}

type AdminPermissionCreateReq struct {
	Code   string `json:"code" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Method string `json:"method" binding:"required"`
	Path   string `json:"path" binding:"required"`
}

type AdminAuditListReq struct {
	AdminUserID    int64  `json:"adminUserId"`
	PermissionCode string `json:"permissionCode"`
	StartTime      string `json:"startTime"`
	EndTime        string `json:"endTime"`
	Page           int64  `json:"page"`
	PageSize       int64  `json:"pageSize"`
}

type AdminHomestayUpdateReq struct {
	ID                  int64   `json:"id" binding:"required"`
	Version             int64   `json:"version"`
	Title               string  `json:"title" binding:"required"`
	SubTitle            string  `json:"subTitle"`
	Banner              string  `json:"banner"`
	Info                string  `json:"info"`
	City                string  `json:"city"`
	Tags                string  `json:"tags"`
	Star                float64 `json:"star"`
	Latitude            float64 `json:"latitude"`
	Longitude           float64 `json:"longitude"`
	PeopleNum           int64   `json:"peopleNum"`
	RowState            int64   `json:"rowState"`
	RowType             int64   `json:"rowType"`
	FoodInfo            string  `json:"foodInfo"`
	FoodPrice           float64 `json:"foodPrice"`
	HomestayPrice       float64 `json:"homestayPrice"`
	MarketHomestayPrice float64 `json:"marketHomestayPrice"`
}

type AdminUserView struct {
	ID           int64   `json:"id"`
	Username     string  `json:"username"`
	Nickname     string  `json:"nickname"`
	Status       int64   `json:"status"`
	BusinessID   int64   `json:"businessId"`
	LinkedUserID int64   `json:"linkedUserId"`
	Version      int64   `json:"version"`
	RoleIDs      []int64 `json:"roleIds"`
	CreatedAt    int64   `json:"createdAt"`
	UpdatedAt    int64   `json:"updatedAt"`
}
