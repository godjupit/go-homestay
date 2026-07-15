package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gin-looklook/internal/search"
	"gin-looklook/internal/shared"
	"gin-looklook/internal/travel"

	"github.com/golang-jwt/jwt/v4"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const authCacheTTL = 5 * time.Minute

type Token struct {
	AccessToken  string
	AccessExpire int64
	RefreshAfter int64
}

type Service struct {
	repo       *Repository
	searchRepo *search.Repository
	redis      *redis.Client
	cfg        shared.Config
}

func NewService(repo *Repository, searchRepo *search.Repository, rdb *redis.Client, cfg shared.Config) *Service {
	return &Service{repo: repo, searchRepo: searchRepo, redis: rdb, cfg: cfg}
}

func (s *Service) Bootstrap(ctx context.Context) error {
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
	id, err := s.repo.CreateAdmin(ctx, &AdminUser{Username: s.cfg.AdminInitialUser, PasswordHash: string(hash), Nickname: "超级管理员", Status: 1})
	if err != nil {
		return err
	}
	role, err := s.repo.RoleByCode(ctx, "super_admin")
	if err != nil {
		return err
	}
	return s.repo.AssignAdminRoles(ctx, id, []int64{role.ID})
}

func (s *Service) Login(ctx context.Context, username, password string) (Token, error) {
	admin, err := s.repo.AdminByUsername(ctx, strings.TrimSpace(username))
	if err == gorm.ErrRecordNotFound || (err == nil && admin.Status != 1) {
		return Token{}, shared.E(shared.CodeCommon, "账号或密码不正确", nil)
	}
	if err != nil {
		return Token{}, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		return Token{}, shared.E(shared.CodeCommon, "账号或密码不正确", nil)
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

func authKey(id int64) string { return fmt.Sprintf("gin:looklook:rbac:v1:admin:%d", id) }

func (s *Service) Authorization(ctx context.Context, adminID int64) (*Authorization, error) {
	key := authKey(adminID)
	if data, err := s.redis.Get(ctx, key).Bytes(); err == nil {
		var auth Authorization
		if json.Unmarshal(data, &auth) == nil {
			return &auth, nil
		}
	}
	auth, err := s.repo.AdminAuthorization(ctx, adminID)
	if err == gorm.ErrRecordNotFound {
		return nil, shared.E(shared.CodeToken, "管理员已停用或不存在", nil)
	}
	if err != nil {
		return nil, shared.E(shared.CodeDB, "读取权限失败", err)
	}
	if data, err := json.Marshal(auth); err == nil {
		_ = s.redis.Set(ctx, key, data, authCacheTTL).Err()
	}
	return auth, nil
}

func (s *Service) InvalidateAuthorization(ctx context.Context, adminIDs ...int64) {
	keys := make([]string, 0, len(adminIDs))
	for _, id := range adminIDs {
		if id > 0 {
			keys = append(keys, authKey(id))
		}
	}
	if len(keys) > 0 {
		_ = s.redis.Del(ctx, keys...).Err()
	}
}

func (s *Service) Users(ctx context.Context, page, pageSize int64) ([]AdminUser, int64, error) {
	return s.repo.AdminUsers(ctx, page, pageSize)
}

func (s *Service) CreateUser(ctx context.Context, v *AdminUser, password string, roleIDs []int64) (int64, error) {
	if len(strings.TrimSpace(v.Username)) < 3 || len(password) < 8 {
		return 0, shared.E(shared.CodeParam, "用户名至少3位，密码至少8位", nil)
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
		return 0, shared.E(shared.CodeDB, "创建管理员失败", err)
	}
	if err = s.repo.AssignAdminRoles(ctx, id, roleIDs); err != nil {
		return 0, shared.E(shared.CodeDB, "分配角色失败", err)
	}
	return id, nil
}

func (s *Service) UpdateUser(ctx context.Context, v *AdminUser, password string) error {
	hash := ""
	var err error
	if password != "" {
		if len(password) < 8 {
			return shared.E(shared.CodeParam, "密码至少8位", nil)
		}
		encoded, encodeErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		err = encodeErr
		hash = string(encoded)
	}
	if err != nil {
		return err
	}
	if err = s.repo.UpdateAdmin(ctx, v, hash); err != nil {
		return shared.E(shared.CodeDB, "更新管理员失败", err)
	}
	s.InvalidateAuthorization(ctx, v.ID)
	return nil
}

func (s *Service) AssignRoles(ctx context.Context, adminID int64, roleIDs []int64) error {
	if err := s.repo.AssignAdminRoles(ctx, adminID, roleIDs); err != nil {
		return shared.E(shared.CodeDB, "分配角色失败", err)
	}
	s.InvalidateAuthorization(ctx, adminID)
	return nil
}

func (s *Service) Roles(ctx context.Context) ([]AdminRole, error) { return s.repo.AdminRoles(ctx) }

func (s *Service) CreateRole(ctx context.Context, v *AdminRole) (int64, error) {
	if strings.TrimSpace(v.Code) == "" || strings.TrimSpace(v.Name) == "" || !validScope(v.ScopeType) {
		return 0, shared.E(shared.CodeParam, "角色参数错误", nil)
	}
	if v.Status == 0 {
		v.Status = 1
	}
	id, err := s.repo.CreateRole(ctx, v)
	if err != nil {
		return 0, shared.E(shared.CodeDB, "创建角色失败", err)
	}
	return id, nil
}

func (s *Service) ConfigureRole(ctx context.Context, v *AdminRole) error {
	if !validScope(v.ScopeType) {
		return shared.E(shared.CodeParam, "数据范围类型错误", nil)
	}
	adminIDs, err := s.repo.AdminIDsByRole(ctx, v.ID)
	if err != nil {
		return shared.E(shared.CodeDB, "读取角色用户失败", err)
	}
	if err = s.repo.ConfigureRole(ctx, v); err != nil {
		return shared.E(shared.CodeDB, "配置角色失败", err)
	}
	s.InvalidateAuthorization(ctx, adminIDs...)
	return nil
}

func validScope(scope int64) bool {
	// TODO(practice-08): 只允许已定义的数据范围枚举。
	return false
}

func (s *Service) Permissions(ctx context.Context) ([]AdminPermission, error) {
	return s.repo.AdminPermissions(ctx)
}

func (s *Service) CreatePermission(ctx context.Context, v *AdminPermission) (int64, error) {
	if strings.TrimSpace(v.Code) == "" || strings.TrimSpace(v.Name) == "" || strings.TrimSpace(v.Method) == "" || !strings.HasPrefix(v.Path, "/") {
		return 0, shared.E(shared.CodeParam, "权限参数错误", nil)
	}
	id, err := s.repo.CreatePermission(ctx, v)
	if err != nil {
		return 0, shared.E(shared.CodeDB, "创建权限失败", err)
	}
	return id, nil
}

func (s *Service) Audits(ctx context.Context, adminID int64, permission string, start, end *time.Time, page, pageSize int64) ([]AdminAudit, int64, error) {
	return s.repo.AdminAudits(ctx, adminID, permission, start, end, page, pageSize)
}

func (s *Service) SaveAudit(ctx context.Context, audit *AdminAudit) error {
	return s.repo.InsertAdminAudit(ctx, audit)
}

func (s *Service) Homestays(ctx context.Context, adminID, page, pageSize int64) ([]travel.Homestay, int64, error) {
	auth, err := s.Authorization(ctx, adminID)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.AdminHomestays(ctx, auth, page, pageSize)
}

func (s *Service) UpdateHomestay(ctx context.Context, adminID int64, v *travel.Homestay) error {
	auth, err := s.Authorization(ctx, adminID)
	if err != nil {
		return err
	}
	scopeSQL, scopeArgs := scopeCondition(auth)
	if err = s.searchRepo.UpdateAdminHomestay(ctx, v, scopeSQL, scopeArgs); err == gorm.ErrRecordNotFound {
		return shared.E(shared.CodeForbidden, "无权访问该民宿", nil)
	} else if err != nil {
		return shared.E(shared.CodeDB, "更新民宿失败", err)
	}
	return nil
}

func (s *Service) RebuildSearch(ctx context.Context) (int64, error) {
	return s.searchRepo.RebuildSearchOutbox(ctx, fmt.Sprintf("%d", time.Now().UnixNano()))
}
