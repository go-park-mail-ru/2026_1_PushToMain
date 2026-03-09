package middleware

import (
	"log"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/response"
)

func Panic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			if err := recover(); err != nil {

				log.Println("panic recovered:", err)

				response.InternalError(w)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
