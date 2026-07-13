package order

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var (
	ErrSeckillSoldOut = errors.New("seckill sold out")
)

type Repository struct {
	DB *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{DB: db} }

func (r *Repository) CreateOrder(ctx context.Context, v *HomestayOrder) error {
	return r.DB.WithContext(ctx).Create(v).Error
}

func (r *Repository) OrderBySN(ctx context.Context, sn string) (*HomestayOrder, error) {
	var v HomestayOrder
	err := r.DB.WithContext(ctx).Where("sn = ? AND del_state = 0", sn).First(&v).Error
	return &v, err
}

func (r *Repository) OrdersByUser(ctx context.Context, userID, lastID, pageSize, tradeState int64) ([]HomestayOrder, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	db := r.DB.WithContext(ctx).Where("user_id = ? AND del_state = 0", userID)
	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}
	if tradeState >= -1 && tradeState <= 4 {
		db = db.Where("trade_state = ?", tradeState)
	}
	var out []HomestayOrder
	err := db.Order("id DESC").Limit(int(pageSize)).Find(&out).Error
	return out, err
}

func (r *Repository) UpdateOrderState(ctx context.Context, id, oldVersion, newState int64) error {
	result := r.DB.WithContext(ctx).
		Model(&HomestayOrder{}).
		Where("id = ? AND version = ?", id, oldVersion).
		Updates(map[string]any{"trade_state": newState, "version": gorm.Expr("version + 1")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("optimistic lock conflict")
	}
	return nil
}

type SeckillActivity struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	HomestayID int64     `gorm:"column:homestay_id"`
	Title      string    `gorm:"column:title"`
	Price      int64     `gorm:"column:price"`
	Stock      int64     `gorm:"column:stock"`
	SoldCount  int64     `gorm:"column:sold_count"`
	StartTime  time.Time `gorm:"column:start_time"`
	EndTime    time.Time `gorm:"column:end_time"`
	Status     int64     `gorm:"column:status"`
}

func (r *Repository) SeckillActivityByID(ctx context.Context, id int64) (*SeckillActivity, error) {
	var v SeckillActivity
	err := r.DB.WithContext(ctx).Where("id = ?", id).First(&v).Error
	return &v, err
}

func (r *Repository) ActiveSeckillActivities(ctx context.Context) ([]SeckillActivity, error) {
	var out []SeckillActivity
	err := r.DB.WithContext(ctx).
		Where("status = 1 AND end_time > NOW()").
		Order("start_time, id").
		Find(&out).Error
	return out, err
}

func (r *Repository) CreateSeckillOrder(ctx context.Context, reservationSN string, activityID, userID int64, order *HomestayOrder) (orderSN string, existed, restoreStock bool, err error) {
	err = r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing struct {
			OrderSN string `gorm:"column:order_sn"`
		}
		e := tx.Table("seckill_order").
			Select("order_sn").
			Where("reservation_sn = ?", reservationSN).
			First(&existing).Error
		if e == nil && existing.OrderSN != "" {
			orderSN = existing.OrderSN
			existed = true
			return nil
		}
		if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return e
		}
		res := tx.Exec("INSERT INTO seckill_order(reservation_sn,activity_id,user_id,status) VALUES(?,?,?,0)",
			reservationSN, activityID, userID)
		if res.Error != nil {
			if !errors.Is(res.Error, gorm.ErrDuplicatedKey) {
				return res.Error
			}
			var dup struct{ OrderSN string }
			if e2 := tx.Table("seckill_order").
				Select("order_sn").
				Where("activity_id = ? AND user_id = ?", activityID, userID).
				First(&dup).Error; e2 != nil {
				return e2
			}
			if dup.OrderSN == "" {
				return errors.New("seckill order is still processing")
			}
			orderSN = dup.OrderSN
			existed = true
			restoreStock = true
			return nil
		}
		stockResult := tx.Model(&SeckillActivity{}).
			Where("id = ? AND status = 1 AND sold_count < stock", activityID).
			Update("sold_count", gorm.Expr("sold_count + 1"))
		if stockResult.Error != nil {
			return stockResult.Error
		}
		if stockResult.RowsAffected == 0 {
			return ErrSeckillSoldOut
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		if err := tx.Table("seckill_order").
			Where("reservation_sn = ?", reservationSN).
			Updates(map[string]any{"status": 1, "order_sn": order.SN}).Error; err != nil {
			return err
		}
		orderSN = order.SN
		return nil
	})
	return
}
