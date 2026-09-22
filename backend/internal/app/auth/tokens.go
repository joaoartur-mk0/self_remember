// Package auth orquestra login e refresh: verifica credenciais e hashes via
// internal/domain/user, persiste refresh tokens via internal/adapters/postgres
// e emite/valida os JWT de acesso.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// AccessTokenTTL é fixo em 15 minutos por ARCHITECTURE.md §7.
	AccessTokenTTL = 15 * time.Minute
	// RefreshTokenTTL é a janela máxima que um refresh token pode ficar sem
	// ser usado antes de expirar, independentemente de rotação.
	RefreshTokenTTL = 30 * 24 * time.Hour

	refreshTokenBytes = 32
)

// TokenIssuer assina e valida os JWT de acesso.
type TokenIssuer struct {
	secret []byte
}

// NewTokenIssuer recebe o segredo de assinatura já carregado da config —
// nunca hardcoded.
func NewTokenIssuer(secret []byte) *TokenIssuer {
	return &TokenIssuer{secret: secret}
}

// IssueAccessToken emite um JWT cujo subject é o ID do usuário.
func (i *TokenIssuer) IssueAccessToken(userID uuid.UUID, now time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(i.secret)
}

// ParseAccessToken valida a assinatura e a expiração de um JWT de acesso e
// devolve o ID do usuário autenticado.
func (i *TokenIssuer) ParseAccessToken(tokenString string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return i.secret, nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("token inválido: %w", err)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("subject inválido: %w", err)
	}

	return userID, nil
}

// GenerateRefreshToken gera um token opaco de alta entropia. Ele nunca é
// persistido em texto puro — só seu hash (HashRefreshToken).
func GenerateRefreshToken() (string, error) {
	buf := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("gerar refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashRefreshToken usa sha256 puro (sem custo computacional deliberado): um
// refresh token já tem entropia alta, diferente de senha, então não precisa
// de argon2 para resistir a força bruta sobre o hash.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
