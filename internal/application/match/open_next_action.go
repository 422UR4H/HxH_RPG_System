package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type OpenNextActionResult struct {
	ClosedTurn *turn.Turn
	OpenedTurn *turn.Turn
	Resolution *service.TurnResolution
}

type IOpenNextAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*OpenNextActionResult, error)
}

type OpenNextActionUC struct{}

func NewOpenNextActionUC() *OpenNextActionUC { return &OpenNextActionUC{} }

func (uc *OpenNextActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*OpenNextActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	closed, opened, err := session.OpenNextAction()
	if err != nil {
		return nil, err
	}
	resolution := service.TurnResolver{}.Resolve(opened, nil, session)
	return &OpenNextActionResult{ClosedTurn: closed, OpenedTurn: opened, Resolution: resolution}, nil
}
