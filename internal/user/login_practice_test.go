package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"gin-looklook/internal/shared"

	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

func loginTestService(find func(context.Context, string) (*User, error)) *Service {
	return &Service{repo: &mockRepo{userByMobileFn: find}, cfg: shared.Config{JWTSecret: "practice-secret", JWTExpire: time.Hour}}
}

func TestPracticeLoginSuccess(t *testing.T) {
	svc := loginTestService(func(context.Context, string) (*User, error) {
		return &User{ID: 42, Password: shared.MD5("123456")}, nil
	})
	got, err := svc.Login(context.Background(), "13800138000", "123456")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	token, err := jwt.Parse(got.AccessToken, func(*jwt.Token) (any, error) { return []byte("practice-secret"), nil })
	if err != nil || !token.Valid {
		t.Fatalf("invalid JWT: %v", err)
	}
	claims := token.Claims.(jwt.MapClaims)
	if claims["jwtUserId"] != float64(42) {
		t.Fatalf("jwtUserId = %v, want 42", claims["jwtUserId"])
	}
}

func TestPracticeLoginFailures(t *testing.T) {
	tests := []struct {
		name     string
		find     func(context.Context, string) (*User, error)
		password string
		wantCode uint32
		wantMsg  string
	}{
		{"not found", func(context.Context, string) (*User, error) { return nil, gorm.ErrRecordNotFound }, "123456", shared.CodeCommon, "用户不存在"},
		{"wrong password", func(context.Context, string) (*User, error) { return &User{Password: shared.MD5("correct")}, nil }, "wrong", shared.CodeCommon, "账号或密码不正确"},
		{"database error", func(context.Context, string) (*User, error) { return nil, errors.New("db down") }, "123456", shared.CodeDB, "数据库繁忙,请稍后再试"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loginTestService(tt.find).Login(context.Background(), "13800138000", tt.password)
			code, msg := shared.Public(err)
			if code != tt.wantCode || msg != tt.wantMsg {
				t.Fatalf("Public(error) = (%d, %q), want (%d, %q)", code, msg, tt.wantCode, tt.wantMsg)
			}
		})
	}
}
