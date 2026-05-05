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
	authHttp "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/delivery/http"
	profileDbRepo "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/repository/db"
	profileS3Repo "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/repository/storage"
	userService "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/service"

	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/minio"
	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/postgres"
	"go.uber.org/zap"

	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	"github.com/gorilla/mux"

	"net"

	grpcDelivery "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/user/delivery/grpc"
	userpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/user"

	"google.golang.org/grpc"
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

	defer app.Logger.Sync()

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

	s3Client, err := minio.New(context.TODO(), app.Config.S3)
	if err != nil {
		app.Logger.Errorf("minio error: %v", err)
	}

	profileDbRepo := profileDbRepo.New(db)
	profileS3Repo, err := profileS3Repo.New(s3Client)
	if err != nil {
		app.Logger.Fatalf("s3 storage init error: %v", err)
	}
	userService := userService.New(profileDbRepo, profileS3Repo, &app.Config.JWTManager)
	grpcServer := grpc.NewServer()

	userGrpcHandler := grpcDelivery.New(userService)

	userpb.RegisterUserServiceServer(
		grpcServer,
		userGrpcHandler,
	)

	lis, err := net.Listen(
		"tcp",
		":"+app.Config.GRPC.UserPort,
	)

	if err != nil {
		app.Logger.Fatalf("grpc listen error: %v", err)
	}

	go func() {
		app.Logger.Infof(
			"grpc started on %s",
			app.Config.GRPC.UserPort,
		)

		if err := grpcServer.Serve(lis); err != nil {
			app.Logger.Fatalf("grpc serve error: %v", err)
		}
	}()

	authHandler := authHttp.New(userService, authHttp.Config{
		TTL:           app.Config.JWTManager.TTL(),
		MaxAvatarSize: app.Config.Avatar.MaxSizeMB * 1024 * 1024,
		AllowedTypes:  app.Config.Avatar.AllowedTypes,
	})

	router := mux.NewRouter()
	router.Use(middleware.Logging(app.Logger))

	public := router.PathPrefix("/api/v1").Subrouter()
	public.Use(middleware.Panic)
	public.Use(middleware.CORS(app.Config.CORS))
	public.Use(middleware.JSON)

	private := public.PathPrefix("").Subrouter()
	private.Use(middleware.AuthMiddleware(&app.Config.JWTManager))
	private.Use(middleware.CSRFMiddleware)

	authHandler.InitRoutes(public, private)

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
