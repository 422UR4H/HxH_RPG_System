package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type IListMatches interface {
	ListMatches(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
}

type ListMatchesUC struct {
	repo IRepository
}

func NewListMatchesUC(repo IRepository) *ListMatchesUC {
	return &ListMatchesUC{
		repo: repo,
	}
}

func (uc *ListMatchesUC) ListMatches(
	ctx context.Context, masterUUID uuid.UUID,
) ([]*match.Summary, error) {
	matches, err := uc.repo.ListMatchesByMasterUUID(ctx, masterUUID)
	if err != nil {
		return nil, err
	}
	return matches, nil
}
