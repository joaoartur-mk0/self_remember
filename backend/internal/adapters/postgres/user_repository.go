// Package postgres contém os repositórios GORM que implementam o acesso a
// dados sobre o schema aplicado pelas migrations em backend/migrations.
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User é o modelo GORM da tabela users. O hash de senha nunca deve atravessar
// para um DTO de resposta HTTP.
type User struct {
	ID           uuid.UUID `gorm:"column:id"`
	Email        string    `gorm:"column:email"`
	PasswordHash string    `gorm:"column:password_hash"`
	Name         string    `gorm:"column:name"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName fixa o nome da tabela, evitando a pluralização automática do GORM
// divergir do nome definido na migration.
func (User) TableName() string {
	return "users"
}

// UserRepository dá acesso à tabela users via GORM.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository cria um UserRepository sobre uma conexão já aberta.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create insere um usuário já com o hash de senha calculado pela camada de
// domínio.
func (r *UserRepository) Create(ctx context.Context, u *User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

// FindByEmail busca um usuário pelo e-mail. Retorna gorm.ErrRecordNotFound se
// não existir.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
