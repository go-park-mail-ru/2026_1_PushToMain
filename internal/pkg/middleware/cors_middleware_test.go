package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSHeadersSet(t *testing.T) {

	cfg := CORSConfig{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	middleware := CORS(cfg)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("Access-Control-Allow-Origin header not set correctly")
	}

	if rec.Header().Get("Access-Control-Allow-Methods") != "GET,POST" {
		t.Error("Access-Control-Allow-Methods header not set correctly")
	}

	if rec.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Error("Access-Control-Allow-Headers header not set correctly")
	}

	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Access-Control-Allow-Credentials header not set")
	}

	if !called {
		t.Error("next handler was not called")
	}
}

func TestCORSOptionsRequest(t *testing.T) {

	cfg := CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
	}

	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	middleware := CORS(cfg)(next)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d got %d", http.StatusOK, rec.Code)
	}

	if called {
		t.Error("next handler should not be called for OPTIONS")
	}
}
