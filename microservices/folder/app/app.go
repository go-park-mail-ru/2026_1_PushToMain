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
	"google.golang.org/grpc"

	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/postgres"
	"go.uber.org/zap"

	folderHttp "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/delivery/http"
	folderRepo "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/repository"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/service"

	"net"

	grpcDelivery "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/folder/delivery/grpc"
	folderpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/folder"

	emailClient "github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/clients/email"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/metrics"
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

	folderRepo := folderRepo.New(db)
	grpcEmailClient, err := emailClient.New(
		app.Config.GRPCClients.EmailService,
	)

	if err != nil {
		app.Logger.Fatalf(
			"failed to init email grpc client: %v",
			err,
		)
	}

	defer func() {
		if err := grpcEmailClient.Close(); err != nil {
			app.Logger.Errorf("grpc client close error: %v", err)
		}
	}()

	folderService := service.New(
		folderRepo,
		grpcEmailClient,
	)
	grpcServer := grpc.NewServer()

	folderGrpcHandler := grpcDelivery.New(
		folderService,
	)

	folderpb.RegisterFolderServiceServer(
		grpcServer,
		folderGrpcHandler,
	)

	lis, err := net.Listen(
		"tcp",
		":"+app.Config.GRPC.FolderPort,
	)

	if err != nil {
		app.Logger.Fatalf(
			"grpc listen error: %v",
			err,
		)
	}

	go func() {
		app.Logger.Infof(
			"grpc started on %s",
			app.Config.GRPC.FolderPort,
		)

		if err := grpcServer.Serve(lis); err != nil {
			app.Logger.Fatalf(
				"grpc serve error: %v",
				err,
			)
		}
	}()

	folderHandler := folderHttp.New(folderService)

	m := metrics.New("folder", "backend")

	router := mux.NewRouter()
	router.Handle("/metrics", m.Handler())
	router.Use(middleware.Logging(app.Logger))
	router.Use(middleware.Metrics(m))

	public := router.PathPrefix("/api/v1/folder").Subrouter()
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
