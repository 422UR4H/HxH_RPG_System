package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type ICancelAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, playerUUID, actionID uuid.UUID) error
}

type CancelActionUC struct{}

func NewCancelActionUC() *CancelActionUC { return &CancelActionUC{} }

func (uc *CancelActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	playerUUID, actionID uuid.UUID,
) error {
	return session.CancelAction(playerUUID, actionID)
}
