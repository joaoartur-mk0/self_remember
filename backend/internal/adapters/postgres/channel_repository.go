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
	ID                    uuid.UUID  `gorm:"column:id"`
	UserID                uuid.UUID  `gorm:"column:user_id"`
	Kind                  string     `gorm:"column:kind"`
	Address               string     `gorm:"column:address"`
	VerifiedAt            *time.Time `gorm:"column:verified_at"`
	FallbackOrder         int        `gorm:"column:fallback_order"`
	VerificationCodeHash  *string    `gorm:"column:verification_code_hash"`
	VerificationExpiresAt *time.Time `gorm:"column:verification_expires_at"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at"`
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

// Create insere um novo Canal, ainda não verificado.
func (r *ChannelRepository) Create(ctx context.Context, m *ChannelModel) error {
	return r.db.WithContext(ctx).Create(m).Error
}

// FindByUser lista os Canais de um usuário, na ordem de fallback.
func (r *ChannelRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]ChannelModel, error) {
	var channels []ChannelModel
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("fallback_order ASC").
		Find(&channels).Error
	return channels, err
}

// CountByUser conta os Canais de um usuário — usado para calcular o próximo
// fallback_order ao criar um Canal (sempre ao fim da cadeia; reordenar fica
// para quando existir uma tela que precise disso).
func (r *ChannelRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&ChannelModel{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// Delete remove um Canal. Não é escopado por usuário — quem chama precisa
// confirmar posse antes (ex.: via FindByIDForUser), como já é o padrão de
// CommitmentRepository.Archive.
func (r *ChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&ChannelModel{}, "id = ?", id).Error
}

// FindByIDForUser busca um Canal pelo ID, restrito ao usuário informado —
// isolamento por user_id (ADR-0001).
func (r *ChannelRepository) FindByIDForUser(ctx context.Context, id, userID uuid.UUID) (*ChannelModel, error) {
	var m ChannelModel
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// SetVerificationCode grava o hash do código de verificação e sua validade.
// Um Canal só pode receber Entregas depois de Verificado (ARCHITECTURE.md
// §7).
func (r *ChannelRepository) SetVerificationCode(ctx context.Context, id uuid.UUID, codeHash string, expiresAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&ChannelModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"verification_code_hash":  codeHash,
			"verification_expires_at": expiresAt,
		}).Error
}

// MarkVerified marca o Canal como verificado e limpa o código usado.
func (r *ChannelRepository) MarkVerified(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&ChannelModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"verified_at":             at,
			"verification_code_hash":  nil,
			"verification_expires_at": nil,
		}).Error
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
