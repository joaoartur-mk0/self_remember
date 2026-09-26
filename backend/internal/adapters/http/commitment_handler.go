package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
	commitmentapp "self-remember/backend/internal/app/commitment"
	commitmentdomain "self-remember/backend/internal/domain/commitment"
)

// CommitmentHandler expõe o CRUD parcial de Compromissos (Create, List,
// Archive) — edição (PUT) fica fora desta fase: depende de regenerar
// Ocorrências futuras (ADR-0002 ainda não existe).
type CommitmentHandler struct {
	create      *commitmentapp.CreateUseCase
	commitments *postgres.CommitmentRepository
}

func NewCommitmentHandler(create *commitmentapp.CreateUseCase, commitments *postgres.CommitmentRepository) *CommitmentHandler {
	return &CommitmentHandler{create: create, commitments: commitments}
}

type commitmentResponse struct {
	ID              string     `json:"id"`
	AreaID          string     `json:"area_id"`
	Title           string     `json:"title"`
	RecurrenceStart time.Time  `json:"recurrence_start"`
	RecurrenceRule  string     `json:"recurrence_rule,omitempty"`
	NagProfile      string     `json:"nag_profile"`
	CarryOverPolicy string     `json:"carry_over_policy"`
	ArchivedAt      *time.Time `json:"archived_at,omitempty"`
	LabelIDs        []string   `json:"label_ids"`
}

func toCommitmentResponse(c commitmentdomain.Commitment, labelIDs []uuid.UUID) commitmentResponse {
	ids := make([]string, 0, len(labelIDs))
	for _, id := range labelIDs {
		ids = append(ids, id.String())
	}
	return commitmentResponse{
		ID:              c.ID.String(),
		AreaID:          c.AreaID.String(),
		Title:           c.Title,
		RecurrenceStart: c.RecurrenceStart,
		RecurrenceRule:  c.RecurrenceRule,
		NagProfile:      string(c.NagProfile),
		CarryOverPolicy: string(c.CarryOverPolicy),
		ArchivedAt:      c.ArchivedAt,
		LabelIDs:        ids,
	}
}

type createCommitmentRequest struct {
	Title           string      `json:"title" binding:"required"`
	AreaID          uuid.UUID   `json:"area_id" binding:"required"`
	RecurrenceStart time.Time   `json:"recurrence_start" binding:"required"`
	RecurrenceRule  string      `json:"recurrence_rule"`
	NagProfile      string      `json:"nag_profile" binding:"required"`
	CarryOverPolicy string      `json:"carry_over_policy" binding:"required"`
	LabelIDs        []uuid.UUID `json:"label_ids"`
}

// Create godoc
// @Summary      Criar Compromisso
// @Tags         commitments
// @Accept       json
// @Produce      json
// @Param        commitment  body  createCommitmentRequest  true  "Compromisso a criar"
// @Success      201  {object}  commitmentResponse
// @Failure      400  "Corpo inválido, Compromisso inválido, ou Área/Etiqueta não pertence ao usuário"
// @Router       /commitments [post]
func (h *CommitmentHandler) Create(c *gin.Context) {
	var req createCommitmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	domainCommitment := commitmentdomain.Commitment{
		UserID:          userIDFrom(c),
		AreaID:          req.AreaID,
		Title:           req.Title,
		RecurrenceStart: req.RecurrenceStart,
		RecurrenceRule:  req.RecurrenceRule,
		NagProfile:      commitmentdomain.NagProfile(req.NagProfile),
		CarryOverPolicy: commitmentdomain.CarryOverPolicy(req.CarryOverPolicy),
	}

	created, err := h.create.Create(c.Request.Context(), domainCommitment, req.LabelIDs)
	if err != nil {
		// Área/Etiqueta de outro usuário devolve 400, não 404: um 404
		// revelaria a quem está tentando que aquele ID existe para outra
		// pessoa (mesmo raciocínio de auth.ErrInvalidCredentials).
		if errors.Is(err, commitmentapp.ErrAreaNotFound) || errors.Is(err, commitmentapp.ErrLabelNotFound) ||
			errors.Is(err, commitmentdomain.ErrTitleRequired) || errors.Is(err, commitmentdomain.ErrInvalidNagProfile) ||
			errors.Is(err, commitmentdomain.ErrInvalidCarryOverPolicy) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, toCommitmentResponse(created, req.LabelIDs))
}

// List godoc
// @Summary      Listar Compromissos
// @Tags         commitments
// @Produce      json
// @Param        include_archived  query  bool  false  "Incluir Compromissos arquivados"
// @Success      200  {array}  commitmentResponse
// @Router       /commitments [get]
func (h *CommitmentHandler) List(c *gin.Context) {
	includeArchived := c.Query("include_archived") == "true"

	commitments, err := h.commitments.FindByUser(c.Request.Context(), userIDFrom(c), includeArchived)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	response := make([]commitmentResponse, 0, len(commitments))
	for _, cm := range commitments {
		labelIDs, err := h.commitments.FindLabelIDs(c.Request.Context(), cm.ID)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		response = append(response, toCommitmentResponse(cm, labelIDs))
	}
	c.JSON(http.StatusOK, response)
}

// Archive godoc
// @Summary      Arquivar Compromisso
// @Tags         commitments
// @Param        id  path  string  true  "ID do Compromisso"
// @Success      204
// @Failure      404  "Compromisso não encontrado ou não pertence ao usuário"
// @Router       /commitments/{id}/archive [post]
func (h *CommitmentHandler) Archive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if _, err := h.commitments.FindByIDForUser(c.Request.Context(), id, userIDFrom(c)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err := h.commitments.Archive(c.Request.Context(), id, time.Now()); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}
