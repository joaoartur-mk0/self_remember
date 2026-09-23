package commitment

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func validCommitment() Commitment {
	return Commitment{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		AreaID:          uuid.New(),
		Title:           "Tomar remédio",
		NagProfile:      NagProfileNormal,
		CarryOverPolicy: CarryOverStack,
	}
}

func TestCommitment_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c Commitment) Commitment
		wantErr error
	}{
		{
			name:    "válido",
			mutate:  func(c Commitment) Commitment { return c },
			wantErr: nil,
		},
		{
			name:    "título vazio",
			mutate:  func(c Commitment) Commitment { c.Title = ""; return c },
			wantErr: ErrTitleRequired,
		},
		{
			name:    "título só com espaços",
			mutate:  func(c Commitment) Commitment { c.Title = "   "; return c },
			wantErr: ErrTitleRequired,
		},
		{
			name:    "perfil de insistência inválido",
			mutate:  func(c Commitment) Commitment { c.NagProfile = "urgente"; return c },
			wantErr: ErrInvalidNagProfile,
		},
		{
			name:    "política de acúmulo inválida",
			mutate:  func(c Commitment) Commitment { c.CarryOverPolicy = "acumular"; return c },
			wantErr: ErrInvalidCarryOverPolicy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.mutate(validCommitment()).Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommitment_IsRecurring(t *testing.T) {
	c := validCommitment()
	if c.IsRecurring() {
		t.Fatal("Compromisso sem RecurrenceRule não deveria ser recorrente")
	}

	c.RecurrenceRule = "FREQ=DAILY"
	if !c.IsRecurring() {
		t.Fatal("Compromisso com RecurrenceRule deveria ser recorrente")
	}
}

func TestCommitment_Archived(t *testing.T) {
	c := validCommitment()
	if c.Archived() {
		t.Fatal("Compromisso novo não deveria estar arquivado")
	}

	now := time.Now()
	c.ArchivedAt = &now
	if !c.Archived() {
		t.Fatal("Compromisso com ArchivedAt deveria estar arquivado")
	}
}
