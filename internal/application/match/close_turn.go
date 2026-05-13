package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type ICloseTurn interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*turn.Turn, error)
}

type CloseTurnUC struct{}

func NewCloseTurnUC() *CloseTurnUC { return &CloseTurnUC{} }

func (uc *CloseTurnUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*turn.Turn, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	return session.CloseTurn()
}
