package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type ICloseRound interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*round.Round, error)
}

type CloseRoundUC struct {
	roundRepo IRoundRepository
}

func NewCloseRoundUC(roundRepo IRoundRepository) *CloseRoundUC {
	return &CloseRoundUC{roundRepo: roundRepo}
}

func (uc *CloseRoundUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*round.Round, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	wasPersisted := session.IsRoundPersisted()
	closedRound, err := session.CloseRound()
	if err != nil {
		return nil, err
	}
	if wasPersisted && closedRound.GetFinishedAt() != nil {
		if dbErr := uc.roundRepo.CloseRound(ctx, closedRound.GetID(), *closedRound.GetFinishedAt()); dbErr != nil {
			_ = dbErr // log in production
		}
	}
	return closedRound, nil
}
