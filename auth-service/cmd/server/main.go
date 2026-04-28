package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	postgresqladapter "github.com/meindokuse/cloud-drive/auth-service/internal/adapters/postgresql"
	redisadapter "github.com/meindokuse/cloud-drive/auth-service/internal/adapters/redis"
	"github.com/meindokuse/cloud-drive/auth-service/internal/config"
	httpcontrollers "github.com/meindokuse/cloud-drive/auth-service/internal/controllers/http"
	googleinfra "github.com/meindokuse/cloud-drive/auth-service/internal/infra"
	"github.com/meindokuse/cloud-drive/auth-service/internal/pkg/postgres"
	usecase "github.com/meindokuse/cloud-drive/auth-service/internal/usecase/auth"
	oauthusecase "github.com/meindokuse/cloud-drive/auth-service/internal/usecase/oauth"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/jwt"
	sharedlogger "github.com/meindokuse/cloud-drive/auth-service/pkg/logger"
	"github.com/meindokuse/cloud-drive/auth-service/pkg/pass"
	redispkg "github.com/meindokuse/cloud-drive/auth-service/pkg/redis"
)

func main() {
	sharedlogger.Init()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	slog.InfoContext(context.Background(), "config loaded", slog.String("http_addr", cfg.Server.HTTPAddr))

	// Создаём PostgreSQL pool
	pgConfig := postgres.Config{
		Host:            cfg.Postgres.Host,
		Port:            cfg.Postgres.Port,
		User:            cfg.Postgres.User,
		Password:        cfg.Postgres.Password,
		Database:        cfg.Postgres.Database,
		SSLMode:         cfg.Postgres.SSLMode,
		MaxConns:        25,
		MinConns:        5,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: 1 * time.Minute,
		ConnectTimeout:  10 * time.Second,
	}

	pool, err := postgres.NewPool(context.Background(), pgConfig)
	if err != nil {
		log.Fatalf("create postgres pool: %v", err)
	}
	defer pool.Close()
	slog.InfoContext(context.Background(), "postgres connected", slog.String("host", cfg.Postgres.Host), slog.Int("port", cfg.Postgres.Port), slog.String("database", cfg.Postgres.Database))

	redisClient, err := redispkg.NewClient(cfg.Redis)
	if err != nil {
		log.Fatalf("open redis: %v", err)
	}
	defer redisClient.Close()
	slog.InfoContext(context.Background(), "redis connected", slog.String("addr", cfg.Redis.Addr), slog.Int("db", cfg.Redis.DB))

	jwtManager, err := jwt.NewManager(cfg.JWT)
	if err != nil {
		log.Fatalf("create jwt manager: %v", err)
	}
	slog.InfoContext(context.Background(), "jwt manager initialized", slog.String("issuer", cfg.JWT.Issuer))

	googleProvider := googleinfra.NewGoogleService(cfg.OAuth.Google)
	accountRepo := postgresqladapter.NewAuthRepository(pool)
	oauthRepo := postgresqladapter.NewOAuthRepo(pool)
	sessionRepo := redisadapter.NewManager(redisClient, cfg.JWT.RefreshTTL)
	hasher := pass.New(cfg.Pass)

	authUC := usecase.NewAuth(accountRepo, sessionRepo, hasher, jwtManager)
	oauthUC := oauthusecase.NewOAuth(accountRepo, oauthRepo, sessionRepo, jwtManager, cfg.JWT.RefreshTTL)
	oauthUC.RegisterProvider("google", googleProvider)

	corsConfig := config.DefaultCORSConfig()
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     corsConfig.AllowOrigins,
		AllowMethods:     corsConfig.AllowMethods,
		AllowHeaders:     corsConfig.AllowHeaders,
		AllowCredentials: corsConfig.AllowCredentials,
		ExposeHeaders:    corsConfig.ExposeHeaders,
		MaxAge:           corsConfig.MaxAge,
	}))

	router.Use(httpcontrollers.RequestContextLogger())
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	builder := httpcontrollers.NewRoutesBuilder(authUC, oauthUC, googleProvider, cfg.Controller)
	builder.MountAll(router)

	server := &http.Server{
		Addr:              cfg.Server.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.InfoContext(context.Background(), "auth-service starting", slog.String("http_addr", cfg.Server.HTTPAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownSignals

	shutdownTimeout, err := time.ParseDuration(cfg.Server.ShutdownTimeout)
	if err != nil {
		shutdownTimeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
		return
	}

	slog.InfoContext(context.Background(), "auth-service stopped")
}
