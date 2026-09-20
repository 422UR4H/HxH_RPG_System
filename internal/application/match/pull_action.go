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
	// Resolution is the newly opened turn's projection — a dry run, nothing applied.
	Resolution *service.TurnResolution
	// ClosedResolution is the resolution that was actually applied when the previous turn
	// closed. Nil when nothing was open.
	ClosedResolution *service.TurnResolution
	Damaged          []matchsession.DamagedCharacter
}

type IPullAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, actionID uuid.UUID) (*PullActionResult, error)
}

type PullActionUC struct {
	statusWriter ISheetStatusWriter
	// closeRound is held, not used. PullAction never reports exhaustion: the master named an
	// action explicitly, so there is always something to open. It is kept for the explicit
	// round-close path a later phase adds — removing the parameter would churn four call sites
	// for nothing, and re-adding it later would churn them again.
	closeRound ICloseRound //nolint:unused // reserved for the explicit round-close path
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

	res := &PullActionResult{
		ClosedTurn:       tr.Closed,
		OpenedTurn:       tr.Opened,
		Resolution:       tr.OpenedResolution,
		ClosedResolution: tr.ClosedResolution,
		Damaged:          tr.Damaged,
	}

	if err != nil {
		// A non-nil result alongside a non-nil error is unusual, but tr.Closed != nil means
		// session.PullAction already closed the previous turn and applied its damage (persisted
		// above) before it failed to find the requested action — see OpenNextActionUC.Execute
		// for why discarding res here is worse than returning both.
		if tr.Closed != nil {
			return res, err
		}
		return nil, err
	}
	return res, nil
}
