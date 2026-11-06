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
	// Provisional is the price an UNFROZEN bar is going to freeze at. When set, keyOnBar scores
	// (key and gate) against it instead of treating the bar as price-less. Projections set it, and
	// so does OpenNextAction: it selects against the price it will freeze only once an action opens.
	Provisional map[action.Bar]int
}

// OrderSlot is one entry of the projected order: who acts, which clocks it charges, and where
// in the round it lands. It deliberately carries no action identity — the queue is secret, the
// order is public.
type OrderSlot struct {
	ActorID uuid.UUID
	Bars    []action.Bar
	// Speeds is the action's speed on each bar it charges. Internal: the wire type is
	// BarSlotPayload, which does not carry it.
	Speeds map[action.Bar]int
	Key    float64
}

// RoundScheduler decides which pending action opens next and when the round is over. It is a
// stateless domain service: all the state lives in the Round and behind BarStateSource.
type RoundScheduler struct{}

// ProvisionalPrices is the price each not-yet-frozen bar WILL freeze at: the smallest speed
// among the queued actions charging it. Bars already frozen in the round, and bars without
// queued work, are absent. It is Race only — nil for any other mode (or a missing round/queue).
func (RoundScheduler) ProvisionalPrices(in ScheduleInput) map[action.Bar]int {
	if in.Round == nil || in.Queue == nil || in.Round.GetMode() != enum.Race {
		return nil
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
	return lowest
}

// FreezeAt freezes the given prices on the round — what the session got from
// ProvisionalPrices. The price of a bar is the smallest speed among its pending actions when
// its first action is chosen; it is frozen per bar, not per round, and never moves afterwards:
// a slower action arriving later simply may not reach it and sits the round out with its roll
// intact. A Free round has no price (ProvisionalPrices is nil there), so nothing freezes.
//
// The session computes ProvisionalPrices before an action opens (the action about to open
// still counts in the minimum) and freezes them with this only once something actually
// opened: the price is public from the first opened action, never from a command that failed.
func (RoundScheduler) FreezeAt(r *round.Round, prices map[action.Bar]int) {
	for bar, price := range prices {
		r.FreezePrice(bar, price)
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
// SelectNext and ProjectRound both go through it, and that is the point — the order the table
// plays and the order the table is shown are computed by the same code, so they cannot drift
// apart.
func (rs RoundScheduler) best(in ScheduleInput, candidates []*action.Action) (int, float64) {
	bestIdx, bestKey := -1, 0.0
	for i, a := range candidates {
		// Um movimento empilhado num pendente não é candidato (D4). "Pendente" é candidates: a
		// fila em SelectNext, o que a simulação ainda não escolheu em ProjectRound — então um
		// anterior adiado adia o empilhado junto, e um anterior escolhido o libera na hora.
		if a.IsBlockedIn(candidates) {
			continue
		}
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

// RoundProjection is the round as it is going to be played (Order) plus every queued action
// that will not open in it (Deferred, in queue order, Key zero).
type RoundProjection struct {
	Order    []OrderSlot
	Deferred []OrderSlot
}

// ProjectOrder returns the projected order alone; see ProjectRound.
//
// No production code calls it: the bars (matchsession.BarsView) read ProjectRound, which also
// reports the deferred actions. It stays as the guard round_scheduler_test.go pins — that the
// projection equals the order the round is actually played in, slot by slot.
func (rs RoundScheduler) ProjectOrder(in ScheduleInput) []OrderSlot {
	return rs.ProjectRound(in).Order
}

// ProjectRound returns the round as it is going to be played: every pending action that can
// still pay, in the order the master will open them, and the ones that cannot (Deferred).
// It is what the general bar publishes.
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
//
// Before a bar's first open it has no frozen price. The projection scores it against the
// PROVISIONAL price (the one the first open will freeze), unless the caller already set
// in.Provisional, so the preview shows the gate the table will actually face. The round itself
// is never frozen by projecting.
func (rs RoundScheduler) ProjectRound(in ScheduleInput) RoundProjection {
	if in.Queue == nil {
		return RoundProjection{}
	}
	sim := &simulatedBars{base: in.Bars, extra: map[simKey][]int{}}
	simIn := in
	simIn.Bars = sim
	if simIn.Provisional == nil {
		simIn.Provisional = rs.ProvisionalPrices(in)
	}

	remaining := in.Queue.All() // already a copy — reshaping it cannot touch the queue
	var proj RoundProjection
	for len(remaining) > 0 {
		idx, key := rs.best(simIn, remaining)
		if idx < 0 {
			break
		}
		a := remaining[idx]
		slot := slotOf(a)
		slot.Key = key
		proj.Order = append(proj.Order, slot)
		sim.open(a)
		remaining = append(remaining[:idx], remaining[idx+1:]...)
	}
	for _, a := range remaining {
		proj.Deferred = append(proj.Deferred, slotOf(a))
	}
	return proj
}

// slotOf describes an action as a slot, without a key.
func slotOf(a *action.Action) OrderSlot {
	speeds := map[action.Bar]int{}
	for _, bar := range a.Bars() {
		speeds[bar] = a.SpeedOn(bar)
	}
	return OrderSlot{ActorID: a.GetActorID(), Bars: a.Bars(), Speeds: speeds}
}

// simKey addresses one character's one clock.
type simKey struct {
	char uuid.UUID
	bar  action.Bar
}

// simulatedBars is a read-only overlay on the real bar state, used only while ProjectRound
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
//
// Só os NÃO bloqueados contam (spec 2026-11-06 D6): a cabeça de uma cadeia de movimentos é "o
// momento prestes a ser gasto", não o empilhado mais rápido. Quando a barra não tem nenhum
// candidato desbloqueado, vale qualquer pendente nela, como antes.
func (rs RoundScheduler) BestPendingFor(in ScheduleInput, actorID uuid.UUID, bar action.Bar) *action.Action {
	if in.Queue == nil {
		return nil
	}
	pending := in.Queue.All()
	if best := rs.bestPendingFor(in, pending, actorID, bar, true); best != nil {
		return best
	}
	return rs.bestPendingFor(in, pending, actorID, bar, false)
}

// bestPendingFor é a passada de BestPendingFor; unblockedOnly pula os bloqueados.
func (rs RoundScheduler) bestPendingFor(in ScheduleInput, pending []*action.Action, actorID uuid.UUID, bar action.Bar, unblockedOnly bool) *action.Action {
	var best *action.Action
	bestKey := math.Inf(-1)
	for _, a := range pending {
		if a.GetActorID() != actorID || !slices.Contains(a.Bars(), bar) {
			continue
		}
		if unblockedOnly && a.IsBlockedIn(pending) {
			continue
		}
		// The gate (ok) is deliberately ignored — an action that could not have opened this
		// round is still the one whose moment the reaction takes, per this function's own doc
		// comment. keyOnBar now hands back the REAL key even when ineligible (a real key can be
		// negative — a deep-in-debt carry, see BarEconomy.Key), so there is no more 0 sentinel
		// to distrust here. bestKey still starts at -Inf rather than 0, on the same principle:
		// nothing scored here should be able to lose to an arbitrary constant.
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
		// Not frozen yet: a projection before the bar's first open supplies the price it WILL
		// freeze at, and the bar is scored (key and gate) against it.
		provisional, ok := in.Provisional[bar]
		if !ok {
			// No economy on this bar — a Free round, or a bar that has not priced yet. The
			// rolled speed is the whole order and nothing gates.
			return carry + float64(speed), true
		}
		price = provisional
	}

	eco := BarEconomy{}
	// The key is computed either way. keyOf (via best/SelectNext/ProjectOrder) discards it the
	// instant ok is false and never looks at the value, so a 0 sentinel there cost nothing —
	// but BestPendingFor deliberately keeps scoring an ineligible candidate (see its own
	// comment: the gate is not consulted for a reaction's consumption), and a literal 0 in
	// place of the real key let an ineligible action's sentinel outrank an eligible one's
	// genuinely negative key. Returning the real key regardless of ok closes that without
	// touching what ok means to either caller.
	key := eco.Key(carry, acted, speed, price)
	if !eco.IsEligible(carry, acted, speed, price) {
		return key, false
	}
	return key, true
}
