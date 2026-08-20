package match

import "github.com/google/uuid"

// Dimension is WHAT a Modifier modifies.
//
// The system has more than one kind of accumulated reserve, and they are not
// interchangeable: the duel reserve (repel/parry) moves actionSpeed, while the closed
// dodge's reserve moves the dodge itself. Without this field the ledger cannot hold both,
// and a caller would have to know which entries it was allowed to read — which is exactly
// how the "always an actionSpeed adjustment" over-generalization happened.
type Dimension string

const (
	DimActionSpeed Dimension = "action_speed"
	DimDodge       Dimension = "dodge"
)

// Lifetime is when a Modifier stops applying.
//
// LifetimeNextTurn is the rung the repel bonus needs. A bonus created during turn N with
// LifetimeEndOfTurn would die at the close of N itself — but the bonus is earned in N and
// spent in N+1. It is implemented by demotion in AdvanceTurn, not by a clock.
type Lifetime string

const (
	LifetimeEndOfTurn  Lifetime = "end_of_turn"
	LifetimeNextTurn   Lifetime = "next_turn"
	LifetimeEndOfRound Lifetime = "end_of_round"
)

// Source records who created a Modifier. The system generates disadvantage on its own
// (swapping a declared action, converting an action into a reaction); the master grants
// or cancels advantage by hand. Keeping them apart is what lets the master cancel the
// system's disadvantage without either one overwriting the other, and it is what the
// audit trail reads to tell the two apart.
type Source string

const (
	SourceSystem Source = "system"
	SourceMaster Source = "master"
)

// ScopeKind is the shape of a Scope. It is not used directly — build a Scope through one of
// the three constructors, so an unset ID can never be mistaken for "anyone".
type ScopeKind string

const (
	scopeAnyone ScopeKind = "anyone"
	scopeOnly   ScopeKind = "only"
	scopeAllBut ScopeKind = "all_but"
)

// Scope is WHOM a Modifier counts against, and it has three forms rather than two.
//
// "All but X" is not a convenience: it is the only shape that expresses the closed dodge's
// reserve — the dodge a character did not need to spend, kept for whoever comes at them from
// outside the duel. A nil-or-one-target pointer cannot say it.
type Scope struct {
	kind ScopeKind
	id   uuid.UUID
}

func ScopeAnyone() Scope             { return Scope{kind: scopeAnyone} }
func ScopeOnly(id uuid.UUID) Scope   { return Scope{kind: scopeOnly, id: id} }
func ScopeAllBut(id uuid.UUID) Scope { return Scope{kind: scopeAllBut, id: id} }

// AppliesTo reports whether this scope counts for a roll made against targetID.
//
// A roll with no named target is "nobody in particular": a targeted bonus does not reach it,
// and an all-but exclusion does not exclude it.
func (s Scope) AppliesTo(targetID *uuid.UUID) bool {
	switch s.kind {
	case scopeOnly:
		return targetID != nil && *targetID == s.id
	case scopeAllBut:
		return targetID == nil || *targetID != s.id
	default:
		return true
	}
}

// Modifier is one accumulated bonus or penalty carried by a character.
//
// Amount and Bias are different currencies and never substitute for each other:
// Amount is a flat adjustment to the total; Bias is advantage/disadvantage on the dice
// (−1/0/+1, accumulating), which is a change in how the roll is made, not a number that
// can be added to it.
type Modifier struct {
	Amount    int
	Bias      int
	Applies   Dimension
	Source    Source
	Against   Scope
	ExpiresAt Lifetime
	Reason    string // surfaced in the Action History and in the audit trail
}

// ModifierLedger is the accumulated difference a character carries: the bonuses and
// penalties that the repel ladder, the closed dodge, the system and the master have piled on.
//
// This is deliberately NOT RollCondition. RollCondition is the master's struct for one
// roll — their dice bias and their manual adjustment. The ledger is the character's
// standing state across turns and rounds.
type ModifierLedger struct {
	modifiers []Modifier
}

func NewModifierLedger() ModifierLedger { return ModifierLedger{} }

func (l *ModifierLedger) Add(m Modifier) { l.modifiers = append(l.modifiers, m) }

// All returns a copy, so a caller iterating the ledger cannot mutate it.
func (l *ModifierLedger) All() []Modifier {
	out := make([]Modifier, len(l.modifiers))
	copy(out, l.modifiers)
	return out
}

// TotalAmount sums the flat adjustments on one dimension that apply against targetID.
func (l *ModifierLedger) TotalAmount(dim Dimension, targetID *uuid.UUID) int {
	total := 0
	for _, m := range l.modifiers {
		if m.Applies == dim && m.Against.AppliesTo(targetID) {
			total += m.Amount
		}
	}
	return total
}

// TotalBias sums the dice biases on one dimension that apply against targetID. Advantage and
// disadvantage accumulate and can cancel each other out.
func (l *ModifierLedger) TotalBias(dim Dimension, targetID *uuid.UUID) int {
	total := 0
	for _, m := range l.modifiers {
		if m.Applies == dim && m.Against.AppliesTo(targetID) {
			total += m.Bias
		}
	}
	return total
}

// Expire drops every modifier whose validity ended at lifetime.
func (l *ModifierLedger) Expire(lifetime Lifetime) {
	kept := l.modifiers[:0]
	for _, m := range l.modifiers {
		if m.ExpiresAt != lifetime {
			kept = append(kept, m)
		}
	}
	l.modifiers = kept
}

// AdvanceTurn moves the ledger one turn forward: what was scoped to this turn dies, and what
// was earned FOR the next turn becomes this-turn's.
//
// This is how "next turn" is implemented — a demotion, in one pass, with no clock. A bonus
// created during turn N is demoted at N's close and dropped at N+1's, so it is live for
// exactly one turn.
func (l *ModifierLedger) AdvanceTurn() {
	kept := l.modifiers[:0]
	for _, m := range l.modifiers {
		switch m.ExpiresAt {
		case LifetimeEndOfTurn:
			continue
		case LifetimeNextTurn:
			m.ExpiresAt = LifetimeEndOfTurn
		}
		kept = append(kept, m)
	}
	l.modifiers = kept
}
