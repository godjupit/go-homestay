package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"gin-looklook/internal/config"
	"gin-looklook/internal/model"
	"gin-looklook/internal/platform"
	"gin-looklook/internal/repository"

	"github.com/golang-jwt/jwt/v4"
	wechat "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	miniConfig "github.com/silenceper/wechat/v2/miniprogram/config"
)

type Token struct {
	AccessToken  string
	AccessExpire int64
	RefreshAfter int64
}
type UserService struct {
	repo *repository.Repository
	cfg  config.Config
}

func NewUserService(repo *repository.Repository, cfg config.Config) *UserService {
	return &UserService{repo: repo, cfg: cfg}
}

func (s *UserService) token(userID int64) (Token, error) {
	now := time.Now()
	claims := jwt.MapClaims{"exp": now.Add(s.cfg.JWTExpire).Unix(), "iat": now.Unix(), "jwtUserId": userID}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(s.cfg.JWTSecret))
	return Token{AccessToken: signed, AccessExpire: now.Add(s.cfg.JWTExpire).Unix(), RefreshAfter: now.Add(s.cfg.JWTExpire / 2).Unix()}, err
}
func (s *UserService) Register(ctx context.Context, mobile, password, nickname, authType, authKey string) (Token, error) {
	if _, err := s.repo.UserByMobile(ctx, mobile); err == nil {
		return Token{}, platform.E(platform.CodeCommon, "user has been registered", nil)
	} else if err != sql.ErrNoRows {
		return Token{}, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	if nickname == "" {
		nickname = platform.Random(8)
	}
	user := &model.User{Mobile: mobile, Nickname: nickname}
	if password != "" {
		user.Password = platform.MD5(password)
	}
	if authType == "" {
		authType = model.UserAuthTypeSystem
	}
	if authKey == "" {
		authKey = mobile
	}
	id, err := s.repo.CreateUser(ctx, user, &model.UserAuth{AuthKey: authKey, AuthType: authType})
	if err != nil {
		return Token{}, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return s.token(id)
}
func (s *UserService) Login(ctx context.Context, mobile, password string) (Token, error) {
	u, err := s.repo.UserByMobile(ctx, mobile)
	if err == sql.ErrNoRows {
		return Token{}, platform.E(platform.CodeCommon, "用户不存在", nil)
	}
	if err != nil {
		return Token{}, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	if platform.MD5(password) != u.Password {
		return Token{}, platform.E(platform.CodeCommon, "账号或密码不正确", nil)
	}
	return s.token(u.ID)
}
func (s *UserService) User(ctx context.Context, id int64) (*model.User, error) {
	u, err := s.repo.UserByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, platform.E(platform.CodeCommon, "用户不存在", nil)
	}
	if err != nil {
		return nil, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	return u, nil
}
func (s *UserService) AuthByUser(ctx context.Context, userID int64, authType string) (*model.UserAuth, error) {
	return s.repo.UserAuthByUser(ctx, userID, authType)
}

func (s *UserService) WXMiniAuth(ctx context.Context, code, encryptedData, iv string) (Token, error) {
	if s.cfg.WxAppID == "" || s.cfg.WxAppSecret == "" {
		return Token{}, platform.E(platform.CodeCommon, "wechat mini auth is not configured", nil)
	}
	mini := wechat.NewWechat().GetMiniProgram(&miniConfig.Config{AppID: s.cfg.WxAppID, AppSecret: s.cfg.WxAppSecret, Cache: cache.NewMemory()})
	result, err := mini.GetAuth().Code2Session(code)
	if err != nil || result.ErrCode != 0 || result.OpenID == "" {
		return Token{}, platform.E(platform.CodeCommon, "wechat mini auth fail", err)
	}
	if auth, err := s.repo.UserAuthByKey(ctx, model.UserAuthTypeSmallWX, result.OpenID); err == nil {
		return s.token(auth.UserID)
	} else if err != sql.ErrNoRows {
		return Token{}, platform.E(platform.CodeDB, "数据库繁忙,请稍后再试", err)
	}
	data, err := mini.GetEncryptor().Decrypt(result.SessionKey, encryptedData, iv)
	if err != nil {
		return Token{}, platform.E(platform.CodeCommon, "wechat mini auth fail", err)
	}
	mobile := data.PhoneNumber
	if len(mobile) < 4 {
		return Token{}, platform.E(platform.CodeCommon, "wechat mobile is invalid", nil)
	}
	nickname := fmt.Sprintf("LookLook%s", mobile[len(mobile)-4:])
	return s.Register(ctx, mobile, "", nickname, model.UserAuthTypeSmallWX, result.OpenID)
}
