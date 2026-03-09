package app

import (
	"net/http"
	"log"
	"github.com/gorilla/mux"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/handler"
	// "github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/repository"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	_ "github.com/go-park-mail-ru/2026_1_PushToMain/docs"
)

type App struct {

}

func New() *App {
	return &App{

	}
}

func (app *App) Run() {
	cfg, err := Load()
	if err != nil {
		log.Fatal(err)
	}

	jwtManager := utils.NewJWTManager(cfg.JWTSecret, cfg.JWTExpire)
	repo := repository.NewMemoryUserRepo()
	authService := service.NewAuthService(repo, jwtManager)
	handler := handler.NewHandler(authService)

	router := mux.NewRouter()
	handler.InitRoutes(router)

	// handlerChain :=
	// 	middleware.Panic(
	// 		middleware.CORS(cfg.CORS)(
	// 			middleware.JSON(router),
	// 		),
	// 	)

	http.ListenAndServe(":"+cfg.ServerPort, router)
}

