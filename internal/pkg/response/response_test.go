package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	data := map[string]string{
		"hello": "world",
	}

	WriteJSON(rec, http.StatusCreated, data)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %s", ct)
	}

	var result map[string]string
	err := json.NewDecoder(rec.Body).Decode(&result)
	if err != nil {
		t.Fatalf("failed to decode json: %v", err)
	}

	if result["hello"] != "world" {
		t.Fatalf("expected world, got %s", result["hello"])
	}
}

func TestBadRequest(t *testing.T) {
	rec := httptest.NewRecorder()

	BadRequest(rec)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var resp ErrorResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}

	if resp.Message != "Bad request" {
		t.Fatalf("unexpected message: %s", resp.Message)
	}
}

func TestUnauthorized(t *testing.T) {
	rec := httptest.NewRecorder()

	Unauthorized(rec)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())

	expected := `{ "error": "Unauthorized" }`
	if body != expected {
		t.Fatalf("expected %s, got %s", expected, body)
	}
}

func TestInternalError(t *testing.T) {
	rec := httptest.NewRecorder()

	InternalError(rec)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())

	expected := `{ "error": "Internal server error" }`
	if body != expected {
		t.Fatalf("expected %s, got %s", expected, body)
	}
}

func TestStatusConflict(t *testing.T) {
	rec := httptest.NewRecorder()

	StatusConflict(rec)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())

	expected := `{ "error": "User already exsist" }`
	if body != expected {
		t.Fatalf("expected %s, got %s", expected, body)
	}
}
