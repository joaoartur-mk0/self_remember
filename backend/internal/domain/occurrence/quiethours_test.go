package occurrence

import (
	"testing"
	"time"
)

func TestQuietHours_Active_JanelaNoMesmoDia(t *testing.T) {
	q := QuietHours{Start: 12 * time.Hour, End: 14 * time.Hour}
	loc := time.UTC
	day := func(h, m int) time.Time { return time.Date(2025, 1, 1, h, m, 0, 0, loc) }

	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"antes da janela", day(11, 59), false},
		{"início inclusive", day(12, 0), true},
		{"dentro", day(13, 0), true},
		{"fim exclusivo", day(14, 0), false},
		{"depois", day(15, 0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := q.Active(tt.at, loc); got != tt.want {
				t.Fatalf("Active(%v) = %v, want %v", tt.at, got, tt.want)
			}
		})
	}
}

func TestQuietHours_Active_AtravessaMeiaNoite(t *testing.T) {
	q := QuietHours{Start: 22 * time.Hour, End: 7 * time.Hour}
	loc := time.UTC
	day := func(h, m int) time.Time { return time.Date(2025, 1, 1, h, m, 0, 0, loc) }

	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"antes da janela (à tarde)", day(18, 0), false},
		{"início inclusive", day(22, 0), true},
		{"depois da meia-noite, dentro", day(3, 0), true},
		{"fim exclusivo", day(7, 0), false},
		{"depois do fim", day(8, 0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := q.Active(tt.at, loc); got != tt.want {
				t.Fatalf("Active(%v) = %v, want %v", tt.at, got, tt.want)
			}
		})
	}
}

func TestQuietHours_Active_Desativada(t *testing.T) {
	q := QuietHours{Start: 8 * time.Hour, End: 8 * time.Hour}
	loc := time.UTC
	at := time.Date(2025, 1, 1, 8, 0, 0, 0, loc)

	if q.Active(at, loc) {
		t.Fatal("Start == End deveria significar Janela desativada")
	}
}

func TestQuietHours_NextAvailableInstant(t *testing.T) {
	loc := time.UTC

	t.Run("já fora da janela: devolve o próprio instante", func(t *testing.T) {
		q := QuietHours{Start: 22 * time.Hour, End: 7 * time.Hour}
		at := time.Date(2025, 1, 1, 12, 0, 0, 0, loc)
		got := q.NextAvailableInstant(at, loc)
		if !got.Equal(at) {
			t.Fatalf("got %v, want %v", got, at)
		}
	})

	t.Run("dentro da janela que atravessa a meia-noite, antes dela", func(t *testing.T) {
		q := QuietHours{Start: 22 * time.Hour, End: 7 * time.Hour}
		at := time.Date(2025, 1, 1, 23, 0, 0, 0, loc)
		want := time.Date(2025, 1, 2, 7, 0, 0, 0, loc)
		got := q.NextAvailableInstant(at, loc)
		if !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("dentro da janela que atravessa a meia-noite, depois dela", func(t *testing.T) {
		q := QuietHours{Start: 22 * time.Hour, End: 7 * time.Hour}
		at := time.Date(2025, 1, 2, 3, 0, 0, 0, loc)
		want := time.Date(2025, 1, 2, 7, 0, 0, 0, loc)
		got := q.NextAvailableInstant(at, loc)
		if !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("dentro da janela no mesmo dia", func(t *testing.T) {
		q := QuietHours{Start: 12 * time.Hour, End: 14 * time.Hour}
		at := time.Date(2025, 1, 1, 13, 0, 0, 0, loc)
		want := time.Date(2025, 1, 1, 14, 0, 0, 0, loc)
		got := q.NextAvailableInstant(at, loc)
		if !got.Equal(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func TestOutOfRelevanceHorizon(t *testing.T) {
	dueAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"bem dentro do horizonte", dueAt.Add(24 * time.Hour), false},
		{"exatamente no limite: ainda dentro", dueAt.Add(HorizonteRelevancia), false},
		{"um segundo depois do limite: fora", dueAt.Add(HorizonteRelevancia + time.Second), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := OutOfRelevanceHorizon(dueAt, tt.now); got != tt.want {
				t.Fatalf("OutOfRelevanceHorizon() = %v, want %v", got, tt.want)
			}
		})
	}
}
