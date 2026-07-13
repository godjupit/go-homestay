package user

import (
	"context"
	"errors"
	"testing"

	"gin-looklook/internal/shared"
)

// ---- mock repository ----

type mockRepo struct {
	userByIDFn  func(ctx context.Context, id int64) (*User, error)
	updateUserFn func(ctx context.Context, user *User) error

	// Unused methods — panic if called so the test fails fast.
	userByMobileFn   func(ctx context.Context, mobile string) (*User, error)
	userAuthByUserFn func(ctx context.Context, userID int64, authType string) (*UserAuth, error)
	userAuthByKeyFn  func(ctx context.Context, authType, authKey string) (*UserAuth, error)
	createUserFn     func(ctx context.Context, user *User, auth *UserAuth) (int64, error)
}

func (m *mockRepo) UserByMobile(ctx context.Context, mobile string) (*User, error) {
	if m.userByMobileFn != nil {
		return m.userByMobileFn(ctx, mobile)
	}
	panic("UserByMobile not implemented in mock")
}
func (m *mockRepo) UserByID(ctx context.Context, id int64) (*User, error) {
	return m.userByIDFn(ctx, id)
}
func (m *mockRepo) UserAuthByUser(ctx context.Context, userID int64, authType string) (*UserAuth, error) {
	if m.userAuthByUserFn != nil {
		return m.userAuthByUserFn(ctx, userID, authType)
	}
	panic("UserAuthByUser not implemented in mock")
}
func (m *mockRepo) UserAuthByKey(ctx context.Context, authType, authKey string) (*UserAuth, error) {
	if m.userAuthByKeyFn != nil {
		return m.userAuthByKeyFn(ctx, authType, authKey)
	}
	panic("UserAuthByKey not implemented in mock")
}
func (m *mockRepo) CreateUser(ctx context.Context, user *User, auth *UserAuth) (int64, error) {
	if m.createUserFn != nil {
		return m.createUserFn(ctx, user, auth)
	}
	panic("CreateUser not implemented in mock")
}
func (m *mockRepo) UpdateUser(ctx context.Context, user *User) error {
	return m.updateUserFn(ctx, user)
}

// ---- helpers ----

func ptrStr(s string) *string { return &s }
func ptrInt(i int64) *int64   { return &i }

func newTestService(byID func(ctx context.Context, id int64) (*User, error), update func(ctx context.Context, user *User) error) *Service {
	return &Service{repo: &mockRepo{userByIDFn: byID, updateUserFn: update}, cfg: shared.Config{}}
}

// ---- tests ----

func TestUpdateProfile_AllFields(t *testing.T) {
	original := &User{ID: 1, Mobile: "13800138000", Nickname: "old", Sex: 0, Avatar: "old.png", Info: "old info"}
	var updated *User

	svc := newTestService(
		func(ctx context.Context, id int64) (*User, error) { return original, nil },
		func(ctx context.Context, user *User) error { updated = user; return nil },
	)

	nick := "newNick"
	sex := int64(1)
	avt := "new.png"
	info := "new info"

	err := svc.UpdateProfile(context.Background(), 1, &nick, &sex, &avt, &info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateUser was not called")
	}
	if updated.Nickname != nick {
		t.Errorf("nickname = %q, want %q", updated.Nickname, nick)
	}
	if updated.Sex != sex {
		t.Errorf("sex = %d, want %d", updated.Sex, sex)
	}
	if updated.Avatar != avt {
		t.Errorf("avatar = %q, want %q", updated.Avatar, avt)
	}
	if updated.Info != info {
		t.Errorf("info = %q, want %q", updated.Info, info)
	}
}

