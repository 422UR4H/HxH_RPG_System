package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// fakeBars is a test double for service.BarStateSource. It is deliberately dumb: the
// scheduler's job is the selection, and the arithmetic is BarEconomy's, already tested.
type fakeBars struct {
	carry map[uuid.UUID]map[action.Bar]float64
	acted map[uuid.UUID]map[action.Bar][]int
}

func newFakeBars() *fakeBars {
	return &fakeBars{
		carry: map[uuid.UUID]map[action.Bar]float64{},
		acted: map[uuid.UUID]map[action.Bar][]int{},
	}
}

func (f *fakeBars) BarState(charID uuid.UUID, bar action.Bar) (float64, []int) {
	return f.carry[charID][bar], f.acted[charID][bar]
}

// act records that an action opened, on every bar it charges — which is what moves the average.
func (f *fakeBars) act(a *action.Action) {
	charID := a.GetActorID()
	if f.acted[charID] == nil {
		f.acted[charID] = map[action.Bar][]int{}
	}
	for _, bar := range a.Bars() {
		f.acted[charID][bar] = append(f.acted[charID][bar], a.SpeedOn(bar))
	}
}

func (f *fakeBars) setCarry(charID uuid.UUID, bar action.Bar, carry float64) {
	if f.carry[charID] == nil {
		f.carry[charID] = map[action.Bar]float64{}
	}
	f.carry[charID][bar] = carry
}

// attackAt builds an action-bar action that entered the round at the given speed.
func attackAt(actorID uuid.UUID, speed int) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: speed}},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
}

// moveAt builds a move-bar action that entered the round at the given speed.
func moveAt(actorID uuid.UUID, speed int) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, &action.Move{Category: enum.Dash, FinalSpeed: speed}, nil, nil, nil, nil, nil)
}

// chargeAt builds an investida: ONE action charging both bars.
func chargeAt(actorID uuid.UUID, actionSpeed, moveSpeed int) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: actionSpeed}},
		nil, &action.Move{Category: enum.Dash, FinalSpeed: moveSpeed},
		&action.Attack{}, nil, nil, nil, nil)
}

// TestRoundScheduler_CanonicalExample is the phase's done-criterion, at the level where the
// rule lives. Speeds are injected; nothing here depends on luck.
//
// docs/game/combate/barra-de-acao.md: p1 = 20, p2 = 23, p3 = 11, and p2's second roll = 17.
// Expected order p2 → p1 → p3 → p2, and closing balances p1 = +9, p3 = 0, p2 = −2.
func TestRoundScheduler_CanonicalExample(t *testing.T) {
	sch := service.RoundScheduler{}
	eco := service.BarEconomy{}
	bars := newFakeBars()
	p1, p2, p3 := uuid.New(), uuid.New(), uuid.New()

	r := round.NewRound(enum.Race)
	q := action.NewActionPriorityQueue(nil)
	q.Insert(attackAt(p1, 20))
	q.Insert(attackAt(p2, 23))
	q.Insert(attackAt(p3, 11))

	in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}

	// The master opens the first action: this is the moment the price freezes, at the
	// smallest speed on the bar among everything pending — 11.
	sch.FreezePrices(in)
	price, frozen := r.Price(action.BarAction)
	if !frozen || price != 11 {
		t.Fatalf("frozen price = (%d, %v), want (11, true)", price, frozen)
	}

	// open drains one action and records it as acted, the way MatchSession will.
	open := func() uuid.UUID {
		sch.FreezePrices(in)
		next := sch.SelectNext(in)
		if next == nil {
			t.Fatal("expected an action to open")
		}
		q.ExtractByID(next.GetID())
		r.AppendTurn(turn.NewTurn(*next))
		bars.act(next)
		return next.GetActorID()
	}

	var order []uuid.UUID
	order = append(order, open()) // p2 at 23
	order = append(order, open()) // p1 at 20
	order = append(order, open()) // p3 at 11

	// p2 sees a leftover of 23 − 11 = 12 and sends a second action, which rolls 17. The
	// right to act was granted by the leftover, BEFORE this roll.
	if !eco.IsEligible(0, []int{23}, 17, price) {
		t.Fatal("precondition: a leftover of 12 buys p2 another action")
	}
	q.Insert(attackAt(p2, 17))
	order = append(order, open()) // p2 again, now at key 9

	want := []uuid.UUID{p2, p1, p3, p2}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %v, want %v (full order %v, want %v)", i, order[i], want[i], order, want)
		}
	}

	t.Run("the round is over: nothing pending, nothing eligible", func(t *testing.T) {
		if sch.AnyEligible(in) {
			t.Error("the queue is empty, so the round must be closeable")
		}
	})

	t.Run("closing balances", func(t *testing.T) {
		cases := []struct {
			name string
			char uuid.UUID
			want float64
		}{
			{"p1 acted once at 20 and carries +9", p1, 9},
			{"p3 acted once at 11 and carries 0", p3, 0},
			{"p2 acted twice and carries the debt", p2, -2},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				carry, acted := bars.BarState(c.char, action.BarAction)
				if got := eco.CloseBalance(carry, acted, price); !closeTo(got, c.want) {
					t.Errorf("CloseBalance = %v, want %v", got, c.want)
				}
			})
		}
	})
}

