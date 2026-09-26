package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
	channelapp "self-remember/backend/internal/app/channel"
	domainchannel "self-remember/backend/internal/domain/channel"
)

// ChannelHandler expõe o cadastro e a verificação dos Canais do usuário —
// os contatos e e-mails alvo do aviso (ARCHITECTURE.md §6). Não expõe
// reordenar a cadeia de fallback: um Canal novo sempre entra ao fim.
type ChannelHandler struct {
	channels *postgres.ChannelRepository
	verify   *channelapp.VerificationUseCase
}

func NewChannelHandler(channels *postgres.ChannelRepository, verify *channelapp.VerificationUseCase) *ChannelHandler {
	return &ChannelHandler{channels: channels, verify: verify}
}

type channelResponse struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Address       string `json:"address"`
	Verified      bool   `json:"verified"`
	FallbackOrder int    `json:"fallback_order"`
}

func toChannelResponse(m postgres.ChannelModel) channelResponse {
	return channelResponse{
		ID:            m.ID.String(),
		Kind:          m.Kind,
		Address:       m.Address,
		Verified:      m.VerifiedAt != nil,
		FallbackOrder: m.FallbackOrder,
	}
}

type createChannelRequest struct {
	Kind    string `json:"kind" binding:"required"`
	Address string `json:"address" binding:"required"`
}

// Create godoc
// @Summary      Criar Canal
// @Tags         channels
// @Accept       json
// @Produce      json
// @Param        channel  body  createChannelRequest  true  "Tipo (whatsapp|email) e endereço"
// @Success      201  {object}  channelResponse
// @Failure      400  "Corpo inválido ou tipo de Canal desconhecido"
// @Router       /channels [post]
func (h *ChannelHandler) Create(c *gin.Context) {
	var req createChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if !domainchannel.Kind(req.Kind).Valid() {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	userID := userIDFrom(c)
	count, err := h.channels.CountByUser(c.Request.Context(), userID)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	m := &postgres.ChannelModel{
		ID:            uuid.New(),
		UserID:        userID,
		Kind:          req.Kind,
		Address:       req.Address,
		FallbackOrder: int(count),
	}
	if err := h.channels.Create(c.Request.Context(), m); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, toChannelResponse(*m))
}

// List godoc
// @Summary      Listar Canais
// @Tags         channels
// @Produce      json
// @Success      200  {array}  channelResponse
// @Router       /channels [get]
func (h *ChannelHandler) List(c *gin.Context) {
	channels, err := h.channels.FindByUser(c.Request.Context(), userIDFrom(c))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	response := make([]channelResponse, 0, len(channels))
	for _, ch := range channels {
		response = append(response, toChannelResponse(ch))
	}
	c.JSON(http.StatusOK, response)
}

// Delete godoc
// @Summary      Remover Canal
// @Tags         channels
// @Param        id  path  string  true  "ID do Canal"
// @Success      204
// @Failure      404  "Canal não encontrado ou não pertence ao usuário"
// @Router       /channels/{id} [delete]
func (h *ChannelHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if _, err := h.channels.FindByIDForUser(c.Request.Context(), id, userIDFrom(c)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err := h.channels.Delete(c.Request.Context(), id); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusNoContent)
}

// RequestVerification godoc
// @Summary      Solicitar código de verificação do Canal
// @Tags         channels
// @Param        id  path  string  true  "ID do Canal"
// @Success      204
// @Failure      404  "Canal não encontrado ou não pertence ao usuário"
// @Failure      502  "Falha ao enviar o código pelo próprio Canal"
// @Router       /channels/{id}/verify/request [post]
func (h *ChannelHandler) RequestVerification(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := h.verify.Request(c.Request.Context(), id, userIDFrom(c)); err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.AbortWithStatus(http.StatusNotFound)
		case errors.Is(err, channelapp.ErrUnsupportedKind):
			c.AbortWithStatus(http.StatusBadRequest)
		case errors.Is(err, channelapp.ErrSendFailed):
			// O problema é do provedor externo (SMTP/EvolutionAPI fora do
			// ar), não do pedido do usuário nem do nosso banco.
			c.AbortWithStatus(http.StatusBadGateway)
		default:
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	c.Status(http.StatusNoContent)
}

type confirmVerificationRequest struct {
	Code string `json:"code" binding:"required"`
}

// ConfirmVerification godoc
// @Summary      Confirmar código de verificação do Canal
// @Tags         channels
// @Param        id  path  string  true  "ID do Canal"
// @Param        body  body  confirmVerificationRequest  true  "Código recebido"
// @Success      204
// @Failure      400  "Corpo inválido"
// @Failure      404  "Canal não encontrado ou não pertence ao usuário"
// @Failure      409  "Nenhum código solicitado, código expirado, ou código incorreto"
// @Router       /channels/{id}/verify/confirm [post]
func (h *ChannelHandler) ConfirmVerification(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var req confirmVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := h.verify.Confirm(c.Request.Context(), id, userIDFrom(c), req.Code); err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.AbortWithStatus(http.StatusNotFound)
		case errors.Is(err, channelapp.ErrNoCodeRequested),
			errors.Is(err, channelapp.ErrCodeExpired),
			errors.Is(err, channelapp.ErrInvalidCode):
			c.AbortWithStatus(http.StatusConflict)
		default:
			c.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	c.Status(http.StatusNoContent)
}
