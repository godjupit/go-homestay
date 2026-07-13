package user

import (
	"context"
	"fmt"
	"time"

	"gin-looklook/internal/shared"

	"github.com/golang-jwt/jwt/v4"
	wechat "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	miniConfig "github.com/silenceper/wechat/v2/miniprogram/config"
	"gorm.io/gorm"
)

type Token struct {
	AccessToken  string
	AccessExpire int64
	RefreshAfter int64
}

type Service struct {
	repo *Repository
	cfg  shared.Config
}

func NewService(repo *Repository, cfg shared.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

func (s *Service) token(userID int64) (Token, error) {
	now := time.Now()
	claims := jwt.MapClaims{"exp": now.Add(s.cfg.JWTExpire).Unix(), "iat": now.Unix(), "jwtUserId": userID}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(s.cfg.JWTSecret))
	return Token{AccessToken: signed, AccessExpire: now.Add(s.cfg.JWTExpire).Unix(), RefreshAfter: now.Add(s.cfg.JWTExpire / 2).Unix()}, err
}

func (s *Service) Register(ctx context.Context, mobile, password, nickname, authType, authKey string) (Token, error) {
	if _, err := s.repo.UserByMobile(ctx, mobile); err == nil {
		return Token{}, shared.E(shared.CodeCommon, "user has been registered", nil)
	} else if err != gorm.ErrRecordNotFound {
		return Token{}, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	if nickname != "" {
		nickname = nickname[:min(len(nickname), 15)]
	}
	user := &User{Mobile: mobile, Nickname: nickname}
	if password != "" {
		user.Password = shared.MD5(password)
	}
	if authType == "" {
		authType = AuthTypeSystem
	}
	if authKey == "" {
		authKey = mobile
	}
	id, err := s.repo.CreateUser(ctx, user, &UserAuth{AuthKey: authKey, AuthType: authType})
	if err != nil {
		return Token{}, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return s.token(id)
}

func (s *Service) Login(ctx context.Context, mobile, password string) (Token, error) {
	u, err := s.repo.UserByMobile(ctx, mobile)
	if err == gorm.ErrRecordNotFound {
		return Token{}, shared.E(shared.CodeCommon, "用户不存在", nil)
	}
	if err != nil {
		return Token{}, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	if shared.MD5(password) != u.Password {
		return Token{}, shared.E(shared.CodeCommon, "账号或密码不正确", nil)
	}
	return s.token(u.ID)
}

func (s *Service) User(ctx context.Context, id int64) (*User, error) {
	u, err := s.repo.UserByID(ctx, id)
	if err == gorm.ErrRecordNotFound {
		return nil, shared.E(shared.CodeCommon, "用户不存在", nil)
	}
	if err != nil {
		return nil, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return u, nil
}

func (s *Service) AuthByUser(ctx context.Context, userID int64, authType string) (*UserAuth, error) {
	return s.repo.UserAuthByUser(ctx, userID, authType)
}

func (s *Service) WXMiniAuth(ctx context.Context, code, encryptedData, iv string) (Token, error) {
	if s.cfg.WxAppID == "" || s.cfg.WxAppSecret == "" {
		return Token{}, shared.E(shared.CodeCommon, "wechat mini auth is not configured", nil)
	}
	mini := wechat.NewWechat().GetMiniProgram(&miniConfig.Config{AppID: s.cfg.WxAppID, AppSecret: s.cfg.WxAppSecret, Cache: cache.NewMemory()})
	result, err := mini.GetAuth().Code2Session(code)
	if err != nil || result.ErrCode != 0 || result.OpenID == "" {
		return Token{}, shared.E(shared.CodeCommon, "wechat mini auth fail", err)
	}
	if auth, err := s.repo.UserAuthByKey(ctx, AuthTypeSmallWX, result.OpenID); err == nil {
		return s.token(auth.UserID)
	} else if err != gorm.ErrRecordNotFound {
		return Token{}, shared.E(shared.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	data, err := mini.GetEncryptor().Decrypt(result.SessionKey, encryptedData, iv)
	if err != nil {
		return Token{}, shared.E(shared.CodeCommon, "wechat mini auth fail", err)
	}
	mobile := data.PhoneNumber
	if len(mobile) < 4 {
		return Token{}, shared.E(shared.CodeCommon, "wechat mobile is invalid", nil)
	}
	nickname := fmt.Sprintf("LookLook%s", mobile[len(mobile)-4:])
	return s.Register(ctx, mobile, "", nickname, AuthTypeSmallWX, result.OpenID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, nickname *string, sex *int64, avatar *string, info *string) error {
	u, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return err
	}
	if nickname != nil {
		u.Nickname = *nickname
	}
	if sex != nil {
		u.Sex = *sex
	}
	if avatar != nil {
		u.Avatar = *avatar
	}
	if info != nil {
		u.Info = *info
	}
	return s.repo.UpdateUser(ctx, u)
}
