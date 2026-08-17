package match

import "github.com/google/uuid"

// Source records who created a Modifier. The system generates disadvantage on its own
// (swapping a declared action, converting an action into a reaction); the master grants
// or cancels advantage by hand. Keeping them apart is what lets the master cancel the
// system's disadvantage without either one overwriting the other — and it is what the
// audit trail reads to tell the two apart.
type Source string

const (
	SourceSystem Source = "system"
	SourceMaster Source = "master"
)

// Scope is when a Modifier stops applying.
type Scope string

const (
	ScopeEndOfTurn  Scope = "end_of_turn"
	ScopeEndOfRound Scope = "end_of_round"
)

// Modifier is one accumulated bonus or penalty carried by a character.
//
// Amount and Bias are different currencies and never substitute for each other:
// Amount is a flat adjustment to the total; Bias is advantage/disadvantage on the dice
// (−1/0/+1, accumulating), which is a change in how the roll is made, not a number that
// can be added to it.
type Modifier struct {
	Amount    int
	Bias      int
	Source    Source
	AgainstID *uuid.UUID // nil = applies against anyone
	ExpiresAt Scope
	Reason    string // surfaced in the Action History and in the audit trail
}

// AppliesTo reports whether m counts for a roll made against targetID.
// A targeted modifier never applies to a roll with no named target.
func (m Modifier) AppliesTo(targetID *uuid.UUID) bool {
	if m.AgainstID == nil {
		return true
	}
	if targetID == nil {
		return false
	}
	return *m.AgainstID == *targetID
}

// ModifierLedger is the accumulated difference a character carries: the bonuses and
// penalties that the repel ladder, the system, and the master have piled on.
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

// TotalAmount sums the flat adjustments that apply against targetID.
func (l *ModifierLedger) TotalAmount(targetID *uuid.UUID) int {
	total := 0
	for _, m := range l.modifiers {
		if m.AppliesTo(targetID) {
			total += m.Amount
		}
	}
	return total
}

// TotalBias sums the dice biases that apply against targetID. Advantage and
// disadvantage accumulate and can cancel each other out.
func (l *ModifierLedger) TotalBias(targetID *uuid.UUID) int {
	total := 0
	for _, m := range l.modifiers {
		if m.AppliesTo(targetID) {
			total += m.Bias
		}
	}
	return total
}

// Expire drops every modifier whose validity ended at scope.
func (l *ModifierLedger) Expire(scope Scope) {
	kept := l.modifiers[:0]
	for _, m := range l.modifiers {
		if m.ExpiresAt != scope {
			kept = append(kept, m)
		}
	}
	l.modifiers = kept
}
