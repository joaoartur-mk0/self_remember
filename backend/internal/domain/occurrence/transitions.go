package occurrence

import (
	"fmt"
	"time"
)

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
		return o, fmt.Errorf("novo vencimento (%v) precisa ser depois do atual (%v)", newDueAt, o.DueAt)
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

// Abandon transiciona nagging → abandoned: a escada do Perfil se esgotou sem
// Reconhecimento, ou a Ocorrência saiu do Horizonte de Relevância
// (occurrence/quiethours.go). Diferente de Dismiss: aqui o usuário nunca se
// manifestou (CONTEXT.md).
func (o Occurrence) Abandon() (Occurrence, error) {
	if o.State != StateNagging {
		return o, &TransitionError{From: o.State, Method: "Abandon"}
	}
	o.State = StateAbandoned
	return o, nil
}
