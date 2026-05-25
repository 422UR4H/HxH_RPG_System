package match

import (
	"context"

	pgMatch "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IDeleteMatch interface {
	Delete(ctx context.Context, input *DeleteMatchInput) error
}

type DeleteMatchInput struct {
	MatchUUID  uuid.UUID
	MasterUUID uuid.UUID
}

type DeleteMatchUC struct {
	matchRepo IRepository
}

func NewDeleteMatchUC(matchRepo IRepository) *DeleteMatchUC {
	return &DeleteMatchUC{matchRepo: matchRepo}
}

func (uc *DeleteMatchUC) Delete(ctx context.Context, input *DeleteMatchInput) error {
	m, err := uc.matchRepo.GetMatch(ctx, input.MatchUUID)
	if err != nil {
		if err == pgMatch.ErrMatchNotFound {
			return ErrMatchNotFound
		}
		return err
	}

	if m.MasterUUID != input.MasterUUID {
		return ErrNotMatchMaster
	}

	if m.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}

	if err := uc.matchRepo.DeleteMatch(ctx, input.MatchUUID); err != nil {
		if err == pgMatch.ErrMatchNotFound {
			// Race condition: StartMatch ran concurrently between GetMatch and DeleteMatch
			return ErrMatchAlreadyStarted
		}
		return err
	}

	return nil
}
