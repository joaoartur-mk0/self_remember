package occurrence

import (
	"time"

	"self-remember/backend/internal/domain/commitment"
)

// Degraus por Perfil de Insistência, em atraso desde o vencimento
// (ADR-0004). Índice 0 é o primeiro Degrau (step 1 em DelayForStep).
var (
	leveDegraus       = []time.Duration{0}
	normalDegraus     = []time.Duration{0, 15 * time.Minute, time.Hour, 4 * time.Hour, 24 * time.Hour}
	implacavelDegraus = []time.Duration{0, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute}
)

// implacavelPassoExtra é o intervalo entre Degraus do Implacável depois de
// esgotar sua lista explícita — "depois a cada 1h" (ADR-0004).
const implacavelPassoExtra = time.Hour

// DelayForStep devolve o atraso desde o vencimento do Degrau step (1-based)
// de um Perfil de Insistência, e se esse Degrau existe. Perfis com teto
// (Leve, Normal) devolvem ok=false depois do último Degrau: a escada
// esgotou (ADR-0004). Implacável nunca esgota por contagem de Degrau — sua
// única contenção é a Janela de Silêncio e o Horizonte de Relevância,
// aplicadas em outro lugar (quiethours.go).
func DelayForStep(profile commitment.NagProfile, step int) (delay time.Duration, ok bool) {
	if step < 1 {
		return 0, false
	}

	switch profile {
	case commitment.NagProfileLeve:
		return delayFromTable(leveDegraus, step)
	case commitment.NagProfileNormal:
		return delayFromTable(normalDegraus, step)
	case commitment.NagProfileImplacavel:
		if step <= len(implacavelDegraus) {
			return implacavelDegraus[step-1], true
		}
		last := implacavelDegraus[len(implacavelDegraus)-1]
		extra := step - len(implacavelDegraus)
		return last + time.Duration(extra)*implacavelPassoExtra, true
	default:
		return 0, false
	}
}

func delayFromTable(table []time.Duration, step int) (time.Duration, bool) {
	if step > len(table) {
		return 0, false
	}
	return table[step-1], true
}

// HasReachedCeiling reporta se currentStep já esgotou a escada do Perfil —
// não existe próximo Degrau. Sempre falso para Implacável (ADR-0004: "sem
// teto").
func HasReachedCeiling(profile commitment.NagProfile, currentStep int) bool {
	_, ok := DelayForStep(profile, currentStep+1)
	return !ok
}

// ShouldEscalate reporta se o próximo Degrau da escada já maturou: se now já
// alcançou dueAt mais o atraso do Degrau currentStep+1. Não considera a
// Janela de Silêncio — essa contenção represa a Entrega mesmo quando a
// escada já permitiria escalar (quiethours.go).
func ShouldEscalate(profile commitment.NagProfile, currentStep int, dueAt, now time.Time) bool {
	delay, ok := DelayForStep(profile, currentStep+1)
	if !ok {
		return false
	}
	return !now.Before(dueAt.Add(delay))
}
