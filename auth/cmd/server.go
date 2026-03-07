package main

import (
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/config"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/handlers"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/repository"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/service"
)

func main() {

	cfg := config.Load()

	repo := repository.NewMemoryUserRepo()
	svc := service.NewAuthService(repo)
	handler := handlers.NewAuthHandler(svc)

	mux := http.NewServeMux()

	mux.HandleFunc("/signup", handler.SignUp)
	mux.HandleFunc("/signin", handler.SignIn)

	handlerWithMiddleware :=
		middleware.Panic(
			middleware.CORS(middleware.CORSConfig(cfg.CORS))(mux),
		)

	http.ListenAndServe(":"+cfg.ServerPort, handlerWithMiddleware)
}
