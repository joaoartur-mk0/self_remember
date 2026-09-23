package commitment

import (
	"fmt"
	"time"

	"github.com/teambition/rrule-go"
)

// NextOccurrenceAfter calcula o próximo vencimento deste Compromisso
// estritamente depois de after, no fuso loc.
//
// A expansão acontece no relógio de parede do fuso do usuário, não em UTC
// (ARCHITECTURE.md §3.1): "todo dia às 8h" precisa continuar significando 8h
// no relógio de parede através de uma virada de horário de verão, não
// UTC+deslocamento fixo. Por isso RecurrenceStart é reinterpretado em loc
// antes de qualquer cálculo — ver wallClockIn.
//
// Compromissos não recorrentes (RecurrenceRule vazio) só têm uma Ocorrência:
// devolve RecurrenceStart se ainda não passou de after, ou ok=false se já
// passou (a Ocorrência única já venceu, não há próxima).
func (c Commitment) NextOccurrenceAfter(after time.Time, loc *time.Location) (time.Time, bool, error) {
	anchor := wallClockIn(c.RecurrenceStart, loc)

	if !c.IsRecurring() {
		if anchor.After(after) {
			return anchor, true, nil
		}
		return time.Time{}, false, nil
	}

	option, err := rrule.StrToROptionInLocation(c.RecurrenceRule, loc)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("regra de recorrência inválida: %w", err)
	}
	option.Dtstart = anchor

	rr, err := rrule.NewRRule(*option)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("regra de recorrência inválida: %w", err)
	}

	// after é um instante real (não um valor a reinterpretar): converte para
	// loc sem mudar o instante, ao contrário de wallClockIn.
	next := rr.After(after.In(loc), false)
	if next.IsZero() {
		return time.Time{}, false, nil
	}
	return next, true, nil
}

// wallClockIn reinterpreta os componentes de data/hora de t no fuso loc, sem
// converter o instante. "8h" em t (seja qual for o *time.Location que t traz)
// continua sendo "8h", agora com o offset UTC correto de loc naquele dia —
// inclusive através de uma virada de horário de verão.
func wallClockIn(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}
