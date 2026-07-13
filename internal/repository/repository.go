package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gin-looklook/internal/config"
	"gin-looklook/internal/model"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

var (
	ErrNotFound       = gorm.ErrRecordNotFound
	ErrSeckillSoldOut = errors.New("seckill sold out")
)

type Repository struct {
	UserDB         *gorm.DB
	TravelDB       *gorm.DB
	OrderDB        *gorm.DB
	PaymentDB      *gorm.DB
	Redis          *redis.Client
	userFlight     singleflight.Group
	homestayFlight singleflight.Group
}

func Open(ctx context.Context, c config.Config) (*Repository, error) {
	gormCfg := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy:                           NamingStrategy,
	}

	open := func(dsn string) (*gorm.DB, error) {
		db, err := gorm.Open(mysql.Open(dsn), gormCfg)
		if err != nil {
			return nil, err
		}
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		sqlDB.SetMaxOpenConns(30)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(3 * time.Minute)
		if err := sqlDB.PingContext(ctx); err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
		return db, nil
	}

	userDB, err := open(c.UserDSN)
	if err != nil {
		return nil, fmt.Errorf("open user db: %w", err)
	}
	travelDB, err := open(c.TravelDSN)
	if err != nil {
		closeDB(userDB)
		return nil, fmt.Errorf("open travel db: %w", err)
	}
	orderDB, err := open(c.OrderDSN)
	if err != nil {
		closeDB(userDB)
		closeDB(travelDB)
		return nil, fmt.Errorf("open order db: %w", err)
	}
	paymentDB, err := open(c.PaymentDSN)
	if err != nil {
		closeDB(userDB)
		closeDB(travelDB)
		closeDB(orderDB)
		return nil, fmt.Errorf("open payment db: %w", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: c.RedisAddr, Password: c.RedisPassword})
	if err := rdb.Ping(ctx).Err(); err != nil {
		closeDB(userDB)
		closeDB(travelDB)
		closeDB(orderDB)
		closeDB(paymentDB)
		return nil, fmt.Errorf("open redis: %w", err)
	}
	return &Repository{UserDB: userDB, TravelDB: travelDB, OrderDB: orderDB, PaymentDB: paymentDB, Redis: rdb}, nil
}

func closeDB(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

// NamingStrategy uses singular table names to match the existing schema.
var NamingStrategy = schema.NamingStrategy{SingularTable: true}

func (r *Repository) Close() {
	closeDB(r.UserDB)
	closeDB(r.TravelDB)
	closeDB(r.OrderDB)
	closeDB(r.PaymentDB)
	_ = r.Redis.Close()
}

// ── User ──

func (r *Repository) UserByMobile(ctx context.Context, mobile string) (*model.User, error) {
	var v model.User
	err := r.UserDB.WithContext(ctx).Where("mobile = ? AND del_state = 0", mobile).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repository) UserByID(ctx context.Context, id int64) (*model.User, error) {
	key := fmt.Sprintf("gin:looklook:v2:user:%d", id)
	if data, err := r.Redis.Get(ctx, key).Bytes(); err == nil {
		var v model.User
		if json.Unmarshal(data, &v) == nil {
			return &v, nil
		}
	}
	loaded, err, _ := r.userFlight.Do(key, func() (any, error) {
		var v model.User
		err := r.UserDB.WithContext(ctx).Where("id = ? AND del_state = 0", id).First(&v).Error
		if err != nil {
			return nil, err
		}
		if data, err := json.Marshal(v); err == nil {
			_ = r.Redis.Set(ctx, key, data, 10*time.Minute).Err()
		}
		return &v, nil
	})
	if err != nil {
		return nil, err
	}
	return loaded.(*model.User), nil
}

func (r *Repository) UserAuthByUser(ctx context.Context, userID int64, authType string) (*model.UserAuth, error) {
	var v model.UserAuth
	err := r.UserDB.WithContext(ctx).Where("user_id = ? AND auth_type = ? AND del_state = 0", userID, authType).First(&v).Error
	return &v, err
}

func (r *Repository) UserAuthByKey(ctx context.Context, authType, authKey string) (*model.UserAuth, error) {
	var v model.UserAuth
	err := r.UserDB.WithContext(ctx).Where("auth_type = ? AND auth_key = ? AND del_state = 0", authType, authKey).First(&v).Error
	return &v, err
}

func (r *Repository) CreateUser(ctx context.Context, user *model.User, auth *model.UserAuth) (int64, error) {
	err := r.UserDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		auth.UserID = user.ID
		return tx.Create(auth).Error
	})
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}

// ── Homestay ──

