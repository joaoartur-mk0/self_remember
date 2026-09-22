package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	"self-remember/backend/internal/app/auth"
)

// NewRouter monta as rotas do cmd/api: health check e swagger públicos,
// autenticação, e /ping como rota protegida de referência para o
// AuthMiddleware até que exista uma rota de negócio de verdade.
func NewRouter(authHandler *AuthHandler, tokens *auth.TokenIssuer) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Só para ambiente local/dev: em produção precisa ficar atrás de
	// Traefik/auth antes de ser exposto (ARCHITECTURE.md §9, fase de
	// Produção ainda não implementada).
	router.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

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
