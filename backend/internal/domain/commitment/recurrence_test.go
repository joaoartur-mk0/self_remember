package commitment

import (
	"testing"
	"time"
)

func TestNextOccurrenceAfter_NotRecurring(t *testing.T) {
	loc := time.UTC
	start := time.Date(2025, 1, 10, 8, 0, 0, 0, time.UTC)
	c := Commitment{RecurrenceStart: start}

	t.Run("antes do vencimento", func(t *testing.T) {
		after := time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC)
		next, ok, err := c.NextOccurrenceAfter(after, loc)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if !ok {
			t.Fatal("esperava ok=true, único vencimento ainda não passou")
		}
		if !next.Equal(start) {
			t.Fatalf("next = %v, want %v", next, start)
		}
	})

	t.Run("depois do vencimento", func(t *testing.T) {
		after := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC)
		_, ok, err := c.NextOccurrenceAfter(after, loc)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if ok {
			t.Fatal("esperava ok=false, único vencimento já passou")
		}
	})
}

func TestNextOccurrenceAfter_Daily(t *testing.T) {
	loc := time.UTC
	start := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)
	c := Commitment{RecurrenceStart: start, RecurrenceRule: "FREQ=DAILY"}

	after := time.Date(2025, 1, 3, 9, 0, 0, 0, time.UTC)
	next, ok, err := c.NextOccurrenceAfter(after, loc)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !ok {
		t.Fatal("esperava próxima ocorrência")
	}

	want := time.Date(2025, 1, 4, 8, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v", next, want)
	}
}

// TestNextOccurrenceAfter_WeeklyByDay prova que BYDAY explícito com
// múltiplos dias funciona mesmo quando RecurrenceStart não cai em nenhum
// deles — o frontend (tela Compromissos, recorrência Semanal) sempre manda
// "hoje" como RecurrenceStart e deixa BYDAY decidir os dias de verdade.
func TestNextOccurrenceAfter_WeeklyByDay(t *testing.T) {
	loc := time.UTC
	// 2025-01-01 é quarta-feira; a regra só pede segunda e sexta.
	start := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)
	c := Commitment{RecurrenceStart: start, RecurrenceRule: "FREQ=WEEKLY;BYDAY=MO,FR"}

	after := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC)
	next, ok, err := c.NextOccurrenceAfter(after, loc)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !ok {
		t.Fatal("esperava próxima ocorrência")
	}

	// A próxima sexta depois de 1º de janeiro de 2025 é 3 de janeiro.
	want := time.Date(2025, 1, 3, 8, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v (próxima sexta-feira)", next, want)
	}
}

// TestNextOccurrenceAfter_MonthlyByMonthDay prova o mesmo para BYMONTHDAY
// (recorrência Mensal), também com RecurrenceStart num dia diferente do
// pedido pela regra.
func TestNextOccurrenceAfter_MonthlyByMonthDay(t *testing.T) {
	loc := time.UTC
	// RecurrenceStart no dia 1; a regra pede todo dia 15.
	start := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)
	c := Commitment{RecurrenceStart: start, RecurrenceRule: "FREQ=MONTHLY;BYMONTHDAY=15"}

	after := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	next, ok, err := c.NextOccurrenceAfter(after, loc)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !ok {
		t.Fatal("esperava próxima ocorrência")
	}

	want := time.Date(2025, 2, 15, 8, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next = %v, want %v (dia 15 do mês seguinte)", next, want)
	}
}

func TestNextOccurrenceAfter_InvalidRule(t *testing.T) {
	c := Commitment{
		RecurrenceStart: time.Now(),
		RecurrenceRule:  "isto não é uma RRULE",
	}

	_, _, err := c.NextOccurrenceAfter(time.Now(), time.UTC)
	if err == nil {
		t.Fatal("esperava erro para RRULE inválida")
	}
}

// TestNextOccurrenceAfter_DST prova a garantia central do ARCHITECTURE.md
// §3.1: "todo dia às 8h" continua significando 8h no relógio de parede do
// usuário através de uma virada de horário de verão, não um deslocamento UTC
// fixo. Em 2024-03-10, o horário de verão dos EUA avançou os relógios de
// America/New_York de 2h para 3h.
func TestNextOccurrenceAfter_DST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("carregar fuso: %v", err)
	}

	// Componentes de data/hora "8h" — o *time.Location anexado aqui é
	// irrelevante, wallClockIn os reinterpreta em loc.
	start := time.Date(2024, 3, 8, 8, 0, 0, 0, time.UTC)
	c := Commitment{RecurrenceStart: start, RecurrenceRule: "FREQ=DAILY"}

	// 10 de março de 2024, 8h em Nova York (dia da virada em si).
	after := time.Date(2024, 3, 10, 8, 0, 0, 0, loc)

	next, ok, err := c.NextOccurrenceAfter(after, loc)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if !ok {
		t.Fatal("esperava próxima ocorrência")
	}

	nextLocal := next.In(loc)
	if nextLocal.Hour() != 8 || nextLocal.Minute() != 0 {
		t.Fatalf("horário de parede mudou através do DST: %v", nextLocal)
	}
	if nextLocal.Day() != 11 {
		t.Fatalf("dia inesperado: %v", nextLocal)
	}

	// Antes da virada, 8h em Nova York era UTC-5 (13h UTC); depois, UTC-4
	// (12h UTC). Se o deslocamento UTC não tiver mudado, a reinterpretação
	// de fuso não está de fato acontecendo — o teste passaria por acidente.
	if next.UTC().Hour() != 12 {
		t.Fatalf("deslocamento UTC não reflete o horário de verão: %v UTC", next.UTC())
	}
}
