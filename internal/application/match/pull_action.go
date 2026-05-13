package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type PullActionResult struct {
	ClosedTurn *turn.Turn
	OpenedTurn *turn.Turn
	Resolution *service.TurnResolution
}

type IPullAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, actionID uuid.UUID) (*PullActionResult, error)
}

type PullActionUC struct{}

func NewPullActionUC() *PullActionUC { return &PullActionUC{} }

func (uc *PullActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	actionID uuid.UUID,
) (*PullActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	closed, opened, err := session.PullAction(actionID)
	if err != nil {
		return nil, err
	}
	resolution := service.CombatResolver{}.Resolve(opened, nil)
	return &PullActionResult{ClosedTurn: closed, OpenedTurn: opened, Resolution: resolution}, nil
}
