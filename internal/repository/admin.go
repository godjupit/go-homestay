package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gin-looklook/internal/model"
)

const adminUserFields = "id,username,password_hash,nickname,status,business_id,linked_user_id,version,created_at,updated_at"

func scanAdminUser(row interface{ Scan(...any) error }) (*model.AdminUser, error) {
	var v model.AdminUser
	err := row.Scan(&v.ID, &v.Username, &v.PasswordHash, &v.Nickname, &v.Status, &v.BusinessID, &v.LinkedUserID, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	return &v, err
}

func (r *Repository) AdminByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	return scanAdminUser(r.UserDB.QueryRowContext(ctx, "SELECT "+adminUserFields+" FROM admin_user WHERE username=? LIMIT 1", username))
}

func (r *Repository) AdminByID(ctx context.Context, id int64) (*model.AdminUser, error) {
	return scanAdminUser(r.UserDB.QueryRowContext(ctx, "SELECT "+adminUserFields+" FROM admin_user WHERE id=? LIMIT 1", id))
}

func (r *Repository) CountAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.UserDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_user").Scan(&count)
	return count, err
}

func (r *Repository) CreateAdmin(ctx context.Context, v *model.AdminUser) (int64, error) {
	res, err := r.UserDB.ExecContext(ctx, `INSERT INTO admin_user(username,password_hash,nickname,status,business_id,linked_user_id) VALUES(?,?,?,?,?,?)`, v.Username, v.PasswordHash, v.Nickname, v.Status, v.BusinessID, v.LinkedUserID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) UpdateAdmin(ctx context.Context, v *model.AdminUser, passwordHash string) error {
	query := "UPDATE admin_user SET nickname=?,status=?,business_id=?,linked_user_id=?,version=version+1"
	args := []any{v.Nickname, v.Status, v.BusinessID, v.LinkedUserID}
	if passwordHash != "" {
		query += ",password_hash=?"
		args = append(args, passwordHash)
	}
	query += " WHERE id=? AND version=?"
	args = append(args, v.ID, v.Version)
	res, err := r.UserDB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("admin user not found or version conflict")
	}
	return nil
}

