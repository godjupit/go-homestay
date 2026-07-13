package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gin-looklook/internal/config"
	"gin-looklook/internal/model"
	"gin-looklook/internal/platform"
	"gin-looklook/internal/repository"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

const adminAuthCacheTTL = 5 * time.Minute

type AdminService struct {
	repo *repository.Repository
	cfg  config.Config
}

func NewAdminService(repo *repository.Repository, cfg config.Config) *AdminService {
	return &AdminService{repo: repo, cfg: cfg}
}

func (s *AdminService) Bootstrap(ctx context.Context) error {
	count, err := s.repo.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(s.cfg.AdminInitialPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	id, err := s.repo.CreateAdmin(ctx, &model.AdminUser{Username: s.cfg.AdminInitialUser, PasswordHash: string(hash), Nickname: "超级管理员", Status: 1})
	if err != nil {
		return err
	}
	role, err := s.repo.RoleByCode(ctx, "super_admin")
	if err != nil {
		return err
	}
	return s.repo.AssignAdminRoles(ctx, id, []int64{role.ID})
}

func (s *AdminService) Login(ctx context.Context, username, password string) (Token, error) {
	admin, err := s.repo.AdminByUsername(ctx, strings.TrimSpace(username))
	if err == sql.ErrNoRows || (err == nil && admin.Status != 1) {
		return Token{}, platform.E(platform.CodeCommon, "账号或密码不正确", nil)
	}
	if err != nil {
		return Token{}, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		return Token{}, platform.E(platform.CodeCommon, "账号或密码不正确", nil)
	}
	now := time.Now()
	expires := now.Add(s.cfg.AdminJWTExpire)
	claims := jwt.MapClaims{"exp": expires.Unix(), "iat": now.Unix(), "adminId": admin.ID, "username": admin.Username, "tokenType": "admin"}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.AdminJWTSecret))
	if err != nil {
		return Token{}, err
	}
	return Token{AccessToken: signed, AccessExpire: expires.Unix(), RefreshAfter: now.Add(s.cfg.AdminJWTExpire / 2).Unix()}, nil
}

func adminAuthKey(id int64) string { return fmt.Sprintf("gin:looklook:rbac:v1:admin:%d", id) }

func (s *AdminService) Authorization(ctx context.Context, adminID int64) (*model.AdminAuthorization, error) {
	key := adminAuthKey(adminID)
	if data, err := s.repo.Redis.Get(ctx, key).Bytes(); err == nil {
		var auth model.AdminAuthorization
		if json.Unmarshal(data, &auth) == nil {
			return &auth, nil
		}
	}
	auth, err := s.repo.AdminAuthorization(ctx, adminID)
	if err == sql.ErrNoRows {
		return nil, platform.E(platform.CodeToken, "管理员已停用或不存在", nil)
	}
	if err != nil {
		return nil, platform.E(platform.CodeDB, "读取权限失败", err)
	}
	if data, err := json.Marshal(auth); err == nil {
		_ = s.repo.Redis.Set(ctx, key, data, adminAuthCacheTTL).Err()
	}
	return auth, nil
}

func (s *AdminService) InvalidateAuthorization(ctx context.Context, adminIDs ...int64) {
	keys := make([]string, 0, len(adminIDs))
	for _, id := range adminIDs {
		if id > 0 {
			keys = append(keys, adminAuthKey(id))
		}
	}
	if len(keys) > 0 {
		_ = s.repo.Redis.Del(ctx, keys...).Err()
	}
}

func (s *AdminService) Users(ctx context.Context, page, pageSize int64) ([]model.AdminUser, int64, error) {
	return s.repo.AdminUsers(ctx, page, pageSize)
}

