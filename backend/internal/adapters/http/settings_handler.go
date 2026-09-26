package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
)

// quietHoursLayout é o formato aceito e devolvido para horário de parede,
// sem segundos: o que o formulário de Ajustes (ARCHITECTURE.md §6) pede.
const quietHoursLayout = "15:04"

// SettingsHandler expõe os Ajustes do usuário: fuso IANA e Janela de
// Silêncio (user_settings, migration 0001) — sem seção de Canais, fora de
// escopo desta fase.
type SettingsHandler struct {
	settings *postgres.UserSettingsRepository
}

func NewSettingsHandler(settings *postgres.UserSettingsRepository) *SettingsHandler {
	return &SettingsHandler{settings: settings}
}

type settingsResponse struct {
	Timezone        string  `json:"timezone"`
	QuietHoursStart *string `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd   *string `json:"quiet_hours_end,omitempty"`
}

func toSettingsResponse(m postgres.UserSettingsModel) settingsResponse {
	return settingsResponse{
		Timezone:        m.Timezone,
		QuietHoursStart: reformatQuietHours(m.QuietHoursStart),
		QuietHoursEnd:   reformatQuietHours(m.QuietHoursEnd),
	}
}

// reformatQuietHours converte o texto "HH:MM:SS" devolvido pela coluna TIME
// para "HH:MM" — o que o cliente enviou e espera receber de volta.
func reformatQuietHours(v *string) *string {
	if v == nil {
		return nil
	}
	t, err := time.Parse("15:04:05", *v)
	if err != nil {
		return v
	}
	formatted := t.Format(quietHoursLayout)
	return &formatted
}

// Get godoc
// @Summary      Ver Ajustes
// @Tags         settings
// @Produce      json
// @Success      200  {object}  settingsResponse
// @Failure      404  "Usuário sem Ajustes gravados"
// @Router       /settings [get]
func (h *SettingsHandler) Get(c *gin.Context) {
	m, err := h.settings.FindByUser(c.Request.Context(), userIDFrom(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, toSettingsResponse(*m))
}

type updateSettingsRequest struct {
	Timezone        string  `json:"timezone" binding:"required"`
	QuietHoursStart *string `json:"quiet_hours_start"`
	QuietHoursEnd   *string `json:"quiet_hours_end"`
}

// Update godoc
// @Summary      Gravar Ajustes
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        settings  body  updateSettingsRequest  true  "Fuso IANA e Janela de Silêncio"
// @Success      200  {object}  settingsResponse
// @Failure      400  "Corpo inválido, fuso desconhecido, ou horário malformado"
// @Router       /settings [put]
func (h *SettingsHandler) Update(c *gin.Context) {
	var req updateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if _, err := time.LoadLocation(req.Timezone); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if (req.QuietHoursStart == nil) != (req.QuietHoursEnd == nil) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	start, err := parseQuietHours(req.QuietHoursStart)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	end, err := parseQuietHours(req.QuietHoursEnd)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m := &postgres.UserSettingsModel{
		UserID:          userIDFrom(c),
		Timezone:        req.Timezone,
		QuietHoursStart: start,
		QuietHoursEnd:   end,
	}
	if err := h.settings.Upsert(c.Request.Context(), m); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, toSettingsResponse(*m))
}

// parseQuietHours valida "HH:MM" e devolve o texto normalizado "HH:MM:00"
// que a coluna TIME espera. nil permanece nil — Janela desativada.
func parseQuietHours(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	t, err := time.Parse(quietHoursLayout, *v)
	if err != nil {
		return nil, err
	}
	normalized := t.Format("15:04:05")
	return &normalized, nil
}
