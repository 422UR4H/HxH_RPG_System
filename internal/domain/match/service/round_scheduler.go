package service

import (
	"math"
	"slices"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/google/uuid"
)

// BarStateSource hands the scheduler the per-character bar state it needs. Implemented by
// *matchsession.MatchSession, which owns the CharacterStatus map; declared here so the domain
// service never imports the session.
//
// carry is the balance that crossed over from the previous round. acted is the ordered list
// of speeds this character has ALREADY OPENED on that bar this round — never the pending ones.
type BarStateSource interface {
	BarState(charID uuid.UUID, bar action.Bar) (carry float64, acted []int)
}

// ScheduleInput is everything one scheduling decision reads.
type ScheduleInput struct {
	Queue *action.PriorityQueue
	Round *round.Round
	Bars  BarStateSource
}

// OrderSlot is one entry of the projected order: who acts, which clocks it charges, and where
// in the round it lands. It deliberately carries no action identity — the queue is secret, the
// order is public.
type OrderSlot struct {
	ActorID uuid.UUID
	Bars    []action.Bar
	Key     float64
}

// RoundScheduler decides which pending action opens next and when the round is over. It is a
// stateless domain service: all the state lives in the Round and behind BarStateSource.
type RoundScheduler struct{}

// FreezePrices fixes the price of every bar that has pending work and has not priced yet.
//
// The price of a bar is the smallest speed among the actions pending on it at the moment its
// first action is chosen. It is frozen per bar, not per round: a bar whose first action only
// shows up mid-round prices then, from what is pending then. Once frozen it never moves — a
// slower action arriving afterwards does not re-price the round, it simply may not reach the
// price and sits the round out with its full roll intact.
//
// A Free round is left untouched: it has no price, no average and no carry-over. That economy
// belongs to Race, and Free's own rule has not been designed yet.
func (RoundScheduler) FreezePrices(in ScheduleInput) {
	if in.Round == nil || in.Queue == nil || in.Round.GetMode() != enum.Race {
		return
	}
	lowest := map[action.Bar]int{}
	for _, a := range in.Queue.All() {
		for _, bar := range a.Bars() {
			if _, frozen := in.Round.Price(bar); frozen {
				continue
			}
			speed := a.SpeedOn(bar)
			if current, seen := lowest[bar]; !seen || speed < current {
				lowest[bar] = speed
			}
		}
	}
	for bar, price := range lowest {
		in.Round.FreezePrice(bar, price)
	}
}

// SelectNext returns the pending action that opens next, or nil when nothing passes its gate.
//
// Nil in a Race round is the round-close signal: everything still queued belongs to the next
// round, carrying the roll it already made.
func (rs RoundScheduler) SelectNext(in ScheduleInput) *action.Action {
	if in.Queue == nil {
		return nil
	}
	candidates := in.Queue.All()
	idx, _ := rs.best(in, candidates)
	if idx < 0 {
		return nil
	}
	return candidates[idx]
}

// best is the single scoring pass: the highest-keyed candidate that passes its gate, and where
// it sits. It answers -1 when nothing does.
//
// SelectNext and ProjectOrder both go through it, and that is the point — the order the table
// plays and the order the table is shown are computed by the same code, so they cannot drift
// apart.
func (rs RoundScheduler) best(in ScheduleInput, candidates []*action.Action) (int, float64) {
	bestIdx, bestKey := -1, 0.0
	for i, a := range candidates {
		key, ok := rs.keyOf(in, a)
		if !ok {
			continue
		}
		// Strictly greater, so a tie goes to whoever was inserted first.
		if bestIdx == -1 || key > bestKey {
			bestIdx, bestKey = i, key
		}
	}
	return bestIdx, bestKey
}

// AnyEligible reports whether any pending action passes the gate that applies to it. Its
// negation is the round-close predicate: the round ends when nothing pending can still pay.
//
// It is deliberately NOT "when the bars run out". A bar does not zero — it ends wherever it
// ends, debt included. And it is not about the key either: an action whose key fell below the
// price still happens, as long as it had already been granted.
func (rs RoundScheduler) AnyEligible(in ScheduleInput) bool {
	return rs.SelectNext(in) != nil
}

