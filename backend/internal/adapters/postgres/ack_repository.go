package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AckKind é o tipo de Reconhecimento registrado (CONTEXT.md).
type AckKind string

const (
	AckKindCompletion AckKind = "completion"
	AckKindSnooze     AckKind = "snooze"
	AckKindDismissal  AckKind = "dismissal"
)

// AckSource é de onde o Reconhecimento chegou. Só "web" existe no MVP —
// Reconhecimento por Canal é trabalho futuro (ADR-0006).
type AckSource string

const (
	AckSourceWeb     AckSource = "web"
	AckSourceChannel AckSource = "channel"
	AckSourceSystem  AckSource = "system"
)

// AckModel é o modelo GORM da tabela acks — log de auditoria de
// Reconhecimentos, só INSERT (migration 0012).
type AckModel struct {
	ID            uuid.UUID  `gorm:"column:id"`
	OccurrenceID  uuid.UUID  `gorm:"column:occurrence_id"`
	Kind          string     `gorm:"column:kind"`
	Source        string     `gorm:"column:source"`
	ChannelID     *uuid.UUID `gorm:"column:channel_id"`
	CorrelationID *string    `gorm:"column:correlation_id"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
}

func (AckModel) TableName() string {
	return "acks"
}

// AckRepository dá acesso à tabela acks via GORM.
type AckRepository struct {
	db *gorm.DB
}

func NewAckRepository(db *gorm.DB) *AckRepository {
	return &AckRepository{db: db}
}

// Create insere um novo Ack. Nunca há Update/Delete — é o dado mais valioso
// do sistema (CONTEXT.md), e um log de auditoria não se corrige, só se
// acumula.
func (r *AckRepository) Create(ctx context.Context, m *AckModel) error {
	return r.db.WithContext(ctx).Create(m).Error
}
