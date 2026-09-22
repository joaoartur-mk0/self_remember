package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshToken é o modelo GORM da tabela refresh_tokens. TokenHash é sempre
// sha256(token) — o valor bruto do refresh token nunca é persistido.
type RefreshToken struct {
	ID          uuid.UUID  `gorm:"column:id"`
	UserID      uuid.UUID  `gorm:"column:user_id"`
	TokenHash   string     `gorm:"column:token_hash"`
	RotatedAt   *time.Time `gorm:"column:rotated_at"`
	RotatedToID *uuid.UUID `gorm:"column:rotated_to_id"`
	RevokedAt   *time.Time `gorm:"column:revoked_at"`
	ExpiresAt   time.Time  `gorm:"column:expires_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// RefreshTokenRepository dá acesso à tabela refresh_tokens via GORM.
type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create insere um novo refresh token (já hasheado).
func (r *RefreshTokenRepository) Create(ctx context.Context, t *RefreshToken) error {
	return r.db.WithContext(ctx).Create(t).Error
}

// FindByTokenHash busca um refresh token pelo hash. Retorna
// gorm.ErrRecordNotFound se não existir.
func (r *RefreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	var t RefreshToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// MarkRotated marca um refresh token como usado, apontando para o token que o
// substituiu. Um token com rotated_at preenchido nunca deveria ser
// reapresentado; se for, é sinal de reuso.
func (r *RefreshTokenRepository) MarkRotated(ctx context.Context, id, rotatedToID uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).
		Model(&RefreshToken{}).
		Where("id = ?", id).
		Updates(map[string]any{"rotated_at": now, "rotated_to_id": rotatedToID}).Error
}

// RevokeAllForUser invalida todos os refresh tokens ainda ativos de um
// usuário. Chamado quando um token já rotacionado é reapresentado
// (ADR-0008): o reuso derruba a sessão inteira, não só a cadeia comprometida.
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return r.db.WithContext(ctx).
		Model(&RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error
}
