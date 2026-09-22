package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/domain/user"
)

// ErrInvalidCredentials cobre e-mail inexistente e senha errada — de
// propósito, para não revelar qual dos dois falhou.
var ErrInvalidCredentials = errors.New("credenciais inválidas")

// LoginUseCase verifica credenciais e, se válidas, emite uma nova sessão.
type LoginUseCase struct {
	users   *postgres.UserRepository
	session *sessionIssuer
}

func NewLoginUseCase(
	users *postgres.UserRepository,
	refreshTokens *postgres.RefreshTokenRepository,
	tokens *TokenIssuer,
	now func() time.Time,
) *LoginUseCase {
	return &LoginUseCase{
		users:   users,
		session: &sessionIssuer{refreshTokens: refreshTokens, tokens: tokens, now: now},
	}
}

// Login verifica e-mail e senha e devolve uma nova sessão.
func (uc *LoginUseCase) Login(ctx context.Context, email, password string) (*Session, error) {
	u, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	ok, err := user.VerifyPassword(u.PasswordHash, password)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}

	session, _, err := uc.session.issue(ctx, u.ID)
	return session, err
}
