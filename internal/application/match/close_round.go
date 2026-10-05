package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type ICloseRound interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*round.Round, error)
}

// CloseRoundUC ends the session's active round in memory — settles the bars, expires the
// end-of-round modifiers, starts the next round — and writes nothing. Its one caller,
// OpenNextActionUC, runs under r.mu; the round's end and the successor's birth are written by the
// room after the unlock, together: in the turn's own transaction when that same command closed a
// turn (TurnCloseData.NextRound), in one of their own otherwise (PersistRoundClose). One master
// command, one transaction (owner decision, 2026-10-02).
//
// What CloseRound settles in memory is not durable anywhere: the bars' carry-over and the
// round's modifiers live only in the session (a restart zeroes the bars — lost, not divergent).
type CloseRoundUC struct{}

func NewCloseRoundUC() *CloseRoundUC {
	return &CloseRoundUC{}
}

func (uc *CloseRoundUC) Execute(
	_ context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*round.Round, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	return session.CloseRound()
}
