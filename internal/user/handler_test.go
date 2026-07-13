package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gin-looklook/internal/shared"

	"github.com/gin-gonic/gin"
)

// ---- handler-specific mock (same shape as the service mock, scoped to handler tests) ----

type handlerMockRepo struct {
	user *User
	err  error // returned by both UserByID and UpdateUser
}

func (m *handlerMockRepo) UserByMobile(ctx context.Context, mobile string) (*User, error) {
	panic("not implemented")
}
func (m *handlerMockRepo) UserByID(ctx context.Context, id int64) (*User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}
func (m *handlerMockRepo) UserAuthByUser(ctx context.Context, userID int64, authType string) (*UserAuth, error) {
	panic("not implemented")
}
func (m *handlerMockRepo) UserAuthByKey(ctx context.Context, authType, authKey string) (*UserAuth, error) {
	panic("not implemented")
}
func (m *handlerMockRepo) CreateUser(ctx context.Context, user *User, auth *UserAuth) (int64, error) {
	panic("not implemented")
}
func (m *handlerMockRepo) UpdateUser(ctx context.Context, user *User) error {
	if m.err != nil {
		return m.err
	}
	// apply changes to the stored user so subsequent detail calls can verify
	m.user.Nickname = user.Nickname
	m.user.Sex = user.Sex
	m.user.Avatar = user.Avatar
	m.user.Info = user.Info
	return nil
}

// ---- setup ----

func setupHandler(mock *handlerMockRepo) *Handler {
	gin.SetMode(gin.TestMode)
	svc := &Service{repo: mock, cfg: shared.Config{}}
	return NewHandler(svc)
}

func newJSONRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/usercenter/v1/user/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func assertCode(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()
	if w.Code != want {
		t.Errorf("status = %d, want %d, body = %s", w.Code, want, w.Body.String())
	}
}

func assertMsg(t *testing.T, w *httptest.ResponseRecorder, want string) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	got, _ := m["msg"].(string)
	if got != want {
		t.Errorf("msg = %q, want %q", got, want)
	}
}

// ---- tests ----

func TestUpdateProfileHandler_SuccessAllFields(t *testing.T) {
	mock := &handlerMockRepo{user: &User{ID: 1, Mobile: "13800138000", Nickname: "old", Sex: 0, Avatar: "old.png", Info: "old"}}
	h := setupHandler(mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", int64(1))
	c.Request = newJSONRequest(`{"nickname":"newName","sex":1,"avatar":"new.png","info":"hello"}`)

	h.UpdateProfile(c)

	assertCode(t, w, 200)
	if mock.user.Nickname != "newName" {
		t.Errorf("nickname = %q, want newName", mock.user.Nickname)
	}
	if mock.user.Sex != 1 {
		t.Errorf("sex = %d, want 1", mock.user.Sex)
	}
	if mock.user.Avatar != "new.png" {
		t.Errorf("avatar = %q, want new.png", mock.user.Avatar)
	}
	if mock.user.Info != "hello" {
		t.Errorf("info = %q, want hello", mock.user.Info)
	}
}

func TestUpdateProfileHandler_SuccessPartial(t *testing.T) {
	mock := &handlerMockRepo{user: &User{ID: 1, Nickname: "old", Sex: 2, Avatar: "a.png", Info: "old info"}}
	h := setupHandler(mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", int64(1))
	c.Request = newJSONRequest(`{"nickname":"onlyName"}`)

	h.UpdateProfile(c)

	assertCode(t, w, 200)
	if mock.user.Nickname != "onlyName" {
		t.Errorf("nickname = %q, want onlyName", mock.user.Nickname)
	}
	if mock.user.Sex != 2 {
		t.Errorf("sex = %d, want 2 (unchanged)", mock.user.Sex)
	}
}

func TestUpdateProfileHandler_SuccessEmptyBody(t *testing.T) {
	mock := &handlerMockRepo{user: &User{ID: 1, Nickname: "old"}}
	h := setupHandler(mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", int64(1))
	c.Request = newJSONRequest(`{}`)

	h.UpdateProfile(c)

	assertCode(t, w, 200)
}

func TestUpdateProfileHandler_NicknameTooLong(t *testing.T) {
	mock := &handlerMockRepo{user: &User{ID: 1}}
	h := setupHandler(mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", int64(1))
	c.Request = newJSONRequest(`{"nickname":"thisNameIsWayTooLong12345"}`) // 24 chars, max 15

	h.UpdateProfile(c)

	assertCode(t, w, 400)
	assertMsg(t, w, "参数错误, Key: 'UpdateProfileReq.Nickname' Error:Field validation for 'Nickname' failed on the 'max' tag")
}

func TestUpdateProfileHandler_SexInvalid(t *testing.T) {
	mock := &handlerMockRepo{user: &User{ID: 1}}
	h := setupHandler(mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", int64(1))
	c.Request = newJSONRequest(`{"sex":999}`)

	h.UpdateProfile(c)

	assertCode(t, w, 400)
	body := w.Body.String()
	if !strings.Contains(strings.ToLower(body), "sex") {
		t.Errorf("expected sex validation error, got %s", body)
	}
}

func TestUpdateProfileHandler_AvatarTooLong(t *testing.T) {
	mock := &handlerMockRepo{user: &User{ID: 1}}
	h := setupHandler(mock)

	long := strings.Repeat("x", 501) // exceeds max=500

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", int64(1))
	c.Request = newJSONRequest(`{"avatar":"` + long + `"}`)

	h.UpdateProfile(c)

	assertCode(t, w, 400)
}

func TestUpdateProfileHandler_InfoTooLong(t *testing.T) {
	mock := &handlerMockRepo{user: &User{ID: 1}}
	h := setupHandler(mock)

	long := strings.Repeat("y", 201) // exceeds max=200

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", int64(1))
	c.Request = newJSONRequest(`{"info":"` + long + `"}`)

	h.UpdateProfile(c)

	assertCode(t, w, 400)
}

func TestUpdateProfileHandler_InvalidJSON(t *testing.T) {
	mock := &handlerMockRepo{user: &User{ID: 1}}
	h := setupHandler(mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", int64(1))
	c.Request = newJSONRequest(`not json`)

	h.UpdateProfile(c)

	assertCode(t, w, 400)
}

func TestUpdateProfileHandler_UserNotFound(t *testing.T) {
	// The real repo returns gorm.ErrRecordNotFound; we simulate it with a plain error
	// to verify the error path is exercised.
	mock := &handlerMockRepo{err: context.DeadlineExceeded}
	h := setupHandler(mock)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("userID", int64(999))
	c.Request = newJSONRequest(`{"nickname":"x"}`)

	h.UpdateProfile(c)

	// The error is not an *AppError so fail() maps it to CodeCommon (100001) with the
	// generic fallback message.
	assertCode(t, w, 400)
	body := w.Body.String()
	if !strings.Contains(body, "服务器开小差啦") {
		t.Errorf("expected fallback error message, got %s", body)
	}
}
