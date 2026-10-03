package http

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/domain/occurrence"
)

func dashItem(title string, due time.Time) postgres.DashboardItem {
	return postgres.DashboardItem{
		Occurrence: occurrence.Occurrence{ID: uuid.New(), DueAt: due, State: occurrence.StateDue},
		Title:      title,
		AreaName:   "Saúde",
	}
}

func TestBucketDashboard(t *testing.T) {
	loc, err := time.LoadLocation("America/Sao_Paulo") // UTC-3
	if err != nil {
		t.Fatal(err)
	}
	// 2026-10-02 22:30 em São Paulo = 2026-10-03 01:30 UTC. O servidor em UTC
	// já está no dia 3; o usuário ainda está no dia 2 — o agrupamento tem
	// que seguir o calendário do usuário.
	now := time.Date(2026, 10, 3, 1, 30, 0, 0, time.UTC)

	items := []postgres.DashboardItem{
		dashItem("atrasada", time.Date(2026, 10, 1, 15, 0, 0, 0, loc)),
		dashItem("hoje tarde", time.Date(2026, 10, 2, 23, 50, 0, 0, loc)),
		// 02:00 UTC do dia 3 é 23:00 do dia 2 em São Paulo: ainda "hoje".
		dashItem("hoje por UTC", time.Date(2026, 10, 3, 2, 0, 0, 0, time.UTC)),
		dashItem("amanhã", time.Date(2026, 10, 3, 9, 0, 0, 0, loc)),
		dashItem("sétimo dia", time.Date(2026, 10, 8, 9, 0, 0, 0, loc)),
	}

	got := bucketDashboard(items, now, loc)

	if len(got.Days) != dashboardDays {
		t.Fatalf("days = %d, want %d", len(got.Days), dashboardDays)
	}
	if got.Days[0].Date != "2026-10-02" || got.Days[6].Date != "2026-10-08" {
		t.Fatalf("datas = %s..%s", got.Days[0].Date, got.Days[6].Date)
	}
	if len(got.Overdue) != 1 || got.Overdue[0].Title != "atrasada" {
		t.Fatalf("overdue = %+v", got.Overdue)
	}
	if n := len(got.Days[0].Items); n != 2 {
		t.Fatalf("hoje tem %d itens, want 2 (o fuso do usuário manda, não o UTC)", n)
	}
	if len(got.Days[1].Items) != 1 || got.Days[1].Items[0].Title != "amanhã" {
		t.Fatalf("amanhã = %+v", got.Days[1].Items)
	}
	if len(got.Days[6].Items) != 1 || got.Days[6].Items[0].Title != "sétimo dia" {
		t.Fatalf("sétimo dia = %+v", got.Days[6].Items)
	}
}
