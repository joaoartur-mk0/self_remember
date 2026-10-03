package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
)

// dashboardDays é quantos dias a agenda do painel cobre, contando hoje.
const dashboardDays = 7

type dashboardItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	AreaID      string    `json:"area_id"`
	AreaName    string    `json:"area_name"`
	AreaColor   *string   `json:"area_color,omitempty"`
	NagProfile  string    `json:"nag_profile"`
	DueAt       time.Time `json:"due_at"`
	State       string    `json:"state"`
	CurrentStep int       `json:"current_step"`
}

type dashboardDay struct {
	// Date é AAAA-MM-DD no fuso de Ajustes — o cliente não deve derivar o
	// dia a partir de due_at com o fuso do navegador, que pode ser outro.
	Date  string          `json:"date"`
	Items []dashboardItem `json:"items"`
}

type dashboardResponse struct {
	Timezone string `json:"timezone"`
	// Overdue são as Ocorrências em aberto de dias anteriores a hoje.
	Overdue []dashboardItem `json:"overdue"`
	// Days tem sempre dashboardDays entradas, começando em hoje, mesmo as
	// vazias — a agenda mantém a mesma forma a cada acesso.
	Days []dashboardDay `json:"days"`
}

// bucketDashboard distribui as Ocorrências em aberto entre atrasadas e os
// dias da agenda, usando o calendário de loc. items já vêm filtrados por
// vencimento < início de hoje + dashboardDays dias.
func bucketDashboard(items []postgres.DashboardItem, now time.Time, loc *time.Location) dashboardResponse {
	now = now.In(loc)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	resp := dashboardResponse{
		Timezone: loc.String(),
		Overdue:  []dashboardItem{},
		Days:     make([]dashboardDay, dashboardDays),
	}
	dayIndex := make(map[string]int, dashboardDays)
	for i := range resp.Days {
		date := startOfToday.AddDate(0, 0, i).Format("2006-01-02")
		resp.Days[i] = dashboardDay{Date: date, Items: []dashboardItem{}}
		dayIndex[date] = i
	}

	for _, it := range items {
		due := it.Occurrence.DueAt.In(loc)
		out := dashboardItem{
			ID:          it.Occurrence.ID.String(),
			Title:       it.Title,
			Description: it.Description,
			AreaID:      it.AreaID.String(),
			AreaName:    it.AreaName,
			AreaColor:   it.AreaColor,
			NagProfile:  string(it.NagProfile),
			DueAt:       it.Occurrence.DueAt,
			State:       string(it.Occurrence.State),
			CurrentStep: it.Occurrence.CurrentStep,
		}

		if due.Before(startOfToday) {
			resp.Overdue = append(resp.Overdue, out)
			continue
		}
		if i, ok := dayIndex[due.Format("2006-01-02")]; ok {
			resp.Days[i].Items = append(resp.Days[i].Items, out)
		}
	}
	return resp
}

// Dashboard godoc
// @Summary      Painel inicial: atrasadas e agenda dos próximos 7 dias
// @Description  Dias e atrasadas são calculados no fuso de Ajustes. Só Ocorrências em aberto de Compromissos não arquivados.
// @Tags         occurrences
// @Produce      json
// @Success      200  {object}  dashboardResponse
// @Failure      404  "Usuário sem Ajustes gravados (fuso desconhecido)"
// @Router       /dashboard [get]
func (h *OccurrenceHandler) Dashboard(c *gin.Context) {
	userID := userIDFrom(c)

	userSettings, err := h.settings.FindByUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	loc, err := time.LoadLocation(userSettings.Timezone)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	now := time.Now().In(loc)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	until := startOfToday.AddDate(0, 0, dashboardDays)

	items, err := h.occurrences.FindOpenBefore(c.Request.Context(), userID, until)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, bucketDashboard(items, now, loc))
}
