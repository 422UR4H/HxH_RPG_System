package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type PullActionResult struct {
	ClosedTurn *turn.Turn
	OpenedTurn *turn.Turn
	// Resolution is the newly opened turn's projection — a dry run, nothing applied.
	Resolution *service.TurnResolution
	// ClosedResolution is the resolution that was actually applied when the previous turn
	// closed. Nil when nothing was open.
	ClosedResolution *service.TurnResolution
	Damaged          []matchsession.DamagedCharacter
	// ClosedRound stays nil here: PullAction never reports exhaustion itself — the master
	// named an action explicitly, so there is always something to open. The collaborator is
	// still taken, for symmetry with OpenNextActionUC and because it is what the room
	// already has to hand.
	ClosedRound *round.Round
}

type IPullAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, actionID uuid.UUID) (*PullActionResult, error)
}

type PullActionUC struct {
	statusWriter ISheetStatusWriter
	closeRound   ICloseRound
}

func NewPullActionUC(statusWriter ISheetStatusWriter, closeRound ICloseRound) *PullActionUC {
	return &PullActionUC{statusWriter: statusWriter, closeRound: closeRound}
}

func (uc *PullActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	actionID uuid.UUID,
) (*PullActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	tr, err := session.PullAction(actionID)
	// Persist before the error check — see OpenNextActionUC.Execute for why.
	persistDamage(ctx, uc.statusWriter, tr.Damaged)
	if err != nil {
		return nil, err
	}
	return &PullActionResult{
		ClosedTurn:       tr.Closed,
		OpenedTurn:       tr.Opened,
		Resolution:       tr.OpenedResolution,
		ClosedResolution: tr.ClosedResolution,
		Damaged:          tr.Damaged,
	}, nil
}
