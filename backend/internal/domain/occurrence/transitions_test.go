package occurrence

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newOccurrence(state State) Occurrence {
	return Occurrence{
		ID:           uuid.New(),
		CommitmentID: uuid.New(),
		DueAt:        time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC),
		Timezone:     "America/Sao_Paulo",
		State:        state,
	}
}

func TestFullLifecycle_Completion(t *testing.T) {
	o := newOccurrence(StatePending)

	o, err := o.MarkDue()
	mustNoErr(t, err)
	if o.State != StateDue {
		t.Fatalf("state = %v, want %v", o.State, StateDue)
	}

	o, err = o.Escalate()
	mustNoErr(t, err)
	if o.State != StateNagging || o.CurrentStep != 1 {
		t.Fatalf("state=%v step=%d, want nagging/1", o.State, o.CurrentStep)
	}

	o, err = o.Escalate()
	mustNoErr(t, err)
	if o.CurrentStep != 2 {
		t.Fatalf("CurrentStep = %d, want 2", o.CurrentStep)
	}

	o, err = o.Complete()
	mustNoErr(t, err)
	if o.State != StateDone {
		t.Fatalf("state = %v, want %v", o.State, StateDone)
	}
	if !o.State.Terminal() {
		t.Fatal("done deveria ser terminal")
	}
}

func TestSnoozeResumesEscadaSemReiniciar(t *testing.T) {
	o := newOccurrence(StateNagging)
	o.CurrentStep = 3

	newDue := o.DueAt.Add(24 * time.Hour)
	o, err := o.Snooze(newDue)
	mustNoErr(t, err)
	if o.State != StateSnoozed || !o.DueAt.Equal(newDue) {
		t.Fatalf("state=%v dueAt=%v, want snoozed/%v", o.State, o.DueAt, newDue)
	}
	if o.CurrentStep != 3 {
		t.Fatalf("CurrentStep = %d, want preservado em 3", o.CurrentStep)
	}

	o, err = o.Resume()
	mustNoErr(t, err)
	if o.State != StateDue {
		t.Fatalf("state = %v, want %v", o.State, StateDue)
	}

	// Retomar a escada continua do Degrau 3, não do 0.
	o, err = o.Escalate()
	mustNoErr(t, err)
	if o.CurrentStep != 4 {
		t.Fatalf("CurrentStep = %d, want 4 (retomado de 3, não reiniciado)", o.CurrentStep)
	}
}

func TestSnooze_RequerNovoVencimentoNoFuturo(t *testing.T) {
	o := newOccurrence(StateNagging)

	_, err := o.Snooze(o.DueAt)
	if err == nil {
		t.Fatal("esperava erro: novo vencimento igual ao atual")
	}

	_, err = o.Snooze(o.DueAt.Add(-time.Hour))
	if err == nil {
		t.Fatal("esperava erro: novo vencimento no passado")
	}
}

func TestDismiss(t *testing.T) {
	o := newOccurrence(StateNagging)
	o, err := o.Dismiss()
	mustNoErr(t, err)
	if o.State != StateDismissed {
		t.Fatalf("state = %v, want %v", o.State, StateDismissed)
	}
	if !o.State.Terminal() {
		t.Fatal("dismissed deveria ser terminal")
	}
}

func TestAbandon(t *testing.T) {
	for _, from := range []State{StateDue, StateNagging} {
		t.Run(string(from), func(t *testing.T) {
			o := newOccurrence(from)
			o, err := o.Abandon()
			mustNoErr(t, err)
			if o.State != StateAbandoned {
				t.Fatalf("state = %v, want %v", o.State, StateAbandoned)
			}
			if !o.State.Terminal() {
				t.Fatal("abandoned deveria ser terminal")
			}
		})
	}
}

// TestInvalidTransitions cobre toda transição fora do diagrama de
// ARCHITECTURE.md §4, incluindo qualquer tentativa a partir de um estado
// terminal.
func TestInvalidTransitions(t *testing.T) {
	tests := []struct {
		name  string
		from  State
		apply func(o Occurrence) (Occurrence, error)
	}{
		{"MarkDue de due", StateDue, func(o Occurrence) (Occurrence, error) { return o.MarkDue() }},
		{"MarkDue de nagging", StateNagging, func(o Occurrence) (Occurrence, error) { return o.MarkDue() }},
		{"Escalate de pending", StatePending, func(o Occurrence) (Occurrence, error) { return o.Escalate() }},
		{"Escalate de snoozed", StateSnoozed, func(o Occurrence) (Occurrence, error) { return o.Escalate() }},
		{"Complete de due", StateDue, func(o Occurrence) (Occurrence, error) { return o.Complete() }},
		{"Complete de pending", StatePending, func(o Occurrence) (Occurrence, error) { return o.Complete() }},
		{"Complete de done", StateDone, func(o Occurrence) (Occurrence, error) { return o.Complete() }},
		{"Dismiss de due", StateDue, func(o Occurrence) (Occurrence, error) { return o.Dismiss() }},
		{"Dismiss de dismissed", StateDismissed, func(o Occurrence) (Occurrence, error) { return o.Dismiss() }},
		{"Snooze de due", StateDue, func(o Occurrence) (Occurrence, error) { return o.Snooze(time.Now().Add(time.Hour)) }},
		{"Resume de due", StateDue, func(o Occurrence) (Occurrence, error) { return o.Resume() }},
		{"Resume de nagging", StateNagging, func(o Occurrence) (Occurrence, error) { return o.Resume() }},
		{"Abandon de pending", StatePending, func(o Occurrence) (Occurrence, error) { return o.Abandon() }},
		{"Abandon de abandoned", StateAbandoned, func(o Occurrence) (Occurrence, error) { return o.Abandon() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := newOccurrence(tt.from)
			before := o
			after, err := tt.apply(o)

			var transErr *TransitionError
			if !errors.As(err, &transErr) {
				t.Fatalf("err = %v, want *TransitionError", err)
			}
			if after != before {
				t.Fatalf("transição inválida não deveria alterar a Ocorrência: %+v != %+v", after, before)
			}
		})
	}
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
}
