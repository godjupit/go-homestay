package repository

import (
	"context"
	"fmt"
	"time"

	"gin-looklook/internal/model"

	"gorm.io/gorm"
)

// ── Admin User ──

func (r *Repository) AdminByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	var v model.AdminUser
	err := r.UserDB.WithContext(ctx).Where("username = ?", username).First(&v).Error
	return &v, err
}

func (r *Repository) AdminByID(ctx context.Context, id int64) (*model.AdminUser, error) {
	var v model.AdminUser
	err := r.UserDB.WithContext(ctx).Where("id = ?", id).First(&v).Error
	return &v, err
}

func (r *Repository) CountAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.UserDB.WithContext(ctx).Model(&model.AdminUser{}).Count(&count).Error
	return count, err
}

func (r *Repository) CreateAdmin(ctx context.Context, v *model.AdminUser) (int64, error) {
	err := r.UserDB.WithContext(ctx).Create(v).Error
	if err != nil {
		return 0, err
	}
	return v.ID, nil
}

func (r *Repository) UpdateAdmin(ctx context.Context, v *model.AdminUser, passwordHash string) error {
	updates := map[string]any{
		"nickname":       v.Nickname,
		"status":         v.Status,
		"business_id":    v.BusinessID,
		"linked_user_id": v.LinkedUserID,
		"version":        gorm.Expr("version + 1"),
	}
	if passwordHash != "" {
		updates["password_hash"] = passwordHash
	}
	result := r.UserDB.WithContext(ctx).
		Model(&model.AdminUser{}).
		Where("id = ? AND version = ?", v.ID, v.Version).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("admin user not found or version conflict")
	}
	return nil
}

