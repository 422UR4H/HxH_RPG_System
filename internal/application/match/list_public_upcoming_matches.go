package match

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type IListPublicUpcomingMatches interface {
	ListPublicUpcomingMatches(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
}

type ListPublicUpcomingMatchesUC struct {
	repo IRepository
}

func NewListPublicUpcomingMatchesUC(repo IRepository) *ListPublicUpcomingMatchesUC {
	return &ListPublicUpcomingMatchesUC{
		repo: repo,
	}
}

func (uc *ListPublicUpcomingMatchesUC) ListPublicUpcomingMatches(
	ctx context.Context, masterUUID uuid.UUID,
) ([]*match.Summary, error) {
	now := time.Now()
	matches, err := uc.repo.ListPublicUpcomingMatches(ctx, now, masterUUID)
	if err != nil {
		return nil, err
	}
	return matches, nil
}
