package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/domain/occurrence"
)

// GuardiaoInterval é o intervalo do PeriodicJob (ARCHITECTURE.md §3.4).
const GuardiaoInterval = 5 * time.Minute

// GuardiaoArgs não tem parâmetros: o Guardião sempre varre todas as
// Ocorrências candidatas a estado terminal.
type GuardiaoArgs struct{}

func (GuardiaoArgs) Kind() string { return "guardiao" }

// GuardiaoWorker aplica os estados terminais que ninguém mais aplica
// (ARCHITECTURE.md §3.4): escada esgotada e Horizonte de Relevância.
//
// Deliberadamente fora desta fase (Motor): carry_over=replace (precisa de
// Canais reais e da tela de Compromissos para ser observável de verdade),
// sugestão de arquivamento por Abandono repetido (é UI, fase de API e
// frontend) e health check da EvolutionAPI (não existe adaptador real
// ainda — fase de Canais). Adicionar essas responsabilidades aqui sem que
// nada as consuma seria código morto.
type GuardiaoWorker struct {
	river.WorkerDefaults[GuardiaoArgs]

	occurrences *postgres.OccurrenceRepository
	logger      *slog.Logger
	now         func() time.Time
}

func NewGuardiaoWorker(occurrences *postgres.OccurrenceRepository, logger *slog.Logger, now func() time.Time) *GuardiaoWorker {
	return &GuardiaoWorker{occurrences: occurrences, logger: logger, now: now}
}

func (w *GuardiaoWorker) Work(ctx context.Context, _ *river.Job[GuardiaoArgs]) error {
	now := w.now()

	ceilingAbandoned, err := w.abandonExhausted(ctx)
	if err != nil {
		return fmt.Errorf("abandonar escada esgotada: %w", err)
	}

	horizonAbandoned, err := w.abandonExpired(ctx, now)
	if err != nil {
		return fmt.Errorf("abandonar fora do horizonte: %w", err)
	}

	w.logger.Info("guardião concluído",
		"abandoned_ceiling", ceilingAbandoned, "abandoned_horizon", horizonAbandoned)
	return nil
}

// abandonExhausted abandona Ocorrências em nagging cujo Perfil esgotou a
// escada de Degraus (ADR-0004) sem Reconhecimento.
func (w *GuardiaoWorker) abandonExhausted(ctx context.Context) (int, error) {
	nagging, err := w.occurrences.FindNaggingWithProfile(ctx)
	if err != nil {
		return 0, fmt.Errorf("buscar ocorrências em nagging: %w", err)
	}

	var count int
	for _, n := range nagging {
		if !occurrence.HasReachedCeiling(n.NagProfile, n.Occurrence.CurrentStep) {
			continue
		}
		if err := w.abandon(ctx, n.Occurrence); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// abandonExpired abandona Ocorrências em due ou nagging que saíram do
// Horizonte de Relevância — nunca entregues, mesmo que ainda tivessem
// Degraus disponíveis (ARCHITECTURE.md §3.4).
func (w *GuardiaoWorker) abandonExpired(ctx context.Context, now time.Time) (int, error) {
	threshold := now.Add(-occurrence.HorizonteRelevancia)
	expired, err := w.occurrences.FindActiveDueBefore(ctx, threshold)
	if err != nil {
		return 0, fmt.Errorf("buscar ocorrências fora do horizonte: %w", err)
	}

	var count int
	for _, occ := range expired {
		if err := w.abandon(ctx, occ); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (w *GuardiaoWorker) abandon(ctx context.Context, occ occurrence.Occurrence) error {
	abandoned, err := occ.Abandon()
	if err != nil {
		w.logger.Error("falha ao abandonar ocorrência", "occurrence_id", occ.ID, "error", err)
		return nil
	}
	if err := w.occurrences.UpdateState(ctx, abandoned.ID, abandoned.State); err != nil {
		return fmt.Errorf("persistir abandono da ocorrência %s: %w", occ.ID, err)
	}
	return nil
}
