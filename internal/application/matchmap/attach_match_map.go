package matchmapuc

import (
	"context"
	"errors"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/google/uuid"
)

type IAttachMatchMap interface {
	Attach(ctx context.Context, input *AttachMatchMapInput) (*entity.MatchMap, error)
}

type AttachMatchMapInput struct {
	RequesterUUID uuid.UUID
	MatchUUID     uuid.UUID
	MapUUID       uuid.UUID
}

type AttachMatchMapUC struct {
	repo      IRepository
	matchRepo IMatchRepository
}

func NewAttachMatchMapUC(repo IRepository, matchRepo IMatchRepository) *AttachMatchMapUC {
	return &AttachMatchMapUC{repo: repo, matchRepo: matchRepo}
}

func (uc *AttachMatchMapUC) Attach(ctx context.Context, input *AttachMatchMapInput) (*entity.MatchMap, error) {
	info, err := uc.matchRepo.GetMatchInfo(ctx, input.MatchUUID)
	if err != nil {
		if errors.Is(err, ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	if info.MasterUUID != input.RequesterUUID {
		return nil, ErrNotMatchMaster
	}
	if info.GameStartAt != nil {
		return nil, ErrMatchAlreadyStarted
	}

	mm, err := uc.repo.AttachMap(ctx, input.MatchUUID, input.MapUUID)
	if err != nil {
		return nil, err
	}
	return mm, nil
}
