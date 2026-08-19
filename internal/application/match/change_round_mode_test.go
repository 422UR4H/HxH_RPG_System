package match_test

import (
	"context"
	"testing"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// freeSession builds a session with one participant, active round in the free regime — the
// default a session is created with.
func freeSession(t *testing.T) *matchsession.MatchSession {
	t.Helper()
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := &matchDomain.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: &playerUUID},
	}
	return matchsession.NewMatchSession(matchUUID, nil, []*matchDomain.Participant{participant})
}

func TestChangeRoundMode(t *testing.T) {
	masterUUID := uuid.New()
	uc := appmatch.NewChangeRoundModeUC()

	t.Run("the master turns the disputed turn on", func(t *testing.T) {
		s := freeSession(t)
		if err := uc.Execute(context.Background(), s, masterUUID, masterUUID, enum.Race); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.GetActiveRound().GetMode() != enum.Race {
			t.Error("the round must be racing")
		}
	})

	t.Run("and back off again", func(t *testing.T) {
		s := freeSession(t)
		_ = uc.Execute(context.Background(), s, masterUUID, masterUUID, enum.Race)
		if err := uc.Execute(context.Background(), s, masterUUID, masterUUID, enum.Free); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.GetActiveRound().GetMode() != enum.Free {
			t.Error("the round must be free again")
		}
	})

	t.Run("only the master", func(t *testing.T) {
		s := freeSession(t)
		err := uc.Execute(context.Background(), s, masterUUID, uuid.New(), enum.Race)
		if err == nil {
			t.Error("a player must not switch the regime")
		}
	})

	t.Run("an unknown mode is refused", func(t *testing.T) {
		s := freeSession(t)
		if err := uc.Execute(context.Background(), s, masterUUID, masterUUID, "Sprint"); err == nil {
			t.Error("only Free and Race exist")
		}
	})
}
