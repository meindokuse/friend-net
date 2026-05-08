package internal

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/login"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/flusher"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/gateway/oauth"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/messagebus"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/storage"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/closer"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/connector/postgres"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/pass"
	authv1 "github.com/meindokuse/cloud-drive/auth-service/internal/app/auth/v1"
	oauthv1 "github.com/meindokuse/cloud-drive/auth-service/internal/app/oauth/v1"
	authservice "github.com/meindokuse/cloud-drive/auth-service/internal/application/service/auth"
	oauthservice "github.com/meindokuse/cloud-drive/auth-service/internal/application/service/oauth"
	redisconn "github.com/meindokuse/cloud-drive/auth-service/internal/pkg/connector/redis"
)

// App is the main application structure
type App struct {
	cfg *config.Config

	pool   *pgxpool.Pool
	rdb    *redis.Client
	jwt    *jwt.Manager
	hasher *pass.Hasher

	storages      *storage.Registry
	authServices  *authservice.Registry
	oauthServices *oauthservice.Registry

	oauthGateway *oauth.Registry
	messageBus   *messagebus.Registry

	httpServer *http.Server
}

// New creates a new App instance
func New(ctx context.Context) *App {
	return &App{
		cfg: config.Instance(),
	}
}

// Run starts the application
func (a *App) Run(ctx context.Context) {
	// Initialize
	if err := a.init(ctx); err != nil {
		slog.ErrorContext(ctx, "initialization failed", "error", err)
		os.Exit(1)
	}

	// Start HTTP server
	go func() {
		slog.InfoContext(ctx, "auth-service starting", "http_addr", a.cfg.Server.HTTPAddr)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.ErrorContext(ctx, "http server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	slog.InfoContext(ctx, "shutdown signal received")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, a.cfg.Graceful.Timeout)
	defer cancel()

	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "shutdown error", "error", err)
	}

	// Close resources
	c := closer.New(a.cfg.Graceful.Timeout)
	c.Add(a.pool.Close)
	c.Add(func(ctx context.Context) error {
		return a.rdb.Close()
	})
	if a.messageBus.Producer != nil {
		c.Add(a.messageBus.Producer.Close)
	}
	c.CloseAll()

	slog.InfoContext(ctx, "auth-service stopped")
}

func (a *App) init(ctx context.Context) error {
	// Initialize components
	if err := a.initPostgres(ctx); err != nil {
		return err
	}

	if err := a.initRedis(ctx); err != nil {
		return err
	}

	if err := a.initJWT(); err != nil {
		return err
	}

	a.initHasher()
	a.initStorages()
	a.initOAuthGateway()

	if err := a.initMessageBus(); err != nil {
		return err
	}

	a.initServices()

	if err := a.initHTTPServer(ctx); err != nil {
		return err
	}

	a.initFlusher(ctx)

	return nil
}

func (a *App) initPostgres(ctx context.Context) error {
	pool, err := postgres.NewPool(ctx, a.cfg.Postgres)
	if err != nil {
		return err
	}
	a.pool = pool
	slog.InfoContext(ctx, "postgres connected", "host", a.cfg.Postgres.Host)
	return nil
}

func (a *App) initRedis(ctx context.Context) error {
	rdb, err := redisconn.NewClient(ctx, a.cfg.Redis)
	if err != nil {
		return err
	}
	a.rdb = rdb
	slog.InfoContext(ctx, "redis connected", "addr", a.cfg.Redis.Addr)
	return nil
}

func (a *App) initJWT() error {
	jwtManager, err := jwt.NewManager(
		a.cfg.JWT.SecretKey,
		a.cfg.JWT.RefreshSecret,
		a.cfg.JWT.Issuer,
		a.cfg.JWT.AccessTTL,
		a.cfg.JWT.RefreshTTL,
		a.cfg.JWT.GracePeriod,
	)
	if err != nil {
		return err
	}
	a.jwt = jwtManager
	slog.InfoContext(nil, "jwt manager initialized", "issuer", a.cfg.JWT.Issuer)
	return nil
}

func (a *App) initHasher() {
	a.hasher = pass.New(a.cfg.Pass.Cost)
}

func (a *App) initStorages() {
	refreshTTL := int64(a.cfg.JWT.RefreshTTL.Seconds())
	a.storages = storage.NewRegistry(a.pool, a.rdb, refreshTTL)
}

func (a *App) initOAuthGateway() {
	a.oauthGateway = oauth.NewRegistry(a.cfg.OAuth)
}

func (a *App) initMessageBus() error {
	a.messageBus = messagebus.NewRegistry(a.cfg.Kafka)
	return nil
}

