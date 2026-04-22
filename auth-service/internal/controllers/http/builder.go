package http

import (
	"github.com/gin-gonic/gin"
	"github.com/meindokuse/cloud-drive/auth-service/internal/usecase/oauth"
)

type RoutesBuilder struct {
	authController  *AuthController
	oauthController *OAuthController
}

func NewRoutesBuilder(
	authUseCase AuthUseCase,
	oauthUseCase oauth.OAuthUseCase,
	googleProvider oauth.OAuthProviderService,
	config ControllerConfig,
) *RoutesBuilder {
	var authController *AuthController
	if authUseCase != nil {
		authController = NewAuthController(authUseCase, config)
	}

	var oauthController *OAuthController
	if oauthUseCase != nil {
		oauthController = NewOAuthController(oauthUseCase, googleProvider, config)
	}

	return &RoutesBuilder{
		authController:  authController,
		oauthController: oauthController,
	}
}

func (b *RoutesBuilder) MountAuth(router gin.IRouter) {
	if b.authController == nil {
		return
	}

	authGroup := router.Group("/auth")
	authGroup.POST("/register", b.authController.Register)
	authGroup.POST("/login", b.authController.Login)
	authGroup.POST("/refresh", b.authController.Refresh)
	authGroup.POST("/logout", b.authController.Logout)
	authGroup.POST("/logout-all", b.authController.LogoutAll)
	authGroup.GET("/sessions", b.authController.Sessions)
	authGroup.DELETE("/sessions/:session_id", b.authController.RevokeSession)
	authGroup.POST("/introspect", b.authController.Introspect)
}

func (b *RoutesBuilder) MountOAuth(router gin.IRouter) {
	if b.oauthController == nil {
		return
	}

	authGroup := router.Group("/auth")
	authGroup.GET("/google", b.oauthController.GoogleAuth)
	authGroup.GET("/google/callback", b.oauthController.GoogleCallback)
	authGroup.GET("/link/google", b.oauthController.LinkGoogle)
	authGroup.GET("/link/google/callback", b.oauthController.LinkGoogleCallback)
}

func (b *RoutesBuilder) MountAll(router gin.IRouter) {
	b.MountAuth(router)
	b.MountOAuth(router)
}
