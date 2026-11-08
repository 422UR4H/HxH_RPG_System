package match

import (
	"context"
	"time"
	"unicode/utf8"

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
	if briefFinal != nil && utf8.RuneCountInString(*briefFinal) > 255 {
		return nil, matchsession.ErrBriefFinalTooLong
	}
	return session.EndMatch(storyAt, confirmLongSkip)
}

var _ IEndMatch = (*EndMatchUC)(nil)
