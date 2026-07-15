package payment

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	DB *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

func (r *Repository) CreatePayment(ctx context.Context, v *ThirdPayment) error {
	return r.DB.WithContext(ctx).Create(v).Error
}

func (r *Repository) PaymentBySN(ctx context.Context, sn string) (*ThirdPayment, error) {
	var v ThirdPayment
	err := r.DB.WithContext(ctx).Where("sn = ? AND del_state = 0", sn).First(&v).Error
	return &v, err
}

func (r *Repository) PaymentByOrder(ctx context.Context, orderSN string) (*ThirdPayment, error) {
	var v ThirdPayment
	err := r.DB.WithContext(ctx).
		Where("order_sn = ? AND pay_status IN (1,2) AND del_state = 0", orderSN).
		Order("id DESC").
		First(&v).Error
	return &v, err
}

func (r *Repository) UpdatePayment(ctx context.Context, v *ThirdPayment) error {
	result := r.DB.WithContext(ctx).
		Model(v).
		Where("id = ? AND version = ?", v.ID, v.Version).
		Updates(map[string]any{
			"trade_type":       v.TradeType,
			"trade_state":      v.TradeState,
			"transaction_id":   v.TransactionID,
			"trade_state_desc": v.TradeStateDesc,
			"pay_status":       v.PayStatus,
			"pay_time":         v.PayTime,
			"version":          gorm.Expr("version + 1"),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("optimistic lock conflict")
	}
	v.Version++
	return nil
}

func (r *Repository) UpdatePaymentWithOutbox(ctx context.Context, v *ThirdPayment, event OutboxEvent) error {
	return r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(v).
			Where("id = ? AND version = ?", v.ID, v.Version).
			Updates(map[string]any{
				"trade_type":       v.TradeType,
				"trade_state":      v.TradeState,
				"transaction_id":   v.TransactionID,
				"trade_state_desc": v.TradeStateDesc,
				"pay_status":       v.PayStatus,
				"pay_time":         v.PayTime,
				"version":          gorm.Expr("version + 1"),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("optimistic lock conflict")
		}
		if err := tx.Create(&event).Error; err != nil {
			return err
		}
		v.Version++
		return nil
	})
}

func (r *Repository) PendingOutbox(ctx context.Context, limit int) ([]OutboxEvent, error) {
	if limit < 1 {
		limit = 100
	}
	var out []OutboxEvent
	err := r.DB.WithContext(ctx).
		Where("status = 0 AND next_retry_at <= NOW()").
		Order("id").
		Limit(limit).
		Find(&out).Error
	return out, err
}

func (r *Repository) OutboxStats(ctx context.Context) (int, time.Duration, error) {
	var stats struct {
		Pending int        `gorm:"column:pending"`
		Oldest  *time.Time `gorm:"column:oldest"`
	}
	if err := r.DB.WithContext(ctx).
		Table("event_outbox").
		Select("COUNT(*) AS pending, MIN(created_at) AS oldest").
		Where("status = 0").
		Scan(&stats).Error; err != nil {
		return 0, 0, err
	}
	if stats.Oldest == nil {
		return stats.Pending, 0, nil
	}
	age := time.Since(*stats.Oldest)
	if age < 0 {
		age = 0
	}
	return stats.Pending, age, nil
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, id int64) error {
	return r.DB.WithContext(ctx).
		Table("event_outbox").
		Where("id = ? AND status = 0", id).
		Updates(map[string]any{"status": 1, "published_at": gorm.Expr("NOW()")}).Error
}

func (r *Repository) RetryOutbox(ctx context.Context, id int64) error {
	return r.DB.WithContext(ctx).
		Table("event_outbox").
		Where("id = ? AND status = 0", id).
		Updates(map[string]any{
			"retry_count":   gorm.Expr("retry_count + 1"),
			"next_retry_at": clause.Expr{SQL: "DATE_ADD(NOW(), INTERVAL LEAST(60, POW(2, LEAST(retry_count,5))) SECOND)"},
		}).Error
}
