package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IChangeRoundMode interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, mode enum.RoundMode) error
}

// ChangeRoundModeUC switches the round between the free regime and the disputed one.
//
// Every rule in the bar economy — the price, the average, acting again, the carry-over — was
// designed looking at the disputed turn, and Race is the only regime with a written rule. This
// is the regime by itself; initiative, the game rule that will normally force it, is a later
// slice.
//
// Switching mid-round is allowed on purpose. The economy simply starts from that moment:
// nobody has acted yet as far as the bars are concerned, and the prices freeze on the next
// selection.
type ChangeRoundModeUC struct{}

func NewChangeRoundModeUC() *ChangeRoundModeUC { return &ChangeRoundModeUC{} }

func (uc *ChangeRoundModeUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	mode enum.RoundMode,
) error {
	if callerUUID != masterUUID {
		return ErrNotMatchMaster
	}
	if mode != enum.Free && mode != enum.Race {
		return ErrInvalidRoundMode
	}
	session.SetRoundMode(mode)
	return nil
}