func TestRoundScheduler_Gates(t *testing.T) {
	sch := service.RoundScheduler{}
	eco := service.BarEconomy{}
	p1, p2 := uuid.New(), uuid.New()

	t.Run("an action that never reaches the price sits the round out", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		q.Insert(attackAt(p1, 20))
		q.Insert(attackAt(p2, 7))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}

		sch.FreezePrices(in)
		// The price is the smallest pending speed, so p2 sits exactly ON it and DOES act.
		if p, _ := r.Price(action.BarAction); p != 7 {
			t.Fatalf("price = %d, want 7", p)
		}

		// Now a genuinely late, slow action arrives after the freeze.
		late := attackAt(p2, 3)
		q.Insert(late)
		first := sch.SelectNext(in)
		q.ExtractByID(first.GetID())
		bars.act(first)

		carry, acted := bars.BarState(late.GetActorID(), action.BarAction)
		if eco.IsEligible(carry, acted, 3, 7) {
			t.Error("a speed of 3 against a frozen price of 7 must not act")
		}
	})

	t.Run("a late but fast action jumps the rest of the queue", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		q.Insert(attackAt(p1, 12))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		q.Insert(attackAt(p2, 30))
		next := sch.SelectNext(in)
		if next.GetActorID() != p2 {
			t.Error("the late action rolled higher, so it opens next — the system never reorders what already played, but it does not hold back what has not")
		}
	})

	t.Run("standing credit rescues a roll below the price", func(t *testing.T) {
		bars := newFakeBars()
		bars.setCarry(p2, action.BarAction, 6)
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		q.Insert(attackAt(p1, 9))
		q.Insert(attackAt(p2, 5))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		if p, _ := r.Price(action.BarAction); p != 5 {
			t.Fatalf("price = %d, want 5", p)
		}
		if !sch.AnyEligible(in) {
			t.Error("carry 6 + roll 5 = 11 clears a price of 5")
		}
	})
}

func TestRoundScheduler_BarsPriceIndependently(t *testing.T) {
	sch := service.RoundScheduler{}
	bars := newFakeBars()
	p1 := uuid.New()

	r := round.NewRound(enum.Race)
	q := action.NewActionPriorityQueue(nil)
	q.Insert(attackAt(p1, 20))
	q.Insert(moveAt(p1, 6))
	in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
	sch.FreezePrices(in)

	if p, _ := r.Price(action.BarAction); p != 20 {
		t.Errorf("action price = %d, want 20 — the move action must not price the action bar", p)
	}
	if p, _ := r.Price(action.BarMove); p != 6 {
		t.Errorf("move price = %d, want 6", p)
	}
}

// TestRoundScheduler_CombinedAction covers the investida: ONE action charging both bars,
// opened once, happening at the time of its slower half.
func TestRoundScheduler_CombinedAction(t *testing.T) {
	sch := service.RoundScheduler{}
	p1, p2 := uuid.New(), uuid.New()

	t.Run("it is scheduled at its slower half, not its faster one", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		// p1 is quick of hand (25) and slow of foot (5). The blow does not get to jump the
		// queue at 25: the investida happens when the feet get there.
		q.Insert(chargeAt(p1, 25, 5))
		q.Insert(attackAt(p2, 12))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		next := sch.SelectNext(in)
		if next.GetActorID() != p2 {
			t.Error("the investida is keyed on min(25, 5) = 5, so the plain attack at 12 goes first")
		}
	})

	t.Run("it opens ONCE and charges both bars", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		a := chargeAt(p1, 25, 5)
		q.Insert(a)
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		next := sch.SelectNext(in)
		if next == nil || next.GetID() != a.GetID() {
			t.Fatal("the investida must open")
		}
		q.ExtractByID(next.GetID())
		bars.act(next)

		if sch.SelectNext(in) != nil {
			t.Error("one action, one opening — there is no second half left in the queue")
		}
		if _, acted := bars.BarState(p1, action.BarAction); len(acted) != 1 {
			t.Error("the action bar must have been charged")
		}
		if _, acted := bars.BarState(p1, action.BarMove); len(acted) != 1 {
			t.Error("the move bar must have been charged too — it pays both")
		}
	})

	t.Run("it needs both gates: a spent action bar keeps the whole investida out", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		// Two plain actions price both bars at 10, then the investida arrives able to pay
		// the movement (12) but not the blow (4).
		q.Insert(attackAt(p2, 10))
		q.Insert(moveAt(p2, 10))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		q.Insert(chargeAt(p1, 4, 12))
		for _, slot := range sch.ProjectOrder(in) {
			if slot.ActorID == p1 {
				t.Error("the investida charges both bars, so it must clear both gates — riding the move bar into a free attack would void the economy")
			}
		}
	})
}

