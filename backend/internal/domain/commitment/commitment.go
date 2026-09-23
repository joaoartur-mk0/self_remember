// Package commitment contém o Compromisso: o registro duradouro da intenção
// do usuário (CONTEXT.md). Domínio puro — sem Gin, GORM ou golang-jwt. A
// expansão de recorrência (com fuso) é responsabilidade de recurrence.go, não
// deste arquivo: aqui só a entidade e sua validação.
package commitment

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NagProfile é o Perfil de Insistência (ADR-0004): Leve, Normal ou
// Implacável. Fixo no MVP — não editável pelo usuário.
type NagProfile string

const (
	NagProfileLeve       NagProfile = "leve"
	NagProfileNormal     NagProfile = "normal"
	NagProfileImplacavel NagProfile = "implacavel"
)

// Valid reporta se o valor é um dos três perfis fixos do sistema.
func (p NagProfile) Valid() bool {
	switch p {
	case NagProfileLeve, NagProfileNormal, NagProfileImplacavel:
		return true
	default:
		return false
	}
}

// CarryOverPolicy é a Política de Acúmulo (CONTEXT.md): o que acontece com
// uma Ocorrência não reconhecida quando a próxima do mesmo Compromisso
// recorrente vence.
type CarryOverPolicy string

const (
	// CarryOverStack mantém a Ocorrência anterior cobrando junto da nova.
	CarryOverStack CarryOverPolicy = "stack"
	// CarryOverReplace encerra a Ocorrência anterior como Abandonada quando a
	// nova vence.
	CarryOverReplace CarryOverPolicy = "replace"
)

// Valid reporta se o valor é uma das duas políticas de acúmulo do sistema.
func (p CarryOverPolicy) Valid() bool {
	switch p {
	case CarryOverStack, CarryOverReplace:
		return true
	default:
		return false
	}
}

var (
	ErrTitleRequired          = errors.New("título é obrigatório")
	ErrInvalidNagProfile      = errors.New("perfil de insistência inválido")
	ErrInvalidCarryOverPolicy = errors.New("política de acúmulo inválida")
)

// Commitment é o Compromisso: a intenção duradoura, com ou sem recorrência.
// RecurrenceRule vazio significa Compromisso único, com uma só Ocorrência.
type Commitment struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	AreaID          uuid.UUID
	Title           string
	RecurrenceRule  string
	NagProfile      NagProfile
	CarryOverPolicy CarryOverPolicy
	ArchivedAt      *time.Time
}

// Validate aplica as regras que todo Compromisso precisa satisfazer antes de
// ser persistido, independentemente de recorrência.
func (c Commitment) Validate() error {
	if strings.TrimSpace(c.Title) == "" {
		return ErrTitleRequired
	}
	if !c.NagProfile.Valid() {
		return ErrInvalidNagProfile
	}
	if !c.CarryOverPolicy.Valid() {
		return ErrInvalidCarryOverPolicy
	}
	return nil
}

// IsRecurring reporta se este Compromisso gera mais de uma Ocorrência.
func (c Commitment) IsRecurring() bool {
	return c.RecurrenceRule != ""
}

// Archived reporta se o Compromisso foi arquivado (ADR-0006: sinal de
// Abandono repetido, entre outros motivos).
func (c Commitment) Archived() bool {
	return c.ArchivedAt != nil
}
