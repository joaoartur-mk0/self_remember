package occurrence

import "time"

// HorizonteRelevancia é o prazo além do qual uma Ocorrência vencida é
// Abandonada em vez de entregue, mesmo que ainda tenha Degraus disponíveis
// (ARCHITECTURE.md §3.2, §3.4) — mesmo o Perfil Implacável se limita a ele
// (ADR-0004). Evita que voltar de uma viagem produza uma avalanche de
// Cobranças atrasadas.
const HorizonteRelevancia = 7 * 24 * time.Hour

// OutOfRelevanceHorizon reporta se a Ocorrência já passou do Horizonte de
// Relevância e deveria ser Abandonada em vez de entregue. Exatamente no
// limite (now - dueAt == HorizonteRelevancia) ainda não é considerado fora:
// o Horizonte é uma janela fechada.
func OutOfRelevanceHorizon(dueAt, now time.Time) bool {
	return now.Sub(dueAt) > HorizonteRelevancia
}

// QuietHours é a Janela de Silêncio diária do usuário (CONTEXT.md), em
// relógio de parede no fuso dele. Start pode ser depois de End (ex.:
// 22:00–07:00), atravessando a virada do dia. Start==End representa uma
// Janela desativada — nunca ativa.
type QuietHours struct {
	Start time.Duration // desde 00:00, ex.: 22h = 22*time.Hour
	End   time.Duration
}

// Active reporta se o instante at, convertido para loc, cai dentro da Janela
// de Silêncio. Nenhum Perfil de Insistência a atravessa (ADR-0004) — nem o
// Implacável.
func (q QuietHours) Active(at time.Time, loc *time.Location) bool {
	if q.Start == q.End {
		return false
	}

	sinceMidnight := sinceMidnightIn(at, loc)

	if q.Start < q.End {
		return sinceMidnight >= q.Start && sinceMidnight < q.End
	}
	// Atravessa a meia-noite (ex.: 22h–7h): ativa antes ou depois dela.
	return sinceMidnight >= q.Start || sinceMidnight < q.End
}

// NextAvailableInstant devolve o próximo instante, a partir de at, em que a
// Janela de Silêncio não está ativa — a primeira brecha (CONTEXT.md:
// "Represamento"). Se at já está fora da Janela, devolve o próprio at.
func (q QuietHours) NextAvailableInstant(at time.Time, loc *time.Location) time.Time {
	if !q.Active(at, loc) {
		return at
	}

	local := at.In(loc)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	end := midnight.Add(q.End)
	if !end.After(at) {
		end = end.AddDate(0, 0, 1)
	}
	return end
}

func sinceMidnightIn(at time.Time, loc *time.Location) time.Duration {
	local := at.In(loc)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return local.Sub(midnight)
}
