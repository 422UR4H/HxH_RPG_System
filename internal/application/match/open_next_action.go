package match

import (
	"context"
	"log"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
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
}

type IOpenNextAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*OpenNextActionResult, error)
}

type OpenNextActionUC struct {
	statusWriter ISheetStatusWriter
}

func NewOpenNextActionUC(statusWriter ISheetStatusWriter) *OpenNextActionUC {
	return &OpenNextActionUC{statusWriter: statusWriter}
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
	if err != nil {
		return nil, err
	}
	return &OpenNextActionResult{
		ClosedTurn:       tr.Closed,
		OpenedTurn:       tr.Opened,
		Resolution:       tr.OpenedResolution,
		ClosedResolution: tr.ClosedResolution,
		Damaged:          tr.Damaged,
	}, nil
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
