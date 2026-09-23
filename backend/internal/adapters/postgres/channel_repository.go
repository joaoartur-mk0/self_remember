package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChannelModel é o modelo GORM da tabela channels. Config (credencial
// cifrada) não é lido nem escrito por este repositório ainda — nenhum
// adaptador real existe até a fase de Canais (só o canal falso de log).
type ChannelModel struct {
	ID            uuid.UUID  `gorm:"column:id"`
	UserID        uuid.UUID  `gorm:"column:user_id"`
	Kind          string     `gorm:"column:kind"`
	Address       string     `gorm:"column:address"`
	VerifiedAt    *time.Time `gorm:"column:verified_at"`
	FallbackOrder int        `gorm:"column:fallback_order"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (ChannelModel) TableName() string {
	return "channels"
}

// ChannelRepository dá acesso à tabela channels via GORM.
type ChannelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

// FindByID busca um Canal pelo ID.
func (r *ChannelRepository) FindByID(ctx context.Context, id uuid.UUID) (*ChannelModel, error) {
	var m ChannelModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// FindPrimaryVerified devolve o Canal verificado de menor fallback_order de
// um usuário — o primeiro da cadeia de fallback (ARCHITECTURE.md §3.3).
// Retorna gorm.ErrRecordNotFound se o usuário não tiver nenhum Canal
// verificado.
func (r *ChannelRepository) FindPrimaryVerified(ctx context.Context, userID uuid.UUID) (*ChannelModel, error) {
	var m ChannelModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND verified_at IS NOT NULL", userID).
		Order("fallback_order ASC").
		First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// NextFallback devolve o próximo Canal verificado na cadeia de fallback de
// um usuário, depois de afterOrder. Retorna gorm.ErrRecordNotFound se não
// houver mais nenhum — a cadeia se esgotou.
func (r *ChannelRepository) NextFallback(ctx context.Context, userID uuid.UUID, afterOrder int) (*ChannelModel, error) {
	var m ChannelModel
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND verified_at IS NOT NULL AND fallback_order > ?", userID, afterOrder).
		Order("fallback_order ASC").
		First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}
