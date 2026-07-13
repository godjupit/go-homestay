package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gin-looklook/internal/model"
)

func scopeCondition(auth *model.AdminAuthorization) (string, []any) {
	if auth != nil && auth.AllData {
		return "", nil
	}
	parts := make([]string, 0, 2)
	args := make([]any, 0)
	if auth != nil && len(auth.BusinessIDs) > 0 {
		placeholders := make([]string, 0, len(auth.BusinessIDs))
		for _, id := range auth.BusinessIDs {
			placeholders = append(placeholders, "?")
			args = append(args, id)
		}
		parts = append(parts, "homestay_business_id IN ("+strings.Join(placeholders, ",")+")")
	}
	if auth != nil && auth.LinkedUserID > 0 {
		parts = append(parts, "user_id=?")
		args = append(args, auth.LinkedUserID)
	}
	if len(parts) == 0 {
		return " AND 1=0", nil
	}
	return " AND (" + strings.Join(parts, " OR ") + ")", args
}

func (r *Repository) AdminHomestays(ctx context.Context, auth *model.AdminAuthorization, page, pageSize int64) ([]model.Homestay, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	scopeSQL, scopeArgs := scopeCondition(auth)
	var total int64
	if err := r.TravelDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM homestay WHERE del_state=0"+scopeSQL, scopeArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	args := append(append([]any{}, scopeArgs...), pageSize, (page-1)*pageSize)
	rows, err := r.TravelDB.QueryContext(ctx, "SELECT "+homestayFields+" FROM homestay WHERE del_state=0"+scopeSQL+" ORDER BY id DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	items, err := scanHomestayRows(rows)
	return items, total, err
}

func (r *Repository) UpdateAdminHomestay(ctx context.Context, auth *model.AdminAuthorization, v *model.Homestay) error {
	tx, err := r.TravelDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	scopeSQL, scopeArgs := scopeCondition(auth)
	checkArgs := append([]any{v.ID}, scopeArgs...)
	var currentVersion int64
	if err = tx.QueryRowContext(ctx, "SELECT version FROM homestay WHERE id=? AND del_state=0"+scopeSQL+" FOR UPDATE", checkArgs...).Scan(&currentVersion); err != nil {
		return err
	}
	if currentVersion != v.Version {
		return fmt.Errorf("homestay version conflict")
	}
	res, err := tx.ExecContext(ctx, `UPDATE homestay SET title=?,sub_title=?,banner=?,info=?,city=?,tags=?,star=?,latitude=?,longitude=?,people_num=?,row_state=?,row_type=?,food_info=?,food_price=?,homestay_price=?,market_homestay_price=?,version=version+1 WHERE id=? AND version=?`,
		v.Title, v.SubTitle, v.Banner, v.Info, v.City, v.Tags, v.Star, v.Latitude, v.Longitude, v.PeopleNum, v.RowState, v.RowType, v.FoodInfo, v.FoodPrice, v.HomestayPrice, v.MarketHomestayPrice, v.ID, v.Version)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("homestay not found or version conflict")
	}
	eventKey := fmt.Sprintf("homestay:%d:v%d", v.ID, v.Version+1)
	if _, err = tx.ExecContext(ctx, "INSERT INTO search_event_outbox(event_key,aggregate_id,event_type) VALUES(?,?,?)", eventKey, v.ID, "upsert"); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	_ = r.Redis.Del(ctx, fmt.Sprintf("gin:looklook:v2:homestay:%d", v.ID)).Err()
	return nil
}

func (r *Repository) PendingSearchOutbox(ctx context.Context, limit int64) ([]model.SearchOutboxEvent, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := r.TravelDB.QueryContext(ctx, `SELECT id,event_key,aggregate_id,event_type,retry_count FROM search_event_outbox WHERE status=0 AND next_retry_at<=NOW() ORDER BY id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]model.SearchOutboxEvent, 0)
	for rows.Next() {
		var v model.SearchOutboxEvent
		if err := rows.Scan(&v.ID, &v.EventKey, &v.AggregateID, &v.EventType, &v.RetryCount); err != nil {
			return nil, err
		}
		items = append(items, v)
	}
	return items, rows.Err()
}

func (r *Repository) MarkSearchOutboxPublished(ctx context.Context, id int64) error {
	_, err := r.TravelDB.ExecContext(ctx, "UPDATE search_event_outbox SET status=1,published_at=NOW(),last_error='' WHERE id=? AND status=0", id)
	return err
}

func (r *Repository) RetrySearchOutbox(ctx context.Context, id, retryCount int64, cause error) error {
	delay := time.Second << minInt64(retryCount, 8)
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	message := ""
	if cause != nil {
		message = cause.Error()
		if len(message) > 512 {
			message = message[:512]
		}
	}
	_, err := r.TravelDB.ExecContext(ctx, "UPDATE search_event_outbox SET retry_count=retry_count+1,next_retry_at=?,last_error=? WHERE id=? AND status=0", time.Now().Add(delay), message, id)
	return err
}

func (r *Repository) BootstrapSearchOutbox(ctx context.Context) error {
	_, err := r.TravelDB.ExecContext(ctx, `INSERT IGNORE INTO search_event_outbox(event_key,aggregate_id,event_type)
		SELECT CONCAT('bootstrap:',id,':v',version),id,'upsert' FROM homestay WHERE del_state=0`)
	return err
}

func (r *Repository) RebuildSearchOutbox(ctx context.Context, token string) (int64, error) {
	res, err := r.TravelDB.ExecContext(ctx, `INSERT IGNORE INTO search_event_outbox(event_key,aggregate_id,event_type)
		SELECT CONCAT('rebuild:',?,':',id),id,'upsert' FROM homestay WHERE del_state=0`, token)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *Repository) HomestayForIndex(ctx context.Context, id int64) (*model.Homestay, error) {
	return scanHomestay(r.TravelDB.QueryRowContext(ctx, "SELECT "+homestayFields+" FROM homestay WHERE id=? AND del_state=0 LIMIT 1", id))
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
