package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type AttachReactionResult struct {
	Resolution *service.TurnResolution
}

type IAttachReaction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, callerUUID uuid.UUID, r *action.Action) (*AttachReactionResult, error)
}

type AttachReactionUC struct{}

func NewAttachReactionUC() *AttachReactionUC { return &AttachReactionUC{} }

func (uc *AttachReactionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	callerUUID uuid.UUID,
	r *action.Action,
) (*AttachReactionResult, error) {
	resolution, err := session.AttachReaction(r)
	if err != nil {
		return nil, err
	}
	return &AttachReactionResult{Resolution: resolution}, nil
}