func (a *App) initServices() {
	// Auth services
	refreshTTL := int64(a.cfg.JWT.RefreshTTL.Seconds())
	a.authServices = authservice.NewRegistry(
		a.storages,
		a.oauthGateway,
		a.jwt,
		a.hasher,
		refreshTTL,
	)

	// OAuth services - create providers map
	providers := make(map[entity.OAuthProvider]login.OAuthProviderGateway)
	if a.oauthGateway.Google != nil {
		providers[entity.OAuthGoogle] = &oauthProviderAdapter{client: a.oauthGateway.Google}
	}

	a.oauthServices = oauthservice.NewRegistry(
		a.storages,
		providers,
		a.jwt,
		refreshTTL,
	)
}

func (a *App) initHTTPServer(ctx context.Context) error {
	router := gin.Default()

	// CORS
	corsConfig := config.DefaultCORSConfig()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     corsConfig.AllowOrigins,
		AllowMethods:     corsConfig.AllowMethods,
		AllowHeaders:     corsConfig.AllowHeaders,
		AllowCredentials: corsConfig.AllowCredentials,
		ExposeHeaders:    corsConfig.ExposeHeaders,
		MaxAge:           corsConfig.MaxAge,
	}))

	// Health check
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Auth routes
	authHandler := authv1.NewAuthService(a.authServices, a.cfg.Controller)
	authGroup := router.Group("/auth")
	authGroup.POST("/register", authHandler.Register)
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)
	authGroup.POST("/logout", authHandler.Logout)
	authGroup.POST("/logout-all", authHandler.LogoutAll)
	authGroup.GET("/sessions", authHandler.Sessions)
	authGroup.DELETE("/sessions/:session_id", authHandler.RevokeSession)
	authGroup.POST("/introspect", authHandler.Introspect)

	// OAuth routes
	oauthHandler := oauthv1.NewOAuthService(a.oauthServices, a.oauthGateway.Google, a.cfg.Controller)
	oauthGroup := router.Group("/auth")
	oauthGroup.GET("/google", oauthHandler.GoogleAuth)
	oauthGroup.GET("/google/callback", oauthHandler.GoogleCallback)

	oauthProtected := router.Group("/auth")
	oauthProtected.Use(a.authMiddleware())
	oauthProtected.GET("/link/google", oauthHandler.LinkGoogle)
	oauthProtected.GET("/link/google/callback", oauthHandler.LinkGoogleCallback)
	oauthProtected.GET("/linked", oauthHandler.GetLinkedAccounts)
	oauthProtected.DELETE("/linked/:provider", oauthHandler.Unlink)

	a.httpServer = &http.Server{
		Addr:              a.cfg.Server.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return nil
}

func (a *App) initFlusher(ctx context.Context) {
	if !a.cfg.Outbox.FlushEnabled || !a.cfg.Kafka.Enabled {
		slog.InfoContext(ctx, "outbox flusher disabled")
		return
	}

	var producer sarama.SyncProducer
	if a.messageBus.Producer != nil {
		producer = a.messageBus.Producer.SyncProducer()
	}

	f := flusher.NewFlusher(
		a.storages.Outbox,
		producer,
		a.cfg.Outbox,
		a.cfg.Kafka.Topic,
	)

	go f.Start(ctx)
}

func (a *App) authMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		accessToken := extractBearerToken(ctx.GetHeader("Authorization"))
		if accessToken == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		result, err := a.authServices.Introspect.Introspect(ctx.Request.Context(), accessToken)
		if err != nil || !result.Active {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		ctx.Set("account_id", result.AccountID)
		ctx.Set("session_id", result.SessionID)
		ctx.Next()
	}
}

func extractBearerToken(authHeader string) string {
	value := strings.TrimSpace(authHeader)
	if value == "" {
		return ""
	}

	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// OAuth provider adapter
type oauthProviderAdapter struct {
	client *oauth.GoogleClient
}

func (a *oauthProviderAdapter) ExchangeToken(ctx context.Context, code string) (*login.OAuthToken, error) {
	tokens, err := a.client.ExchangeToken(ctx, code)
	if err != nil {
		return nil, err
	}
	return &login.OAuthToken{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Expiry:       tokens.Expiry,
	}, nil
}

func (a *oauthProviderAdapter) GetUserInfo(ctx context.Context, accessToken string) (*login.OAuthUserInfo, error) {
	info, err := a.client.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return &login.OAuthUserInfo{
		ProviderID: info.ProviderID,
		Email:      info.Email,
		Name:       info.Name,
		AvatarURL:  info.AvatarURL,
	}, nil
}
