// Package occurrence contém a Ocorrência: a instância datada de um
// Compromisso, sua máquina de estados, a escada de Degraus e a Janela de
// Silêncio (ARCHITECTURE.md §5). Domínio puro — sem Gin, GORM ou golang-jwt.
package occurrence

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// State é o estado de uma Ocorrência, conforme o diagrama em
// ARCHITECTURE.md §4:
//
//	pending ──▶ due ──▶ nagging ──┬──▶ done
//	                              ├──▶ snoozed ──▶ due
//	                              ├──▶ dismissed
//	                              └──▶ abandoned
type State string

const (
	StatePending   State = "pending"
	StateDue       State = "due"
	StateNagging   State = "nagging"
	StateDone      State = "done"
	StateSnoozed   State = "snoozed"
	StateDismissed State = "dismissed"
	StateAbandoned State = "abandoned"
)

// Terminal reporta se o estado é final. Nenhuma transição existe a partir de
// um estado terminal — o histórico de por que uma Ocorrência morreu nunca é
// reescrito (CONTEXT.md).
func (s State) Terminal() bool {
	switch s {
	case StateDone, StateDismissed, StateAbandoned:
		return true
	default:
		return false
	}
}

var (
	ErrCommitmentIDRequired = errors.New("commitment id é obrigatório")
	ErrDueAtRequired        = errors.New("due_at é obrigatório")
	ErrInvalidTimezone      = errors.New("timezone inválido")
)

// Occurrence é a Ocorrência: a unidade que vence, é cobrada e é reconhecida.
type Occurrence struct {
	ID           uuid.UUID
	CommitmentID uuid.UUID
	DueAt        time.Time
	// Timezone é o IANA que gerou DueAt (ARCHITECTURE.md §4) — persistido
	// junto do instante para que o cálculo de recorrência seja auditável.
	Timezone    string
	State       State
	CurrentStep int
}

// Validate aplica as regras que toda Ocorrência precisa satisfazer antes de
// ser persistida.
func (o Occurrence) Validate() error {
	if o.CommitmentID == uuid.Nil {
		return ErrCommitmentIDRequired
	}
	if o.DueAt.IsZero() {
		return ErrDueAtRequired
	}
	if _, err := time.LoadLocation(o.Timezone); err != nil {
		return ErrInvalidTimezone
	}
	return nil
}
