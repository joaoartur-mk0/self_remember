package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"self-remember/backend/internal/adapters/evolution"
)

// InstanceHandler expõe o status e o pareamento por QR code da única
// instância WhatsApp desta instalação (ADR-0001: instância, não por
// usuário — não existe uma instância por usuário aqui). Qualquer usuário
// autenticado pode ver e reparear: é um deploy self-hosted de confiança
// (ARCHITECTURE.md §9), não multi-inquilino.
type InstanceHandler struct {
	evolution *evolution.Channel
}

func NewInstanceHandler(evolution *evolution.Channel) *InstanceHandler {
	return &InstanceHandler{evolution: evolution}
}

type instanceStatusResponse struct {
	State string `json:"state"`
}

// Status godoc
// @Summary      Estado da instância WhatsApp
// @Tags         whatsapp-instance
// @Produce      json
// @Success      200  {object}  instanceStatusResponse
// @Failure      502  "EvolutionAPI inacessível"
// @Router       /whatsapp-instance/status [get]
func (h *InstanceHandler) Status(c *gin.Context) {
	state, err := h.evolution.ConnectionState(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}
	c.JSON(http.StatusOK, instanceStatusResponse{State: state})
}

type instanceQRCodeResponse struct {
	// QRCode é uma data URI ("data:image/png;base64,...") pronta para um
	// <img src>, ou vazia se a EvolutionAPI não tiver um QR novo para
	// devolver agora (ex.: já conectada, ou pareamento anterior ainda
	// válido) — o cliente decide o que mostrar consultando Status junto.
	QRCode string `json:"qrcode"`
}

// QRCode godoc
// @Summary      Solicitar QR code de pareamento da instância WhatsApp
// @Tags         whatsapp-instance
// @Produce      json
// @Success      200  {object}  instanceQRCodeResponse
// @Failure      502  "EvolutionAPI inacessível"
// @Router       /whatsapp-instance/qrcode [get]
func (h *InstanceHandler) QRCode(c *gin.Context) {
	qrCode, err := h.evolution.Connect(c.Request.Context())
	if err != nil {
		c.AbortWithStatus(http.StatusBadGateway)
		return
	}
	c.JSON(http.StatusOK, instanceQRCodeResponse{QRCode: qrCode})
}