func TestRoundScheduler_FreeRoundHasNoEconomy(t *testing.T) {
	sch := service.RoundScheduler{}
	bars := newFakeBars()
	p1, p2 := uuid.New(), uuid.New()

	r := round.NewRound(enum.Free)
	q := action.NewActionPriorityQueue(nil)
	q.Insert(attackAt(p1, 4))
	q.Insert(attackAt(p2, 9))
	in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}

	sch.FreezePrices(in)
	if _, frozen := r.Price(action.BarAction); frozen {
		t.Error("a Free round has no price at all — the economy is Race's, and Free's rule is not written yet")
	}

	next := sch.SelectNext(in)
	if next.GetActorID() != p2 {
		t.Error("with no economy the rolled speed is the whole order")
	}
	if !sch.AnyEligible(in) {
		t.Error("a Free round never auto-closes while anything is pending")
	}
}

func TestRoundScheduler_ProjectOrder(t *testing.T) {
	sch := service.RoundScheduler{}
	bars := newFakeBars()
	p1, p2, p3 := uuid.New(), uuid.New(), uuid.New()

	r := round.NewRound(enum.Race)
	q := action.NewActionPriorityQueue(nil)
	q.Insert(attackAt(p1, 20))
	q.Insert(attackAt(p2, 23))
	q.Insert(attackAt(p3, 11))
	in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
	sch.FreezePrices(in)

	slots := sch.ProjectOrder(in)
	if len(slots) != 3 {
		t.Fatalf("slots = %d, want 3", len(slots))
	}
	want := []uuid.UUID{p2, p1, p3}
	for i := range want {
		if slots[i].ActorID != want[i] {
			t.Errorf("slot[%d] = %v, want %v — highest key first", i, slots[i].ActorID, want[i])
		}
	}
	if !closeTo(slots[0].Key, 23) {
		t.Errorf("slot[0].Key = %v, want 23", slots[0].Key)
	}
}

