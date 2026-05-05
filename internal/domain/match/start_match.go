package match

import (
	"context"
	"time"

	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IStartMatch interface {
	Start(ctx context.Context, matchUUID uuid.UUID, masterUUID uuid.UUID) error
}

type StartMatchUC struct {
	matchRepo IRepository
}

func NewStartMatchUC(matchRepo IRepository) *StartMatchUC {
	return &StartMatchUC{matchRepo: matchRepo}
}

func (uc *StartMatchUC) Start(
	ctx context.Context,
	matchUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if err == matchPg.ErrMatchNotFound {
			return ErrMatchNotFound
		}
		return err
	}

	if match.MasterUUID != masterUUID {
		return ErrNotMatchMaster
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}
	if match.StoryEndAt != nil {
		return ErrMatchAlreadyFinished
	}

	return uc.matchRepo.StartMatch(ctx, matchUUID, time.Now())
}
