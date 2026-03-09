package middleware

import (
	"net/http"
	"strings"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
)

func AuthMiddleware(jwtManager *utils.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				response.Unauthorized(w)
				return
			}

			parts := strings.Split(authHeader, " ")

			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Unauthorized(w)
				return
			}

			token := parts[1]

			_, err := jwtManager.ValidateJWT(token)
			if err != nil {
				response.Unauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
