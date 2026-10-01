// Package jobs contém os quatro jobs do motor de cobrança
// (ARCHITECTURE.md §3): Materializador, Scanner, Entregador e Guardião.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"self-remember/backend/internal/adapters/postgres"
	commitmentapp "self-remember/backend/internal/app/commitment"
)

// MaterializadorInterval é o intervalo do PeriodicJob (ARCHITECTURE.md §3.1).
const MaterializadorInterval = time.Hour

// MaterializadorArgs não tem parâmetros: o Materializador sempre varre todos
// os Compromissos ativos.
type MaterializadorArgs struct{}

func (MaterializadorArgs) Kind() string { return "materializador" }

// MaterializadorWorker varre todos os Compromissos ativos e materializa as
// Ocorrências de cada um via commitmentapp.MaterializeUseCase — a mesma
// lógica que CreateUseCase chama uma vez, eager, na criação de um
// Compromisso. Este worker é a varredura periódica: pega recorrências cujo
// próximo vencimento só entrou no Horizonte de Materialização depois da
// criação (ex.: um Compromisso diário, dia seguinte).
type MaterializadorWorker struct {
	river.WorkerDefaults[MaterializadorArgs]

	commitments *postgres.CommitmentRepository
	materialize *commitmentapp.MaterializeUseCase
	logger      *slog.Logger
	now         func() time.Time
}

func NewMaterializadorWorker(
	commitments *postgres.CommitmentRepository,
	materialize *commitmentapp.MaterializeUseCase,
	logger *slog.Logger,
	now func() time.Time,
) *MaterializadorWorker {
	return &MaterializadorWorker{
		commitments: commitments,
		materialize: materialize,
		logger:      logger,
		now:         now,
	}
}

func (w *MaterializadorWorker) Work(ctx context.Context, _ *river.Job[MaterializadorArgs]) error {
	now := w.now()

	actives, err := w.commitments.FindActiveWithTimezone(ctx)
	if err != nil {
		return fmt.Errorf("buscar compromissos ativos: %w", err)
	}

	var created int
	for _, active := range actives {
		n, err := w.materialize.Materialize(ctx, active, now)
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
