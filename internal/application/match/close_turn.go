package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type CloseTurnResult struct {
	ClosedTurn *turn.Turn
	Resolution *service.TurnResolution
	Damaged    []matchsession.DamagedCharacter
	// Refused is the unopened reactions that blocked the close. Non-empty means nothing was
	// closed: the master has to send it again with Confirm.
	Refused []action.Action
}

type ICloseTurn interface {
	Execute(ctx context.Context, session *matchsession.MatchSession,
		masterUUID, callerUUID uuid.UUID, confirm bool) (*CloseTurnResult, error)
}

// CloseTurnUC writes nothing: the HP the close applies (Damaged) is written by the room in the
// turn's own transaction (PersistTurnClose) — one master command, one transaction (owner
// decision, 2026-10-02). It runs under r.mu, so it does no I/O.
type CloseTurnUC struct{}

func NewCloseTurnUC() *CloseTurnUC {
	return &CloseTurnUC{}
}

// Execute ends the open turn on purpose.
//
// The confirmation is the SERVER's, not the front's: refusing here is what makes the
// criterion verifiable without a browser. What is being confirmed away is not the
// calculation — an unopened reaction is in the chain either way — it is the moment to narrate.
//
// Closing a turn does NOT close the round. That no action in the queue can still pay its price
// is detected in exactly one place, OpenNextActionUC, where the scheduling happens. Two detection points is how two versions of
// one rule are born.
func (uc *CloseTurnUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	confirm bool,
) (*CloseTurnResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	if !confirm {
		if pending := session.UnopenedReactions(); len(pending) > 0 {
			return &CloseTurnResult{Refused: pending}, nil
		}
	}
	tr, err := session.CloseOpenTurn()
	if err != nil {
		return nil, err
	}
	return &CloseTurnResult{
		ClosedTurn: tr.Closed,
		Resolution: tr.ClosedResolution,
		Damaged:    tr.Damaged,
	}, nil
}

var _ ICloseTurn = (*CloseTurnUC)(nil)
