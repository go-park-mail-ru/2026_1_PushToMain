package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
)

func TestGetEmails_Success(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/emails", nil)

	claims := &utils.JwtPayload{
		Email: "anna.sidorova@smail.ru",
	}

	ctx := middleware.ContextWithClaims(req.Context(), claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	h.GetEmails(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d got %d", http.StatusOK, rec.Code)
	}

	var emails []map[string]any
	err := json.NewDecoder(rec.Body).Decode(&emails)
	if err != nil {
		t.Fatalf("failed decode response: %v", err)
	}

	if len(emails) == 0 {
		t.Fatal("expected emails but got empty list")
	}
}

func TestGetEmails_NoClaims(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/emails", nil)

	rec := httptest.NewRecorder()

	h.GetEmails(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestGetEmails_EmptyEmail(t *testing.T) {
	h := &Handler{}

	req := httptest.NewRequest(http.MethodGet, "/emails", nil)

	claims := &utils.JwtPayload{
		Email: "",
	}

	ctx := middleware.ContextWithClaims(context.Background(), claims)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	h.GetEmails(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d got %d", http.StatusBadRequest, rec.Code)
	}
}
