package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"self-remember/backend/internal/adapters/postgres"
)

// Session é o par de tokens devolvido por login e refresh.
type Session struct {
	AccessToken  string
	RefreshToken string
}

// sessionIssuer emite uma nova sessão e persiste o refresh token gerado.
// Compartilhado entre LoginUseCase e RefreshUseCase para não duplicar a
// emissão em dois lugares.
type sessionIssuer struct {
	refreshTokens *postgres.RefreshTokenRepository
	tokens        *TokenIssuer
	now           func() time.Time
}

func (s *sessionIssuer) issue(ctx context.Context, userID uuid.UUID) (*Session, *postgres.RefreshToken, error) {
	now := s.now()

	accessToken, err := s.tokens.IssueAccessToken(userID, now)
	if err != nil {
		return nil, nil, err
	}

	refreshTokenPlain, err := GenerateRefreshToken()
	if err != nil {
		return nil, nil, err
	}

	record := &postgres.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: HashRefreshToken(refreshTokenPlain),
		ExpiresAt: now.Add(RefreshTokenTTL),
		CreatedAt: now,
	}
	if err := s.refreshTokens.Create(ctx, record); err != nil {
		return nil, nil, err
	}

	return &Session{AccessToken: accessToken, RefreshToken: refreshTokenPlain}, record, nil
}
