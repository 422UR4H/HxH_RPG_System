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

func TestEscapeLandings(t *testing.T) {
	newTurn := func() *turn.Turn {
		return turn.NewTurn(*action.NewAction(uuid.New(), nil, uuid.Nil, nil,
			action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil))
	}

	t.Run("a fresh turn has no landing chosen", func(t *testing.T) {
		if got := newTurn().EscapeLandings(); len(got) != 0 {
			t.Fatalf("EscapeLandings() = %v, want empty", got)
		}
	})

	t.Run("setting stores the position under the reaction, and setting again replaces it", func(t *testing.T) {
		tn := newTurn()
		reactionID := uuid.New()
		tn.SetEscapeLanding(reactionID, [3]int{7, 6, 0})
		tn.SetEscapeLanding(reactionID, [3]int{5, 2, 0})

		got := tn.EscapeLandings()
		if len(got) != 1 || got[reactionID] != [3]int{5, 2, 0} {
			t.Fatalf("EscapeLandings() = %v, want only %s → [5 2 0]", got, reactionID)
		}
	})

	t.Run("clearing removes only that reaction's choice", func(t *testing.T) {
		tn := newTurn()
		kept, cleared := uuid.New(), uuid.New()
		tn.SetEscapeLanding(kept, [3]int{1, 1, 0})
		tn.SetEscapeLanding(cleared, [3]int{2, 2, 0})
		tn.ClearEscapeLanding(cleared)
		// Clearing what was never set is a no-op, not a panic.
		tn.ClearEscapeLanding(uuid.New())

		got := tn.EscapeLandings()
		if len(got) != 1 || got[kept] != [3]int{1, 1, 0} {
			t.Fatalf("EscapeLandings() = %v, want only %s → [1 1 0]", got, kept)
		}
	})

	t.Run("the map handed out is a copy", func(t *testing.T) {
		tn := newTurn()
		reactionID := uuid.New()
		tn.SetEscapeLanding(reactionID, [3]int{3, 3, 0})

		got := tn.EscapeLandings()
		got[reactionID] = [3]int{9, 9, 9}
		delete(got, reactionID)

		if again := tn.EscapeLandings(); again[reactionID] != [3]int{3, 3, 0} {
			t.Fatalf("mutating the returned map reached the turn: %v", again)
		}
	})
}

func TestTurn_SeatedReactions(t *testing.T) {
	base := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	reaction := func(actor uuid.UUID, kind action.ReactionKind) *action.Action {
		r := action.NewAction(actor, nil, base.GetID(), nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		r.ReactionKind = kind
		return r
	}
	a, b := uuid.New(), uuid.New()

	t.Run("a seated reaction is the target's until they answer, and holds no close", func(t *testing.T) {
		tn := turn.NewTurn(*base)
		seated := reaction(a, action.ReactDefault)
		tn.Seat(seated)
		if !tn.IsSeated(seated.GetID()) || tn.SeatedFor(a) == nil || tn.SeatedFor(b) != nil {
			t.Fatal("the seated reaction must be found by its actor and only by it")
		}
		if got := tn.AwaitingTheFloor(); len(got) != 0 {
			t.Fatalf("AwaitingTheFloor = %d, want 0 — nobody sent the seated one", len(got))
		}
		if got := tn.UnopenedReactions(); len(got) != 1 {
			t.Fatalf("UnopenedReactions = %d, want 1 — it is on the turn, unopened", len(got))
		}
	})

	t.Run("the answer replaces the seated one and takes its place in the opening order", func(t *testing.T) {
		tn := turn.NewTurn(*base)
		other := reaction(b, action.ReactRepel)
		seated := reaction(a, action.ReactDefault)
		tn.Seat(seated)
		tn.AddReaction(other)
		tn.OpenReaction(seated.GetID())
		tn.OpenReaction(other.GetID())

		answer := reaction(a, action.ReactDodge)
		tn.AddReaction(answer)
		if !tn.ReplaceSeated(seated.GetID(), answer.GetID()) {
			t.Fatal("ReplaceSeated must find the seated reaction")
		}
		if tn.ReactionRef(seated.GetID()) != nil || tn.IsSeated(seated.GetID()) {
			t.Fatal("the seated reaction must leave the turn")
		}
		got := tn.OpenedReactionIDs()
		if len(got) != 2 || got[0] != answer.GetID() || got[1] != other.GetID() {
			t.Fatalf("opened = %v, want [answer, other] — the character still holds the floor first", got)
		}
		if tn.ReplaceSeated(answer.GetID(), uuid.New()) {
			t.Fatal("a reaction a player sent is never replaced as seated")
		}
	})

	t.Run("unseat removes only a seated reaction", func(t *testing.T) {
		tn := turn.NewTurn(*base)
		seated := reaction(a, action.ReactDefault)
		sent := reaction(b, action.ReactDodge)
		tn.Seat(seated)
		tn.AddReaction(sent)
		tn.OpenReaction(seated.GetID())
		if tn.Unseat(sent.GetID()) {
			t.Fatal("a sent reaction is never unseated")
		}
		if !tn.Unseat(seated.GetID()) || len(tn.GetReactions()) != 1 || len(tn.OpenedReactionIDs()) != 0 {
			t.Fatal("the seated reaction must leave the turn and the opening order")
		}
	})

	t.Run("a nothing never awaits the floor; a sent default does", func(t *testing.T) {
		tn := turn.NewTurn(*base)
		tn.AddReaction(reaction(a, action.ReactNothing))
		sentDefault := reaction(b, action.ReactDefault)
		tn.AddReaction(sentDefault)
		got := tn.AwaitingTheFloor()
		if len(got) != 1 || got[0].GetID() != sentDefault.GetID() {
			t.Fatalf("AwaitingTheFloor = %+v, want only the declared default", got)
		}
	})
}
