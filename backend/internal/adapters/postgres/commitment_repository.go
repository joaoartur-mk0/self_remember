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

// CommitmentModel é o modelo GORM da tabela commitments, usado pelos métodos
// de escrita (CRUD da API) — FindActiveWithTimezone usa uma projeção própria
// porque precisa de um join que o modelo sozinho não expressa.
type CommitmentModel struct {
	ID              uuid.UUID  `gorm:"column:id"`
	UserID          uuid.UUID  `gorm:"column:user_id"`
	AreaID          uuid.UUID  `gorm:"column:area_id"`
	Title           string     `gorm:"column:title"`
	RecurrenceStart time.Time  `gorm:"column:recurrence_start"`
	RecurrenceRule  string     `gorm:"column:recurrence_rule"`
	NagProfile      string     `gorm:"column:nag_profile"`
	CarryOverPolicy string     `gorm:"column:carry_over_policy"`
	ArchivedAt      *time.Time `gorm:"column:archived_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (CommitmentModel) TableName() string {
	return "commitments"
}

func (m CommitmentModel) toDomain() commitment.Commitment {
	return commitment.Commitment{
		ID:              m.ID,
		UserID:          m.UserID,
		AreaID:          m.AreaID,
		Title:           m.Title,
		RecurrenceStart: m.RecurrenceStart,
		RecurrenceRule:  m.RecurrenceRule,
		NagProfile:      commitment.NagProfile(m.NagProfile),
		CarryOverPolicy: commitment.CarryOverPolicy(m.CarryOverPolicy),
		ArchivedAt:      m.ArchivedAt,
	}
}

func commitmentModelFromDomain(c commitment.Commitment) CommitmentModel {
	return CommitmentModel{
		ID:              c.ID,
		UserID:          c.UserID,
		AreaID:          c.AreaID,
		Title:           c.Title,
		RecurrenceStart: c.RecurrenceStart,
		RecurrenceRule:  c.RecurrenceRule,
		NagProfile:      string(c.NagProfile),
		CarryOverPolicy: string(c.CarryOverPolicy),
		ArchivedAt:      c.ArchivedAt,
	}
}

// Create insere um novo Compromisso, já validado pelo domínio pelo chamador.
func (r *CommitmentRepository) Create(ctx context.Context, c commitment.Commitment) error {
	m := commitmentModelFromDomain(c)
	return r.db.WithContext(ctx).Create(&m).Error
}

// FindByID busca um Compromisso pelo ID. Retorna gorm.ErrRecordNotFound se
// não existir.
func (r *CommitmentRepository) FindByID(ctx context.Context, id uuid.UUID) (commitment.Commitment, error) {
	var m CommitmentModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return commitment.Commitment{}, err
	}
	return m.toDomain(), nil
}

// FindByUser lista os Compromissos de um usuário, mais recentes primeiro.
func (r *CommitmentRepository) FindByUser(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]commitment.Commitment, error) {
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if !includeArchived {
		q = q.Where("archived_at IS NULL")
	}

	var models []CommitmentModel
	if err := q.Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]commitment.Commitment, 0, len(models))
	for _, m := range models {
		result = append(result, m.toDomain())
	}
	return result, nil
}

// Update sobrescreve os campos editáveis de um Compromisso existente.
func (r *CommitmentRepository) Update(ctx context.Context, c commitment.Commitment) error {
	m := commitmentModelFromDomain(c)
	return r.db.WithContext(ctx).
		Model(&CommitmentModel{}).
		Where("id = ?", c.ID).
		Updates(map[string]any{
			"area_id":           m.AreaID,
			"title":             m.Title,
			"recurrence_start":  m.RecurrenceStart,
			"recurrence_rule":   m.RecurrenceRule,
			"nag_profile":       m.NagProfile,
			"carry_over_policy": m.CarryOverPolicy,
		}).Error
}

// Archive marca um Compromisso como arquivado.
func (r *CommitmentRepository) Archive(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.db.WithContext(ctx).
		Model(&CommitmentModel{}).
		Where("id = ?", id).
		Update("archived_at", at).Error
}

// SetLabels substitui o conjunto de Etiquetas de um Compromisso —
// classificação livre, sem efeito na cobrança (CONTEXT.md).
func (r *CommitmentRepository) SetLabels(ctx context.Context, commitmentID uuid.UUID, labelIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM commitment_labels WHERE commitment_id = ?", commitmentID).Error; err != nil {
			return err
		}
		for _, labelID := range labelIDs {
			if err := tx.Exec(
				"INSERT INTO commitment_labels (commitment_id, label_id) VALUES (?, ?)",
				commitmentID, labelID,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindLabelIDs devolve os IDs das Etiquetas associadas a um Compromisso.
func (r *CommitmentRepository) FindLabelIDs(ctx context.Context, commitmentID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).
		Table("commitment_labels").
		Where("commitment_id = ?", commitmentID).
		Pluck("label_id", &ids).Error
	return ids, err
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