// TestRoundScheduler_ProjectOrderMatchesWhatIsPlayed pins the promise combat-engine.md makes
// about the general bar: what ProjectOrder publishes is what SelectNext then plays.
//
// The canonical numbers, with p2's second action already pending: price 11, p2 at 23 and 17,
// p1 at 20, p3 at 11. The published order must be p2 → p1 → p3 → p2, with keys 23, 20, 11 and
// 9 — NOT p2 → p1 → p2 → p3, which is what scoring every entry against the same untouched
// `acted` produces, because it keys p2's second action at 17 as though the first had not
// opened. Position 1 is right either way; everything after it is not.
func TestRoundScheduler_ProjectOrderMatchesWhatIsPlayed(t *testing.T) {
	sch := service.RoundScheduler{}
	bars := newFakeBars()
	p1, p2, p3 := uuid.New(), uuid.New(), uuid.New()

	r := round.NewRound(enum.Race)
	q := action.NewActionPriorityQueue(nil)
	q.Insert(attackAt(p1, 20))
	q.Insert(attackAt(p2, 23))
	q.Insert(attackAt(p3, 11))
	q.Insert(attackAt(p2, 17))
	in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}

	sch.FreezePrices(in)
	if price, frozen := r.Price(action.BarAction); !frozen || price != 11 {
		t.Fatalf("frozen price = (%d, %v), want (11, true)", price, frozen)
	}

	wantOrder := []uuid.UUID{p2, p1, p3, p2}
	wantKeys := []float64{23, 20, 11, 9}

	t.Run("the projection walks the round forward", func(t *testing.T) {
		slots := sch.ProjectOrder(in)
		if len(slots) != len(wantOrder) {
			t.Fatalf("slots = %d, want %d", len(slots), len(wantOrder))
		}
		for i := range wantOrder {
			if slots[i].ActorID != wantOrder[i] {
				t.Errorf("slot[%d] actor = %v, want %v", i, slots[i].ActorID, wantOrder[i])
			}
			if !closeTo(slots[i].Key, wantKeys[i]) {
				t.Errorf("slot[%d].Key = %v, want %v", i, slots[i].Key, wantKeys[i])
			}
		}
	})

	t.Run("and the round is played in exactly that order", func(t *testing.T) {
		projected := sch.ProjectOrder(in)

		var played []uuid.UUID
		var playedKeys []float64
		for {
			// The key SelectNext scores against is read before the action opens — the same
			// instant ProjectOrder read it in its own walk.
			slots := sch.ProjectOrder(in)
			next := sch.SelectNext(in)
			if next == nil {
				break
			}
			played = append(played, next.GetActorID())
			playedKeys = append(playedKeys, slots[0].Key)
			q.ExtractByID(next.GetID())
			r.AppendTurn(turn.NewTurn(*next))
			bars.act(next)
		}

		if len(played) != len(projected) {
			t.Fatalf("played %d actions, projected %d", len(played), len(projected))
		}
		for i := range played {
			if played[i] != projected[i].ActorID {
				t.Errorf("position %d: played %v, projected %v", i, played[i], projected[i].ActorID)
			}
			if !closeTo(playedKeys[i], projected[i].Key) {
				t.Errorf("position %d: played key %v, projected %v", i, playedKeys[i], projected[i].Key)
			}
		}
	})
}

func TestRoundScheduler_BestPendingFor(t *testing.T) {
	t.Run("picks the pending action that would have opened first for that character", func(t *testing.T) {
		actor, other := uuid.New(), uuid.New()
		slow := attackAt(actor, 8)
		fast := attackAt(actor, 20)
		theirs := attackAt(other, 30)
		q := action.NewActionPriorityQueue(&[]*action.Action{slow, theirs, fast})
		r := round.NewRound(enum.Race)
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: newFakeBars()}

		got := service.RoundScheduler{}.BestPendingFor(in, actor, action.BarAction)
		if got == nil || got.GetID() != fast.GetID() {
			t.Fatal("the reaction consumes the moment that was about to be spent, not the leftovers")
		}
	})

	t.Run("answers nil when that character has nothing pending on that bar", func(t *testing.T) {
		actor := uuid.New()
		q := action.NewActionPriorityQueue(nil)
		r := round.NewRound(enum.Race)
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: newFakeBars()}
		if got := (service.RoundScheduler{}).BestPendingFor(in, actor, action.BarAction); got != nil {
			t.Fatal("nothing pending means the reaction simply becomes the action")
		}
	})

	// keyOnBar used to answer (0, false) for an ineligible candidate, discarding its real key.
	// BestPendingFor deliberately ignores the "false" (the gate is not consulted here — see
	// its own doc comment), but it must not end up trusting the discarded literal 0 either: a
	// real key can be negative (a heavily-penalised speed pushes a bar's frozen price
	// negative too), and 0 used to outrank a genuinely negative ELIGIBLE key, consuming the
	// wrong action.
	t.Run("an ineligible candidate's real (negative) key must not be replaced by a 0 that outranks an eligible one", func(t *testing.T) {
		actor := uuid.New()
		r := round.NewRound(enum.Race)
		r.FreezePrice(action.BarAction, -20)

		// First-action rule: eligible iff carry(0) + speed >= price(-20).
		eligible := attackAt(actor, -15)   // key = -15, -15 >= -20: eligible
		ineligible := attackAt(actor, -25) // key = -25, -25 <  -20: ineligible, but its real
		// key (-25) is still LOWER than the eligible action's (-15) — the eligible one must win.

		for _, order := range [][2]*action.Action{{ineligible, eligible}, {eligible, ineligible}} {
			q := action.NewActionPriorityQueue(&[]*action.Action{order[0], order[1]})
			in := service.ScheduleInput{Queue: &q, Round: r, Bars: newFakeBars()}

			got := service.RoundScheduler{}.BestPendingFor(in, actor, action.BarAction)
			if got == nil || got.GetID() != eligible.GetID() {
				t.Fatalf("BestPendingFor = %v, want the eligible action (key -15), not the ineligible one masquerading as key 0 (real key -25)", got)
			}
		}
	})
}
