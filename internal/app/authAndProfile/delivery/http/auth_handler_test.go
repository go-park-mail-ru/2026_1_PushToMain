package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/authAndProfile/service"
)

type mockAuthService struct {
	signUpFunc func(ctx context.Context, cmd service.SignUpInput) (string, error)
	signInFunc func(ctx context.Context, cmd service.SignInInput) (string, error)
}

func (m *mockAuthService) SignUp(ctx context.Context, cmd service.SignUpInput) (string, error) {
	return m.signUpFunc(ctx, cmd)
}

func (m *mockAuthService) SignIn(ctx context.Context, cmd service.SignInInput) (string, error) {
	return m.signInFunc(ctx, cmd)
}

func newTestHandler(auth AuthService) *Handler {
	return &Handler{
		authService: auth,
		ttl:         time.Hour,
	}
}

func TestSignUpSuccess(t *testing.T) {

	mock := &mockAuthService{
		signUpFunc: func(ctx context.Context, cmd service.SignUpInput) (string, error) {
			return "token123", nil
		},
	}

	h := newTestHandler(mock)

	body := SignUpRequest{
		Name:     "John",
		Surname:  "Doe",
		Email:    "test@smail.ru",
		Password: "12345678",
	}

	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewBuffer(jsonBody))
	rec := httptest.NewRecorder()

	h.SignUp(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()

	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}

	if cookies[0].Name != sessionTokenCookie {
		t.Fatalf("expected cookie %s", sessionTokenCookie)
	}
}

func TestSignUpBadRequest(t *testing.T) {

	mock := &mockAuthService{}
	h := newTestHandler(mock)

	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewBuffer([]byte("bad json")))
	rec := httptest.NewRecorder()

	h.SignUp(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSignInSuccess(t *testing.T) {

	mock := &mockAuthService{
		signInFunc: func(ctx context.Context, cmd service.SignInInput) (string, error) {
			return "token123", nil
		},
	}

	h := newTestHandler(mock)

	body := SignInRequest{
		Email:    "test@smail.ru",
		Password: "12345678",
	}

	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/auth/signin", bytes.NewBuffer(jsonBody))
	rec := httptest.NewRecorder()

	h.SignIn(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()

	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}

	if cookies[0].Name != sessionTokenCookie {
		t.Fatalf("expected cookie %s", sessionTokenCookie)
	}
}

func TestLogout(t *testing.T) {

	h := &Handler{
		ttl: time.Hour,
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	cookies := rec.Result().Cookies()

	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}

	if cookies[0].Value != "" {
		t.Fatal("expected cookie to be cleared")
	}
}
