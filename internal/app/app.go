package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"

	_ "github.com/go-park-mail-ru/2026_1_PushToMain/docs"
	authHttp "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/delivery/http"
	authRepo "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/repository"
	authService "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service"

	emailHttp "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/delivery/http"
	emailRepo "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
	emailService "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/utils"
	"github.com/gorilla/mux"
)

const shutdownMaxTime = 5 * time.Second

type App struct {
	Server  http.Server
	Address string
}

func New() *App {
	return &App{}
}

func (app *App) Run() {
	cfg, err := Load()
	if err != nil {
		log.Fatal(err)
		return
	}

	err = logger.Init(logger.DefaultConfig())
	if err != nil {
		log.Fatal(err)
		return
	}
	defer logger.Sync()

	jwtManager := utils.NewJWTManager(cfg.JWTSecret, cfg.JWTExpire)
	authRepo := authRepo.New()
	authService := authService.New(authRepo, jwtManager)
	authHandler := authHttp.New(authService, authHttp.Config{TTL: jwtManager.TTL()})

	emailRepo := emailRepo.New()
	emailService := emailService.New(emailRepo)
	emailHandler := emailHttp.New(emailService, emailHttp.Config{TTL: jwtManager.TTL()})

	router := mux.NewRouter()

	public := router.PathPrefix("/api/v1").Subrouter()
	public.Use(middleware.Panic)
	public.Use(middleware.CORS(cfg.CORS))
	public.Use(middleware.JSON)

	private := public.PathPrefix("").Subrouter()
	private.Use(middleware.AuthMiddleware(jwtManager))

	authHandler.InitRoutes(public, private)
	emailHandler.InitRoutes(public, private)

	app.Server = http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Errorw("server error", "error", err)
		}
	}()

	<-ctx.Done()

	if err := app.shutdownGracefully(); err != nil {
		logger.Logger.Errorw("error during shutdown", "error", err)
	}
}

func (app *App) shutdownGracefully() error {
	shutdownContex, cancel := context.WithTimeout(context.Background(), shutdownMaxTime)
	defer cancel()

	logger.Logger.Info("shutting down server")

	fullShutdown := make(chan struct{}, 1)
	go func() {
		if err := app.Server.Shutdown(shutdownContex); err != nil {
			logger.Logger.Errorw("HTTP server Shutdown", "error", err)
		}
		close(fullShutdown)
	}()
	select {
	case <-shutdownContex.Done():
		return fmt.Errorf("server shutdown: %w", shutdownContex.Err())
	case <-fullShutdown:
		logger.Logger.Info("Server shut down successfully")
	}

	return nil
}
