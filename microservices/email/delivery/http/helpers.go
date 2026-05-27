package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/gorilla/mux"
)

func userIDFromCtx(r *http.Request, w http.ResponseWriter) (int64, bool) {
	payload, err := middleware.ClaimsFromContext(r.Context())
	if err != nil || payload.UserId <= 0 {
		response.InternalError(w)
		return 0, false
	}
	return payload.UserId, true
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	s, ok := mux.Vars(r)[key]
	if !ok || s == "" {
		return 0, errors.New("missing path param")
	}
	return strconv.ParseInt(s, 10, 64)
}

func parsePagination(r *http.Request) (int, int) {
	limit := 20
	if s := r.URL.Query().Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	offset := 0
	if s := r.URL.Query().Get("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decodeIDs(w http.ResponseWriter, r *http.Request) ([]int64, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		response.BadRequest(w)
		return nil, false
	}
	var req IDsRequest
	if err := req.UnmarshalJSON(body); err != nil {
		response.BadRequest(w)
		return nil, false
	}
	if len(req.IDs) == 0 {
		response.BadRequest(w)
		return nil, false
	}
	for _, id := range req.IDs {
		if id <= 0 {
			response.BadRequest(w)
			return nil, false
		}
	}
	return req.IDs, true
}

func validEmails(addrs []string) bool {
	if len(addrs) == 0 {
		return false
	}
	for _, a := range addrs {
		if _, err := mail.ParseAddress(a); err != nil {
			return false
		}
	}
	return true
}

// isMultipart reports whether the request Content-Type is multipart/form-data.
func isMultipart(contentType string) bool {
	return strings.HasPrefix(contentType, "multipart/form-data")
}
