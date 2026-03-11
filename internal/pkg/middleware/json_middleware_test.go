package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONMiddleware(t *testing.T) {

	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := JSON(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	contentType := rec.Header().Get("Content-Type")
	expected := "application/json; charset=utf-8"

	if contentType != expected {
		t.Errorf("expected Content-Type %s, got %s", expected, contentType)
	}

	if !called {
		t.Error("next handler was not called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
