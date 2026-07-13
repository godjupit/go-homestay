package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"

	"gin-looklook/internal/model"
	"gin-looklook/internal/platform"
	"gin-looklook/internal/repository"
)

const (
	ActivityPreferred    = "preferredHomestay"
	ActivityGoodBusiness = "goodBusiness"
)

type TravelService struct {
	repo  *repository.Repository
	users *UserService
}

func NewTravelService(repo *repository.Repository, users *UserService) *TravelService {
	return &TravelService{repo: repo, users: users}
}
func (s *TravelService) Homestay(ctx context.Context, id int64) (*model.Homestay, error) {
	v, err := s.repo.HomestayByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, platform.E(platform.CodeCommon, "This record does not exist", nil)
	}
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}
func (s *TravelService) HomestayList(ctx context.Context, page, pageSize int64) ([]model.Homestay, error) {
	v, err := s.repo.HomestaysByActivity(ctx, ActivityPreferred, page, pageSize)
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}
func (s *TravelService) BusinessHomestays(ctx context.Context, businessID, lastID, pageSize int64) ([]model.Homestay, error) {
	v, err := s.repo.HomestaysByBusiness(ctx, businessID, lastID, pageSize)
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}
func (s *TravelService) Guess(ctx context.Context) ([]model.Homestay, error) {
	v, err := s.repo.GuessHomestays(ctx)
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}
func (s *TravelService) Businesses(ctx context.Context, lastID, pageSize int64) ([]model.HomestayBusiness, error) {
	v, err := s.repo.Businesses(ctx, lastID, pageSize)
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}
func (s *TravelService) BusinessBoss(ctx context.Context, id int64) (*model.User, error) {
	b, err := s.repo.BusinessByID(ctx, id)
	if err == sql.ErrNoRows {
		return &model.User{}, nil
	}
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return s.users.User(ctx, b.UserID)
}
func (s *TravelService) GoodBosses(ctx context.Context) ([]model.User, error) {
	ids, err := s.repo.GoodBossUserIDs(ctx)
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	out := make([]model.User, 0, len(ids))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, id := range ids {
		id := id
		wg.Add(1)
		go func() {
			defer wg.Done()
			if u, e := s.users.User(ctx, id); e == nil {
				mu.Lock()
				out = append(out, *u)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return out, nil
}
func (s *TravelService) Comments(ctx context.Context, lastID, pageSize int64) ([]model.HomestayComment, error) {
	v, err := s.repo.Comments(ctx, lastID, pageSize)
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}
func ParseStar(raw []byte) float64 {
	var value float64
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	var obj map[string]float64
	if json.Unmarshal(raw, &obj) == nil {
		var total float64
		for _, v := range obj {
			total += v
		}
		if len(obj) > 0 {
			return total / float64(len(obj))
		}
	}
	return 0
}
