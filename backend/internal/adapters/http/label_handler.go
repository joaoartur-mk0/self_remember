package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"self-remember/backend/internal/adapters/postgres"
)

// LabelHandler expõe o CRUD de Etiquetas — classificação livre e
// transversal, sem efeito na cobrança (CONTEXT.md).
type LabelHandler struct {
	labels *postgres.LabelRepository
}

func NewLabelHandler(labels *postgres.LabelRepository) *LabelHandler {
	return &LabelHandler{labels: labels}
}

type labelResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toLabelResponse(m postgres.LabelModel) labelResponse {
	return labelResponse{ID: m.ID.String(), Name: m.Name}
}

type createLabelRequest struct {
	Name string `json:"name" binding:"required"`
}

// Create godoc
// @Summary      Criar Etiqueta
// @Tags         labels
// @Accept       json
// @Produce      json
// @Param        label  body  createLabelRequest  true  "Nome da Etiqueta"
// @Success      201  {object}  labelResponse
// @Failure      400  "Corpo da requisição inválido"
// @Failure      409  "Já existe uma Etiqueta com este nome"
// @Router       /labels [post]
func (h *LabelHandler) Create(c *gin.Context) {
	var req createLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m := &postgres.LabelModel{
		ID:     uuid.New(),
		UserID: userIDFrom(c),
		Name:   req.Name,
	}
	if err := h.labels.Create(c.Request.Context(), m); err != nil {
		if errors.Is(err, postgres.ErrDuplicateName) {
			c.AbortWithStatus(http.StatusConflict)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, toLabelResponse(*m))
}

// List godoc
// @Summary      Listar Etiquetas
// @Tags         labels
// @Produce      json
// @Success      200  {array}  labelResponse
// @Router       /labels [get]
func (h *LabelHandler) List(c *gin.Context) {
	labels, err := h.labels.FindByUser(c.Request.Context(), userIDFrom(c))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	response := make([]labelResponse, 0, len(labels))
	for _, l := range labels {
		response = append(response, toLabelResponse(l))
	}
	c.JSON(http.StatusOK, response)
}
