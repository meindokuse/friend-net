package internal

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	authv1 "github.com/meindokuse/cloud-drive/auth-service-new/internal/app/auth/v1"
	oauthv1 "github.com/meindokuse/cloud-drive/auth-service-new/internal/app/oauth/v1"
	authservice "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/auth"
	oauthservice "github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/application/service/oauth/providers"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/domain/entity"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/gateway/oauth"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/messagebus"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/infrastructure/storage"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/connector/postgres"
	redisconn "github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/connector/redis"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/event/flusher/platform"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/jwt"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/logger"
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/pass"
)

func (a *App) init(ctx context.Context) error {
	logger.Init("info")
	if err := a.initPostgres(ctx); err != nil {
		return err
	}
	if err := a.initRedis(ctx); err != nil {
		return err
	}
	if err := a.initJWT(ctx); err != nil {
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

func (a *App) initJWT(ctx context.Context) error {
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
	slog.InfoContext(ctx, "jwt manager initialized", "issuer", a.cfg.JWT.Issuer)
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
	refreshTTL := int64(a.cfg.JWT.RefreshTTL.Seconds())
	a.authServices = authservice.NewRegistry(
		a.storages,
		a.oauthGateway,
		a.jwt,
		a.hasher,
		refreshTTL,
	)

	providerMap := make(map[entity.OAuthProvider]providers.OAuthProviderGateway)
	if a.oauthGateway.Google != nil {
		providerMap[entity.OAuthGoogle] = &oauthProviderAdapter{client: a.oauthGateway.Google}
	}

	a.oauthServices = oauthservice.NewRegistry(
		a.storages,
		providerMap,
		a.jwt,
		refreshTTL,
	)
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		traceID := uuid.NewString()

		ctx := logger.InitRequestContext(c.Request.Context(), traceID, c.FullPath())
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Trace-Id", traceID)
		

		slog.DebugContext(ctx, "request started",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"remote_addr", c.ClientIP(),
		)

		c.Next()

		status := c.Writer.Status()
		duration := time.Since(start)

		// Enrich with user_id after handlers run (set by authMiddleware)
		finalCtx := ctx
		if accountID, exists := c.Get("account_id"); exists {
			if id, ok := accountID.(string); ok && id != "" {
				finalCtx = logger.WithUserIDEntry(finalCtx, id)
			}
		}

		lvl := slog.LevelInfo
		if status >= 500 {
			lvl = slog.LevelError
		} else if status >= 400 {
			lvl = slog.LevelWarn
		}

		slog.Log(finalCtx, lvl, "request completed",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", status,
			"duration_ms", duration.Milliseconds(),
		)
	}
}

func (a *App) initHTTPServer(_ context.Context) error {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(loggingMiddleware())

	corsConfig := config.DefaultCORSConfig()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     corsConfig.AllowOrigins,
		AllowMethods:     corsConfig.AllowMethods,
		AllowHeaders:     corsConfig.AllowHeaders,
		AllowCredentials: corsConfig.AllowCredentials,
		ExposeHeaders:    corsConfig.ExposeHeaders,
		MaxAge:           corsConfig.MaxAge,
	}))

	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

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
	authGroup.GET("/validate", authHandler.Validate)

	oauthHandler := oauthv1.NewOAuthService(a.oauthServices, a.oauthGateway.Google, a.cfg.Controller)
	oauthGroup := router.Group("/auth")
	oauthGroup.GET("/google", oauthHandler.GoogleAuth)
	oauthGroup.GET("/google/callback", oauthHandler.GoogleCallback)

	oauthProtected := router.Group("/auth")
	// oauthProtected.Use(a.authMiddleware())
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

	f := platform.NewFlusher(
		a.pool,
		producer,
		a.cfg.Outbox,
		a.cfg.Kafka.Topic,
	)
	a.flusher = f

	flushCtx, cancel := context.WithCancel(ctx)
	a.flushCancel = cancel

	go f.Start(flushCtx)
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

		// Propagate user_id into request context so service-level logs include it.
		enriched := logger.WithUserIDEntry(ctx.Request.Context(), result.AccountID)
		ctx.Request = ctx.Request.WithContext(enriched)

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

// oauthProviderAdapter adapts oauth.GoogleClient to providers.OAuthProviderGateway.
type oauthProviderAdapter struct {
	client *oauth.GoogleClient
}

func (a *oauthProviderAdapter) ExchangeToken(ctx context.Context, code string) (*providers.OAuthToken, error) {
	tokens, err := a.client.ExchangeToken(ctx, code)
	if err != nil {
		return nil, err
	}
	return &providers.OAuthToken{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		Expiry:       tokens.Expiry,
	}, nil
}

func (a *oauthProviderAdapter) GetUserInfo(ctx context.Context, accessToken string) (*providers.OAuthUserInfo, error) {
	info, err := a.client.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return &providers.OAuthUserInfo{
		ProviderID: info.ProviderID,
		Email:      info.Email,
		Name:       info.Name,
		AvatarURL:  info.AvatarURL,
	}, nil
}
