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

	userClient "github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/clients/user"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/metrics"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	emailHttp "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/http"
	emailRepo "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/repository/db"
	emailStorage "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/repository/storage"
	emailService "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	"github.com/gorilla/mux"

	"net"

	grpcDelivery "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/grpc"

	emailpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/email"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

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

	repo := emailRepo.New(db)

	grpcUserClient, err := userClient.New(app.Config.GRPCClients.UserService)
	if err != nil {
		app.Logger.Fatalf("failed to init user grpc client: %v", err)
	}
	defer grpcUserClient.Close()

	svc := emailService.New(
		repo,
		grpcUserClient,
		emailService.DraftsConfig{MaxPerUser: app.Config.Drafts.MaxPerUser},
	)

	// Initialise MinIO / S3-compatible object storage for attachments.
	minioClient, err := newMinioClient(app.Config.Minio)
	if err != nil {
		app.Logger.Errorf("minio init error (attachments disabled): %v", err)
	} else {
		svc.WithStorage(emailStorage.New(minioClient))
		app.Logger.Info("object storage connected")
	}

	grpcServer := grpc.NewServer()
	emailGrpcHandler := grpcDelivery.New(svc)
	emailpb.RegisterEmailServiceServer(grpcServer, emailGrpcHandler)

	lis, err := net.Listen("tcp", ":"+app.Config.GRPC.EmailPort)
	if err != nil {
		app.Logger.Fatalf("grpc listen error: %v", err)
	}

	go func() {
		app.Logger.Infof("grpc started on %s", app.Config.GRPC.EmailPort)
		if err := grpcServer.Serve(lis); err != nil {
			app.Logger.Fatalf("grpc serve error: %v", err)
		}
	}()

	emailHandler := emailHttp.New(svc, emailHttp.Config{TTL: app.Config.JWTManager.TTL()})

	m := metrics.New("email", "backend")

	router := mux.NewRouter()
	router.Handle("/metrics", m.Handler())
	router.Use(middleware.Logging(app.Logger))
	router.Use(middleware.Metrics(m))

	public := router.PathPrefix("/api/v1/email").Subrouter()
	public.Use(middleware.Panic)
	public.Use(middleware.CORS(app.Config.CORS))
	public.Use(middleware.JSON)

	private := public.PathPrefix("").Subrouter()
	private.Use(middleware.AuthMiddleware(&app.Config.JWTManager))
	private.Use(middleware.CSRFMiddleware)

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
}

// newMinioClient builds an S3-compatible client pointing at MinIO.
func newMinioClient(cfg MinioConfig) (*s3.Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				SigningRegion:     cfg.Region,
				HostnameImmutable: true,
			}, nil
		},
	)

	awsCfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("minio aws config: %w", err)
	}

	return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // required for MinIO
	}), nil
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
