package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AreaModel é o modelo GORM da tabela areas. Color é opcional — uma Área
// sem cor ainda é uma Área válida (migration 0011).
type AreaModel struct {
	ID        uuid.UUID `gorm:"column:id"`
	UserID    uuid.UUID `gorm:"column:user_id"`
	Name      string    `gorm:"column:name"`
	Color     *string   `gorm:"column:color"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (AreaModel) TableName() string {
	return "areas"
}

// AreaRepository dá acesso à tabela areas via GORM.
type AreaRepository struct {
	db *gorm.DB
}

func NewAreaRepository(db *gorm.DB) *AreaRepository {
	return &AreaRepository{db: db}
}

// Create insere uma nova Área. Devolve ErrDuplicateName se o usuário já tiver
// uma Área com o mesmo nome (UNIQUE(user_id, name), migration 0010).
func (r *AreaRepository) Create(ctx context.Context, a *AreaModel) error {
	if err := r.db.WithContext(ctx).Create(a).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateName
		}
		return err
	}
	return nil
}

// FindByUser lista as Áreas de um usuário.
func (r *AreaRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]AreaModel, error) {
	var areas []AreaModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("name ASC").Find(&areas).Error
	return areas, err
}
