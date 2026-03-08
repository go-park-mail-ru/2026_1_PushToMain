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
	if cfg == nil {
		return
	}
	repo := repository.NewMemoryUserRepo()
	svc := service.NewAuthService(repo)
	handler := handlers.NewAuthHandler(svc)

	mux := http.NewServeMux()

	mux.HandleFunc("/signup", handler.SignUp)
	mux.HandleFunc("/signin", handler.SignIn)

	handlerChain :=
		middleware.Panic(
			middleware.CORS(cfg.CORS)(
				middleware.JSON(mux),
			),
		)

	http.ListenAndServe(":"+cfg.ServerPort, handlerChain)
}
