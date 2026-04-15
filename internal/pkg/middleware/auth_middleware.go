package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/response"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
)

type ctxKey string

const (
	ClaimsKey ctxKey = "claims"
)

func AuthMiddleware(jwtManager *utils.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			cookie, err := r.Cookie("session_token")

			if err != nil {
				response.Unauthorized(w)
				return
			}

			claims, err := jwtManager.ValidateJWT(cookie.Value)
			if err != nil {
				response.Unauthorized(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(ContextWithClaims(r.Context(), claims)))
		})
	}
}

func ContextWithClaims(ctx context.Context, claims *utils.JwtPayload) context.Context {
	return context.WithValue(ctx, ClaimsKey, claims)
}

func ClaimsFromContext(ctx context.Context) (*utils.JwtPayload, error) {
	claims, ok := ctx.Value(ClaimsKey).(*utils.JwtPayload)
	if !ok {
		return nil, fmt.Errorf("no claims found in context")
	}
	return claims, nil
}
