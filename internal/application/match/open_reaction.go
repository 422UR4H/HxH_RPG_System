package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type OpenReactionResult struct {
	Resolution *service.TurnResolution
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
	resolution, err := session.OpenReaction(reactionID)
	if err != nil {
		return nil, err
	}
	return &OpenReactionResult{Resolution: resolution}, nil
}
