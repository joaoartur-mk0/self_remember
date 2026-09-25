package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LabelModel é o modelo GORM da tabela labels.
type LabelModel struct {
	ID        uuid.UUID `gorm:"column:id"`
	UserID    uuid.UUID `gorm:"column:user_id"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (LabelModel) TableName() string {
	return "labels"
}

// LabelRepository dá acesso à tabela labels via GORM.
type LabelRepository struct {
	db *gorm.DB
}

func NewLabelRepository(db *gorm.DB) *LabelRepository {
	return &LabelRepository{db: db}
}

// Create insere uma nova Etiqueta. Devolve ErrDuplicateName se o usuário já
// tiver uma Etiqueta com o mesmo nome (UNIQUE(user_id, name), migration
// 0008).
func (r *LabelRepository) Create(ctx context.Context, l *LabelModel) error {
	if err := r.db.WithContext(ctx).Create(l).Error; err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateName
		}
		return err
	}
	return nil
}

// FindByUser lista as Etiquetas de um usuário.
func (r *LabelRepository) FindByUser(ctx context.Context, userID uuid.UUID) ([]LabelModel, error) {
	var labels []LabelModel
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("name ASC").Find(&labels).Error
	return labels, err
}
