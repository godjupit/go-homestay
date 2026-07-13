package travel

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type Repository struct {
	DB     *gorm.DB
	Redis  *redis.Client
	flight singleflight.Group
}

func NewRepository(db *gorm.DB, rdb *redis.Client) *Repository {
	return &Repository{DB: db, Redis: rdb}
}

func (r *Repository) HomestayByID(ctx context.Context, id int64) (*Homestay, error) {
	key := fmt.Sprintf("gin:looklook:v2:homestay:%d", id)
	if data, err := r.Redis.Get(ctx, key).Bytes(); err == nil {
		var v Homestay
		if json.Unmarshal(data, &v) == nil {
			return &v, nil
		}
	}
	loaded, err, _ := r.flight.Do(key, func() (any, error) {
		var v Homestay
		err := r.DB.WithContext(ctx).Where("id = ? AND del_state = 0", id).First(&v).Error
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
	return loaded.(*Homestay), nil
}

func (r *Repository) HomestaysByActivity(ctx context.Context, rowType string, page, pageSize int64) ([]Homestay, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	var out []Homestay
	err := r.DB.WithContext(ctx).
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

func (r *Repository) HomestaysByBusiness(ctx context.Context, businessID, lastID, pageSize int64) ([]Homestay, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	db := r.DB.WithContext(ctx).Where("homestay_business_id = ? AND del_state = 0", businessID)
	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}
	var out []Homestay
	err := db.Order("id DESC").Limit(int(pageSize)).Find(&out).Error
	return out, err
}

func (r *Repository) GuessHomestays(ctx context.Context) ([]Homestay, error) {
	var out []Homestay
	err := r.DB.WithContext(ctx).Where("del_state = 0").Order("id DESC").Limit(5).Find(&out).Error
	return out, err
}

func (r *Repository) Businesses(ctx context.Context, lastID, pageSize int64) ([]HomestayBusiness, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	db := r.DB.WithContext(ctx).Where("del_state = 0")
	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}
	var out []HomestayBusiness
	err := db.Order("id DESC").Limit(int(pageSize)).Find(&out).Error
	return out, err
}

func (r *Repository) BusinessByID(ctx context.Context, id int64) (*HomestayBusiness, error) {
	var v HomestayBusiness
	err := r.DB.WithContext(ctx).Where("id = ? AND del_state = 0", id).First(&v).Error
	return &v, err
}

func (r *Repository) GoodBossUserIDs(ctx context.Context) ([]int64, error) {
	var out []int64
	err := r.DB.WithContext(ctx).
		Table("homestay_activity").
		Select("data_id").
		Where("row_type = 'goodBusiness' AND row_status = 1 AND del_state = 0").
		Order("data_id DESC").
		Limit(10).
		Pluck("data_id", &out).Error
	return out, err
}

func (r *Repository) Comments(ctx context.Context, lastID, pageSize int64) ([]HomestayComment, error) {
	if pageSize < 1 {
		pageSize = 10
	}
	db := r.DB.WithContext(ctx).Where("del_state = 0")
	if lastID > 0 {
		db = db.Where("id < ?", lastID)
	}
	var out []HomestayComment
	err := db.Order("id DESC").Limit(int(pageSize)).Find(&out).Error
	return out, err
}
