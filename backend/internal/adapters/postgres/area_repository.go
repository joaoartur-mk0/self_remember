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

// FindByIDForUser busca uma Área pelo ID, restrito ao usuário informado —
// isolamento por user_id (ADR-0001). Usada pelo caso de uso de Compromisso
// para validar que a AreaID informada pertence a quem está criando.
func (r *AreaRepository) FindByIDForUser(ctx context.Context, id, userID uuid.UUID) (*AreaModel, error) {
	var m AreaModel
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&m).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateForUser sobrescreve nome e cor de uma Área, restrito ao usuário
// informado. Devolve gorm.ErrRecordNotFound se não existir ou não pertencer
// a ele, ErrDuplicateName se o novo nome colidir com outra Área do mesmo
// usuário.
func (r *AreaRepository) UpdateForUser(ctx context.Context, id, userID uuid.UUID, name string, color *string) error {
	result := r.db.WithContext(ctx).
		Model(&AreaModel{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]any{"name": name, "color": color})
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return ErrDuplicateName
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteForUser remove uma Área, restrito ao usuário informado. Devolve
// gorm.ErrRecordNotFound se não existir ou não pertencer a ele, ErrInUse se
// algum Compromisso ainda referenciar a Área (commitments.area_id é NOT
// NULL REFERENCES areas(id), sem ON DELETE — migration 0005).
func (r *AreaRepository) DeleteForUser(ctx context.Context, id, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&AreaModel{})
	if result.Error != nil {
		if isForeignKeyViolation(result.Error) {
			return ErrInUse
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
