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
		// Only the next pending (23) counts: 0 + 23 - 1*11 = 12. The 17 behind it does not
		// weigh on the bar until it is the next one.
		if !slices.Equal(c2.Action.Pending, []int{23, 17}) || c2.Action.Balance != 12 {
			t.Errorf("p2 = %+v, want Pending [23 17], Balance 12", c2.Action)
		}
		if b := charView(t, v, p1).Action.Balance; b != 9 {
			t.Errorf("p1 Balance = %v, want 9", b)
		}
		if b := charView(t, v, p3).Action.Balance; b != 0 {
			t.Errorf("p3 Balance = %v, want 0", b)
		}
	})

	t.Run("master balance counts acted plus only the next pending", func(t *testing.T) {
		s, _, p2, _ := canonicalBars(t)
		c := charView(t, s.BarsView(true), p2).Action
		// Pending [23 17], provisional price 11: only the 23 counts — 0 + 23 − 1×11 = 12.
		if !slices.Equal(c.Pending, []int{23, 17}) || c.Balance != 12 {
			t.Errorf("before open = %+v, want Pending [23 17], Balance 12", c)
		}
		mustOpen(t, s)
		// O 23 abriu: agiram [23] ++ a próxima [17] → (23+17)/2 − 2×11 = −2.
		c = charView(t, s.BarsView(true), p2).Action
		if !slices.Equal(c.Acted, []int{23}) || !slices.Equal(c.Pending, []int{17}) || c.Balance != -2 {
			t.Errorf("after open = %+v, want Acted [23], Pending [17], Balance -2", c)
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
		// Fixed dice (a low attack roll, then high ones): the balances must not depend on luck.
		s, playerB, b, actID := swapSetupWithDice(t, dashAction, []int{2, 2, 2, 2, 6})
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

// The price becomes public when an action actually opens (spec 2026-11-04 D7): a command that
// fails before opening anything must leave the round unpriced.
func TestMatchSession_PriceFreezesOnlyWhenAnActionOpens(t *testing.T) {
	t.Run("a pull of an unknown id freezes nothing", func(t *testing.T) {
		s, _, _, _ := canonicalBars(t)
		if _, err := s.PullAction(uuid.New()); err == nil {
			t.Fatal("pulling an action that is not queued must fail")
		}
		if got := s.RoundPrices(); len(got) != 0 {
			t.Errorf("RoundPrices = %v, want empty", got)
		}
	})

	t.Run("open_next with nothing able to pay and no turn opened freezes nothing", func(t *testing.T) {
		// Round 1 leaves p2 at carry -2 (acted 23 and 17 against price 11). In round 2 a
		// lone p2 action is the price itself, yet the debt keeps it from paying.
		u1, u2, u3 := uuid.New(), uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(u1, u2, u3)
		s.GetActiveRound().SetMode(enum.Race)
		p2 := chars[1]
		for _, e := range []struct {
			player, char uuid.UUID
			speed        int
		}{{u2, p2, 23}, {u1, chars[0], 20}, {u2, p2, 17}, {u3, chars[2], 11}} {
			if err := s.EnqueueAction(e.player, makeActionWithSpeed(e.char, e.speed)); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
		}
		closeRoundWhenNoActionCanPay(t, s)
		if carry, _ := s.BarState(p2, action.BarAction); carry >= 0 {
			t.Fatalf("fixture did not leave p2 in debt (carry %v)", carry)
		}
		s.GetActiveRound().SetMode(enum.Race)
		if err := s.EnqueueAction(u2, makeActionWithSpeed(p2, 5)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		if got := s.RoundPrices(); len(got) != 0 {
			t.Fatalf("precondition: new round already priced: %v", got)
		}
		tr, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("OpenNextAction: %v", err)
		}
		if !tr.NoActionCanPay {
			t.Fatal("fixture: something could pay in the new round")
		}
		if got := s.RoundPrices(); len(got) != 0 {
			t.Errorf("RoundPrices = %v, want empty", got)
		}
	})

	t.Run("a successful open freezes the provisional price and opens the fastest", func(t *testing.T) {
		s, _, _, _ := canonicalBars(t)
		tr, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("OpenNextAction: %v", err)
		}
		if tr.Opened == nil {
			t.Fatal("nothing opened")
		}
		opened := tr.Opened.GetAction()
		if opened.SpeedOn(action.BarAction) != 23 {
			t.Fatalf("opened speed = %d, want 23", opened.SpeedOn(action.BarAction))
		}
		if got := s.RoundPrices()[action.BarAction]; got != 11 {
			t.Errorf("price = %d, want 11", got)
		}
	})

	t.Run("pulling the slowest still counts it in the minimum", func(t *testing.T) {
		s, _, _, _ := canonicalBars(t)
		var slowest uuid.UUID
		for _, a := range s.PendingActions() {
			if a.SpeedOn(action.BarAction) == 11 {
				slowest = a.GetID()
			}
		}
		if _, err := s.PullAction(slowest); err != nil {
			t.Fatalf("PullAction: %v", err)
		}
		if got := s.RoundPrices()[action.BarAction]; got != 11 {
			t.Errorf("price = %d, want 11", got)
		}
	})

	t.Run("a round closing after a turn opened still freezes the pending bar", func(t *testing.T) {
		// Round 1 leaves p2 in debt on the MOVE bar (23 and 17 against price 11). In round 2
		// p1's attack opens (action bar prices), then p2's move arrives: the move bar is still
		// unfrozen, p2 cannot pay, and the look that closes the round must freeze it so
		// settleBars settles that bar like it always did.
		u1, u2, u3 := uuid.New(), uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(u1, u2, u3)
		s.GetActiveRound().SetMode(enum.Race)
		move := func(char uuid.UUID, speed int) *action.Action {
			a := dashAction(char)
			a.Move.FinalSpeed = speed
			return a
		}
		for _, e := range []struct {
			player, char uuid.UUID
			speed        int
		}{{u2, chars[1], 23}, {u1, chars[0], 20}, {u2, chars[1], 17}, {u3, chars[2], 11}} {
			if err := s.EnqueueAction(e.player, move(e.char, e.speed)); err != nil {
				t.Fatalf("enqueue: %v", err)
			}
		}
		closeRoundWhenNoActionCanPay(t, s)
		if carry, _ := s.BarState(chars[1], action.BarMove); carry >= 0 {
			t.Fatalf("fixture did not leave p2 in move debt (carry %v)", carry)
		}
		s.GetActiveRound().SetMode(enum.Race)
		if err := s.EnqueueAction(u1, makeActionWithSpeed(chars[0], 9)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		if tr, err := s.OpenNextAction(); err != nil || tr.Opened == nil {
			t.Fatalf("the attack must open: %v", err)
		}
		if _, ok := s.RoundPrices()[action.BarMove]; ok {
			t.Fatal("precondition: the move bar must still be unpriced")
		}
		if err := s.EnqueueAction(u2, move(chars[1], 5)); err != nil {
			t.Fatalf("enqueue: %v", err)
		}
		tr, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("OpenNextAction: %v", err)
		}
		if !tr.NoActionCanPay {
			t.Fatal("fixture: the indebted move must not be able to pay")
		}
		if got := s.RoundPrices()[action.BarMove]; got != 5 {
			t.Errorf("move price = %d, want 5 — the closing look freezes the bar so it settles", got)
		}
	})
}
