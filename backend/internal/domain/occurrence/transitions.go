package occurrence

import (
	"errors"
	"fmt"
	"time"
)

// ErrSnoozeMustBeAfterCurrent é devolvido por Snooze quando newDueAt não é
// estritamente posterior ao vencimento atual — distinto de TransitionError
// porque o estado de partida (nagging) era válido; só o valor pedido não.
var ErrSnoozeMustBeAfterCurrent = errors.New("novo vencimento precisa ser depois do atual")

// TransitionError reporta uma transição de estado que o diagrama em
// ARCHITECTURE.md §4 não permite.
type TransitionError struct {
	From   State
	Method string
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("%s: transição inválida a partir de %q", e.Method, e.From)
}

// MarkDue transiciona pending → due: o vencimento chegou, ainda sem nenhuma
// tentativa de Cobrança.
func (o Occurrence) MarkDue() (Occurrence, error) {
	if o.State != StatePending {
		return o, &TransitionError{From: o.State, Method: "MarkDue"}
	}
	o.State = StateDue
	return o, nil
}

// Escalate transiciona due → nagging (primeiro Degrau) ou nagging → nagging
// (Degrau seguinte), avançando CurrentStep. É o Escalonamento do CONTEXT.md:
// a subida de um Degrau quando a Ocorrência continua sem Reconhecimento.
func (o Occurrence) Escalate() (Occurrence, error) {
	if o.State != StateDue && o.State != StateNagging {
		return o, &TransitionError{From: o.State, Method: "Escalate"}
	}
	o.State = StateNagging
	o.CurrentStep++
	return o, nil
}

// Complete transiciona nagging → done: o Reconhecimento de Conclusão.
func (o Occurrence) Complete() (Occurrence, error) {
	if o.State != StateNagging {
		return o, &TransitionError{From: o.State, Method: "Complete"}
	}
	o.State = StateDone
	return o, nil
}

// Dismiss transiciona nagging → dismissed: o Reconhecimento de Desistência.
// É de primeira classe (CONTEXT.md) — esta transição não deve ser tratada
// como caminho secundário em nenhuma camada acima do domínio.
func (o Occurrence) Dismiss() (Occurrence, error) {
	if o.State != StateNagging {
		return o, &TransitionError{From: o.State, Method: "Dismiss"}
	}
	o.State = StateDismissed
	return o, nil
}

// Snooze transiciona nagging → snoozed: o Reconhecimento de Adiamento.
// CurrentStep é preservado — "a escada retoma, não reinicia" (CONTEXT.md).
// newDueAt precisa ser estritamente posterior ao vencimento atual.
func (o Occurrence) Snooze(newDueAt time.Time) (Occurrence, error) {
	if o.State != StateNagging {
		return o, &TransitionError{From: o.State, Method: "Snooze"}
	}
	if !newDueAt.After(o.DueAt) {
		return o, fmt.Errorf("%w: pedido %v, atual %v", ErrSnoozeMustBeAfterCurrent, newDueAt, o.DueAt)
	}
	o.State = StateSnoozed
	o.DueAt = newDueAt
	return o, nil
}

// Resume transiciona snoozed → due: o novo vencimento do Adiamento chegou.
func (o Occurrence) Resume() (Occurrence, error) {
	if o.State != StateSnoozed {
		return o, &TransitionError{From: o.State, Method: "Resume"}
	}
	o.State = StateDue
	return o, nil
}

// Abandon transiciona due ou nagging → abandoned: a escada do Perfil se
// esgotou sem Reconhecimento, ou a Ocorrência saiu do Horizonte de
// Relevância (occurrence/quiethours.go) — nesse segundo caso, mesmo uma
// Ocorrência que nunca chegou a ser cobrada (ainda em due) é Abandonada
// "nunca entregue" (ARCHITECTURE.md §3.4). Diferente de Dismiss: aqui o
// usuário nunca se manifestou (CONTEXT.md).
func (o Occurrence) Abandon() (Occurrence, error) {
	if o.State != StateDue && o.State != StateNagging {
		return o, &TransitionError{From: o.State, Method: "Abandon"}
	}
	o.State = StateAbandoned
	return o, nil
}
