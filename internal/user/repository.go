package user

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

func (r *Repository) UserByMobile(ctx context.Context, mobile string) (*User, error) {
	var v User
	err := r.DB.WithContext(ctx).Where("mobile = ? AND del_state = 0", mobile).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *Repository) UserByID(ctx context.Context, id int64) (*User, error) {
	key := fmt.Sprintf("gin:looklook:v2:user:%d", id)
	if data, err := r.Redis.Get(ctx, key).Bytes(); err == nil {
		var v User
		if json.Unmarshal(data, &v) == nil {
			return &v, nil
		}
	}
	loaded, err, _ := r.flight.Do(key, func() (any, error) {
		var v User
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
	return loaded.(*User), nil
}

func (r *Repository) UserAuthByUser(ctx context.Context, userID int64, authType string) (*UserAuth, error) {
	var v UserAuth
	err := r.DB.WithContext(ctx).Where("user_id = ? AND auth_type = ? AND del_state = 0", userID, authType).First(&v).Error
	return &v, err
}

func (r *Repository) UserAuthByKey(ctx context.Context, authType, authKey string) (*UserAuth, error) {
	var v UserAuth
	err := r.DB.WithContext(ctx).Where("auth_type = ? AND auth_key = ? AND del_state = 0", authType, authKey).First(&v).Error
	return &v, err
}

func (r *Repository) CreateUser(ctx context.Context, user *User, auth *UserAuth) (int64, error) {
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		auth.UserID = user.ID
		return tx.Create(auth).Error
	})
	if err != nil {
		return 0, err
	}
	slog.InfoContext(
		ctx,
		"created user",
		"userID", user.ID,
	)
	return user.ID, nil
}

func (r *Repository) UpdateUser(ctx context.Context, user *User) error {
	return r.DB.WithContext(ctx).Model(user).Updates(user).Error
}
