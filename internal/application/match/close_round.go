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

type CloseRoundUC struct{}

func NewCloseRoundUC() *CloseRoundUC { return &CloseRoundUC{} }

func (uc *CloseRoundUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*round.Round, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	return session.CloseRound()
}