func (r *Repository) AssignAdminRoles(ctx context.Context, adminID int64, roleIDs []int64) error {
	tx, err := r.UserDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "DELETE FROM admin_user_role WHERE admin_user_id=?", adminID); err != nil {
		return err
	}
	for _, roleID := range uniquePositive(roleIDs) {
		if _, err = tx.ExecContext(ctx, "INSERT INTO admin_user_role(admin_user_id,role_id) VALUES(?,?)", adminID, roleID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) AdminUsers(ctx context.Context, page, pageSize int64) ([]model.AdminUser, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	if err := r.UserDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_user").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.UserDB.QueryContext(ctx, "SELECT "+adminUserFields+" FROM admin_user ORDER BY id DESC LIMIT ? OFFSET ?", pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminUser, 0)
	for rows.Next() {
		v, err := scanAdminUser(rows)
		if err != nil {
			return nil, 0, err
		}
		roleRows, err := r.UserDB.QueryContext(ctx, "SELECT role_id FROM admin_user_role WHERE admin_user_id=? ORDER BY role_id", v.ID)
		if err != nil {
			return nil, 0, err
		}
		for roleRows.Next() {
			var roleID int64
			if err = roleRows.Scan(&roleID); err != nil {
				roleRows.Close()
				return nil, 0, err
			}
			v.RoleIDs = append(v.RoleIDs, roleID)
		}
		if err = roleRows.Close(); err != nil {
			return nil, 0, err
		}
		items = append(items, *v)
	}
	return items, total, rows.Err()
}

func scanAdminRole(row interface{ Scan(...any) error }) (*model.AdminRole, error) {
	var v model.AdminRole
	err := row.Scan(&v.ID, &v.Code, &v.Name, &v.Status, &v.ScopeType, &v.Version, &v.CreatedAt, &v.UpdatedAt)
	return &v, err
}

const adminRoleFields = "id,code,name,status,scope_type,version,created_at,updated_at"

func (r *Repository) RoleByCode(ctx context.Context, code string) (*model.AdminRole, error) {
	return scanAdminRole(r.UserDB.QueryRowContext(ctx, "SELECT "+adminRoleFields+" FROM admin_role WHERE code=? LIMIT 1", code))
}

func (r *Repository) CreateRole(ctx context.Context, v *model.AdminRole) (int64, error) {
	res, err := r.UserDB.ExecContext(ctx, "INSERT INTO admin_role(code,name,status,scope_type) VALUES(?,?,?,?)", v.Code, v.Name, v.Status, v.ScopeType)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) AdminRoles(ctx context.Context) ([]model.AdminRole, error) {
	rows, err := r.UserDB.QueryContext(ctx, "SELECT "+adminRoleFields+" FROM admin_role ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.AdminRole, 0)
	for rows.Next() {
		v, err := scanAdminRole(rows)
		if err != nil {
			return nil, err
		}
		permissionRows, err := r.UserDB.QueryContext(ctx, "SELECT permission_id FROM admin_role_permission WHERE role_id=? ORDER BY permission_id", v.ID)
		if err != nil {
			return nil, err
		}
		for permissionRows.Next() {
			var id int64
			if err = permissionRows.Scan(&id); err != nil {
				permissionRows.Close()
				return nil, err
			}
			v.PermissionIDs = append(v.PermissionIDs, id)
		}
		permissionRows.Close()
		businessRows, err := r.UserDB.QueryContext(ctx, "SELECT business_id FROM admin_role_data_scope WHERE role_id=? ORDER BY business_id", v.ID)
		if err != nil {
			return nil, err
		}
		for businessRows.Next() {
			var id int64
			if err = businessRows.Scan(&id); err != nil {
				businessRows.Close()
				return nil, err
			}
			v.BusinessIDs = append(v.BusinessIDs, id)
		}
		businessRows.Close()
		items = append(items, *v)
	}
	return items, rows.Err()
}

func (r *Repository) ConfigureRole(ctx context.Context, v *model.AdminRole) error {
	tx, err := r.UserDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "UPDATE admin_role SET name=?,status=?,scope_type=?,version=version+1 WHERE id=? AND version=?", v.Name, v.Status, v.ScopeType, v.ID, v.Version)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("role not found or version conflict")
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM admin_role_permission WHERE role_id=?", v.ID); err != nil {
		return err
	}
	for _, permissionID := range uniquePositive(v.PermissionIDs) {
		if _, err = tx.ExecContext(ctx, "INSERT INTO admin_role_permission(role_id,permission_id) VALUES(?,?)", v.ID, permissionID); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM admin_role_data_scope WHERE role_id=?", v.ID); err != nil {
		return err
	}
	if v.ScopeType == model.DataScopeCustom {
		for _, businessID := range uniquePositive(v.BusinessIDs) {
			if _, err = tx.ExecContext(ctx, "INSERT INTO admin_role_data_scope(role_id,business_id) VALUES(?,?)", v.ID, businessID); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *Repository) AdminPermissions(ctx context.Context) ([]model.AdminPermission, error) {
	rows, err := r.UserDB.QueryContext(ctx, "SELECT id,code,name,method,path,created_at FROM admin_permission ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.AdminPermission, 0)
	for rows.Next() {
		var v model.AdminPermission
		if err := rows.Scan(&v.ID, &v.Code, &v.Name, &v.Method, &v.Path, &v.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (r *Repository) CreatePermission(ctx context.Context, v *model.AdminPermission) (int64, error) {
	res, err := r.UserDB.ExecContext(ctx, "INSERT INTO admin_permission(code,name,method,path) VALUES(?,?,?,?)", v.Code, v.Name, strings.ToUpper(v.Method), v.Path)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) AdminAuthorization(ctx context.Context, adminID int64) (*model.AdminAuthorization, error) {
	admin, err := r.AdminByID(ctx, adminID)
	if err != nil {
		return nil, err
	}
	if admin.Status != 1 {
		return nil, sql.ErrNoRows
	}
	auth := &model.AdminAuthorization{Permissions: make(map[string]struct{})}
	rows, err := r.UserDB.QueryContext(ctx, `SELECT DISTINCT p.code
		FROM admin_user_role ur JOIN admin_role r ON r.id=ur.role_id AND r.status=1
		JOIN admin_role_permission rp ON rp.role_id=r.id
		JOIN admin_permission p ON p.id=rp.permission_id WHERE ur.admin_user_id=?`, adminID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var code string
		if err = rows.Scan(&code); err != nil {
			rows.Close()
			return nil, err
		}
		auth.Permissions[code] = struct{}{}
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	scopeRows, err := r.UserDB.QueryContext(ctx, `SELECT r.scope_type,COALESCE(s.business_id,0)
		FROM admin_user_role ur JOIN admin_role r ON r.id=ur.role_id AND r.status=1
		LEFT JOIN admin_role_data_scope s ON s.role_id=r.id WHERE ur.admin_user_id=?`, adminID)
	if err != nil {
		return nil, err
	}
	businesses := make(map[int64]struct{})
	for scopeRows.Next() {
		var scopeType, customBusinessID int64
		if err = scopeRows.Scan(&scopeType, &customBusinessID); err != nil {
			scopeRows.Close()
			return nil, err
		}
		switch scopeType {
		case model.DataScopeAll:
			auth.AllData = true
		case model.DataScopeBusiness:
			if admin.BusinessID > 0 {
				businesses[admin.BusinessID] = struct{}{}
			}
		case model.DataScopeCustom:
			if customBusinessID > 0 {
				businesses[customBusinessID] = struct{}{}
			}
		case model.DataScopeSelf:
			auth.LinkedUserID = admin.LinkedUserID
		}
	}
	scopeRows.Close()
	for id := range businesses {
		auth.BusinessIDs = append(auth.BusinessIDs, id)
	}
	return auth, nil
}

func (r *Repository) AdminIDsByRole(ctx context.Context, roleID int64) ([]int64, error) {
	rows, err := r.UserDB.QueryContext(ctx, "SELECT admin_user_id FROM admin_user_role WHERE role_id=?", roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) InsertAdminAudit(ctx context.Context, v *model.AdminAudit) error {
	_, err := r.UserDB.ExecContext(ctx, `INSERT INTO admin_audit_log(admin_user_id,username,permission_code,method,path,request_id,ip,http_status,success,duration_ms,request_body,error_message)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, v.AdminUserID, v.Username, v.PermissionCode, v.Method, v.Path, v.RequestID, v.IP, v.HTTPStatus, v.Success, v.DurationMS, v.RequestBody, v.ErrorMessage)
	return err
}

func (r *Repository) AdminAudits(ctx context.Context, adminID int64, permission string, start, end *time.Time, page, pageSize int64) ([]model.AdminAudit, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	where := " WHERE 1=1"
	args := make([]any, 0)
	if adminID > 0 {
		where += " AND admin_user_id=?"
		args = append(args, adminID)
	}
	if permission != "" {
		where += " AND permission_code=?"
		args = append(args, permission)
	}
	if start != nil {
		where += " AND created_at>=?"
		args = append(args, *start)
	}
	if end != nil {
		where += " AND created_at<=?"
		args = append(args, *end)
	}
	var total int64
	if err := r.UserDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_audit_log"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append(append([]any{}, args...), pageSize, (page-1)*pageSize)
	rows, err := r.UserDB.QueryContext(ctx, `SELECT id,admin_user_id,username,permission_code,method,path,request_id,ip,http_status,success,duration_ms,request_body,error_message,created_at FROM admin_audit_log`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]model.AdminAudit, 0)
	for rows.Next() {
		var v model.AdminAudit
		if err := rows.Scan(&v.ID, &v.AdminUserID, &v.Username, &v.PermissionCode, &v.Method, &v.Path, &v.RequestID, &v.IP, &v.HTTPStatus, &v.Success, &v.DurationMS, &v.RequestBody, &v.ErrorMessage, &v.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, v)
	}
	return items, total, rows.Err()
}

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
