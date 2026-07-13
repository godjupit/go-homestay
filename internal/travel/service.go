package travel

import (
	"context"
	"sync"

	"gin-looklook/internal/shared"
	"gin-looklook/internal/user"

	"gorm.io/gorm"
)

const (
	ActivityPreferred    = "preferredHomestay"
	ActivityGoodBusiness = "goodBusiness"
)

type Service struct {
	repo  *Repository
	users *user.Service
}

func NewService(repo *Repository, users *user.Service) *Service {
	return &Service{repo: repo, users: users}
}

func (s *Service) Homestay(ctx context.Context, id int64) (*Homestay, error) {
	v, err := s.repo.HomestayByID(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, shared.E(shared.CodeCommon, "This record does not exist", nil)
	}
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}

func (s *Service) HomestayList(ctx context.Context, page, pageSize int64) ([]Homestay, error) {
	v, err := s.repo.HomestaysByActivity(ctx, ActivityPreferred, page, pageSize)
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}

func (s *Service) BusinessHomestays(ctx context.Context, businessID, lastID, pageSize int64) ([]Homestay, error) {
	v, err := s.repo.HomestaysByBusiness(ctx, businessID, lastID, pageSize)
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}

func (s *Service) Guess(ctx context.Context) ([]Homestay, error) {
	v, err := s.repo.GuessHomestays(ctx)
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}

func (s *Service) Businesses(ctx context.Context, lastID, pageSize int64) ([]HomestayBusiness, error) {
	v, err := s.repo.Businesses(ctx, lastID, pageSize)
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}

func (s *Service) BusinessBoss(ctx context.Context, id int64) (*user.User, error) {
	b, err := s.repo.BusinessByID(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return &user.User{}, nil
	}
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return s.users.User(ctx, b.UserID)
}

func (s *Service) GoodBosses(ctx context.Context) ([]user.User, error) {
	ids, err := s.repo.GoodBossUserIDs(ctx)
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	out := make([]user.User, 0, len(ids))
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

func (s *Service) Comments(ctx context.Context, lastID, pageSize int64) ([]HomestayComment, error) {
	v, err := s.repo.Comments(ctx, lastID, pageSize)
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return v, nil
}

func ParseStar(value float64) float64 { return value }
