package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserSettingsModel é o modelo GORM da tabela user_settings (migration 0001).
// QuietHoursStart/End são colunas TIME lidas/escritas como texto
// "HH:MM:SS" — o fuso e a Janela de Silêncio em si são interpretados pelo
// domínio (internal/domain/occurrence.QuietHours), não por este repositório.
type UserSettingsModel struct {
	UserID          uuid.UUID `gorm:"column:user_id"`
	Timezone        string    `gorm:"column:timezone"`
	QuietHoursStart *string   `gorm:"column:quiet_hours_start"`
	QuietHoursEnd   *string   `gorm:"column:quiet_hours_end"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (UserSettingsModel) TableName() string {
	return "user_settings"
}

// UserSettingsRepository dá acesso à tabela user_settings via GORM.
type UserSettingsRepository struct {
	db *gorm.DB
}

func NewUserSettingsRepository(db *gorm.DB) *UserSettingsRepository {
	return &UserSettingsRepository{db: db}
}

// FindByUser busca os Ajustes de um usuário. Retorna gorm.ErrRecordNotFound
// se o usuário não tiver linha em user_settings — hoje isso não deveria
// acontecer (cmd/admin cria a linha junto do usuário), mas o chamador
// decide o que fazer, não este repositório.
func (r *UserSettingsRepository) FindByUser(ctx context.Context, userID uuid.UUID) (*UserSettingsModel, error) {
	var m UserSettingsModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// Upsert grava os Ajustes de um usuário, criando a linha se ainda não
// existir — PUT é idempotente por natureza, então não há Create separado.
func (r *UserSettingsRepository) Upsert(ctx context.Context, m *UserSettingsModel) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"timezone", "quiet_hours_start", "quiet_hours_end", "updated_at",
			}),
		}).
		Create(m).Error
}
