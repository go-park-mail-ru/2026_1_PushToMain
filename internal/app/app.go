package app

import (
	"net/http"
	"smail/internal/app/handler"
	"github.com/gorilla/mux"
	"context"
	"os/signal"
	"syscall"
	"time"
	"fmt"
)

const (
	shutdownTime 	= 5 * time.Second
	serverAddress 	= "127.0.0.1:8087"
)

type App struct {
	Server http.Server
	Router *mux.Router
}

func New() *App {
	handler := handler.NewHandler()
	router := handler.InitRoutes()
	
	return &App{
		Server: http.Server {
			Addr: 		serverAddress,
			Handler: 	router,
		},
	}
}

func (app *App) shutdownGracefully() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTime)
	defer cancel()

	fmt.Println("shutting down server")

	fullShutdown := make(chan struct{}, 1)
	go func() {
		if err := app.Server.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("HTTP server Shutdown: %v", err)
		}
		close(fullShutdown)
	}()

	select {
	case <- shutdownCtx.Done():
		return fmt.Errorf("server shutdown: %w", shutdownCtx.Err())
	case <- fullShutdown:
		fmt.Println("Server shut down successfully")
	}

	return nil
}

func (app *App) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := app.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("server error: %v", err)
		}
	}()

	fmt.Printf("listening on %s", serverAddress)
	<-ctx.Done()
	
	return app.shutdownGracefully()
}


