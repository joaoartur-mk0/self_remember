package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
	occurrenceapp "self-remember/backend/internal/app/occurrence"
	domainoccurrence "self-remember/backend/internal/domain/occurrence"
)

// OccurrenceHandler expõe a tela Hoje e o Reconhecimento de Ocorrências.
type OccurrenceHandler struct {
	occurrences *postgres.OccurrenceRepository
	settings    *postgres.UserSettingsRepository
	ack         *occurrenceapp.AckUseCase
}

func NewOccurrenceHandler(
	occurrences *postgres.OccurrenceRepository,
	settings *postgres.UserSettingsRepository,
	ack *occurrenceapp.AckUseCase,
) *OccurrenceHandler {
	return &OccurrenceHandler{occurrences: occurrences, settings: settings, ack: ack}
}

type occurrenceResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	DueAt       time.Time `json:"due_at"`
	State       string    `json:"state"`
	CurrentStep int       `json:"current_step"`
}

// Today godoc
// @Summary      Ocorrências de hoje
// @Tags         occurrences
// @Produce      json
// @Success      200  {array}  occurrenceResponse
// @Failure      400  "Usuário sem Ajustes gravados (fuso desconhecido)"
// @Router       /occurrences/today [get]
func (h *OccurrenceHandler) Today(c *gin.Context) {
	userID := userIDFrom(c)

	userSettings, err := h.settings.FindByUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusBadRequest)
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
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc)

	items, err := h.occurrences.FindDueForUser(c.Request.Context(), userID, endOfDay)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	response := make([]occurrenceResponse, 0, len(items))
	for _, item := range items {
		response = append(response, occurrenceResponse{
			ID:          item.Occurrence.ID.String(),
			Title:       item.Title,
			DueAt:       item.Occurrence.DueAt,
			State:       string(item.Occurrence.State),
			CurrentStep: item.Occurrence.CurrentStep,
		})
	}
	c.JSON(http.StatusOK, response)
}

// Complete godoc
// @Summary      Reconhecer Conclusão
// @Tags         occurrences
// @Param        id  path  string  true  "ID da Ocorrência"
// @Success      204
// @Failure      404  "Ocorrência não encontrada ou não pertence ao usuário"
// @Failure      409  "Transição inválida a partir do estado atual"
// @Router       /occurrences/{id}/complete [post]
func (h *OccurrenceHandler) Complete(c *gin.Context) {
	h.handleAck(c, func(id, userID uuid.UUID) error {
		return h.ack.Complete(c.Request.Context(), id, userID)
	})
}

// Dismiss godoc
// @Summary      Reconhecer Desistência
// @Tags         occurrences
// @Param        id  path  string  true  "ID da Ocorrência"
// @Success      204
// @Failure      404  "Ocorrência não encontrada ou não pertence ao usuário"
// @Failure      409  "Transição inválida a partir do estado atual"
// @Router       /occurrences/{id}/dismiss [post]
func (h *OccurrenceHandler) Dismiss(c *gin.Context) {
	h.handleAck(c, func(id, userID uuid.UUID) error {
		return h.ack.Dismiss(c.Request.Context(), id, userID)
	})
}

type snoozeRequest struct {
	DueAt time.Time `json:"due_at" binding:"required"`
}

// Snooze godoc
// @Summary      Reconhecer Adiamento
// @Tags         occurrences
// @Param        id  path  string  true  "ID da Ocorrência"
// @Param        body  body  snoozeRequest  true  "Novo vencimento"
// @Success      204
// @Failure      400  "Corpo inválido"
// @Failure      404  "Ocorrência não encontrada ou não pertence ao usuário"
// @Failure      409  "Transição inválida, ou novo vencimento não é depois do atual"
// @Router       /occurrences/{id}/snooze [post]
func (h *OccurrenceHandler) Snooze(c *gin.Context) {
	var req snoozeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	h.handleAck(c, func(id, userID uuid.UUID) error {
		return h.ack.Snooze(c.Request.Context(), id, userID, req.DueAt)
	})
}

// handleAck resolve o :id da rota e traduz os erros comuns aos três
// Reconhecimentos para status HTTP — só a chamada em si muda entre eles.
func (h *OccurrenceHandler) handleAck(c *gin.Context, do func(id, userID uuid.UUID) error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := do(id, userIDFrom(c)); err != nil {
		var transitionErr *domainoccurrence.TransitionError
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.AbortWithStatus(http.StatusNotFound)
		case errors.As(err, &transitionErr), errors.Is(err, domainoccurrence.ErrSnoozeMustBeAfterCurrent):
			c.AbortWithStatus(http.StatusConflict)
		default:
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	c.Status(http.StatusNoContent)
}
