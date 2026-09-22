package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type OpenReactionResult struct {
	Resolution *service.TurnResolution
	// Opened is the reaction that just got the floor. It travels because an escape
	// DISPLACES: the delivery layer walks its Move onto the board the same way it walks the
	// opened action's, and the reaction is the only thing that carries it. Nil is impossible
	// on a successful Execute — a reaction that could not be found is the error path.
	//
	// It ALIASES the turn's own reaction rather than copying it, so a caller that reads it
	// outside the lock that serialized Execute must copy what it needs first.
	Opened *action.Action
}

type IOpenReaction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, callerUUID, reactionID uuid.UUID) (*OpenReactionResult, error)
}

type OpenReactionUC struct{}

func NewOpenReactionUC() *OpenReactionUC { return &OpenReactionUC{} }

// Execute opens one reaction on the open turn. Master-only — the caller check lives in the
// delivery layer, exactly as it does for open_next_action.
func (uc *OpenReactionUC) Execute(
	ctx context.Context, session *matchsession.MatchSession, callerUUID, reactionID uuid.UUID,
) (*OpenReactionResult, error) {
	opened, resolution, err := session.OpenReaction(reactionID)
	if err != nil {
		return nil, err
	}
	return &OpenReactionResult{Resolution: resolution, Opened: opened}, nil
}
