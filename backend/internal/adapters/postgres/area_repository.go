package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AreaModel é o modelo GORM da tabela areas.
type AreaModel struct {
	ID        uuid.UUID `gorm:"column:id"`
	UserID    uuid.UUID `gorm:"column:user_id"`
	Name      string    `gorm:"column:name"`
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

func (r *AreaRepository) Create(ctx context.Context, a *AreaModel) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// FindByUser lista as Áreas de um usuário.
func (r *AreaRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]AreaModel, error) {
	var areas []AreaModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("name ASC").Find(&areas).Error
	return areas, err
}

// FindByID busca uma Área pelo ID. Retorna gorm.ErrRecordNotFound se não
// existir.
func (r *AreaRepository) FindByID(ctx context.Context, id uuid.UUID) (*AreaModel, error) {
	var a AreaModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}
