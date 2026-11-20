package turn_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// O instante real de abertura do turno e de cada reação (Task 15b): o primeiro registro fica,
// a reação que substitui uma sentada já aberta herda o instante dela, e a que sai não tem um.
func TestTurn_OpeningTimes(t *testing.T) {
	t0 := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	newReaction := func() *action.Action {
		return action.NewAction(uuid.New(), nil, uuid.New(), nil, action.ActionSpeed{},
			nil, nil, nil, nil, &action.Dodge{}, nil, nil)
	}

	t.Run("the turn keeps its opening", func(t *testing.T) {
		tr := turn.NewTurn(action.Action{})
		if !tr.GetOpenedAt().IsZero() {
			t.Fatal("a new turn has an opening")
		}
		tr.SetOpenedAt(t0)
		if !tr.GetOpenedAt().Equal(t0) {
			t.Fatalf("openedAt = %v, want %v", tr.GetOpenedAt(), t0)
		}
	})

	t.Run("a reaction's first opening stays", func(t *testing.T) {
		tr := turn.NewTurn(action.Action{})
		r := newReaction()
		tr.AddReaction(r)
		if _, ok := tr.ReactionOpenedAt(r.GetID()); ok {
			t.Fatal("an unopened reaction has an opening")
		}
		tr.OpenReactionAt(r.GetID(), t0)
		tr.OpenReactionAt(r.GetID(), t0.Add(time.Second))
		if at, ok := tr.ReactionOpenedAt(r.GetID()); !ok || !at.Equal(t0) {
			t.Fatalf("reaction openedAt = %v/%v, want the first %v", at, ok, t0)
		}
	})

	t.Run("a replacement inherits the seated one's opening; an unseated one loses it", func(t *testing.T) {
		tr := turn.NewTurn(action.Action{})
		seated, own, other := newReaction(), newReaction(), newReaction()
		tr.Seat(seated)
		tr.Seat(other)
		tr.OpenReactionAt(seated.GetID(), t0)
		tr.OpenReactionAt(other.GetID(), t0)
		tr.AddReaction(own)
		tr.ReplaceSeated(seated.GetID(), own.GetID())
		if at, ok := tr.ReactionOpenedAt(own.GetID()); !ok || !at.Equal(t0) {
			t.Fatalf("replacement openedAt = %v/%v, want %v", at, ok, t0)
		}
		if _, ok := tr.ReactionOpenedAt(seated.GetID()); ok {
			t.Fatal("the replaced seated reaction kept an opening")
		}
		tr.Unseat(other.GetID())
		if _, ok := tr.ReactionOpenedAt(other.GetID()); ok {
			t.Fatal("the unseated reaction kept an opening")
		}
	})
}
