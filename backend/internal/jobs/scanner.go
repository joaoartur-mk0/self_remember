package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"self-remember/backend/internal/domain/commitment"
	"self-remember/backend/internal/domain/occurrence"
)

// ScannerInterval é o intervalo do PeriodicJob (ARCHITECTURE.md §3.2).
const ScannerInterval = time.Minute

// ScannerArgs não tem parâmetros: o Scanner sempre varre todas as Ocorrências
// elegíveis.
type ScannerArgs struct{}

func (ScannerArgs) Kind() string { return "scanner" }

// ScannerWorker é o único produtor de jobs de Entrega do sistema (ADR-0003).
// A cada execução, em uma única transação pgx: seleciona Ocorrências
// elegíveis com FOR UPDATE SKIP LOCKED, escala as que já maturaram e
// enfileira o Entregador via InsertTx — tudo na mesma transação que avança
// occurrences.current_step, para que nada seja enfileirado sem o avanço
// correspondente ter sido persistido (e vice-versa).
//
// Usa pgx nativo, não o GORM dos outros repositórios: river.InsertTx exige
// uma transação pgx, e a API GORM sobre database/sql não expõe uma.
type ScannerWorker struct {
	river.WorkerDefaults[ScannerArgs]

	pool   *pgxpool.Pool
	logger *slog.Logger
	now    func() time.Time
}

func NewScannerWorker(pool *pgxpool.Pool, logger *slog.Logger, now func() time.Time) *ScannerWorker {
	return &ScannerWorker{pool: pool, logger: logger, now: now}
}

const scannerSelectSQL = `
	SELECT o.id, o.commitment_id, o.due_at, o.timezone, o.state, o.current_step,
	       c.nag_profile, c.user_id
	FROM occurrences o
	JOIN commitments c ON c.id = o.commitment_id
	WHERE o.state IN ('pending', 'due', 'nagging')
	  AND o.due_at <= $1
	  AND o.due_at >= $2
	FOR UPDATE OF o SKIP LOCKED
`

type scannerCandidate struct {
	ID           uuid.UUID
	CommitmentID uuid.UUID
	DueAt        time.Time
	Timezone     string
	State        string
	CurrentStep  int
	NagProfile   string
	UserID       uuid.UUID
}

