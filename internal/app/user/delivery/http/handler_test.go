package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestInitRoutes_PublicRoutes(t *testing.T) {
	public := mux.NewRouter()
	private := mux.NewRouter()

	h := NewHandler(&mockAuthService{}, time.Hour)

	h.InitRoutes(public, private)

	tests := []struct {
		name   string
		method string
		url    string
	}{
		{"signup", http.MethodPost, "/signup"},
		{"signin", http.MethodPost, "/signin"},
		{"logout", http.MethodPost, "/logout"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.url, nil)
		rec := httptest.NewRecorder()

		public.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Fatalf("route %s not registered", tt.url)
		}
	}
}

func TestDocsRouteRegistered(t *testing.T) {
	public := mux.NewRouter()
	private := mux.NewRouter()

	h := NewHandler(&mockAuthService{}, time.Hour)

	h.InitRoutes(public, private)

	req := httptest.NewRequest(http.MethodGet, "/docs/index.html", nil)
	rec := httptest.NewRecorder()

	public.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatal("docs route not registered")
	}
}