func (r *Repository) HomestayByID(ctx context.Context, id int64) (*model.Homestay, error) {
	key := fmt.Sprintf("gin:looklook:v2:homestay:%d", id)
	if data, err := r.Redis.Get(ctx, key).Bytes(); err == nil {
		var v model.Homestay
		if json.Unmarshal(data, &v) == nil {
			return &v, nil
		}
	}
	loaded, err, _ := r.homestayFlight.Do(key, func() (any, error) {
		var v model.Homestay
		err := r.TravelDB.WithContext(ctx).Where("id = ? AND del_state = 0", id).First(&v).Error
		if err != nil {
			return nil, err
		}
		if data, err := json.Marshal(v); err == nil {
			_ = r.Redis.Set(ctx, key, data, 10*time.Minute).Err()
		}
		return &v, nil
	})
	if err != nil {
		return nil, err
	}
	return loaded.(*model.Homestay), nil
}

func (r *Repository) HomestaysByActivity(ctx context.Context, rowType string, page, pageSize int64) ([]model.Homestay, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var out []model.Homestay
	err := r.TravelDB.WithContext(ctx).
		Table("homestay h").
		Select("h.*").
		Joins("JOIN homestay_activity a ON a.data_id = h.id").
		Where("a.row_type = ? AND a.row_status = 1 AND a.del_state = 0 AND h.del_state = 0", rowType).
		Order("a.data_id DESC").
		Limit(int(pageSize)).
		Offset(int((page - 1) * pageSize)).
		Find(&out).Error
	return out, err
}

func (r *Repository) HomestaysByBusiness(ctx context.Context, businessID, lastID, pageSize int64) ([]model.Homestay, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	db := r.TravelDB.WithContext(ctx).Where("homestay_business_id = ? AND del_state = 0", businessID)
	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}
	var out []model.Homestay
	err := db.Order("id DESC").Limit(int(pageSize)).Find(&out).Error
	return out, err
}

func (r *Repository) GuessHomestays(ctx context.Context) ([]model.Homestay, error) {
	var out []model.Homestay
	err := r.TravelDB.WithContext(ctx).Where("del_state = 0").Order("id DESC").Limit(5).Find(&out).Error
	return out, err
}

func (r *Repository) Businesses(ctx context.Context, lastID, pageSize int64) ([]model.HomestayBusiness, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	db := r.TravelDB.WithContext(ctx).Where("del_state = 0")
	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}
	var out []model.HomestayBusiness
	err := db.Order("id DESC").Limit(int(pageSize)).Find(&out).Error
	return out, err
}

func (r *Repository) BusinessByID(ctx context.Context, id int64) (*model.HomestayBusiness, error) {
	var v model.HomestayBusiness
	err := r.TravelDB.WithContext(ctx).Where("id = ? AND del_state = 0", id).First(&v).Error
	return &v, err
}

func (r *Repository) GoodBossUserIDs(ctx context.Context) ([]int64, error) {
	var out []int64
	err := r.TravelDB.WithContext(ctx).
		Table("homestay_activity").
		Select("data_id").
		Where("row_type = 'goodBusiness' AND row_status = 1 AND del_state = 0").
		Order("data_id DESC").
		Limit(10).
		Pluck("data_id", &out).Error
	return out, err
}

func (r *Repository) Comments(ctx context.Context, lastID, pageSize int64) ([]model.HomestayComment, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	db := r.TravelDB.WithContext(ctx).Where("del_state = 0")
	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}
	var out []model.HomestayComment
	err := db.Order("id DESC").Limit(int(pageSize)).Find(&out).Error
	return out, err
}

// ── Order ──

func (r *Repository) CreateOrder(ctx context.Context, v *model.HomestayOrder) error {
	return r.OrderDB.WithContext(ctx).Create(v).Error
}

func (r *Repository) OrderBySN(ctx context.Context, sn string) (*model.HomestayOrder, error) {
	var v model.HomestayOrder
	err := r.OrderDB.WithContext(ctx).Where("sn = ? AND del_state = 0", sn).First(&v).Error
	return &v, err
}

func (r *Repository) OrdersByUser(ctx context.Context, userID, lastID, pageSize, tradeState int64) ([]model.HomestayOrder, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	db := r.OrderDB.WithContext(ctx).Where("user_id = ? AND del_state = 0", userID)
	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}
	if tradeState >= -1 && tradeState <= 4 {
		db = db.Where("trade_state = ?", tradeState)
	}
	var out []model.HomestayOrder
	err := db.Order("id DESC").Limit(int(pageSize)).Find(&out).Error
	return out, err
}

