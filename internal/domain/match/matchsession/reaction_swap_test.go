package matchsession_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// B is attacked by A while B's own Dash to (2,2) waits in the queue; B escapes. Fleeing to
// where the queued move was going is not a swap (owner decision, 2026-10-11): it still
// consumes the move, but costs no Disadvantage. Anywhere else, or over an action that did
// more than move, the swap Disadvantage stays.
func swapSetup(t *testing.T, queuedFor func(actor uuid.UUID) *action.Action) (*matchsession.MatchSession, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	playerA, playerB := uuid.New(), uuid.New()
	s, chars := raceSessionWithSheets(t, enum.Race, playerA, playerB)
	a := makeActionWithSpeed(chars[0], 10)
	a.TargetID = []uuid.UUID{chars[1]}
	if err := s.EnqueueAction(playerA, a); err != nil {
		t.Fatalf("enqueue attack: %v", err)
	}
	opened := mustOpen(t, s)
	act := opened.GetAction()
	if err := s.EnqueueAction(playerB, queuedFor(chars[1])); err != nil {
		t.Fatalf("enqueue B's own action: %v", err)
	}
	return s, playerB, chars[1], act.GetID()
}

func escapeTo(actor, reactTo uuid.UUID, col, row int) *action.Action {
	r := action.NewAction(actor, nil, reactTo, nil, action.ActionSpeed{}, nil,
		&action.Move{Category: enum.Dash, Position: [3]int{col, row, 0}}, nil, nil, &action.Dodge{}, nil, nil)
	r.ReactionKind = action.ReactEscape
	return r
}

func TestMatchSession_EscapeToTheQueuedDestinationIsNotASwap(t *testing.T) {
	t.Run("same destination: consumed, no Disadvantage", func(t *testing.T) {
		s, playerB, b, actID := swapSetup(t, dashAction)
		r := escapeTo(b, actID, 2, 2)
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if len(r.ConsumedActionIDs) != 1 {
			t.Fatalf("ConsumedActionIDs = %v, want the queued move", r.ConsumedActionIDs)
		}
		if r.SystemBias != 0 {
			t.Errorf("SystemBias = %d, want 0 — the flight is the move that was queued", r.SystemBias)
		}
	})

	t.Run("another destination: the swap Disadvantage stays", func(t *testing.T) {
		s, playerB, b, actID := swapSetup(t, dashAction)
		r := escapeTo(b, actID, 3, 2)
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if r.SystemBias != -1 {
			t.Errorf("SystemBias = %d, want -1", r.SystemBias)
		}
	})

	t.Run("a queued move that also attacked is a swap, same square or not", func(t *testing.T) {
		combined := func(actor uuid.UUID) *action.Action {
			a := dashAction(actor)
			sword := enum.Sword
			a.Attack = &action.Attack{Weapon: &sword}
			a.TargetID = []uuid.UUID{uuid.New()}
			return a
		}
		s, playerB, b, actID := swapSetup(t, combined)
		r := escapeTo(b, actID, 2, 2)
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if r.SystemBias != -1 {
			t.Errorf("SystemBias = %d, want -1", r.SystemBias)
		}
	})
}

// The master's Advantage on the movement of an escape that carries the system's Disadvantage:
// the two cancel (bias -1 + 1 = 0) and the roll reads the PRIMARY set — neutral, not
// "advantage". When the primary set was already the worse one, nothing moves.
func TestMatchSession_MasterAdvantageCancelsTheSwapDisadvantage(t *testing.T) {
	s, playerB, b, actID := swapSetup(t, dashAction)
	r := escapeTo(b, actID, 5, 5)
	if _, err := s.AttachReaction(playerB, r); err != nil {
		t.Fatalf("AttachReaction: %v", err)
	}
	if r.SystemBias != -1 {
		t.Fatalf("SystemBias = %d, want -1 (a real swap)", r.SystemBias)
	}
	att := r.Move.Speed.Attempts
	skill := r.Move.Speed.SkillValue
	worse := min(sum(att.Primary), sum(att.Secondary))
	if got := r.Move.Speed.Result; got != skill+worse {
		t.Fatalf("with the swap Disadvantage: moveSpeed = %d, want skill %d + the worse set %d", got, skill, worse)
	}

	edit := action.NewMasterAction()
	edit.ActionID = r.GetID()
	edit.Conditions = []action.ConditionEdit{{Field: action.FieldMoveSpeed, Condition: action.RollCondition{Bias: 1}}}
	if _, err := s.ApplyMasterAction(edit, uuid.Nil); err != nil {
		t.Fatalf("ApplyMasterAction: %v", err)
	}
	if got := r.Move.Speed.Result; got != skill+sum(att.Primary) {
		t.Errorf("Advantage over the Disadvantage: moveSpeed = %d, want skill %d + the primary set %d (neutral)",
			got, skill, sum(att.Primary))
	}
}

func sum(d []int) int {
	t := 0
	for _, v := range d {
		t += v
	}
	return t
}
