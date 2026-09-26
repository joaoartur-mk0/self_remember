// Package occurrence contém o caso de uso de Reconhecimento: a transição de
// domínio de uma Ocorrência, seguida do registro em acks (o "quando/como" —
// CONTEXT.md) — o próprio domínio não sabe persistir nada.
package occurrence

import (
	"context"
	"time"

	"github.com/google/uuid"

	"self-remember/backend/internal/adapters/postgres"
	domainoccurrence "self-remember/backend/internal/domain/occurrence"
)

// AckUseCase aplica um Reconhecimento a uma Ocorrência: confirma posse,
// aplica a transição de domínio, persiste, e grava o Ack de auditoria.
type AckUseCase struct {
	occurrences *postgres.OccurrenceRepository
	acks        *postgres.AckRepository
}

func NewAckUseCase(occurrences *postgres.OccurrenceRepository, acks *postgres.AckRepository) *AckUseCase {
	return &AckUseCase{occurrences: occurrences, acks: acks}
}

// Complete aplica o Reconhecimento de Conclusão. Devolve
// gorm.ErrRecordNotFound se a Ocorrência não existir ou não pertencer ao
// usuário, ou *occurrence.TransitionError se o estado atual não permitir.
func (uc *AckUseCase) Complete(ctx context.Context, occurrenceID, userID uuid.UUID) error {
	return uc.apply(ctx, occurrenceID, userID, postgres.AckKindCompletion, func(o domainoccurrence.Occurrence) (domainoccurrence.Occurrence, error) {
		return o.Complete()
	})
}

// Dismiss aplica o Reconhecimento de Desistência.
func (uc *AckUseCase) Dismiss(ctx context.Context, occurrenceID, userID uuid.UUID) error {
	return uc.apply(ctx, occurrenceID, userID, postgres.AckKindDismissal, func(o domainoccurrence.Occurrence) (domainoccurrence.Occurrence, error) {
		return o.Dismiss()
	})
}

// Snooze aplica o Reconhecimento de Adiamento, movendo o vencimento para
// newDueAt (precisa ser estritamente posterior ao atual — occurrence.Snooze
// valida isso).
func (uc *AckUseCase) Snooze(ctx context.Context, occurrenceID, userID uuid.UUID, newDueAt time.Time) error {
	return uc.apply(ctx, occurrenceID, userID, postgres.AckKindSnooze, func(o domainoccurrence.Occurrence) (domainoccurrence.Occurrence, error) {
		return o.Snooze(newDueAt)
	})
}

// apply é o esqueleto comum aos três Reconhecimentos: buscar com posse
// confirmada, transicionar, persistir, e gravar o Ack — só a transição em
// si muda entre eles.
func (uc *AckUseCase) apply(
	ctx context.Context,
	occurrenceID, userID uuid.UUID,
	kind postgres.AckKind,
	transition func(domainoccurrence.Occurrence) (domainoccurrence.Occurrence, error),
) error {
	occ, err := uc.occurrences.FindByIDForUser(ctx, occurrenceID, userID)
	if err != nil {
		return err
	}

	occ, err = transition(occ)
	if err != nil {
		return err
	}

	if err := uc.occurrences.Save(ctx, occ); err != nil {
		return err
	}

	return uc.acks.Create(ctx, &postgres.AckModel{
		ID:           uuid.New(),
		OccurrenceID: occ.ID,
		Kind:         string(kind),
		Source:       string(postgres.AckSourceWeb),
	})
}
