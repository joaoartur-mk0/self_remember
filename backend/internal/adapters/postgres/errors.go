package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrDuplicateName é devolvido por Create quando o nome já existe para o
// usuário (violação da constraint UNIQUE(user_id, name) de areas/labels).
// Repositórios traduzem o erro bruto do Postgres para este sentinel; os
// handlers HTTP nunca inspecionam código de erro do driver diretamente —
// mesmo padrão de auth.ErrInvalidCredentials.
var ErrDuplicateName = errors.New("nome já existe para este usuário")

// isUniqueViolation reconhece a SQLSTATE 23505 (unique_violation) do
// Postgres através das camadas do GORM.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
