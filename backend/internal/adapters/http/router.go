package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"self-remember/backend/internal/app/auth"
)

// NewRouter monta as rotas do cmd/api: health check público, autenticação e
// as rotas de negócio (ainda nenhuma além do healthz) atrás do
// AuthMiddleware.
func NewRouter(authHandler *AuthHandler, tokens *auth.TokenIssuer) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	v1 := router.Group("/api/v1")

	authGroup := v1.Group("/auth")
	authGroup.POST("/login", authHandler.Login)
	authGroup.POST("/refresh", authHandler.Refresh)

	protected := v1.Group("")
	protected.Use(AuthMiddleware(tokens))
	protected.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return router
}
