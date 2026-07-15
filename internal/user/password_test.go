package user

import (
	"context"
	"testing"

	"gin-looklook/internal/shared"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestVerifyPassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		encoded    string
		password   string
		wantValid  bool
		wantLegacy bool
	}{
		{name: "bcrypt", encoded: string(hash), password: "correct", wantValid: true},
		{name: "bcrypt wrong", encoded: string(hash), password: "wrong"},
		{name: "legacy md5", encoded: shared.MD5("correct"), password: "correct", wantValid: true, wantLegacy: true},
		{name: "legacy wrong", encoded: shared.MD5("correct"), password: "wrong", wantLegacy: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, legacy := verifyPassword(tt.encoded, tt.password)
			if valid != tt.wantValid || legacy != tt.wantLegacy {
				t.Fatalf("verifyPassword() = (%v,%v), want (%v,%v)", valid, legacy, tt.wantValid, tt.wantLegacy)
			}
		})
	}
}

func TestLoginMigratesLegacyMD5(t *testing.T) {
	var migrated string
	repo := &mockRepo{
		userByMobileFn: func(context.Context, string) (*User, error) {
			return &User{ID: 7, Password: shared.MD5("correct")}, nil
		},
		updatePasswordHash: func(_ context.Context, userID int64, hash string) error {
			if userID != 7 {
				t.Fatalf("userID = %d, want 7", userID)
			}
			migrated = hash
			return nil
		},
	}
	svc := NewService(repo, shared.Config{JWTSecret: "test-secret", JWTExpire: 60})
	if _, err := svc.Login(context.Background(), "13800138000", "correct"); err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(migrated), []byte("correct")); err != nil {
		t.Fatalf("password was not migrated to bcrypt: %v", err)
	}
}

func TestLoginDoesNotRevealMissingAccount(t *testing.T) {
	repo := &mockRepo{userByMobileFn: func(context.Context, string) (*User, error) {
		return nil, gorm.ErrRecordNotFound
	}}
	svc := NewService(repo, shared.Config{})
	_, err := svc.Login(context.Background(), "13800138000", "wrong")
	_, msg := shared.Public(err)
	if msg != "账号或密码不正确" {
		t.Fatalf("message = %q, want generic credential error", msg)
	}
}
