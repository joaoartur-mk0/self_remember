// Package commitment contém os casos de uso de Compromisso: a camada que
// valida posse de Área/Etiquetas antes de persistir, algo que o domínio
// puro (internal/domain/commitment) não sabe fazer, pois não conhece o
// banco.
package commitment

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/domain/commitment"
)

// ErrAreaNotFound cobre tanto Área inexistente quanto Área de outro
// usuário — de propósito, para não revelar a um usuário que um ID de Área
// de outra pessoa existe (mesma lógica de auth.ErrInvalidCredentials).
var ErrAreaNotFound = errors.New("área não encontrada")

// ErrLabelNotFound cobre Etiqueta inexistente ou de outro usuário, pelo
// mesmo motivo de ErrAreaNotFound.
var ErrLabelNotFound = errors.New("etiqueta não encontrada")

// CreateUseCase cria um Compromisso, confirmando que a Área e as Etiquetas
// informadas pertencem a quem está criando antes de persistir.
type CreateUseCase struct {
	commitments *postgres.CommitmentRepository
	areas       *postgres.AreaRepository
	labels      *postgres.LabelRepository
}

func NewCreateUseCase(
	commitments *postgres.CommitmentRepository,
	areas *postgres.AreaRepository,
	labels *postgres.LabelRepository,
) *CreateUseCase {
	return &CreateUseCase{commitments: commitments, areas: areas, labels: labels}
}

// Create valida o Compromisso (domínio puro), confirma posse de Área e
// Etiquetas, e persiste os dois juntos.
func (uc *CreateUseCase) Create(ctx context.Context, c commitment.Commitment, labelIDs []uuid.UUID) (commitment.Commitment, error) {
	if err := c.Validate(); err != nil {
		return commitment.Commitment{}, err
	}

	if _, err := uc.areas.FindByIDForUser(ctx, c.AreaID, c.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return commitment.Commitment{}, ErrAreaNotFound
		}
		return commitment.Commitment{}, err
	}

	if err := uc.confirmLabelsOwnership(ctx, c.UserID, labelIDs); err != nil {
		return commitment.Commitment{}, err
	}

	c.ID = uuid.New()
	if err := uc.commitments.Create(ctx, c); err != nil {
		return commitment.Commitment{}, err
	}
	if err := uc.commitments.SetLabels(ctx, c.ID, labelIDs); err != nil {
		return commitment.Commitment{}, err
	}

	return c, nil
}

// confirmLabelsOwnership verifica que cada Etiqueta em labelIDs pertence a
// userID, com uma única consulta em vez de uma por Etiqueta.
func (uc *CreateUseCase) confirmLabelsOwnership(ctx context.Context, userID uuid.UUID, labelIDs []uuid.UUID) error {
	if len(labelIDs) == 0 {
		return nil
	}

	owned, err := uc.labels.FindByUser(ctx, userID)
	if err != nil {
		return err
	}

	ownedSet := make(map[uuid.UUID]bool, len(owned))
	for _, l := range owned {
		ownedSet[l.ID] = true
	}
	for _, id := range labelIDs {
		if !ownedSet[id] {
			return ErrLabelNotFound
		}
	}
	return nil
}
