package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"self-remember/backend/internal/domain/commitment"
)

// ActiveCommitment é um Compromisso ativo junto do fuso do seu usuário — o
// Materializador precisa dos dois juntos para expandir a recorrência
// (ARCHITECTURE.md §3.1).
type ActiveCommitment struct {
	Commitment commitment.Commitment
	Timezone   string
}

// CommitmentRepository dá acesso à tabela commitments via GORM. Só tem o
// método que o Materializador usa hoje — escrita (criar/editar Compromisso)
// entra na fase de API/CRUD.
type CommitmentRepository struct {
	db *gorm.DB
}

func NewCommitmentRepository(db *gorm.DB) *CommitmentRepository {
	return &CommitmentRepository{db: db}
}

type activeCommitmentRow struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	AreaID          uuid.UUID
	Title           string
	RecurrenceStart time.Time
	RecurrenceRule  string
	NagProfile      string
	CarryOverPolicy string
	Timezone        string
}

// FindActiveWithTimezone devolve todos os Compromissos não arquivados, junto
// do fuso IANA do usuário dono de cada um (user_settings.timezone).
func (r *CommitmentRepository) FindActiveWithTimezone(ctx context.Context) ([]ActiveCommitment, error) {
	var rows []activeCommitmentRow

	err := r.db.WithContext(ctx).
		Table("commitments AS c").
		Select("c.id, c.user_id, c.area_id, c.title, c.recurrence_start, c.recurrence_rule, c.nag_profile, c.carry_over_policy, us.timezone").
		Joins("JOIN user_settings us ON us.user_id = c.user_id").
		Where("c.archived_at IS NULL").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]ActiveCommitment, 0, len(rows))
	for _, row := range rows {
		result = append(result, ActiveCommitment{
			Commitment: commitment.Commitment{
				ID:              row.ID,
				UserID:          row.UserID,
				AreaID:          row.AreaID,
				Title:           row.Title,
				RecurrenceStart: row.RecurrenceStart,
				RecurrenceRule:  row.RecurrenceRule,
				NagProfile:      commitment.NagProfile(row.NagProfile),
				CarryOverPolicy: commitment.CarryOverPolicy(row.CarryOverPolicy),
			},
			Timezone: row.Timezone,
		})
	}
	return result, nil
}
