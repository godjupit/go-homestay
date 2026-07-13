package search

import (
	"context"
	"fmt"
	"time"

	"gin-looklook/internal/travel"

	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

func (r *Repository) HomestayForIndex(ctx context.Context, id int64) (*travel.Homestay, error) {
	var v travel.Homestay
	err := r.DB.WithContext(ctx).Where("id = ? AND del_state = 0", id).First(&v).Error
	return &v, err
}

func (r *Repository) PendingSearchOutbox(ctx context.Context, limit int64) ([]OutboxEvent, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	var items []OutboxEvent
	err := r.DB.WithContext(ctx).
		Where("status = 0 AND next_retry_at <= NOW()").
		Order("id").
		Limit(int(limit)).
		Find(&items).Error
	return items, err
}

func (r *Repository) MarkSearchOutboxPublished(ctx context.Context, id int64) error {
	return r.DB.WithContext(ctx).
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
	return r.DB.WithContext(ctx).
		Table("search_event_outbox").
		Where("id = ? AND status = 0", id).
		Updates(map[string]any{
			"retry_count":   gorm.Expr("retry_count + 1"),
			"next_retry_at": time.Now().Add(delay),
			"last_error":    message,
		}).Error
}

func (r *Repository) BootstrapSearchOutbox(ctx context.Context) error {
	return r.DB.WithContext(ctx).Exec(
		"INSERT IGNORE INTO search_event_outbox(event_key,aggregate_id,event_type) SELECT CONCAT('bootstrap:',id,':v',version),id,'upsert' FROM homestay WHERE del_state=0",
	).Error
}

func (r *Repository) RebuildSearchOutbox(ctx context.Context, token string) (int64, error) {
	result := r.DB.WithContext(ctx).Exec(
		"INSERT IGNORE INTO search_event_outbox(event_key,aggregate_id,event_type) SELECT CONCAT('rebuild:',?,':',id),id,'upsert' FROM homestay WHERE del_state=0", token,
	)
	return result.RowsAffected, result.Error
}

func (r *Repository) UpdateAdminHomestay(ctx context.Context, v *travel.Homestay, scopeSQL string, scopeArgs []any) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(v).
			Where("id = ? AND version = ?", v.ID, v.Version).
			Where(scopeSQL, scopeArgs...).
			Updates(map[string]any{
				"title": v.Title, "sub_title": v.SubTitle, "banner": v.Banner,
				"info": v.Info, "city": v.City, "tags": v.Tags, "star": v.Star,
				"latitude": v.Latitude, "longitude": v.Longitude, "people_num": v.PeopleNum,
				"row_state": v.RowState, "row_type": v.RowType,
				"food_info": v.FoodInfo, "food_price": v.FoodPrice,
				"homestay_price": v.HomestayPrice, "market_homestay_price": v.MarketHomestayPrice,
				"version": gorm.Expr("version + 1"),
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

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
