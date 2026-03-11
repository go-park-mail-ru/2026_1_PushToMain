package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
)

func TestAuthMiddleware_NoCookie(t *testing.T) {
	jwtManager := utils.NewJWTManager("secret", time.Hour)

	mw := AuthMiddleware(jwtManager)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	if nextCalled {
		t.Fatal("next handler should not be called")
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	jwtManager := utils.NewJWTManager("secret", time.Hour)

	mw := AuthMiddleware(jwtManager)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: "invalid.token",
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	if nextCalled {
		t.Fatal("next handler should not be called")
	}
}

func TestAuthMiddleware_Success(t *testing.T) {
	jwtManager := utils.NewJWTManager("secret", time.Hour)

	token, err := jwtManager.GenerateJWT("test@mail.com")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	mw := AuthMiddleware(jwtManager)

	nextCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		nextCalled = true

		claims, err := ClaimsFromContext(r.Context())
		if err != nil {
			t.Fatalf("expected claims in context: %v", err)
		}

		if claims.Email != "test@mail.com" {
			t.Fatalf("expected email test@mail.com, got %s", claims.Email)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := mw(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: token,
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("next handler was not called")
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestContextWithClaimsAndClaimsFromContext(t *testing.T) {

	claims := &utils.JwtPayload{
		Email: "user@mail.com",
	}

	ctx := ContextWithClaims(context.Background(), claims)

	result, err := ClaimsFromContext(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Email != claims.Email {
		t.Fatalf("expected %s got %s", claims.Email, result.Email)
	}
}

func TestClaimsFromContext_NoClaims(t *testing.T) {

	ctx := context.Background()

	_, err := ClaimsFromContext(ctx)

	if err == nil {
		t.Fatal("expected error when no claims in context")
	}
}