func TestUpdateProfile_SingleField(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(u *User)
		check    func(t *testing.T, u *User)
	}{
		{
			name: "nickname only",
			setup: func(u *User) {
				nick := "newNick"
				u.Nickname = "old"
				_ = &nick
			},
			check: func(t *testing.T, u *User) {
				if u.Nickname != "newNick" {
					t.Errorf("nickname = %q, want %q", u.Nickname, "newNick")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := &User{ID: 1, Mobile: "13800138000", Nickname: "old", Sex: 2, Avatar: "a.png", Info: "i"}
			var updated *User

			svc := newTestService(
				func(ctx context.Context, id int64) (*User, error) { return original, nil },
				func(ctx context.Context, user *User) error { updated = user; return nil },
			)

			// only update nickname
			nick := "newNick"
			err := svc.UpdateProfile(context.Background(), 1, &nick, nil, nil, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated.Nickname != nick {
				t.Errorf("nickname = %q, want %q", updated.Nickname, nick)
			}
			// other fields unchanged
			if updated.Sex != 2 {
				t.Errorf("sex = %d, want 2", updated.Sex)
			}
			if updated.Avatar != "a.png" {
				t.Errorf("avatar = %q, want a.png", updated.Avatar)
			}
			if updated.Info != "i" {
				t.Errorf("info = %q, want i", updated.Info)
			}
		})
	}
}

func TestUpdateProfile_EachFieldIndividually(t *testing.T) {
	original := &User{ID: 1, Nickname: "old", Sex: 0, Avatar: "old.png", Info: "old"}

	t.Run("sex only", func(t *testing.T) {
		var updated *User
		svc := newTestService(
			func(ctx context.Context, id int64) (*User, error) { return original, nil },
			func(ctx context.Context, user *User) error { updated = user; return nil },
		)
		sex := int64(1)
		if err := svc.UpdateProfile(context.Background(), 1, nil, &sex, nil, nil); err != nil {
			t.Fatal(err)
		}
		if updated.Sex != 1 {
			t.Errorf("sex = %d, want 1", updated.Sex)
		}
	})

	t.Run("avatar only", func(t *testing.T) {
		var updated *User
		svc := newTestService(
			func(ctx context.Context, id int64) (*User, error) { return original, nil },
			func(ctx context.Context, user *User) error { updated = user; return nil },
		)
		avt := "avatar.png"
		if err := svc.UpdateProfile(context.Background(), 1, nil, nil, &avt, nil); err != nil {
			t.Fatal(err)
		}
		if updated.Avatar != avt {
			t.Errorf("avatar = %q, want %q", updated.Avatar, avt)
		}
	})

	t.Run("info only", func(t *testing.T) {
		var updated *User
		svc := newTestService(
			func(ctx context.Context, id int64) (*User, error) { return original, nil },
			func(ctx context.Context, user *User) error { updated = user; return nil },
		)
		info := "new info"
		if err := svc.UpdateProfile(context.Background(), 1, nil, nil, nil, &info); err != nil {
			t.Fatal(err)
		}
		if updated.Info != info {
			t.Errorf("info = %q, want %q", updated.Info, info)
		}
	})
}

func TestUpdateProfile_NoFields(t *testing.T) {
	original := &User{ID: 1, Nickname: "old", Sex: 0, Avatar: "old.png", Info: "old"}
	var updated *User

	svc := newTestService(
		func(ctx context.Context, id int64) (*User, error) { return original, nil },
		func(ctx context.Context, user *User) error { updated = user; return nil },
	)

	err := svc.UpdateProfile(context.Background(), 1, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateUser was not called")
	}
	if updated.Nickname != "old" || updated.Sex != 0 || updated.Avatar != "old.png" || updated.Info != "old" {
		t.Error("fields should remain unchanged when all pointers are nil")
	}
}

func TestUpdateProfile_UserByIDError(t *testing.T) {
	svc := newTestService(
		func(ctx context.Context, id int64) (*User, error) { return nil, errors.New("db down") },
		nil,
	)
	err := svc.UpdateProfile(context.Background(), 1, ptrStr("x"), nil, nil, nil)
	if err == nil || err.Error() != "db down" {
		t.Fatalf("expected 'db down', got %v", err)
	}
}

func TestUpdateProfile_UpdateUserError(t *testing.T) {
	svc := newTestService(
		func(ctx context.Context, id int64) (*User, error) { return &User{ID: 1}, nil },
		func(ctx context.Context, user *User) error { return errors.New("write failed") },
	)
	err := svc.UpdateProfile(context.Background(), 1, ptrStr("x"), nil, nil, nil)
	if err == nil || err.Error() != "write failed" {
		t.Fatalf("expected 'write failed', got %v", err)
	}
}

func TestUpdateProfile_NicknameEmptyStringVsNil(t *testing.T) {
	// An empty string pointer (pointing to "") should still update nickname to "".
	original := &User{ID: 1, Nickname: "old"}
	var updated *User

	svc := newTestService(
		func(ctx context.Context, id int64) (*User, error) { return original, nil },
		func(ctx context.Context, user *User) error { updated = user; return nil },
	)

	empty := ""
	err := svc.UpdateProfile(context.Background(), 1, &empty, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Nickname != "" {
		t.Errorf("nickname = %q, want empty string", updated.Nickname)
	}
}
