package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"

	"self-remember/backend/internal/app/auth"
)

// Handlers agrupa os handlers de negócio do cmd/api — um parâmetro só, em
// vez de uma lista de argumentos que cresce a cada rota nova.
type Handlers struct {
	Auth       *AuthHandler
	Area       *AreaHandler
	Label      *LabelHandler
	Settings   *SettingsHandler
	Commitment *CommitmentHandler
	Occurrence *OccurrenceHandler
}

// NewRouter monta as rotas do cmd/api: health check e swagger públicos,
// autenticação, e as rotas de negócio atrás do AuthMiddleware.
func NewRouter(h Handlers, tokens *auth.TokenIssuer) *gin.Engine {
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
	authGroup.POST("/login", h.Auth.Login)
	authGroup.POST("/refresh", h.Auth.Refresh)

	protected := v1.Group("")
	protected.Use(AuthMiddleware(tokens))
	protected.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	protected.POST("/areas", h.Area.Create)
	protected.GET("/areas", h.Area.List)

	protected.POST("/labels", h.Label.Create)
	protected.GET("/labels", h.Label.List)

	protected.GET("/settings", h.Settings.Get)
	protected.PUT("/settings", h.Settings.Update)

	protected.POST("/commitments", h.Commitment.Create)
	protected.GET("/commitments", h.Commitment.List)
	protected.POST("/commitments/:id/archive", h.Commitment.Archive)

	protected.GET("/occurrences/today", h.Occurrence.Today)
	protected.POST("/occurrences/:id/complete", h.Occurrence.Complete)
	protected.POST("/occurrences/:id/dismiss", h.Occurrence.Dismiss)
	protected.POST("/occurrences/:id/snooze", h.Occurrence.Snooze)

	return router
}
