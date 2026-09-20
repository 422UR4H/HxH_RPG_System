package match

import (
	"context"
	"log"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
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
	// ClosedRound is set when the round ran out: nothing pending could still pay, so the
	// round closed instead of opening anything. The caller announces round_closed.
	ClosedRound *round.Round
}

type IOpenNextAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*OpenNextActionResult, error)
}

type OpenNextActionUC struct {
	statusWriter ISheetStatusWriter
	closeRound   ICloseRound
}

func NewOpenNextActionUC(statusWriter ISheetStatusWriter, closeRound ICloseRound) *OpenNextActionUC {
	return &OpenNextActionUC{statusWriter: statusWriter, closeRound: closeRound}
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
	// Persist before the error check: the previous turn closed and its damage was applied
	// even when there is no next action to open, so bailing out first would leave the
	// in-memory sheet and the row disagreeing.
	persistDamage(ctx, uc.statusWriter, tr.Damaged)

	res := &OpenNextActionResult{
		ClosedTurn:       tr.Closed,
		OpenedTurn:       tr.Opened,
		Resolution:       tr.OpenedResolution,
		ClosedResolution: tr.ClosedResolution,
		Damaged:          tr.Damaged,
	}

	// The round ran out. Nothing pending passes the gate that applies to it, so the round
	// ends — and whatever is still queued keeps the roll it already made and belongs to the
	// next one. This is the moment CloseRoundUC finally has a caller.
	if tr.RoundExhausted {
		if uc.closeRound == nil {
			return res, nil
		}
		closed, closeErr := uc.closeRound.Execute(ctx, session, masterUUID, callerUUID)
		if closeErr != nil {
			// Same policy as persistDamage: the turn is already closed and applied, so
			// refusing the whole operation would leave the table without the baton.
			log.Printf("auto-close round: %v", closeErr)
			return res, nil
		}
		res.ClosedRound = closed
		return res, nil
	}

	if err != nil {
		// A non-nil result alongside a non-nil error is unusual, but tr.Closed != nil means
		// session.OpenNextAction already closed the previous turn and applied its damage
		// (persisted above) before it failed to open the next one — the same half-success this
		// file already accepts for persistDamage and the round auto-close above. Discarding res
		// here would silently drop that closed turn's resolution and its persistence: the
		// caller's error branch never announces resolution_updated and never calls
		// PersistTurnClose.
		if tr.Closed != nil {
			return res, err
		}
		return nil, err
	}
	return res, nil
}

// persistDamage writes the applied HP reductions through.
//
// A failure is logged, not returned: the damage is already applied in memory and the turn
// is already closed, so refusing the whole operation would leave the table stuck with a
// turn that will not open. Losing a row is recoverable; losing the baton is not.
func persistDamage(ctx context.Context, w ISheetStatusWriter, damaged []matchsession.DamagedCharacter) {
	if w == nil {
		return
	}
	for _, d := range damaged {
		if d.Sheet == nil {
			continue
		}
		bars := d.Sheet.GetAllStatusBar()
		if err := w.UpdateStatusBars(
			ctx, d.CharacterID.String(),
			bars[enum.Health], bars[enum.Stamina], bars[enum.Aura],
		); err != nil {
			log.Printf("UpdateStatusBars for %s: %v", d.CharacterID, err)
		}
	}
}
