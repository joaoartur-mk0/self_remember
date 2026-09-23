package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/domain/channel"
)

// EntregadorArgs identifica uma tentativa física de Entrega: uma Ocorrência,
// um Degrau e um Canal específicos.
type EntregadorArgs struct {
	OccurrenceID uuid.UUID `json:"occurrence_id"`
	StepIndex    int       `json:"step_index"`
	ChannelID    uuid.UUID `json:"channel_id"`
}

func (EntregadorArgs) Kind() string { return "entregador" }

// EntregadorWorker faz a tentativa física de Entrega. A idempotência é
// garantida no banco antes da chamada externa (ARCHITECTURE.md §3.3): a
// linha em deliveries é inserida primeiro; se colidir com uma já existente,
// esta Entrega já foi feita e o job termina com sucesso sem enviar nada.
//
// sender é o único Channel disponível na fase de Motor — o canal falso de
// log (internal/adapters/logchannel). Escolher o adaptador real por Kind é
// tarefa da fase de Canais reais.
type EntregadorWorker struct {
	river.WorkerDefaults[EntregadorArgs]

	deliveries  *postgres.DeliveryRepository
	channels    *postgres.ChannelRepository
	occurrences *postgres.OccurrenceRepository
	sender      channel.Channel
	logger      *slog.Logger
}

func NewEntregadorWorker(
	deliveries *postgres.DeliveryRepository,
	channels *postgres.ChannelRepository,
	occurrences *postgres.OccurrenceRepository,
	sender channel.Channel,
	logger *slog.Logger,
) *EntregadorWorker {
	return &EntregadorWorker{
		deliveries:  deliveries,
		channels:    channels,
		occurrences: occurrences,
		sender:      sender,
		logger:      logger,
	}
}

func (w *EntregadorWorker) Work(ctx context.Context, job *river.Job[EntregadorArgs]) error {
	args := job.Args

	ch, err := w.channels.FindByID(ctx, args.ChannelID)
	if err != nil {
		return fmt.Errorf("buscar canal %s: %w", args.ChannelID, err)
	}

	correlationID := uuid.NewString()
	delivery := &postgres.DeliveryModel{
		ID:            uuid.New(),
		OccurrenceID:  args.OccurrenceID,
		StepIndex:     args.StepIndex,
		ChannelID:     args.ChannelID,
		CorrelationID: correlationID,
		Status:        postgres.DeliveryStatusPending,
	}

	created, err := w.deliveries.CreateIfNotExists(ctx, delivery)
	if err != nil {
		return fmt.Errorf("criar registro de entrega: %w", err)
	}
	if !created {
		w.logger.Info("entrega já registrada, ignorando (idempotência)",
			"occurrence_id", args.OccurrenceID, "step_index", args.StepIndex, "channel_id", args.ChannelID)
		return nil
	}

	summary, err := w.occurrences.FindSummary(ctx, args.OccurrenceID)
	if err != nil {
		return fmt.Errorf("buscar resumo da ocorrência: %w", err)
	}

	providerID, sendErr := w.sender.Send(ctx, channel.Message{
		To:            ch.Address,
		Body:          fmt.Sprintf("%s venceu em %s", summary.Title, summary.DueAt.Format("02/01 15:04")),
		CorrelationID: correlationID,
	})
	if sendErr != nil {
		return w.handleSendFailure(ctx, delivery.ID, ch, args, sendErr)
	}

	if err := w.deliveries.MarkSent(ctx, delivery.ID, string(providerID)); err != nil {
		return fmt.Errorf("marcar entrega como enviada: %w", err)
	}
	return nil
}

// handleSendFailure registra a falha e, se houver próximo Canal verificado
// na cadeia de fallback do usuário, enfileira uma nova tentativa. O
// Entregador nunca estoura por falha de Canal (ARCHITECTURE.md §3.3) — quem
// decide abandonar por esgotamento de Degraus é o Guardião.
func (w *EntregadorWorker) handleSendFailure(
	ctx context.Context,
	deliveryID uuid.UUID,
	ch *postgres.ChannelModel,
	args EntregadorArgs,
	sendErr error,
) error {
	w.logger.Error("falha ao entregar", "delivery_id", deliveryID, "channel_id", ch.ID, "error", sendErr)

	if err := w.deliveries.MarkFailed(ctx, deliveryID); err != nil {
		return fmt.Errorf("marcar entrega como falha: %w", err)
	}

	next, err := w.channels.NextFallback(ctx, ch.UserID, ch.FallbackOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.logger.Info("cadeia de fallback esgotada", "occurrence_id", args.OccurrenceID, "step_index", args.StepIndex)
			return nil
		}
		return fmt.Errorf("buscar próximo canal de fallback: %w", err)
	}

	riverClient := river.ClientFromContext[pgx.Tx](ctx)
	if _, err := riverClient.Insert(ctx, EntregadorArgs{
		OccurrenceID: args.OccurrenceID,
		StepIndex:    args.StepIndex,
		ChannelID:    next.ID,
	}, nil); err != nil {
		return fmt.Errorf("enfileirar fallback: %w", err)
	}
	return nil
}
