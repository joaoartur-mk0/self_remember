package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
)

// AreaHandler expõe o CRUD de Áreas — a divisão de topo da vida do usuário
// à qual um Compromisso pertence (CONTEXT.md).
type AreaHandler struct {
	areas *postgres.AreaRepository
}

func NewAreaHandler(areas *postgres.AreaRepository) *AreaHandler {
	return &AreaHandler{areas: areas}
}

type areaResponse struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

func toAreaResponse(m postgres.AreaModel) areaResponse {
	return areaResponse{ID: m.ID.String(), Name: m.Name, Color: m.Color}
}

type createAreaRequest struct {
	Name  string  `json:"name" binding:"required"`
	Color *string `json:"color"`
}

// Create godoc
// @Summary      Criar Área
// @Tags         areas
// @Accept       json
// @Produce      json
// @Param        area  body  createAreaRequest  true  "Nome e cor (opcional) da Área"
// @Success      201  {object}  areaResponse
// @Failure      400  "Corpo da requisição inválido"
// @Failure      409  "Já existe uma Área com este nome"
// @Router       /areas [post]
func (h *AreaHandler) Create(c *gin.Context) {
	var req createAreaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	m := &postgres.AreaModel{
		ID:     uuid.New(),
		UserID: userIDFrom(c),
		Name:   req.Name,
		Color:  req.Color,
	}
	if err := h.areas.Create(c.Request.Context(), m); err != nil {
		if errors.Is(err, postgres.ErrDuplicateName) {
			c.AbortWithStatus(http.StatusConflict)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, toAreaResponse(*m))
}

// List godoc
// @Summary      Listar Áreas
// @Tags         areas
// @Produce      json
// @Success      200  {array}  areaResponse
// @Router       /areas [get]
func (h *AreaHandler) List(c *gin.Context) {
	areas, err := h.areas.FindByUser(c.Request.Context(), userIDFrom(c))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	response := make([]areaResponse, 0, len(areas))
	for _, a := range areas {
		response = append(response, toAreaResponse(a))
	}
	c.JSON(http.StatusOK, response)
}

type updateAreaRequest struct {
	Name  string  `json:"name" binding:"required"`
	Color *string `json:"color"`
}

// Update godoc
// @Summary      Editar Área
// @Tags         areas
// @Accept       json
// @Produce      json
// @Param        id    path  string             true  "ID da Área"
// @Param        area  body  updateAreaRequest  true  "Novo nome e cor (opcional)"
// @Success      200  {object}  areaResponse
// @Failure      400  "Corpo da requisição inválido"
// @Failure      404  "Área não encontrada ou não pertence ao usuário"
// @Failure      409  "Já existe uma Área com este nome"
// @Router       /areas/{id} [put]
func (h *AreaHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var req updateAreaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	userID := userIDFrom(c)
	if err := h.areas.UpdateForUser(c.Request.Context(), id, userID, req.Name, req.Color); err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.AbortWithStatus(http.StatusNotFound)
		case errors.Is(err, postgres.ErrDuplicateName):
			c.AbortWithStatus(http.StatusConflict)
		default:
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	c.JSON(http.StatusOK, areaResponse{ID: id.String(), Name: req.Name, Color: req.Color})
}

// Delete godoc
// @Summary      Remover Área
// @Tags         areas
// @Param        id  path  string  true  "ID da Área"
// @Success      204
// @Failure      404  "Área não encontrada ou não pertence ao usuário"
// @Failure      409  "Área ainda tem Compromissos associados"
// @Router       /areas/{id} [delete]
func (h *AreaHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := h.areas.DeleteForUser(c.Request.Context(), id, userIDFrom(c)); err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.AbortWithStatus(http.StatusNotFound)
		case errors.Is(err, postgres.ErrInUse):
			c.AbortWithStatus(http.StatusConflict)
		default:
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	c.Status(http.StatusNoContent)
}
