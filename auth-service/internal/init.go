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

	"github.com/meindokuse/cloud-drive/auth-service-new/config"
	appmiddleware "github.com/meindokuse/cloud-drive/auth-service-new/internal/app/middleware"
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
	"github.com/meindokuse/cloud-drive/auth-service-new/internal/pkg/pass"
)

func (a *App) init(ctx context.Context) error {
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

func (a *App) initHTTPServer(_ context.Context) error {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(appmiddleware.Logging())

	corsConfig := config.DefaultCORSConfig()
	router.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return strings.HasPrefix(origin, "http://localhost") ||
				strings.HasPrefix(origin, "http://127.0.0.1")
		},
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
	oauthHandler := oauthv1.NewOAuthService(a.oauthServices, a.oauthGateway.Google, a.cfg.Controller)

	// --- Public auth routes (no JWT required) ---
	authPublic := router.Group("/auth")
	authPublic.POST("/register", authHandler.Register)
	authPublic.POST("/login", authHandler.Login)
	authPublic.POST("/refresh", authHandler.Refresh)
	authPublic.POST("/introspect", authHandler.Introspect)
	authPublic.GET("/validate", authHandler.Validate)

	// --- Private auth routes (Traefik validates JWT, X-Account-ID forwarded) ---
	authPrivate := router.Group("/auth")
	authPrivate.Use(appmiddleware.RequireAccountID())
	authPrivate.POST("/logout", authHandler.Logout)
	authPrivate.POST("/logout-all", authHandler.LogoutAll)
	authPrivate.GET("/sessions", authHandler.Sessions)
	authPrivate.DELETE("/sessions/:session_id", authHandler.RevokeSession)

	// --- Public OAuth routes ---
	oauthPublic := router.Group("/auth")
	oauthPublic.GET("/google", oauthHandler.GoogleAuth)
	oauthPublic.GET("/google/callback", oauthHandler.GoogleCallback)

	// --- Private OAuth routes (Traefik validates JWT, X-Account-ID forwarded) ---
	oauthPrivate := router.Group("/auth")
	oauthPrivate.Use(appmiddleware.RequireAccountID())
	oauthPrivate.GET("/link/google", oauthHandler.LinkGoogle)
	oauthPrivate.GET("/link/google/callback", oauthHandler.LinkGoogleCallback)
	oauthPrivate.GET("/linked", oauthHandler.GetLinkedAccounts)
	oauthPrivate.DELETE("/linked/:provider", oauthHandler.Unlink)

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
