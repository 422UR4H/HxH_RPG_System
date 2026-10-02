package match

import (
	"context"
	"log"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type OpenNextActionResult struct {
	ClosedTurn *turn.Turn
	OpenedTurn *turn.Turn
	// Resolution is the newly opened turn's projection — a dry run, nothing applied.
	Resolution *service.TurnResolution
	// ClosedResolution is the resolution that was actually applied when the previous turn
	// closed. Nil on the first open of a round.
	ClosedResolution *service.TurnResolution
	Damaged          []matchsession.DamagedCharacter
	// ClosedRound is set when the round ended because no action in the queue could still pay its
	// price, so it closed instead of opening anything. The caller writes it and announces
	// round_closed.
	ClosedRound *round.Round
}

type IOpenNextAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*OpenNextActionResult, error)
}

// OpenNextActionUC writes nothing. The HP its close applies (Damaged) and the round it may end
// (ClosedRound) are written by the room in the closed turn's own transaction (PersistTurnClose),
// or — a round that ends with no turn closed — in a transaction of their own: one master
// command, one transaction (owner decision, 2026-10-02). It runs under r.mu, so it does no I/O.
type OpenNextActionUC struct {
	closeRound ICloseRound
}

func NewOpenNextActionUC(closeRound ICloseRound) *OpenNextActionUC {
	return &OpenNextActionUC{closeRound: closeRound}
}

func (uc *OpenNextActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*OpenNextActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	tr, err := session.OpenNextAction()

	res := &OpenNextActionResult{
		ClosedTurn:       tr.Closed,
		OpenedTurn:       tr.Opened,
		Resolution:       tr.OpenedResolution,
		ClosedResolution: tr.ClosedResolution,
		Damaged:          tr.Damaged,
	}

	// No action in the queue can still pay its price — none passes the gate that applies to
	// it — so the round ends, and whatever is still queued keeps the roll it already made and
	// belongs to the next one. This is the moment CloseRoundUC finally has a caller.
	if tr.NoActionCanPay {
		if uc.closeRound == nil {
			return res, nil
		}
		closed, closeErr := uc.closeRound.Execute(ctx, session, masterUUID, callerUUID)
		if closeErr != nil {
			// The turn is already closed and applied, so refusing the whole operation would
			// leave the table without the baton.
			log.Printf("auto-close round: %v", closeErr)
			return res, nil
		}
		res.ClosedRound = closed
		return res, nil
	}

	if err != nil {
		// A non-nil result alongside a non-nil error is unusual, but tr.Closed != nil means
		// session.OpenNextAction already closed the previous turn and applied its damage
		// before it failed to open the next one — the same half-success this file already
		// accepts for the round auto-close above. Discarding res here would silently drop that
		// closed turn's resolution and its persistence — its HP included, which travels in
		// Damaged to the turn's own PersistTurnClose: the caller's error branch would never
		// announce resolution_updated and never write the turn.
		if tr.Closed != nil {
			return res, err
		}
		return nil, err
	}
	return res, nil
}
