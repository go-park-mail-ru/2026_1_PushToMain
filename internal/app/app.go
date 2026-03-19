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
	authRepo "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/authAndProfile/repository"
	authService "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/authAndProfile/service"

	authHttp "github.com/go-park-mail-ru/2026_1_PushToMain/internal/app/authAndProfile/delivery/http"
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

	jwtManager := utils.NewJWTManager(cfg.JWTSecret, cfg.JWTExpire)
	authRepo := authRepo.NewMemoryUserRepo()
	authService := authService.NewAuthService(authRepo, jwtManager)
	authHandler := authHttp.NewHandler(authService, jwtManager.TTL())

	emailRepo := emailRepo.NewMemoryEmailRepo()
	emailService := emailService.NewEmailService(emailRepo)
	emailHandler := emailHttp.NewHandler(emailService, jwtManager.TTL())

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
			fmt.Printf("server error: %v", err)
		}
	}()

	<-ctx.Done()

	if err := app.shutdownGracefully(); err != nil {
		fmt.Printf("An error during shutdown: %v", err)
	}
}

func (app *App) shutdownGracefully() error {
	shutdownContex, cancel := context.WithTimeout(context.Background(), shutdownMaxTime)
	defer cancel()

	fmt.Println("shutting down server")

	fullShutdown := make(chan struct{}, 1)
	go func() {
		if err := app.Server.Shutdown(shutdownContex); err != nil {
			fmt.Printf("HTTP server Shutdown: %v", err)
		}
		close(fullShutdown)
	}()
	select {
	case <-shutdownContex.Done():
		return fmt.Errorf("server shutdown: %w", shutdownContex.Err())
	case <-fullShutdown:
		fmt.Println("Server shut down successfully")
	}

	return nil
}
