package app

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"go.uber.org/zap"

	_ "github.com/go-park-mail-ru/2026_1_PushToMain/docs"
	authHttp "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/delivery/http"
	authRepo "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/repository"
	authService "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/user/service"

	emailHttp "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/delivery/http"
	emailRepo "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/repository"
	emailService "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/email/service"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/gorilla/mux"
)

const shutdownMaxTime = 5 * time.Second

type App struct {
	Server  http.Server
	Address string
	Config  *Config
	Logger  *zap.SugaredLogger
}

func New(configPath string) *App {
	app := App{}

	cfg, err := Load(configPath)
	if err != nil {
		return nil
	}
	defer app.Logger.Sync()

	app.Config = cfg
	return &app
}

func (app *App) Run(configPath string) {
	authRepo := authRepo.New()
	authService := authService.New(authRepo, &app.Config.JWTManager)
	authHandler := authHttp.New(authService, authHttp.Config{TTL: app.Config.JWTManager.TTL()})

	emailRepo := emailRepo.New()
	emailService := emailService.New(emailRepo)
	emailHandler := emailHttp.New(emailService, emailHttp.Config{TTL: app.Config.JWTManager.TTL()})

	router := mux.NewRouter()

	public := router.PathPrefix("/api/v1").Subrouter()
	public.Use(middleware.Panic)
	public.Use(middleware.CORS(app.Config.CORS))
	public.Use(middleware.JSON)

	private := public.PathPrefix("").Subrouter()
	private.Use(middleware.AuthMiddleware(&app.Config.JWTManager))

	authHandler.InitRoutes(public, private)
	emailHandler.InitRoutes(public, private)

	app.Server = http.Server{
		Addr:    ":" + app.Config.ServerPort,
		Handler: router,
	}

	fmt.Printf("Starting server at port %s\n", app.Config.ServerPort)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Logger.Errorf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	if err := app.shutdownGracefully(); err != nil {
		app.Logger.Errorf("error during shutdown: %v", err)
	}

	return nil
}

func (app *App) shutdownGracefully() error {
	shutdownContex, cancel := context.WithTimeout(context.Background(), shutdownMaxTime)
	defer cancel()

	app.Logger.Info("shutting down server")

	fullShutdown := make(chan struct{}, 1)
	go func() {
		if err := app.Server.Shutdown(shutdownContex); err != nil {
			app.Logger.Errorf("HTTP server Shutdown: %v", err)
		}
		close(fullShutdown)
	}()
	select {
	case <-shutdownContex.Done():
		app.Logger.Errorf("server shutdown: %w", shutdownContex.Err())
		return fmt.Errorf("server shutdown: %w", shutdownContex.Err())
	case <-fullShutdown:
		app.Logger.Info("Server shut down successfully")
	}

	return nil
}
