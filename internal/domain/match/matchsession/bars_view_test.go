package matchsession_test

import (
	"slices"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// canonicalBars queues the canonical example: price 11; p2 at 23 and 17, p1 at 20, p3 at 11.
func canonicalBars(t *testing.T) (s *matchsession.MatchSession, p1, p2, p3 uuid.UUID) {
	t.Helper()
	u1, u2, u3 := uuid.New(), uuid.New(), uuid.New()
	s, chars := sessionWithParticipants(u1, u2, u3)
	s.GetActiveRound().SetMode(enum.Race)
	for _, e := range []struct {
		player uuid.UUID
		char   uuid.UUID
		speed  int
	}{{u2, chars[1], 23}, {u1, chars[0], 20}, {u2, chars[1], 17}, {u3, chars[2], 11}} {
		if err := s.EnqueueAction(e.player, makeActionWithSpeed(e.char, e.speed)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
	}
	return s, chars[0], chars[1], chars[2]
}

func charView(t *testing.T, v matchsession.BarsView, id uuid.UUID) matchsession.CharacterBarsView {
	t.Helper()
	for _, c := range v.Characters {
		if c.CharacterID == id {
			return c
		}
	}
	t.Fatalf("character %s not in the view", id)
	return matchsession.CharacterBarsView{}
}

func TestMatchSession_BarsView(t *testing.T) {
	t.Run("master sees the provisional price, pending speeds and live balance", func(t *testing.T) {
		s, p1, p2, p3 := canonicalBars(t)
		v := s.BarsView(true)
		if len(v.Prices) != 0 {
			t.Errorf("Prices = %v, want empty", v.Prices)
		}
		if len(v.Provisional) != 1 || v.Provisional[action.BarAction] != 11 {
			t.Errorf("Provisional = %v, want {action: 11}", v.Provisional)
		}
		if len(v.Order) != 4 {
			t.Errorf("Order has %d slots, want 4", len(v.Order))
		}
		c2 := charView(t, v, p2)
		if !slices.Equal(c2.Action.Pending, []int{23, 17}) || c2.Action.Balance != -2 {
			t.Errorf("p2 = %+v, want Pending [23 17], Balance -2", c2.Action)
		}
		if b := charView(t, v, p1).Action.Balance; b != 9 {
			t.Errorf("p1 Balance = %v, want 9", b)
		}
		if b := charView(t, v, p3).Action.Balance; b != 0 {
			t.Errorf("p3 Balance = %v, want 0", b)
		}
	})

	t.Run("player sees nothing the master has not opened", func(t *testing.T) {
		s, _, _, _ := canonicalBars(t)
		v := s.BarsView(false)
		if v.Provisional != nil || v.Order != nil {
			t.Errorf("Provisional = %v, Order = %v, want both nil", v.Provisional, v.Order)
		}
		for _, c := range v.Characters {
			for _, b := range []matchsession.BarView{c.Action, c.Move} {
				if len(b.Pending) != 0 || len(b.Deferred) != 0 || len(b.Acted) != 0 {
					t.Errorf("%s leaks %+v", c.CharacterID, b)
				}
				if b.Balance != b.Carry {
					t.Errorf("Balance = %v, want carry %v", b.Balance, b.Carry)
				}
			}
		}
	})

	t.Run("after the first action opens", func(t *testing.T) {
		s, _, p2, _ := canonicalBars(t)
		mustOpen(t, s)
		pv := s.BarsView(false)
		if pv.Prices[action.BarAction] != 11 {
			t.Errorf("player Prices = %v, want {action: 11}", pv.Prices)
		}
		c := charView(t, pv, p2)
		if !slices.Equal(c.Action.Acted, []int{23}) || c.Action.Balance != 12 {
			t.Errorf("player p2 = %+v, want Acted [23], Balance 12", c.Action)
		}
		mc := charView(t, s.BarsView(true), p2)
		if !slices.Equal(mc.Action.Acted, []int{23}) || !slices.Equal(mc.Action.Pending, []int{17}) || mc.Action.Balance != -2 {
			t.Errorf("master p2 = %+v, want Acted [23], Pending [17], Balance -2", mc.Action)
		}
	})

	t.Run("characters are sorted by id string", func(t *testing.T) {
		s, _, _, _ := canonicalBars(t)
		v := s.BarsView(true)
		if len(v.Characters) != 3 {
			t.Fatalf("got %d characters, want 3", len(v.Characters))
		}
		for i := 1; i < len(v.Characters); i++ {
			if v.Characters[i-1].CharacterID.String() >= v.Characters[i].CharacterID.String() {
				t.Errorf("not sorted at %d", i)
			}
		}
	})

	t.Run("cancelling the only queued action drops the provisional price", func(t *testing.T) {
		u := uuid.New()
		s, chars := sessionWithParticipants(u)
		s.GetActiveRound().SetMode(enum.Race)
		a := makeActionWithSpeed(chars[0], 15)
		if err := s.EnqueueAction(u, a); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		if _, err := s.CancelAction(u, a.GetID()); err != nil {
			t.Fatalf("cancel: %v", err)
		}
		v := s.BarsView(true)
		if _, has := v.Provisional[action.BarAction]; has {
			t.Errorf("Provisional = %v, want no action bar", v.Provisional)
		}
		c := charView(t, v, chars[0])
		if c.Action.Balance != c.Action.Carry {
			t.Errorf("Balance = %v, want carry %v", c.Action.Balance, c.Action.Carry)
		}
	})

	t.Run("a combined action is pending on both bars", func(t *testing.T) {
		u := uuid.New()
		s, chars := sessionWithParticipants(u)
		s.GetActiveRound().SetMode(enum.Race)
		a := makeActionWithSpeed(chars[0], 14)
		a.Move = dashAction(chars[0]).Move
		a.Move.FinalSpeed = 9
		a.Attack = &action.Attack{}
		if err := s.EnqueueAction(u, a); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		c := charView(t, s.BarsView(true), chars[0])
		if !slices.Equal(c.Action.Pending, []int{14}) || !slices.Equal(c.Move.Pending, []int{9}) {
			t.Errorf("Action.Pending = %v, Move.Pending = %v, want [14] and [9]", c.Action.Pending, c.Move.Pending)
		}
	})

	t.Run("a secret reaction is in the master's Acted and not in the player's", func(t *testing.T) {
		s, playerB, b, actID := swapSetup(t, dashAction)
		r := escapeTo(b, actID, 3, 2)
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		mc := charView(t, s.BarsView(true), b)
		pc := charView(t, s.BarsView(false), b)
		if len(mc.Action.Acted) != len(pc.Action.Acted)+1 {
			t.Errorf("master Acted = %v, player Acted = %v, want one more for the master", mc.Action.Acted, pc.Action.Acted)
		}
		_, all := s.BarState(b, action.BarAction)
		_, public := s.PublicBarState(b, action.BarAction)
		if !slices.Equal(mc.Action.Acted, all) || !slices.Equal(pc.Action.Acted, public) {
			t.Errorf("Acted differs from BarState/PublicBarState")
		}
		if mc.Action.Balance == pc.Action.Balance {
			t.Errorf("both balances are %v, the secret speed must only move the master's", mc.Action.Balance)
		}
	})
}
