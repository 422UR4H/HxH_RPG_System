package service_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestCombatResolver_Resolve(t *testing.T) {
	resolver := service.CombatResolver{}

	t.Run("returns non-nil TurnResolution for a Turn with only an action", func(t *testing.T) {
		tRn := makeCombatTurn()

		res := resolver.Resolve(tRn, nil)

		if res == nil {
			t.Fatal("expected non-nil TurnResolution")
		}
	})

	t.Run("IsSettled is false when turn has no finishedAt", func(t *testing.T) {
		tRn := makeCombatTurn()
		res := resolver.Resolve(tRn, nil)
		if res.IsSettled {
			t.Error("expected IsSettled=false for open turn")
		}
	})

	t.Run("IsSettled is true when turn is closed", func(t *testing.T) {
		tRn := makeCombatTurn()
		tRn.Close(time.Now())
		res := resolver.Resolve(tRn, nil)
		if !res.IsSettled {
			t.Error("expected IsSettled=true for closed turn")
		}
	})

	t.Run("ReactionResults has one entry per reaction", func(t *testing.T) {
		tRn := makeCombatTurn()
		act := tRn.GetAction()
		reaction := makeCombatReactionTo((&act).GetID())
		tRn.AddReaction(reaction)

		res := resolver.Resolve(tRn, nil)

		if len(res.ReactionResults) != 1 {
			t.Errorf("expected 1 ReactionResult, got %d", len(res.ReactionResults))
		}
	})
}

func makeCombatTurn() *turn.Turn {
	a := action.NewAction(
		uuid.New(),
		[]uuid.UUID{uuid.New()},
		uuid.Nil,
		nil,
		action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
	)
	return turn.NewTurn(*a)
}

func makeCombatReactionTo(targetID uuid.UUID) *action.Action {
	a := action.NewAction(
		uuid.New(),
		nil,
		targetID,
		nil,
		action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
	)
	a.ReactToID = targetID
	return a
}
