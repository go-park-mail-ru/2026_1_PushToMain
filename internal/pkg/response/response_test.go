package response

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponses(t *testing.T) {
	tests := []struct {
		name       string
		handler    func(http.ResponseWriter)
		statusCode int
		bodyPart   string
	}{
		{"BadRequest", BadRequest, http.StatusBadRequest, "Bad request"},
		{"Unauthorized", Unauthorized, http.StatusUnauthorized, "Unauthorized"},
		{"InternalError", InternalError, http.StatusInternalServerError, "Internal server error"},
		{"StatusConflict", StatusConflict, http.StatusConflict, "Already exsist"},
		{"Forbidden", Forbidden, http.StatusForbidden, "Don't have access"},
		{"NotFound", NotFound, http.StatusNotFound, "Not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			tt.handler(rr)

			if rr.Code != tt.statusCode {
				t.Fatalf("expected status %d, got %d", tt.statusCode, rr.Code)
			}

			if !strings.Contains(rr.Body.String(), tt.bodyPart) {
				t.Fatalf("expected body to contain %q, got %s", tt.bodyPart, rr.Body.String())
			}
		})
	}
}
