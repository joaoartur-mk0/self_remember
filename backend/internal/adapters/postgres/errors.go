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

// ErrInUse é devolvido por Delete quando o registro ainda é referenciado por
// outro (ex.: Área com Compromissos) — violação de FK sem ON DELETE
// (commitments.area_id é NOT NULL REFERENCES areas(id), migration 0005).
// Apagar a Área deletaria a referência de um Compromisso existente em
// silêncio; melhor recusar e deixar o usuário arquivar os Compromissos
// primeiro.
var ErrInUse = errors.New("ainda está em uso e não pode ser removido")

// isUniqueViolation reconhece a SQLSTATE 23505 (unique_violation) do
// Postgres através das camadas do GORM.
func isUniqueViolation(err error) bool {
	return isPgError(err, "23505")
}

// isForeignKeyViolation reconhece a SQLSTATE 23503 (foreign_key_violation).
func isForeignKeyViolation(err error) bool {
	return isPgError(err, "23503")
}

func isPgError(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
