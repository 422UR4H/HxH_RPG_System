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

// EscapeLandingEdit is edit_action's escapeLanding: where the master puts the piece of an
// escape IF it fails (front-combat-phases.md §6A.5, B13). ReactionID is the escape reaction's
// own ID (edit_action's actionId); a nil Position clears the choice. It travels beside the
// MasterAction rather than inside it: it is not a correction of any roll, and MasterAction is
// the record of those.
type EscapeLandingEdit struct {
	ReactionID uuid.UUID
	Position   *[3]int
}

type IEditAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession,
		masterUUID, callerUUID uuid.UUID, ma *action.MasterAction,
		landing *EscapeLandingEdit) (*EditActionResult, error)
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
	landing *EscapeLandingEdit,
) (*EditActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	// Both parts of one edit are validated before either lands: a landing refused AFTER the
	// condition edit had applied would report failure to a master whose edit, in fact, partly
	// went through — the rule ApplyMasterAction already keeps for its own parts.
	if landing != nil {
		if err := session.CheckEscapeLanding(landing.ReactionID, landing.Position); err != nil {
			return nil, err
		}
	}
	var res *service.TurnResolution
	// A payload with ONLY escapeLanding edits no roll, so it does not go through
	// ApplyMasterAction — which would append an empty MasterAction to the turn's edit history.
	// Without a landing the call is made exactly as before, empty or not.
	if landing == nil || hasRollEdits(ma) {
		var err error
		res, err = session.ApplyMasterAction(ma, masterUUID)
		if err != nil {
			return nil, err
		}
	}
	if landing != nil {
		var err error
		res, err = session.SetEscapeLanding(landing.ReactionID, landing.Position)
		if err != nil {
			// Unreachable: CheckEscapeLanding above ran the same validation under the same
			// lock. If it fires, the two have drifted — fail loudly.
			return nil, err
		}
	}
	return &EditActionResult{Resolution: res, TurnID: session.CurrentTurnID()}, nil
}

// hasRollEdits reports whether a MasterAction carries any of the sections ApplyMasterAction
// applies. nil Skills/TargetID mean "not sent"; an empty-but-present list is an edit.
func hasRollEdits(ma *action.MasterAction) bool {
	return ma != nil && (len(ma.Conditions) > 0 || ma.Skills != nil || ma.TargetID != nil)
}

var _ IEditAction = (*EditActionUC)(nil)
