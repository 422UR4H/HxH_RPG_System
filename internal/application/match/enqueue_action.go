package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IEnqueueAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, playerUUID uuid.UUID, a *action.Action) error
}

type EnqueueActionUC struct{}

func NewEnqueueActionUC() *EnqueueActionUC { return &EnqueueActionUC{} }

func (uc *EnqueueActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	playerUUID uuid.UUID,
	a *action.Action,
) error {
	return session.EnqueueAction(playerUUID, a)
}
