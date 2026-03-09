package main

import (
	"log"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/config"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/handlers"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/repository"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/tools"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	jwtManager := tools.NewJWTManager(cfg.JWTSecret, cfg.JWTExpire)
	repo := repository.NewMemoryUserRepo()
	svc := service.NewAuthService(repo, jwtManager)
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
