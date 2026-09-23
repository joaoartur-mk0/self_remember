// Package logchannel implementa a porta Channel só gravando em log
// estruturado — o "canal falso" da fase de Motor (README.md), para provar o
// motor de cobrança de ponta a ponta antes de existir qualquer adaptador
// real (EvolutionAPI, SMTP).
package logchannel

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"self-remember/backend/internal/domain/channel"
)

// Channel grava cada tentativa de Entrega em log, sem enviar nada de
// verdade. Nunca loga o corpo da mensagem — a mesma regra de
// ARCHITECTURE.md §8 vale mesmo para um canal que não sai do servidor.
type Channel struct {
	kind   channel.Kind
	logger *slog.Logger
}

// New cria um canal falso para o Kind informado.
func New(kind channel.Kind, logger *slog.Logger) *Channel {
	return &Channel{kind: kind, logger: logger}
}

func (c *Channel) Kind() channel.Kind {
	return c.kind
}

// Send só loga que a Entrega "aconteceu" e devolve um ProviderID sintético.
func (c *Channel) Send(_ context.Context, msg channel.Message) (channel.ProviderID, error) {
	providerID := channel.ProviderID(uuid.NewString())
	c.logger.Info("entrega simulada (canal falso)",
		"kind", c.kind,
		"to", msg.To,
		"correlation_id", msg.CorrelationID,
		"provider_id", providerID,
	)
	return providerID, nil
}

// HealthCheck nunca falha: não há sessão externa nenhuma para cair.
func (c *Channel) HealthCheck(_ context.Context) error {
	return nil
}

// SupportsInboundAck é false como em todo adaptador do MVP (ADR-0006).
func (c *Channel) SupportsInboundAck() bool {
	return false
}
