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
	smtppkg "github.com/go-park-mail-ru/2026_1_PushToMain/pkg/smtp"
	"go.uber.org/zap"

	userClient "github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/clients/user"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/logger"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/metrics"
	"github.com/go-park-mail-ru/2026_1_PushToMain/internal/pkg/middleware"
	lmtpDelivery "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/lmtp"
	emailHttp "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/delivery/http"
	emailRepo "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/repository"
	emailStorage "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/repository/storage"
	emailService "github.com/go-park-mail-ru/2026_1_PushToMain/microservices/email/service"
	"github.com/gorilla/mux"

	"net"

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

	// SMTP-клиент для исходящих писем через Postfix.
	// Если хост не задан — smtpClient будет nil, исходящие через Postfix отключены.
	var smtpClient emailService.SmtpClient
	if app.Config.SMTP.Host != "" {
		smtpClient = smtppkg.NewSmtpClient(
			app.Config.SMTP.Host,
			app.Config.SMTP.Port,
			"", "", // auth не нужен, Postfix доверяет mynetworks
		)
		app.Logger.Infof("smtp client configured: %s:%s", app.Config.SMTP.Host, app.Config.SMTP.Port)
	}

	svc := emailService.New(
		repo,
		grpcUserClient,
		emailService.DraftsConfig{MaxPerUser: app.Config.Drafts.MaxPerUser},
		smtpClient,
	)

	// MinIO / S3-совместимое хранилище для вложений.
	// Если не сконфигурировано — вложения недоступны, сервис продолжает работу.
	minioClient, err := minio.New(context.TODO(), app.Config.S3)
	if err != nil {
		app.Logger.Errorf("minio init error (attachments disabled): %v", err)
	} else {
		strg, err := emailStorage.New(minioClient)
		if err != nil {
			app.Logger.Warnf("failed to init s3 Storage: %v", err)
		} else {
			svc.WithStorage(strg)
			app.Logger.Info("object storage connected")
		}
	}

	// LMTP-сервер для входящих писем от Postfix.
	// Запускаем в горутине — он работает параллельно с HTTP и gRPC.
	if app.Config.LMTP.Addr != "" {
		lmtpSrv := lmtpDelivery.NewServer(svc, app.Config.LMTP.Addr)
		go func() {
			app.Logger.Infof("lmtp server started on %s", app.Config.LMTP.Addr)
			if err := lmtpSrv.ListenAndServe(); err != nil {
				app.Logger.Errorf("lmtp server error: %v", err)
			}
		}()
	}

	// gRPC сервер
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

	// HTTP сервер
	emailHandler := emailHttp.New(svc, emailHttp.Config{
		TTL: app.Config.JWTManager.TTL()})

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
