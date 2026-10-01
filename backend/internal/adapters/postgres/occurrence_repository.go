package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"self-remember/backend/internal/domain/commitment"
	"self-remember/backend/internal/domain/occurrence"
)

// OccurrenceModel é o modelo GORM da tabela occurrences.
type OccurrenceModel struct {
	ID           uuid.UUID `gorm:"column:id"`
	CommitmentID uuid.UUID `gorm:"column:commitment_id"`
	DueAt        time.Time `gorm:"column:due_at"`
	Timezone     string    `gorm:"column:timezone"`
	State        string    `gorm:"column:state"`
	CurrentStep  int       `gorm:"column:current_step"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (OccurrenceModel) TableName() string {
	return "occurrences"
}

func (m OccurrenceModel) toDomain() occurrence.Occurrence {
	return occurrence.Occurrence{
		ID:           m.ID,
		CommitmentID: m.CommitmentID,
		DueAt:        m.DueAt,
		Timezone:     m.Timezone,
		State:        occurrence.State(m.State),
		CurrentStep:  m.CurrentStep,
	}
}

// OccurrenceWithProfile é uma Ocorrência junto do NagProfile do seu
// Compromisso — o Guardião precisa dos dois para checar o teto da escada.
type OccurrenceWithProfile struct {
	Occurrence occurrence.Occurrence
	NagProfile commitment.NagProfile
}

// OccurrenceRepository dá acesso à tabela occurrences via GORM. O Scanner não
// usa este repositório — ele precisa de uma transação pgx nativa para
// enfileirar no River atomicamente (ADR-0003) e acessa o banco direto em
// internal/jobs/scanner.go.
type OccurrenceRepository struct {
	db *gorm.DB
}

func NewOccurrenceRepository(db *gorm.DB) *OccurrenceRepository {
	return &OccurrenceRepository{db: db}
}

// LatestDueAt devolve o maior due_at já materializado para um Compromisso, e
// se existe algum. O Materializador usa isso como cursor para não regerar
// Ocorrências já existentes.
func (r *OccurrenceRepository) LatestDueAt(ctx context.Context, commitmentID uuid.UUID) (time.Time, bool, error) {
	var result sql.NullTime
	err := r.db.WithContext(ctx).
		Model(&OccurrenceModel{}).
		Where("commitment_id = ?", commitmentID).
		Select("MAX(due_at)").
		Scan(&result).Error
	if err != nil {
		return time.Time{}, false, err
	}
	if !result.Valid {
		return time.Time{}, false, nil
	}
	return result.Time, true, nil
}

// Create insere uma nova Ocorrência. Devolve created=false sem erro se já
// existir uma com o mesmo (commitment_id, due_at) — a idempotência do
// Materializador (ADR-0002) é o índice único, não uma checagem prévia.
func (r *OccurrenceRepository) Create(ctx context.Context, occ occurrence.Occurrence) (bool, error) {
	m := OccurrenceModel{
		ID:           occ.ID,
		CommitmentID: occ.CommitmentID,
		DueAt:        occ.DueAt,
		Timezone:     occ.Timezone,
		State:        string(occ.State),
		CurrentStep:  occ.CurrentStep,
	}
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&m)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// UpdateState persiste uma transição de estado feita no domínio.
func (r *OccurrenceRepository) UpdateState(ctx context.Context, id uuid.UUID, state occurrence.State) error {
	return r.db.WithContext(ctx).
		Model(&OccurrenceModel{}).
		Where("id = ?", id).
		Update("state", string(state)).Error
}

// FindByIDForUser busca uma Ocorrência pelo ID, restrito a Compromissos do
// usuário informado — isolamento por user_id (ADR-0001). Retorna
// gorm.ErrRecordNotFound se não existir ou não pertencer a ele.
func (r *OccurrenceRepository) FindByIDForUser(ctx context.Context, id, userID uuid.UUID) (occurrence.Occurrence, error) {
	var m OccurrenceModel
	err := r.db.WithContext(ctx).
		Select("occurrences.*").
		Joins("JOIN commitments ON commitments.id = occurrences.commitment_id").
		Where("occurrences.id = ? AND commitments.user_id = ?", id, userID).
		First(&m).Error
	if err != nil {
		return occurrence.Occurrence{}, err
	}
	return m.toDomain(), nil
}

// OccurrenceListItem é uma Ocorrência junto do título do seu Compromisso —
// o que a tela Hoje precisa para listar.
type OccurrenceListItem struct {
	Occurrence occurrence.Occurrence
	Title      string
}

// FindDueForUser lista, para a tela Hoje, as Ocorrências não terminais de um
// usuário cujo vencimento já chegou ou chega até endOfDay (calculado pelo
// chamador no fuso do usuário — este repositório não sabe de fusos).
func (r *OccurrenceRepository) FindDueForUser(ctx context.Context, userID uuid.UUID, endOfDay time.Time) ([]OccurrenceListItem, error) {
	type row struct {
		OccurrenceModel `gorm:"embedded"`
		Title           string
	}
	var rows []row

	err := r.db.WithContext(ctx).
		Table("occurrences AS o").
		Select("o.*, c.title").
		Joins("JOIN commitments c ON c.id = o.commitment_id").
		Where("c.user_id = ? AND o.state NOT IN (?, ?, ?) AND o.due_at <= ?",
			userID, string(occurrence.StateDone), string(occurrence.StateDismissed), string(occurrence.StateAbandoned), endOfDay).
		Order("o.due_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]OccurrenceListItem, 0, len(rows))
	for _, rw := range rows {
		result = append(result, OccurrenceListItem{
			Occurrence: rw.OccurrenceModel.toDomain(),
			Title:      rw.Title,
		})
	}
	return result, nil
}

// Save persiste uma Ocorrência inteira após uma transição de domínio
// (Complete, Dismiss, Snooze, Resume) — todas mudam state, e Snooze também
// muda due_at, então um único método genérico evita um por transição.
func (r *OccurrenceRepository) Save(ctx context.Context, occ occurrence.Occurrence) error {
	return r.db.WithContext(ctx).
		Model(&OccurrenceModel{}).
		Where("id = ?", occ.ID).
		Updates(map[string]any{
			"state":        string(occ.State),
			"current_step": occ.CurrentStep,
			"due_at":       occ.DueAt,
		}).Error
}

// FindNaggingWithProfile devolve as Ocorrências em nagging junto do
// NagProfile do seu Compromisso, para o Guardião checar teto da escada.
func (r *OccurrenceRepository) FindNaggingWithProfile(ctx context.Context) ([]OccurrenceWithProfile, error) {
	type row struct {
		OccurrenceModel `gorm:"embedded"`
		NagProfile      string
	}
	var rows []row

	err := r.db.WithContext(ctx).
		Table("occurrences AS o").
		Select("o.*, c.nag_profile").
		Joins("JOIN commitments c ON c.id = o.commitment_id").
		Where("o.state = ?", string(occurrence.StateNagging)).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]OccurrenceWithProfile, 0, len(rows))
	for _, rw := range rows {
		result = append(result, OccurrenceWithProfile{
			Occurrence: rw.OccurrenceModel.toDomain(),
			NagProfile: commitment.NagProfile(rw.NagProfile),
		})
	}
	return result, nil
}

// OccurrenceSummary é o que o Entregador precisa para compor o texto de uma
// Cobrança: título e descrição do Compromisso, a Área a que pertence, e o
// vencimento da Ocorrência (ARCHITECTURE.md §3.3 — título no topo para
// reconhecimento, Área e descrição abaixo para dizer o que é).
type OccurrenceSummary struct {
	Title       string
	Description string
	AreaName    string
	DueAt       time.Time
}

// FindSummary busca o título, a descrição, o nome da Área e o vencimento de
// uma Ocorrência.
func (r *OccurrenceRepository) FindSummary(ctx context.Context, id uuid.UUID) (OccurrenceSummary, error) {
	var s OccurrenceSummary
	err := r.db.WithContext(ctx).
		Table("occurrences AS o").
		Select("c.title, c.description, a.name AS area_name, o.due_at").
		Joins("JOIN commitments c ON c.id = o.commitment_id").
		Joins("JOIN areas a ON a.id = c.area_id").
		Where("o.id = ?", id).
		Scan(&s).Error
	return s, err
}

// FindActiveDueBefore devolve Ocorrências em due ou nagging cujo vencimento
// é anterior a threshold — candidatas ao Horizonte de Relevância
// (ARCHITECTURE.md §3.4).
func (r *OccurrenceRepository) FindActiveDueBefore(ctx context.Context, threshold time.Time) ([]occurrence.Occurrence, error) {
	var models []OccurrenceModel
	err := r.db.WithContext(ctx).
		Where("state IN ? AND due_at < ?", []string{string(occurrence.StateDue), string(occurrence.StateNagging)}, threshold).
		Find(&models).Error
	if err != nil {
		return nil, err
	}

	result := make([]occurrence.Occurrence, 0, len(models))
	for _, m := range models {
		result = append(result, m.toDomain())
	}
	return result, nil
}