func (r *Repository) AssignAdminRoles(ctx context.Context, adminID int64, roleIDs []int64) error {
	return r.UserDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("admin_user_id = ?", adminID).Delete(&struct {
			AdminUserID int64 `gorm:"column:admin_user_id"`
			RoleID      int64 `gorm:"column:role_id"`
		}{}).Error; err != nil {
			return err
		}
		for _, roleID := range uniquePositive(roleIDs) {
			if err := tx.Exec("INSERT INTO admin_user_role(admin_user_id,role_id) VALUES(?,?)", adminID, roleID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) AdminUsers(ctx context.Context, page, pageSize int64) ([]model.AdminUser, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	if err := r.UserDB.WithContext(ctx).Model(&model.AdminUser{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.AdminUser
	if err := r.UserDB.WithContext(ctx).Order("id DESC").Limit(int(pageSize)).Offset(int((page - 1) * pageSize)).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	for i := range items {
		var roleIDs []int64
		if err := r.UserDB.WithContext(ctx).
			Table("admin_user_role").
			Select("role_id").
			Where("admin_user_id = ?", items[i].ID).
			Order("role_id").
			Pluck("role_id", &roleIDs).Error; err != nil {
			return nil, 0, err
		}
		items[i].RoleIDs = roleIDs
	}
	return items, total, nil
}

// ── Admin Role ──

func (r *Repository) RoleByCode(ctx context.Context, code string) (*model.AdminRole, error) {
	var v model.AdminRole
	err := r.UserDB.WithContext(ctx).Where("code = ?", code).First(&v).Error
	return &v, err
}

func (r *Repository) CreateRole(ctx context.Context, v *model.AdminRole) (int64, error) {
	err := r.UserDB.WithContext(ctx).Create(v).Error
	if err != nil {
		return 0, err
	}
	return v.ID, nil
}

func (r *Repository) AdminRoles(ctx context.Context) ([]model.AdminRole, error) {
	var items []model.AdminRole
	if err := r.UserDB.WithContext(ctx).Order("id").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		if err := r.UserDB.WithContext(ctx).
			Table("admin_role_permission").
			Select("permission_id").
			Where("role_id = ?", items[i].ID).
			Order("permission_id").
			Pluck("permission_id", &items[i].PermissionIDs).Error; err != nil {
			return nil, err
		}
		if err := r.UserDB.WithContext(ctx).
			Table("admin_role_data_scope").
			Select("business_id").
			Where("role_id = ?", items[i].ID).
			Order("business_id").
			Pluck("business_id", &items[i].BusinessIDs).Error; err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *Repository) ConfigureRole(ctx context.Context, v *model.AdminRole) error {
	return r.UserDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.AdminRole{}).
			Where("id = ? AND version = ?", v.ID, v.Version).
			Updates(map[string]any{
				"name":       v.Name,
				"status":     v.Status,
				"scope_type": v.ScopeType,
				"version":    gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("role not found or version conflict")
		}
		if err := tx.Where("role_id = ?", v.ID).Delete(&struct {
			RoleID       int64 `gorm:"column:role_id"`
			PermissionID int64 `gorm:"column:permission_id"`
		}{}).Error; err != nil {
			return err
		}
		for _, permissionID := range uniquePositive(v.PermissionIDs) {
			if err := tx.Exec("INSERT INTO admin_role_permission(role_id,permission_id) VALUES(?,?)", v.ID, permissionID).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("role_id = ?", v.ID).Delete(&struct {
			RoleID     int64 `gorm:"column:role_id"`
			BusinessID int64 `gorm:"column:business_id"`
		}{}).Error; err != nil {
			return err
		}
		if v.ScopeType == model.DataScopeCustom {
			for _, businessID := range uniquePositive(v.BusinessIDs) {
				if err := tx.Exec("INSERT INTO admin_role_data_scope(role_id,business_id) VALUES(?,?)", v.ID, businessID).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// ── Admin Permission ──

func (r *Repository) AdminPermissions(ctx context.Context) ([]model.AdminPermission, error) {
	var items []model.AdminPermission
	err := r.UserDB.WithContext(ctx).Order("id").Find(&items).Error
	return items, err
}

func (r *Repository) CreatePermission(ctx context.Context, v *model.AdminPermission) (int64, error) {
	err := r.UserDB.WithContext(ctx).Create(v).Error
	if err != nil {
		return 0, err
	}
	return v.ID, nil
}

// ── Admin Authorization ──

func (r *Repository) AdminAuthorization(ctx context.Context, adminID int64) (*model.AdminAuthorization, error) {
	admin, err := r.AdminByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if admin.Status != 1 {
		return nil, gorm.ErrRecordNotFound
	}
	auth := &model.AdminAuthorization{Permissions: make(map[string]struct{})}

	var codes []string
	if err := r.UserDB.WithContext(ctx).
		Table("admin_user_role ur").
		Select("DISTINCT p.code").
		Joins("JOIN admin_role r ON r.id = ur.role_id AND r.status = 1").
		Joins("JOIN admin_role_permission rp ON rp.role_id = r.id").
		Joins("JOIN admin_permission p ON p.id = rp.permission_id").
		Where("ur.admin_user_id = ?", adminID).
		Pluck("p.code", &codes).Error; err != nil {
		return nil, err
	}
	for _, code := range codes {
		auth.Permissions[code] = struct{}{}
	}

	type scopeRow struct {
		ScopeType       int64 `gorm:"column:scope_type"`
		CustomBusinessID int64 `gorm:"column:custom_business_id"`
	}
	var scopeRows []scopeRow
	if err := r.UserDB.WithContext(ctx).
		Table("admin_user_role ur").
		Select("r.scope_type, COALESCE(s.business_id, 0) AS custom_business_id").
		Joins("JOIN admin_role r ON r.id = ur.role_id AND r.status = 1").
		Joins("LEFT JOIN admin_role_data_scope s ON s.role_id = r.id").
		Where("ur.admin_user_id = ?", adminID).
		Find(&scopeRows).Error; err != nil {
		return nil, err
	}
	businesses := make(map[int64]struct{})
	for _, row := range scopeRows {
		switch row.ScopeType {
		case model.DataScopeAll:
			auth.AllData = true
		case model.DataScopeBusiness:
			if admin.BusinessID > 0 {
				businesses[admin.BusinessID] = struct{}{}
			}
		case model.DataScopeCustom:
			if row.CustomBusinessID > 0 {
				businesses[row.CustomBusinessID] = struct{}{}
			}
		case model.DataScopeSelf:
			auth.LinkedUserID = admin.LinkedUserID
		}
	}
	for id := range businesses {
		auth.BusinessIDs = append(auth.BusinessIDs, id)
	}
	return auth, nil
}

func (r *Repository) AdminIDsByRole(ctx context.Context, roleID int64) ([]int64, error) {
	var ids []int64
	err := r.UserDB.WithContext(ctx).
		Table("admin_user_role").
		Select("admin_user_id").
		Where("role_id = ?", roleID).
		Pluck("admin_user_id", &ids).Error
	return ids, err
}

// ── Admin Audit ──

func (r *Repository) InsertAdminAudit(ctx context.Context, v *model.AdminAudit) error {
	return r.UserDB.WithContext(ctx).Create(v).Error
}

func (r *Repository) AdminAudits(ctx context.Context, adminID int64, permission string, start, end *time.Time, page, pageSize int64) ([]model.AdminAudit, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	db := r.UserDB.WithContext(ctx).Model(&model.AdminAudit{})
	if adminID > 0 {
		db = db.Where("admin_user_id = ?", adminID)
	}
	if permission != "" {
		db = db.Where("permission_code = ?", permission)
	}
	if start != nil {
		db = db.Where("created_at >= ?", *start)
	}
	if end != nil {
		db = db.Where("created_at <= ?", *end)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.AdminAudit
	if err := db.Order("id DESC").Limit(int(pageSize)).Offset(int((page - 1) * pageSize)).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ── Helpers ──

func normalizePage(page, pageSize int64) (int64, int64) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func uniquePositive(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