func (s *AdminService) CreateUser(ctx context.Context, v *model.AdminUser, password string, roleIDs []int64) (int64, error) {
	if len(strings.TrimSpace(v.Username)) < 3 || len(password) < 8 {
		return 0, platform.E(platform.CodeParam, "用户名至少3位，密码至少8位", nil)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	v.PasswordHash = string(hash)
	if v.Status == 0 {
		v.Status = 1
	}
	id, err := s.repo.CreateAdmin(ctx, v)
	if err != nil {
		return 0, platform.E(platform.CodeDB, "创建管理员失败", err)
	}
	if err = s.repo.AssignAdminRoles(ctx, id, roleIDs); err != nil {
		return 0, platform.E(platform.CodeDB, "分配角色失败", err)
	}
	return id, nil
}

func (s *AdminService) UpdateUser(ctx context.Context, v *model.AdminUser, password string) error {
	hash := ""
	var err error
	if password != "" {
		if len(password) < 8 {
			return platform.E(platform.CodeParam, "密码至少8位", nil)
		}
		encoded, encodeErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		err = encodeErr
		hash = string(encoded)
	}
	if err != nil {
		return err
	}
	if err = s.repo.UpdateAdmin(ctx, v, hash); err != nil {
		return platform.E(platform.CodeDB, "更新管理员失败", err)
	}
	s.InvalidateAuthorization(ctx, v.ID)
	return nil
}

func (s *AdminService) AssignRoles(ctx context.Context, adminID int64, roleIDs []int64) error {
	if err := s.repo.AssignAdminRoles(ctx, adminID, roleIDs); err != nil {
		return platform.E(platform.CodeDB, "分配角色失败", err)
	}
	s.InvalidateAuthorization(ctx, adminID)
	return nil
}

func (s *AdminService) Roles(ctx context.Context) ([]model.AdminRole, error) {
	return s.repo.AdminRoles(ctx)
}

func (s *AdminService) CreateRole(ctx context.Context, v *model.AdminRole) (int64, error) {
	if strings.TrimSpace(v.Code) == "" || strings.TrimSpace(v.Name) == "" || !validScope(v.ScopeType) {
		return 0, platform.E(platform.CodeParam, "角色参数错误", nil)
	}
	if v.Status == 0 {
		v.Status = 1
	}
	id, err := s.repo.CreateRole(ctx, v)
	if err != nil {
		return 0, platform.E(platform.CodeDB, "创建角色失败", err)
	}
	return id, nil
}

func (s *AdminService) ConfigureRole(ctx context.Context, v *model.AdminRole) error {
	if !validScope(v.ScopeType) {
		return platform.E(platform.CodeParam, "数据范围类型错误", nil)
	}
	adminIDs, err := s.repo.AdminIDsByRole(ctx, v.ID)
	if err != nil {
		return platform.E(platform.CodeDB, "读取角色用户失败", err)
	}
	if err = s.repo.ConfigureRole(ctx, v); err != nil {
		return platform.E(platform.CodeDB, "配置角色失败", err)
	}
	s.InvalidateAuthorization(ctx, adminIDs...)
	return nil
}

func validScope(scope int64) bool {
	return scope >= model.DataScopeAll && scope <= model.DataScopeSelf
}

func (s *AdminService) Permissions(ctx context.Context) ([]model.AdminPermission, error) {
	return s.repo.AdminPermissions(ctx)
}

func (s *AdminService) CreatePermission(ctx context.Context, v *model.AdminPermission) (int64, error) {
	if strings.TrimSpace(v.Code) == "" || strings.TrimSpace(v.Name) == "" || strings.TrimSpace(v.Method) == "" || !strings.HasPrefix(v.Path, "/") {
		return 0, platform.E(platform.CodeParam, "权限参数错误", nil)
	}
	id, err := s.repo.CreatePermission(ctx, v)
	if err != nil {
		return 0, platform.E(platform.CodeDB, "创建权限失败", err)
	}
	return id, nil
}

func (s *AdminService) Audits(ctx context.Context, adminID int64, permission string, start, end *time.Time, page, pageSize int64) ([]model.AdminAudit, int64, error) {
	return s.repo.AdminAudits(ctx, adminID, permission, start, end, page, pageSize)
}

func (s *AdminService) SaveAudit(ctx context.Context, audit *model.AdminAudit) error {
	return s.repo.InsertAdminAudit(ctx, audit)
}

func (s *AdminService) Homestays(ctx context.Context, adminID, page, pageSize int64) ([]model.Homestay, int64, error) {
	auth, err := s.Authorization(ctx, adminID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.AdminHomestays(ctx, auth, page, pageSize)
}

func (s *AdminService) UpdateHomestay(ctx context.Context, adminID int64, v *model.Homestay) error {
	auth, err := s.Authorization(ctx, adminID)
	if err != nil {
		return err
	}
	if err = s.repo.UpdateAdminHomestay(ctx, auth, v); err == sql.ErrNoRows {
		return platform.E(platform.CodeForbidden, "无权访问该民宿", nil)
	} else if err != nil {
		return platform.E(platform.CodeDB, "更新民宿失败", err)
	}
	return nil
}

func (s *AdminService) RebuildSearch(ctx context.Context) (int64, error) {
	return s.repo.RebuildSearchOutbox(ctx, fmt.Sprintf("%d", time.Now().UnixNano()))
}
