package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IEnqueueMasterAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, ma *action.MasterAction) error
}

type EnqueueMasterActionUC struct{}

func NewEnqueueMasterActionUC() *EnqueueMasterActionUC {
	return &EnqueueMasterActionUC{}
}

func (uc *EnqueueMasterActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	ma *action.MasterAction,
) error {
	if callerUUID != masterUUID {
		return ErrNotMatchMaster
	}
	return session.EnqueueMasterAction(ma)
}

var _ IEnqueueMasterAction = (*EnqueueMasterActionUC)(nil)
