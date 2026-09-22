package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"self-remember/backend/internal/app/auth"
)

// AuthHandler expõe login e refresh via cookie HttpOnly + SameSite=Strict
// (ARCHITECTURE.md §7).
type AuthHandler struct {
	login   *auth.LoginUseCase
	refresh *auth.RefreshUseCase
}

func NewAuthHandler(login *auth.LoginUseCase, refresh *auth.RefreshUseCase) *AuthHandler {
	return &AuthHandler{login: login, refresh: refresh}
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login autentica por e-mail e senha e devolve os cookies de sessão.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	session, err := h.login.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	setSessionCookies(c, session)
	c.Status(http.StatusNoContent)
}

// Refresh rotaciona o refresh token do cookie e devolve uma nova sessão.
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	session, err := h.refresh.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidRefreshToken) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	setSessionCookies(c, session)
	c.Status(http.StatusNoContent)
}

// setSessionCookies grava os dois cookies HttpOnly + SameSite=Strict.
// refresh_token é escopado a /api/v1/auth para não ser enviado a rotas que
// não precisam dele. Secure fica false até a fase de Produção trazer TLS via
// Traefik (ARCHITECTURE.md §9); não é omissão, é decisão explícita de escopo.
func setSessionCookies(c *gin.Context, session *auth.Session) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("access_token", session.AccessToken, int(auth.AccessTokenTTL.Seconds()), "/", "", false, true)
	c.SetCookie("refresh_token", session.RefreshToken, int(auth.RefreshTokenTTL.Seconds()), "/api/v1/auth", "", false, true)
}
