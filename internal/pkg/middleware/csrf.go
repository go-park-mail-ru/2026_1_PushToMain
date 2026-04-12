package middleware

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
)

func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(csrfCookieName)
		if err != nil || cookie.Value == "" {
			response.Forbidden(w)
			return
		}

		headerToken := r.Header.Get(csrfHeaderName)
		if headerToken == "" {
			response.Forbidden(w)
			return
		}

		if cookie.Value != headerToken {
			response.Forbidden(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}
