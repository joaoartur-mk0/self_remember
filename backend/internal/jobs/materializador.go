// Package jobs contém os quatro jobs do motor de cobrança
// (ARCHITECTURE.md §3): Materializador, Scanner, Entregador e Guardião.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/domain/occurrence"
)

// HorizonteMaterializacao é a janela de antecedência em que o Materializador
// cria Ocorrências futuras (ARCHITECTURE.md §3.1).
const HorizonteMaterializacao = 48 * time.Hour

// MaterializadorInterval é o intervalo do PeriodicJob (ARCHITECTURE.md §3.1).
const MaterializadorInterval = time.Hour

// materializacaoMaxIteracoes limita quantas Ocorrências o Materializador cria
// por Compromisso em uma única execução — trava de segurança contra um bug
// de expansão de recorrência que nunca avança o cursor.
const materializacaoMaxIteracoes = 1000

// MaterializadorArgs não tem parâmetros: o Materializador sempre varre todos
// os Compromissos ativos.
type MaterializadorArgs struct{}

func (MaterializadorArgs) Kind() string { return "materializador" }

// MaterializadorWorker cria as Ocorrências que vencem dentro do Horizonte de
// Materialização, para cada Compromisso ativo. Idempotente: reexecutar não
// duplica nada (ADR-0002, via UNIQUE(commitment_id, due_at) em occurrences).
type MaterializadorWorker struct {
	river.WorkerDefaults[MaterializadorArgs]

	commitments *postgres.CommitmentRepository
	occurrences *postgres.OccurrenceRepository
	logger      *slog.Logger
	now         func() time.Time
}

func NewMaterializadorWorker(
	commitments *postgres.CommitmentRepository,
	occurrences *postgres.OccurrenceRepository,
	logger *slog.Logger,
	now func() time.Time,
) *MaterializadorWorker {
	return &MaterializadorWorker{
		commitments: commitments,
		occurrences: occurrences,
		logger:      logger,
		now:         now,
	}
}

func (w *MaterializadorWorker) Work(ctx context.Context, _ *river.Job[MaterializadorArgs]) error {
	now := w.now()
	horizon := now.Add(HorizonteMaterializacao)

	actives, err := w.commitments.FindActiveWithTimezone(ctx)
	if err != nil {
		return fmt.Errorf("buscar compromissos ativos: %w", err)
	}

	var created int
	for _, active := range actives {
		n, err := w.materialize(ctx, active, now, horizon)
		if err != nil {
			w.logger.Error("falha ao materializar compromisso",
				"commitment_id", active.Commitment.ID, "error", err)
			continue
		}
		created += n
	}

	w.logger.Info("materializador concluído", "occurrences_created", created)
	return nil
}

func (w *MaterializadorWorker) materialize(ctx context.Context, active postgres.ActiveCommitment, now, horizon time.Time) (int, error) {
	loc, err := time.LoadLocation(active.Timezone)
	if err != nil {
		return 0, fmt.Errorf("carregar fuso %q: %w", active.Timezone, err)
	}

	cursor, ok, err := w.occurrences.LatestDueAt(ctx, active.Commitment.ID)
	if err != nil {
		return 0, fmt.Errorf("buscar último vencimento materializado: %w", err)
	}
	if !ok {
		cursor = active.Commitment.RecurrenceStart.Add(-time.Nanosecond)
	}

	created := 0
	for i := 0; i < materializacaoMaxIteracoes; i++ {
		next, ok, err := active.Commitment.NextOccurrenceAfter(cursor, loc)
		if err != nil {
			return created, fmt.Errorf("calcular próxima ocorrência: %w", err)
		}
		if !ok || next.After(horizon) {
			return created, nil
		}

		wasCreated, err := w.occurrences.Create(ctx, occurrence.Occurrence{
			ID:           uuid.New(),
			CommitmentID: active.Commitment.ID,
			DueAt:        next,
			Timezone:     active.Timezone,
			State:        occurrence.StatePending,
			CurrentStep:  0,
		})
		if err != nil {
			return created, fmt.Errorf("criar ocorrência: %w", err)
		}
		if wasCreated {
			created++
		}

		cursor = next
	}

	return created, fmt.Errorf("compromisso %s excedeu %d iterações de materialização — possível regra de recorrência sem avanço", active.Commitment.ID, materializacaoMaxIteracoes)
}
