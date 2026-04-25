package middleware

import (
	"context"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
)

type AdminChecker interface {
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

func AdminMiddleware(checker AdminChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := GetLogger(r.Context())

			claims, err := ClaimsFromContext(r.Context())
			if err != nil {
				logger.Errorf("admin middleware: no claims in context: %v", err)
				response.Unauthorized(w)
				return
			}

			ok, err := checker.IsAdmin(r.Context(), claims.UserId)
			if err != nil {
				logger.Errorf("admin middleware: check failed: %v", err)
				response.InternalError(w)
				return
			}

			if !ok {
				response.Forbidden(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
