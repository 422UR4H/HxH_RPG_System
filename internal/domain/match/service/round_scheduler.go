package service

import (
	"math"
	"sort"

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
	var best *action.Action
	var bestKey float64
	for _, a := range in.Queue.All() {
		key, ok := rs.keyOf(in, a)
		if !ok {
			continue
		}
		// Strictly greater, so a tie goes to whoever was inserted first.
		if best == nil || key > bestKey {
			best, bestKey = a, key
		}
	}
	return best
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

// ProjectOrder returns every pending action that passes its gate, highest key first. It is
// SelectNext generalised from "the next one" to "all of them, in order", which is what the
// general bar shows — including a character who is going to act more than once.
func (rs RoundScheduler) ProjectOrder(in ScheduleInput) []OrderSlot {
	var slots []OrderSlot
	for _, a := range in.Queue.All() {
		key, ok := rs.keyOf(in, a)
		if !ok {
			continue
		}
		slots = append(slots, OrderSlot{ActorID: a.GetActorID(), Bars: a.Bars(), Key: key})
	}
	// Stable, so equal keys keep insertion order — the same tie-break SelectNext uses.
	sort.SliceStable(slots, func(i, j int) bool { return slots[i].Key > slots[j].Key })
	return slots
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
