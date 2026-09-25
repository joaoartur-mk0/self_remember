package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"self-remember/backend/internal/app/auth"
)

// contextUserIDKey é a chave usada para guardar o ID do usuário autenticado
// no gin.Context, depois de passar pelo AuthMiddleware.
const contextUserIDKey = "user_id"

// AuthMiddleware exige um access_token JWT válido no cookie HttpOnly. Nunca
// aceita o token por header Authorization: ARCHITECTURE.md §7 define o
// cookie como o único transporte.
func AuthMiddleware(tokens *auth.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := c.Cookie("access_token")
		if err != nil || accessToken == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		userID, err := tokens.ParseAccessToken(accessToken)
		if err != nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Set(contextUserIDKey, userID)
		c.Next()
	}
}

// userIDFrom lê o ID do usuário autenticado, gravado pelo AuthMiddleware.
// Só é chamado por handlers atrás dele, então a asserção de tipo nunca
// deveria falhar — um pânico aqui denuncia uma rota sem o middleware, não
// uma entrada do usuário.
func userIDFrom(c *gin.Context) uuid.UUID {
	return c.MustGet(contextUserIDKey).(uuid.UUID)
}
