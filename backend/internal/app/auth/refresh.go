package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
)

// ErrInvalidRefreshToken cobre token inexistente, expirado, revogado e
// reusado — sem distinção externa, para não dar pistas a quem está tentando
// os quatro casos.
var ErrInvalidRefreshToken = errors.New("refresh token inválido")

// RefreshUseCase rotaciona um refresh token válido e detecta reuso.
type RefreshUseCase struct {
	refreshTokens *postgres.RefreshTokenRepository
	session       *sessionIssuer
}

func NewRefreshUseCase(
	refreshTokens *postgres.RefreshTokenRepository,
	tokens *TokenIssuer,
	now func() time.Time,
) *RefreshUseCase {
	return &RefreshUseCase{
		refreshTokens: refreshTokens,
		session:       &sessionIssuer{refreshTokens: refreshTokens, tokens: tokens, now: now},
	}
}

// Refresh troca um refresh token válido por uma nova sessão. Ver ADR-0008: se
// o token apresentado já tinha sido rotacionado antes, isso é tratado como
// sinal de roubo e revoga todos os refresh tokens ativos do usuário.
func (uc *RefreshUseCase) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	now := uc.session.now()
	hash := HashRefreshToken(refreshToken)

	record, err := uc.refreshTokens.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	if record.RevokedAt != nil || record.ExpiresAt.Before(now) {
		return nil, ErrInvalidRefreshToken
	}

	if record.RotatedAt != nil {
		if revokeErr := uc.refreshTokens.RevokeAllForUser(ctx, record.UserID, now); revokeErr != nil {
			return nil, revokeErr
		}
		return nil, ErrInvalidRefreshToken
	}

	session, newRecord, err := uc.session.issue(ctx, record.UserID)
	if err != nil {
		return nil, err
	}

	if err := uc.refreshTokens.MarkRotated(ctx, record.ID, newRecord.ID, now); err != nil {
		return nil, err
	}

	return session, nil
}
