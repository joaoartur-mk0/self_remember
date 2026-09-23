package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeliveryModel é o modelo GORM da tabela deliveries.
type DeliveryModel struct {
	ID                uuid.UUID `gorm:"column:id"`
	OccurrenceID      uuid.UUID `gorm:"column:occurrence_id"`
	StepIndex         int       `gorm:"column:step_index"`
	ChannelID         uuid.UUID `gorm:"column:channel_id"`
	CorrelationID     string    `gorm:"column:correlation_id"`
	ProviderMessageID *string   `gorm:"column:provider_message_id"`
	Status            string    `gorm:"column:status"`
	CreatedAt         time.Time `gorm:"column:created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

func (DeliveryModel) TableName() string {
	return "deliveries"
}

const (
	DeliveryStatusPending = "pending"
	DeliveryStatusSent    = "sent"
	DeliveryStatusFailed  = "failed"
)

// DeliveryRepository dá acesso à tabela deliveries via GORM.
type DeliveryRepository struct {
	db *gorm.DB
}

func NewDeliveryRepository(db *gorm.DB) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

// CreateIfNotExists insere a linha de Entrega antes de qualquer chamada
// externa (ARCHITECTURE.md §3.3). Devolve created=false sem erro se a
// colisão em (occurrence_id, step_index, channel_id) indicar que esta
// Entrega já foi tentada — é essa colisão que garante a idempotência, não
// uma checagem prévia.
func (r *DeliveryRepository) CreateIfNotExists(ctx context.Context, d *DeliveryModel) (bool, error) {
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(d)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// MarkSent registra o envio bem-sucedido e o ID que o provedor atribuiu.
func (r *DeliveryRepository) MarkSent(ctx context.Context, id uuid.UUID, providerID string) error {
	return r.db.WithContext(ctx).
		Model(&DeliveryModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":              DeliveryStatusSent,
			"provider_message_id": providerID,
		}).Error
}

// MarkFailed registra falha de envio por este Canal. O Entregador não
// estoura em falha (ARCHITECTURE.md §3.3): ele grava a falha e, se houver
// fallback, enfileira uma nova tentativa por outro Canal.
func (r *DeliveryRepository) MarkFailed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&DeliveryModel{}).
		Where("id = ?", id).
		Update("status", DeliveryStatusFailed).Error
}
