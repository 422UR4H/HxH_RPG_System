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
	// Cancelled são os movimentos que caíram em cascata (spec 2026-11-06 D6): os empilhados no
	// que a reação consumiu, quando ela não foi para onde ele ia. Em ordem de cadeia; vazio quando
	// nada caiu. A Room grava cada um e manda um action_cancelled por action.
	Cancelled []*action.Action
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
	resolution, cancelled, err := session.AttachReactionWithCancels(callerUUID, r)
	if err != nil {
		return nil, err
	}
	return &AttachReactionResult{Resolution: resolution, Cancelled: cancelled}, nil
}