func (r *Repository) UpdateOrderState(ctx context.Context, id, oldVersion, newState int64) error {
	result := r.OrderDB.WithContext(ctx).
		Model(&model.HomestayOrder{}).
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

// ── Seckill ──

func (r *Repository) ActiveSeckillActivities(ctx context.Context) ([]model.SeckillActivity, error) {
	var out []model.SeckillActivity
	err := r.OrderDB.WithContext(ctx).
		Where("status = 1 AND end_time > NOW()").
		Order("start_time, id").
		Find(&out).Error
	return out, err
}

func (r *Repository) SeckillActivityByID(ctx context.Context, id int64) (*model.SeckillActivity, error) {
	var v model.SeckillActivity
	err := r.OrderDB.WithContext(ctx).Where("id = ?", id).First(&v).Error
	return &v, err
}

// CreateSeckillOrder uses MySQL as the final oversell and idempotency barrier.
func (r *Repository) CreateSeckillOrder(ctx context.Context, reservation model.SeckillReservation, order *model.HomestayOrder) (string, bool, bool, error) {
	var orderSN string
	var existed, restoreStock bool

	err := r.OrderDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check existing by reservation SN
		var existing struct {
			OrderSN string `gorm:"column:order_sn"`
		}
		err := tx.Table("seckill_order").
			Select("order_sn").
			Where("reservation_sn = ?", reservation.ReservationSN).
			First(&existing).Error
		if err == nil && existing.OrderSN != "" {
			orderSN = existing.OrderSN
			existed = true
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// Insert reservation record
		res := tx.Exec("INSERT INTO seckill_order(reservation_sn,activity_id,user_id,status) VALUES(?,?,?,0)",
			reservation.ReservationSN, reservation.ActivityID, reservation.UserID)
		if res.Error != nil {
			if !errors.Is(res.Error, gorm.ErrDuplicatedKey) {
				return res.Error
			}
			// Duplicate key — check if it was already processed
			var dup struct{ OrderSN string }
			if err := tx.Table("seckill_order").
				Select("order_sn").
				Where("activity_id = ? AND user_id = ?", reservation.ActivityID, reservation.UserID).
				First(&dup).Error; err != nil {
				return err
			}
			if dup.OrderSN == "" {
				return errors.New("seckill order is still processing")
			}
			orderSN = dup.OrderSN
			existed = true
			restoreStock = true
			return nil
		}

		// Decrement stock with condition
		stockResult := tx.Model(&model.SeckillActivity{}).
			Where("id = ? AND status = 1 AND sold_count < stock", reservation.ActivityID).
			Update("sold_count", gorm.Expr("sold_count + 1"))
		if stockResult.Error != nil {
			return stockResult.Error
		}
		if stockResult.RowsAffected == 0 {
			return ErrSeckillSoldOut
		}

		// Insert order
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// Link reservation to order
		if err := tx.Table("seckill_order").
			Where("reservation_sn = ?", reservation.ReservationSN).
			Updates(map[string]any{"status": 1, "order_sn": order.SN}).Error; err != nil {
			return err
		}
		orderSN = order.SN
		return nil
	})
	return orderSN, existed, restoreStock, err
}

// ── Payment ──

func (r *Repository) CreatePayment(ctx context.Context, v *model.ThirdPayment) error {
	return r.PaymentDB.WithContext(ctx).Create(v).Error
}

func (r *Repository) PaymentBySN(ctx context.Context, sn string) (*model.ThirdPayment, error) {
	var v model.ThirdPayment
	err := r.PaymentDB.WithContext(ctx).Where("sn = ? AND del_state = 0", sn).First(&v).Error
	return &v, err
}

func (r *Repository) PaymentByOrder(ctx context.Context, orderSN string) (*model.ThirdPayment, error) {
	var v model.ThirdPayment
	err := r.PaymentDB.WithContext(ctx).
		Where("order_sn = ? AND pay_status IN (1,2) AND del_state = 0", orderSN).
		Order("id DESC").
		First(&v).Error
	return &v, err
}

func (r *Repository) UpdatePayment(ctx context.Context, v *model.ThirdPayment) error {
	result := r.PaymentDB.WithContext(ctx).
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

// UpdatePaymentWithOutbox commits the aggregate update and integration event atomically.
func (r *Repository) UpdatePaymentWithOutbox(ctx context.Context, v *model.ThirdPayment, event model.OutboxEvent) error {
	return r.PaymentDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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

// ── Outbox ──

func (r *Repository) PendingOutbox(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	if limit < 1 {
		limit = 100
	}
	var out []model.OutboxEvent
	err := r.PaymentDB.WithContext(ctx).
		Where("status = 0 AND next_retry_at <= NOW()").
		Order("id").
		Limit(limit).
		Find(&out).Error
	return out, err
}

func (r *Repository) MarkOutboxPublished(ctx context.Context, id int64) error {
	return r.PaymentDB.WithContext(ctx).
		Table("event_outbox").
		Where("id = ? AND status = 0", id).
		Updates(map[string]any{"status": 1, "published_at": gorm.Expr("NOW()")}).Error
}

func (r *Repository) RetryOutbox(ctx context.Context, id int64) error {
	return r.PaymentDB.WithContext(ctx).
		Table("event_outbox").
		Where("id = ? AND status = 0", id).
		Updates(map[string]any{
			"retry_count":  gorm.Expr("retry_count + 1"),
			"next_retry_at": clause.Expr{SQL: "DATE_ADD(NOW(), INTERVAL LEAST(60, POW(2, LEAST(retry_count,5))) SECOND)"},
		}).Error
}
