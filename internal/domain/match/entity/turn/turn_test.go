package turn_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

func TestTurn_GetID(t *testing.T) {
	a := action.Action{}
	tRn := turn.NewTurn(a)
	id := tRn.GetID()
	if id == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
}

func TestTurn_AddMasterAction(t *testing.T) {
	a := action.Action{}
	tRn := turn.NewTurn(a)
	ma := action.NewMasterAction()
	tRn.AddMasterAction(*ma)
	got := tRn.GetMasterActions()
	if len(got) != 1 {
		t.Fatalf("expected 1 master action, got %d", len(got))
	}
}

func TestTurn_OpenReaction(t *testing.T) {
	base := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	tn := turn.NewTurn(*base)

	first := action.NewAction(uuid.New(), nil, base.GetID(), nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	second := action.NewAction(uuid.New(), nil, base.GetID(), nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	tn.AddReaction(first)
	tn.AddReaction(second)

	t.Run("records the order the master opened, not the order they arrived", func(t *testing.T) {
		if !tn.OpenReaction(second.GetID()) {
			t.Fatal("OpenReaction must find an attached reaction")
		}
		if !tn.OpenReaction(first.GetID()) {
			t.Fatal("OpenReaction must find an attached reaction")
		}
		got := tn.OpenedReactionIDs()
		if len(got) != 2 || got[0] != second.GetID() || got[1] != first.GetID() {
			t.Fatal("the opening order is the master's, and it is what the chain walks")
		}
	})

	t.Run("opening the same reaction twice does not duplicate it", func(t *testing.T) {
		tn.OpenReaction(first.GetID()) //nolint:errcheck
		if len(tn.OpenedReactionIDs()) != 2 {
			t.Fatal("a reaction is opened once; re-opening is a no-op, not a second slot")
		}
	})

	t.Run("refuses an id that is not attached to this turn", func(t *testing.T) {
		if tn.OpenReaction(uuid.New()) {
			t.Fatal("only a reaction attached to this turn can be opened")
		}
	})
}

func TestUnopenedReactions(t *testing.T) {
	newReaction := func(actor uuid.UUID) *action.Action {
		a := action.NewAction(actor, nil, uuid.New(), nil, action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil)
		a.ReactionKind = action.ReactDodge
		return a
	}

	t.Run("every attached reaction is unopened before the master opens any", func(t *testing.T) {
		tn := turn.NewTurn(*action.NewAction(uuid.New(), nil, uuid.Nil, nil,
			action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil))
		r1, r2 := newReaction(uuid.New()), newReaction(uuid.New())
		tn.AddReaction(r1)
		tn.AddReaction(r2)

		if got := len(tn.UnopenedReactions()); got != 2 {
			t.Fatalf("UnopenedReactions() = %d, want 2", got)
		}
	})

	t.Run("opening one removes it, and only it", func(t *testing.T) {
		tn := turn.NewTurn(*action.NewAction(uuid.New(), nil, uuid.Nil, nil,
			action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil))
		r1, r2 := newReaction(uuid.New()), newReaction(uuid.New())
		tn.AddReaction(r1)
		tn.AddReaction(r2)
		tn.OpenReaction(r1.GetID())

		left := tn.UnopenedReactions()
		if len(left) != 1 {
			t.Fatalf("UnopenedReactions() = %d, want 1", len(left))
		}
		if left[0].GetID() != r2.GetID() {
			t.Fatalf("the wrong reaction stayed unopened: %v", left[0].GetID())
		}
	})
}