// ProjectOrder returns the round as it is going to be played: every pending action that can
// still pay, in the order the master will open them. It is what the general bar publishes.
//
// It SIMULATES THE ROUND FORWARD rather than scoring everything once. Scoring once is wrong
// past the first slot, because a character's second pending action is keyed and gated against
// the state their FIRST one leaves behind — the average slides as soon as that first action
// opens. In the canonical example (price 11; p2 pending at 23 and 17, p1 at 20, p3 at 11) a
// single pass publishes p2 → p1 → p2 → p3, keying p2's second action at 17 as if it were their
// first; the table then plays p2 → p1 → p3 → p2, because that second key drops to
// mean(23,17) − 11 = 9 the moment the first one opens. Position 1 was the only one guaranteed
// right.
//
// So it walks a copy of the bar state, picks the best candidate exactly as SelectNext would,
// records it as opened in the copy, drops it from the candidates, and repeats until nothing
// passes its gate. Nothing real is mutated: the queue hands out a copy and the bar state is
// read through an overlay.
func (rs RoundScheduler) ProjectOrder(in ScheduleInput) []OrderSlot {
	if in.Queue == nil {
		return nil
	}
	sim := &simulatedBars{base: in.Bars, extra: map[simKey][]int{}}
	simIn := in
	simIn.Bars = sim

	remaining := in.Queue.All() // already a copy — reshaping it cannot touch the queue
	var slots []OrderSlot
	for len(remaining) > 0 {
		idx, key := rs.best(simIn, remaining)
		if idx < 0 {
			break
		}
		a := remaining[idx]
		slots = append(slots, OrderSlot{ActorID: a.GetActorID(), Bars: a.Bars(), Key: key})
		sim.open(a)
		remaining = append(remaining[:idx], remaining[idx+1:]...)
	}
	return slots
}

// simKey addresses one character's one clock.
type simKey struct {
	char uuid.UUID
	bar  action.Bar
}

// simulatedBars is a read-only overlay on the real bar state, used only while ProjectOrder
// walks the round forward. It never writes through: extra holds the speeds the simulation has
// opened, appended to whatever the session actually reports.
//
// It keeps RoundScheduler stateless — it is created per call and dies with it.
type simulatedBars struct {
	base  BarStateSource
	extra map[simKey][]int
}

func (s *simulatedBars) BarState(charID uuid.UUID, bar action.Bar) (float64, []int) {
	var carry float64
	var acted []int
	if s.base != nil {
		carry, acted = s.base.BarState(charID, bar)
	}
	extra := s.extra[simKey{char: charID, bar: bar}]
	if len(extra) == 0 {
		return carry, acted
	}
	out := make([]int, 0, len(acted)+len(extra))
	out = append(out, acted...)
	out = append(out, extra...)
	return carry, out
}

// open records, inside the simulation only, that an action opened — on every bar it charges,
// because a combined action moves both averages.
func (s *simulatedBars) open(a *action.Action) {
	for _, bar := range a.Bars() {
		k := simKey{char: a.GetActorID(), bar: bar}
		s.extra[k] = append(s.extra[k], a.SpeedOn(bar))
	}
}

// keyOf returns where an action sits in the order, and whether it may open at all.
//
// An action charging BOTH bars — an investida — is scheduled at its SLOWER half and must clear
// the gate on BOTH. The higher key acts first, so the slower half is the smaller key, hence
// the min. And it pays both bars, so it has to be able to afford both: letting it through on
// the move bar alone would hand a character with a spent action bar a free attack.
func (rs RoundScheduler) keyOf(in ScheduleInput, a *action.Action) (float64, bool) {
	key := math.Inf(1)
	for _, bar := range a.Bars() {
		barKey, ok := rs.keyOnBar(in, a, bar)
		if !ok {
			return 0, false
		}
		if barKey < key {
			key = barKey
		}
	}
	if math.IsInf(key, 1) {
		return 0, false
	}
	return key, true
}

// BestPendingFor returns the pending action that would open first for one character on one
// bar, or nil when they have none there.
//
// It exists for the reaction: reacting actively consumes the action you had queued, and the
// one it consumes is the moment that was about to be spent — the best key. Scoring reuses
// keyOnBar, so this invents no criterion the round did not already have.
//
// The gate is deliberately NOT consulted. An action that could not have opened this round is
// still the one whose moment the reaction takes; refusing to consume it would let a character
// react AND keep a queued action, which is the trade the Disadvantage is paying for.
func (rs RoundScheduler) BestPendingFor(in ScheduleInput, actorID uuid.UUID, bar action.Bar) *action.Action {
	if in.Queue == nil {
		return nil
	}
	var best *action.Action
	bestKey := 0.0
	for _, a := range in.Queue.All() {
		if a.GetActorID() != actorID || !slices.Contains(a.Bars(), bar) {
			continue
		}
		key, _ := rs.keyOnBar(in, a, bar)
		if best == nil || key > bestKey {
			best, bestKey = a, key
		}
	}
	return best
}

// keyOnBar scores one action against one of the bars it charges.
func (rs RoundScheduler) keyOnBar(in ScheduleInput, a *action.Action, bar action.Bar) (float64, bool) {
	speed := a.SpeedOn(bar)

	var carry float64
	var acted []int
	if in.Bars != nil {
		carry, acted = in.Bars.BarState(a.GetActorID(), bar)
	}

	price, frozen := 0, false
	if in.Round != nil {
		price, frozen = in.Round.Price(bar)
	}
	if !frozen {
		// No economy on this bar — a Free round, or a bar that has not priced yet. The rolled
		// speed is the whole order and nothing gates.
		return carry + float64(speed), true
	}

	eco := BarEconomy{}
	if !eco.IsEligible(carry, acted, speed, price) {
		return 0, false
	}
	return eco.Key(carry, acted, speed, price), true
}
