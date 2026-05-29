package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-park-mail-ru/2026_1_PushToMain/docs"

	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/postgres"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/metrics"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	supportHttp "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/support/delivery/http"
	supportRepo "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/support/repository"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/support/service"
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
		log.Fatalf("failed to load config: %v", err)
	}

	app.Logger, err = logger.New(&cfg.Logger)
	if err != nil {
		return nil
	}

	defer func() {
		if err := app.Logger.Sync(); err != nil {
			log.Printf("logger sync error: %v", err)
		}
	}()

	app.Config = cfg
	return &app
}

func (app *App) Run(configPath string) {
	db, err := postgres.New(app.Config.Db)
	if err != nil {
		app.Logger.Errorf("postgres error: %v", err)
	}
	err = postgres.RunMigrations(app.Config.Db)
	if err != nil {
		app.Logger.Errorf("migrations error: %v", err)
	}
	supportRepo := supportRepo.New(db)
	supportService := service.New(
		supportRepo,
	)
	folderHandler := supportHttp.New(supportService)
	m := metrics.New("support", "backend")

	router := mux.NewRouter()
	router.Handle("/metrics", m.Handler())
	router.Use(middleware.Logging(app.Logger))
	router.Use(middleware.Metrics(m))

	public := router.PathPrefix("/api/v1/support").Subrouter()
	public.Use(middleware.Panic)
	public.Use(middleware.CORS(app.Config.CORS))
	public.Use(middleware.JSON)

	private := public.PathPrefix("").Subrouter()
	private.Use(middleware.AuthMiddleware(&app.Config.JWTManager))
	private.Use(middleware.CSRFMiddleware)

	folderHandler.InitRoutes(public, private)

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
