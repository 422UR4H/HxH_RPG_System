package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type EditActionResult struct {
	Resolution *service.TurnResolution
	TurnID     uuid.UUID
}

type IEditAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession,
		masterUUID, callerUUID uuid.UUID, ma *action.MasterAction) (*EditActionResult, error)
}

type EditActionUC struct{}

func NewEditActionUC() *EditActionUC { return &EditActionUC{} }

// Execute applies the master's edit and recomputes. There is deliberately NO confirmation
// verb: the master edits, the resolution recomputes on the spot, and passing the baton — open
// the next action, open the next reaction, close the turn — is the confirmation. What a
// confirm button would really offer is cancel, and cancelling is editing back to the original,
// which the master already has in hand.
func (uc *EditActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	ma *action.MasterAction,
) (*EditActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	res, err := session.ApplyMasterAction(ma, masterUUID)
	if err != nil {
		return nil, err
	}
	return &EditActionResult{Resolution: res, TurnID: session.CurrentTurnID()}, nil
}

var _ IEditAction = (*EditActionUC)(nil)
