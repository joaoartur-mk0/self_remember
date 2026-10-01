package commitment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/domain/occurrence"
)

// HorizonteMaterializacao é a janela de antecedência em que Ocorrências
// futuras são criadas (ARCHITECTURE.md §3.1). Usado tanto pela varredura
// periódica (internal/jobs.MaterializadorWorker, de hora em hora) quanto
// pela materialização eager de um Compromisso recém-criado em CreateUseCase
// — sem a segunda, o primeiro vencimento de um Compromisso criado logo
// depois de um tick do Materializador ficaria até 1h sem aparecer na tela
// Hoje nem disparar nada, mesmo já tendo vencido.
const HorizonteMaterializacao = 48 * time.Hour

// materializacaoMaxIteracoes limita quantas Ocorrências uma chamada a
// Materialize cria por Compromisso — trava de segurança contra uma regra de
// recorrência que nunca avança o cursor.
const materializacaoMaxIteracoes = 1000

// MaterializeUseCase cria as Ocorrências de um Compromisso que vencem
// dentro do Horizonte de Materialização. Idempotente via
// UNIQUE(commitment_id, due_at) (ADR-0002): chamar de novo para o mesmo
// Compromisso nunca duplica, então é seguro chamar tanto na criação quanto
// na varredura periódica sem coordenar os dois.
type MaterializeUseCase struct {
	occurrences *postgres.OccurrenceRepository
}

func NewMaterializeUseCase(occurrences *postgres.OccurrenceRepository) *MaterializeUseCase {
	return &MaterializeUseCase{occurrences: occurrences}
}

// Materialize cria as Ocorrências de um único Compromisso (active) com
// vencimento entre o maior já materializado (ou RecurrenceStart, se nenhum
// existir ainda) e now+HorizonteMaterializacao. Devolve quantas Ocorrências
// novas foram criadas.
func (uc *MaterializeUseCase) Materialize(ctx context.Context, active postgres.ActiveCommitment, now time.Time) (int, error) {
	loc, err := time.LoadLocation(active.Timezone)
	if err != nil {
		return 0, fmt.Errorf("carregar fuso %q: %w", active.Timezone, err)
	}

	horizon := now.Add(HorizonteMaterializacao)

	cursor, ok, err := uc.occurrences.LatestDueAt(ctx, active.Commitment.ID)
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

		wasCreated, err := uc.occurrences.Create(ctx, occurrence.Occurrence{
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
