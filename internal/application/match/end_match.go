package match

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IEndMatch interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, storyAt time.Time, briefFinal *string, confirmLongSkip bool) (*matchsession.MatchEnd, error)
}

// EndMatchUC encerra a partida na memória da sessão. Só o mestre pode. A persistência (e o
// anúncio à mesa) ficam com quem chama.
type EndMatchUC struct{}

func NewEndMatchUC() *EndMatchUC { return &EndMatchUC{} }

func (uc *EndMatchUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	storyAt time.Time,
	briefFinal *string,
	confirmLongSkip bool,
) (*matchsession.MatchEnd, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	brief, err := matchsession.NormalizeBriefFinal(briefFinal)
	if err != nil {
		return nil, err
	}
	end, err := session.EndMatch(storyAt, confirmLongSkip)
	if err != nil {
		return nil, err
	}
	end.BriefFinalDescription = brief
	return end, nil
}

var _ IEndMatch = (*EndMatchUC)(nil)
