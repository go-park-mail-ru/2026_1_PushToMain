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

	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/minio"
	"github.com/go-park-mail-ru/2026_1_PushToMain/pkg/postgres"
	smtp "github.com/go-park-mail-ru/2026_1_PushToMain/pkg/smtp"
	"go.uber.org/zap"

	"net"

	userClient "github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/clients/user"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/metrics"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	emailHttp "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/http"
	"github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/lmtp"
	emailRepo "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/repository/db"
	emailStorage "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/repository/storage"
	emailService "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	"github.com/gorilla/mux"

	grpcDelivery "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/grpc"

	emailpb "github.com/go-park-mail-ru/2026_1_PushToMain/proto/email"

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

	repo := emailRepo.New(db)

	grpcUserClient, err := userClient.New(app.Config.GRPCClients.UserService)
	if err != nil {
		app.Logger.Fatalf("failed to init user grpc client: %v", err)
	}

	defer func() {
		if err := grpcUserClient.Close(); err != nil {
			app.Logger.Errorf("grpc client close error: %v", err)
		}
	}()

	svc := emailService.New(
		repo,
		grpcUserClient,
		emailService.DraftsConfig{MaxPerUser: app.Config.Drafts.MaxPerUser},
	)

	minioClient, err := minio.New(context.TODO(), app.Config.S3)
	if err != nil {
		app.Logger.Errorf("minio init error (attachments disabled): %v", err)
	} else {
		strg, err := emailStorage.New(minioClient)
		if err != nil {
			app.Logger.Warnf("failed to init s3 Storage: %v", err)
		}
		svc.WithStorage(strg)
		app.Logger.Info("object storage connected")
	}

	if app.Config.SMTP.Host != "" {
		smtpClient := smtp.NewSmtpClient(app.Config.SMTP.Host, app.Config.SMTP.Port, "", "")
		svc.WithSmtp(smtpClient)
		app.Logger.Infof("smtp client configured: %s:%s", app.Config.SMTP.Host, app.Config.SMTP.Port)
	} else {
		app.Logger.Error("empty SMTP client host config")
	}

	// LMTP
	lmtpServer := lmtp.NewServer(svc, app.Config.LMTP.Addr)
	lmtpListener, err := net.Listen("tcp", app.Config.LMTP.Addr)
	if err != nil {
		app.Logger.Fatalf("lmtp bind error: %v", err)
	}
	app.Logger.Infof("lmtp listening on %s", app.Config.LMTP.Addr)
	go func() {
		if err := lmtpServer.Serve(lmtpListener); err != nil {
			app.Logger.Errorf("lmtp serve stopped: %v", err)
		}
	}()

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

func (app *App) shutdownGracefully() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownMaxTime)
	defer cancel()

	app.Logger.Info("shutting down server")

	fullShutdown := make(chan struct{}, 1)
	go func() {
		if err := app.Server.Shutdown(shutdownCtx); err != nil {
			app.Logger.Errorf("HTTP server Shutdown: %v", err)
		}
		close(fullShutdown)
	}()
	select {
	case <-shutdownCtx.Done():
		return fmt.Errorf("server shutdown: %w", shutdownCtx.Err())
	case <-fullShutdown:
		app.Logger.Info("Server shut down successfully")
	}

	return nil
}
