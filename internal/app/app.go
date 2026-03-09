package app

import (
	"net/http"
	"log"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
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
	svc := service.NewAuthService(repo, jwtManager)
	h := handler.NewAuthHandler(svc)

	router := mux.NewRouter()

	router.HandleFunc("/signup", h.SignUp)
	router.HandleFunc("/signin", h.SignIn)
	router.HandleFunc("/inbox/", h.GetEmails).Methods(http.MethodGet, http.MethodOptions)
	router.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)

	handlerChain :=
		middleware.Panic(
			middleware.CORS(cfg.CORS)(
				middleware.JSON(router),
			),
		)

	http.ListenAndServe(":"+cfg.ServerPort, handlerChain)
}

