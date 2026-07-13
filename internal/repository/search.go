package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gin-looklook/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		parts = append(parts, "user_id = ?")
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

	db := r.TravelDB.WithContext(ctx).Where("del_state = 0")
	if scopeSQL != "" {
		db = db.Where(scopeSQL, scopeArgs...)
	}

	var total int64
	if err := db.Model(&model.Homestay{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.Homestay
	err := db.Order("id DESC").Limit(int(pageSize)).Offset(int((page - 1) * pageSize)).Find(&items).Error
	return items, total, err
}

func (r *Repository) UpdateAdminHomestay(ctx context.Context, auth *model.AdminAuthorization, v *model.Homestay) error {
	return r.TravelDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		checkDB := tx.Where("id = ? AND del_state = 0", v.ID)
		scopeSQL, scopeArgs := scopeCondition(auth)
		if scopeSQL != "" {
			checkDB = checkDB.Where(scopeSQL, scopeArgs...)
		}
		var currentVersion int64
		if err := checkDB.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("version").First(&model.Homestay{}).Scan(&currentVersion).Error; err != nil {
			return err
		}
		if currentVersion != v.Version {
			return fmt.Errorf("homestay version conflict")
		}

		result := tx.Model(v).
			Where("id = ? AND version = ?", v.ID, v.Version).
			Updates(map[string]any{
				"title":                v.Title,
				"sub_title":            v.SubTitle,
				"banner":               v.Banner,
				"info":                 v.Info,
				"city":                 v.City,
				"tags":                 v.Tags,
				"star":                 v.Star,
				"latitude":             v.Latitude,
				"longitude":            v.Longitude,
				"people_num":           v.PeopleNum,
				"row_state":            v.RowState,
				"row_type":             v.RowType,
				"food_info":            v.FoodInfo,
				"food_price":           v.FoodPrice,
				"homestay_price":       v.HomestayPrice,
				"market_homestay_price": v.MarketHomestayPrice,
				"version":              gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fmt.Errorf("homestay not found or version conflict")
		}

		eventKey := fmt.Sprintf("homestay:%d:v%d", v.ID, v.Version+1)
		if err := tx.Exec("INSERT INTO search_event_outbox(event_key,aggregate_id,event_type) VALUES(?,?,?)", eventKey, v.ID, "upsert").Error; err != nil {
			return err
		}
		return nil
	})
}

// ── Search Outbox ──

func (r *Repository) PendingSearchOutbox(ctx context.Context, limit int64) ([]model.SearchOutboxEvent, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var items []model.SearchOutboxEvent
	err := r.TravelDB.WithContext(ctx).
		Where("status = 0 AND next_retry_at <= NOW()").
		Order("id").
		Limit(int(limit)).
		Find(&items).Error
	return items, err
}

func (r *Repository) MarkSearchOutboxPublished(ctx context.Context, id int64) error {
	return r.TravelDB.WithContext(ctx).
		Table("search_event_outbox").
		Where("id = ? AND status = 0", id).
		Updates(map[string]any{"status": 1, "published_at": gorm.Expr("NOW()"), "last_error": ""}).Error
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
	return r.TravelDB.WithContext(ctx).
		Table("search_event_outbox").
		Where("id = ? AND status = 0", id).
		Updates(map[string]any{
			"retry_count":  gorm.Expr("retry_count + 1"),
			"next_retry_at": time.Now().Add(delay),
			"last_error":    message,
		}).Error
}

func (r *Repository) BootstrapSearchOutbox(ctx context.Context) error {
	return r.TravelDB.WithContext(ctx).Exec(
		"INSERT IGNORE INTO search_event_outbox(event_key,aggregate_id,event_type) SELECT CONCAT('bootstrap:',id,':v',version),id,'upsert' FROM homestay WHERE del_state=0",
	).Error
}

func (r *Repository) RebuildSearchOutbox(ctx context.Context, token string) (int64, error) {
	result := r.TravelDB.WithContext(ctx).Exec(
		"INSERT IGNORE INTO search_event_outbox(event_key,aggregate_id,event_type) SELECT CONCAT('rebuild:',?,':',id),id,'upsert' FROM homestay WHERE del_state=0", token,
	)
	return result.RowsAffected, result.Error
}

func (r *Repository) HomestayForIndex(ctx context.Context, id int64) (*model.Homestay, error) {
	var v model.Homestay
	err := r.TravelDB.WithContext(ctx).Where("id = ? AND del_state = 0", id).First(&v).Error
	return &v, err
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
