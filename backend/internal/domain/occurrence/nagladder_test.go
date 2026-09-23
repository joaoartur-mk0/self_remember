package occurrence

import (
	"testing"
	"time"

	"self-remember/backend/internal/domain/commitment"
)

func TestDelayForStep_Leve(t *testing.T) {
	delay, ok := DelayForStep(commitment.NagProfileLeve, 1)
	if !ok || delay != 0 {
		t.Fatalf("step 1 = (%v, %v), want (0, true)", delay, ok)
	}

	if _, ok := DelayForStep(commitment.NagProfileLeve, 2); ok {
		t.Fatal("Leve não deveria ter Degrau 2 — teto é 1 (ADR-0004)")
	}
}

func TestDelayForStep_Normal(t *testing.T) {
	want := []time.Duration{0, 15 * time.Minute, time.Hour, 4 * time.Hour, 24 * time.Hour}
	for i, w := range want {
		step := i + 1
		delay, ok := DelayForStep(commitment.NagProfileNormal, step)
		if !ok || delay != w {
			t.Fatalf("step %d = (%v, %v), want (%v, true)", step, delay, ok, w)
		}
	}

	if _, ok := DelayForStep(commitment.NagProfileNormal, 6); ok {
		t.Fatal("Normal não deveria ter Degrau 6 — teto é 5 (ADR-0004)")
	}
}

func TestDelayForStep_Implacavel(t *testing.T) {
	want := []time.Duration{0, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute}
	for i, w := range want {
		step := i + 1
		delay, ok := DelayForStep(commitment.NagProfileImplacavel, step)
		if !ok || delay != w {
			t.Fatalf("step %d = (%v, %v), want (%v, true)", step, delay, ok, w)
		}
	}

	// "depois a cada 1h" — sem teto (ADR-0004).
	for step := 5; step <= 10; step++ {
		delay, ok := DelayForStep(commitment.NagProfileImplacavel, step)
		if !ok {
			t.Fatalf("Implacável nunca esgota por contagem de Degrau — step %d retornou ok=false", step)
		}
		wantDelay := 30*time.Minute + time.Duration(step-4)*time.Hour
		if delay != wantDelay {
			t.Fatalf("step %d delay = %v, want %v", step, delay, wantDelay)
		}
	}
}

func TestDelayForStep_StepMenorQue1(t *testing.T) {
	if _, ok := DelayForStep(commitment.NagProfileNormal, 0); ok {
		t.Fatal("step 0 não é um Degrau válido")
	}
}

func TestHasReachedCeiling(t *testing.T) {
	tests := []struct {
		profile     commitment.NagProfile
		currentStep int
		want        bool
	}{
		{commitment.NagProfileLeve, 0, false},
		{commitment.NagProfileLeve, 1, true},
		{commitment.NagProfileNormal, 4, false},
		{commitment.NagProfileNormal, 5, true},
		{commitment.NagProfileImplacavel, 100, false},
	}

	for _, tt := range tests {
		got := HasReachedCeiling(tt.profile, tt.currentStep)
		if got != tt.want {
			t.Errorf("HasReachedCeiling(%v, %d) = %v, want %v", tt.profile, tt.currentStep, got, tt.want)
		}
	}
}

func TestShouldEscalate(t *testing.T) {
	dueAt := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		profile     commitment.NagProfile
		currentStep int
		now         time.Time
		want        bool
	}{
		{
			name:        "Normal, primeiro degrau, exatamente no vencimento",
			profile:     commitment.NagProfileNormal,
			currentStep: 0,
			now:         dueAt,
			want:        true,
		},
		{
			name:        "Normal, segundo degrau, um minuto antes de maturar",
			profile:     commitment.NagProfileNormal,
			currentStep: 1,
			now:         dueAt.Add(15*time.Minute - time.Minute),
			want:        false,
		},
		{
			name:        "Normal, segundo degrau, exatamente ao maturar",
			profile:     commitment.NagProfileNormal,
			currentStep: 1,
			now:         dueAt.Add(15 * time.Minute),
			want:        true,
		},
		{
			name:        "Leve, depois do único degrau: teto atingido",
			profile:     commitment.NagProfileLeve,
			currentStep: 1,
			now:         dueAt.Add(365 * 24 * time.Hour),
			want:        false,
		},
		{
			name:        "Implacável, muito tempo depois: sempre maturado, nunca teto",
			profile:     commitment.NagProfileImplacavel,
			currentStep: 50,
			now:         dueAt.Add(365 * 24 * time.Hour),
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldEscalate(tt.profile, tt.currentStep, dueAt, tt.now)
			if got != tt.want {
				t.Fatalf("ShouldEscalate() = %v, want %v", got, tt.want)
			}
		})
	}
}