func (w *ScannerWorker) Work(ctx context.Context, _ *river.Job[ScannerArgs]) error {
	riverClient := river.ClientFromContext[pgx.Tx](ctx)
	now := w.now()

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx)

	candidates, err := w.selectCandidates(ctx, tx, now)
	if err != nil {
		return err
	}

	var escalated, held int
	for _, cand := range candidates {
		didEscalate, err := w.processCandidate(ctx, tx, riverClient, cand, now)
		if err != nil {
			return fmt.Errorf("processar ocorrência %s: %w", cand.ID, err)
		}
		if didEscalate {
			escalated++
		} else {
			held++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	w.logger.Info("scanner concluído", "candidates", len(candidates), "escalated", escalated, "held", held)
	return nil
}

func (w *ScannerWorker) selectCandidates(ctx context.Context, tx pgx.Tx, now time.Time) ([]scannerCandidate, error) {
	rows, err := tx.Query(ctx, scannerSelectSQL, now, now.Add(-occurrence.HorizonteRelevancia))
	if err != nil {
		return nil, fmt.Errorf("selecionar candidatas: %w", err)
	}
	defer rows.Close()

	var candidates []scannerCandidate
	for rows.Next() {
		var c scannerCandidate
		if err := rows.Scan(&c.ID, &c.CommitmentID, &c.DueAt, &c.Timezone, &c.State, &c.CurrentStep, &c.NagProfile, &c.UserID); err != nil {
			return nil, fmt.Errorf("ler candidata: %w", err)
		}
		candidates = append(candidates, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar candidatas: %w", err)
	}
	return candidates, nil
}

// processCandidate decide o destino de uma Ocorrência elegível: escalar
// (enfileirando o Entregador) ou represar (só persistir pending→due, se for
// o caso, e esperar o próximo tick). Devolve true se escalou.
func (w *ScannerWorker) processCandidate(
	ctx context.Context,
	tx pgx.Tx,
	riverClient *river.Client[pgx.Tx],
	cand scannerCandidate,
	now time.Time,
) (bool, error) {
	loc, err := time.LoadLocation(cand.Timezone)
	if err != nil {
		return false, fmt.Errorf("carregar fuso %q: %w", cand.Timezone, err)
	}

	occ := occurrence.Occurrence{
		ID:           cand.ID,
		CommitmentID: cand.CommitmentID,
		DueAt:        cand.DueAt,
		Timezone:     cand.Timezone,
		State:        occurrence.State(cand.State),
		CurrentStep:  cand.CurrentStep,
	}

	if occ.State == occurrence.StatePending {
		occ, err = occ.MarkDue()
		if err != nil {
			return false, fmt.Errorf("marcar due: %w", err)
		}
	}

	quietHours, err := loadQuietHours(ctx, tx, cand.UserID)
	if err != nil {
		return false, fmt.Errorf("carregar janela de silêncio: %w", err)
	}

	shouldEscalate := occurrence.ShouldEscalate(commitment.NagProfile(cand.NagProfile), occ.CurrentStep, occ.DueAt, now)

	var channelID uuid.UUID
	var hasChannel bool
	if shouldEscalate && !quietHours.Active(now, loc) {
		channelID, hasChannel, err = findPrimaryChannel(ctx, tx, cand.UserID)
		if err != nil {
			return false, fmt.Errorf("buscar canal primário: %w", err)
		}
	}

	if !shouldEscalate || quietHours.Active(now, loc) || !hasChannel {
		if err := persistOccurrenceState(ctx, tx, occ); err != nil {
			return false, fmt.Errorf("persistir estado represado: %w", err)
		}
		return false, nil
	}

	occ, err = occ.Escalate()
	if err != nil {
		return false, fmt.Errorf("escalar: %w", err)
	}

	if err := persistOccurrenceState(ctx, tx, occ); err != nil {
		return false, fmt.Errorf("persistir escalonamento: %w", err)
	}

	if _, err := riverClient.InsertTx(ctx, tx, EntregadorArgs{
		OccurrenceID: occ.ID,
		StepIndex:    occ.CurrentStep,
		ChannelID:    channelID,
	}, nil); err != nil {
		return false, fmt.Errorf("enfileirar entregador: %w", err)
	}

	return true, nil
}

func persistOccurrenceState(ctx context.Context, tx pgx.Tx, occ occurrence.Occurrence) error {
	_, err := tx.Exec(ctx,
		`UPDATE occurrences SET state = $1, current_step = $2, updated_at = now() WHERE id = $3`,
		string(occ.State), occ.CurrentStep, occ.ID,
	)
	return err
}

// loadQuietHours carrega a Janela de Silêncio do usuário. Ausência de
// configuração (NULL) equivale a Janela desativada — ver QuietHours.
func loadQuietHours(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (occurrence.QuietHours, error) {
	var startMicros, endMicros int64
	err := tx.QueryRow(ctx, `
		SELECT
			EXTRACT(EPOCH FROM COALESCE(quiet_hours_start, TIME '00:00:00')) * 1000000,
			EXTRACT(EPOCH FROM COALESCE(quiet_hours_end, TIME '00:00:00')) * 1000000
		FROM user_settings WHERE user_id = $1
	`, userID).Scan(&startMicros, &endMicros)
	if err != nil {
		return occurrence.QuietHours{}, err
	}
	return occurrence.QuietHours{
		Start: time.Duration(startMicros) * time.Microsecond,
		End:   time.Duration(endMicros) * time.Microsecond,
	}, nil
}

// findPrimaryChannel devolve o Canal verificado de menor fallback_order de um
// usuário — o primeiro da cadeia de fallback (ARCHITECTURE.md §3.3).
func findPrimaryChannel(ctx context.Context, tx pgx.Tx, userID uuid.UUID) (uuid.UUID, bool, error) {
	var channelID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id FROM channels
		WHERE user_id = $1 AND verified_at IS NOT NULL
		ORDER BY fallback_order ASC
		LIMIT 1
	`, userID).Scan(&channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, false, nil
		}
		return uuid.Nil, false, err
	}
	return channelID, true, nil
}
