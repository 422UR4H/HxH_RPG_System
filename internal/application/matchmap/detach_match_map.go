package matchmapuc

import (
	"context"
	"errors"

	pgmatchmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
	"github.com/google/uuid"
)

type IDetachMatchMap interface {
	Detach(ctx context.Context, input *DetachMatchMapInput) error
}

type DetachMatchMapInput struct {
	RequesterUUID uuid.UUID
	MatchUUID     uuid.UUID
}

type DetachMatchMapUC struct {
	repo      IRepository
	matchRepo IMatchRepository
}

func NewDetachMatchMapUC(repo IRepository, matchRepo IMatchRepository) *DetachMatchMapUC {
	return &DetachMatchMapUC{repo: repo, matchRepo: matchRepo}
}

func (uc *DetachMatchMapUC) Detach(ctx context.Context, input *DetachMatchMapInput) error {
	info, err := uc.matchRepo.GetMatchInfo(ctx, input.MatchUUID)
	if err != nil {
		if errors.Is(err, ErrMatchNotFound) {
			return ErrMatchNotFound
		}
		return err
	}
	if info.MasterUUID != input.RequesterUUID {
		return ErrNotMatchMaster
	}
	if info.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}

	if err := uc.repo.DetachMap(ctx, input.MatchUUID); err != nil {
		if errors.Is(err, pgmatchmap.ErrMatchMapNotFound) {
			return ErrMatchMapNotFound
		}
		return err
	}
	return nil
}
