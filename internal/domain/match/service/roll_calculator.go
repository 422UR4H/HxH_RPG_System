package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// RollInput is everything Derive needs besides the dice themselves.
//
// Condition is the master's struct for this one roll: their dice bias and their manual
// adjustment. Ledger is the character's standing accumulated difference. They are summed
// here, never merged upstream, so the master can cancel the system's disadvantage without
// either one overwriting the other.
type RollInput struct {
	SkillName  string
	SkillValue int                   // already resolved from the sheet by the caller
	Passive    bool                  // take the set's average instead of rolling
	Condition  *action.RollCondition // master-owned; nil = neutral
	// Ledger is character-owned; nil = empty. WHAT each entry modifies is now the entry's own
	// business (match.Modifier.Applies), read against Dimension below — the caller no longer
	// decides by passing nil. An earlier comment here claimed the accumulated difference was
	// "always an actionSpeed adjustment, never a hit adjustment": that was true of the duel
	// reserve and false as a general law. The closed dodge's reserve modifies the dodge.
	Ledger *match.ModifierLedger
	// Dimension is which reserve this roll reads out of the ledger. The zero value reads
	// nothing, which is the right default for a test no reserve applies to.
	Dimension match.Dimension
	// SystemBias is advantage/disadvantage the ENGINE imposes on this one roll — the
	// action→reaction conversion is the first one. It is not the ledger (that is the
	// character's, and lasts until it expires) and not RollCondition (that is the master's).
	// Three origins, three homes, none overwriting another.
	SystemBias int
	AgainstID  *uuid.UUID // whom the roll is against; nil = nobody in particular
}

// RollOutcome is the derived result of one test. The individual dice survive because a
// critical is the combination, not the sum.
type RollOutcome struct {
	SkillName         string
	SkillValue        int
	Dice              []int // empty on a passive test
	DiceTotal         int
	Bias              int // net accumulated bias, master + system
	Modifier          int // net accumulated flat adjustment, master + ledger
	Passive           bool
	Total             int
	IsCritical        bool
	IsCriticalFailure bool
}

// Margin is the reading of this outcome against a difficulty class. The result of a test
// is the margin, not a boolean — success and failure are readings of the margin against
// thresholds.
//
// It is a method rather than a field because the CD comes from the opposed roll, which
// does not exist until the collision is implemented.
func (o RollOutcome) Margin(cd int) int { return o.Total - cd }

// RollCalculator is a stateless domain service that turns dice plus a character's numbers
// into a result. It rolls once (Roll) and derives as many times as the master edits
// (Derive).
type RollCalculator struct{}

// Roll rolls both sets for the given rules. Called once, when the action or reaction
// arrives. A nil src means the production roller.
func (rc RollCalculator) Roll(rules match.MatchRules, src RollSource) action.RollAttempts {
	return action.RollAttempts{
		Primary:   rc.RollDice(rules.DiceSet.Dice(), src),
		Secondary: rc.RollDice(rules.DiceSet.Dice(), src),
	}
}

// RollDice rolls an arbitrary set of dice, in order. Damage needs it: a weapon carries its
// own dice (a Sword is D10 + D4), which are not the match's test set.
func (rc RollCalculator) RollDice(sides []enum.DieSides, src RollSource) []int {
	s := sourceOrDefault(src)
	out := make([]int, 0, len(sides))
	for _, face := range sides {
		out = append(out, s.RollDie(face))
	}
	return out
}

// Derive computes the outcome from dice already rolled. Pure: same inputs, same output,
// no new dice. Every master edit goes through here.
func (rc RollCalculator) Derive(
	rules match.MatchRules, attempts action.RollAttempts, in RollInput,
) RollOutcome {
	bias, modifier := in.SystemBias, 0
	if in.Condition != nil {
		bias += in.Condition.Bias
		modifier += in.Condition.Modifier
	}
	if in.Ledger != nil {
		bias += in.Ledger.TotalBias(in.Dimension, in.AgainstID)
		modifier += in.Ledger.TotalAmount(in.Dimension, in.AgainstID)
	}

	out := RollOutcome{
		SkillName:  in.SkillName,
		SkillValue: in.SkillValue,
		Bias:       bias,
		Modifier:   modifier,
		Passive:    in.Passive,
	}

	if in.Passive {
		// A passive test takes the average of the set instead of rolling, so rolling has
		// exactly zero expected gain. No dice means no critical either way.
		out.DiceTotal = rules.PassiveValue()
		out.Total = out.DiceTotal + in.SkillValue + modifier
		return out
	}

	dice := pickAttempt(attempts, bias)
	out.Dice = append([]int(nil), dice...)
	out.DiceTotal = sumDice(dice)
	out.IsCritical = allDiceShow(dice, rules.DiceSet.MaxFace())
	out.IsCriticalFailure = allDiceShow(dice, 1)
	out.Total = out.DiceTotal + in.SkillValue + modifier
	return out
}

// pickAttempt applies advantage and disadvantage: the better set on a positive bias, the
// worse one on a negative bias, the primary set when neutral. Magnitude beyond ±1 only
// settles the sign — accumulated advantage and disadvantage cancel each other out first.
//
// Ties fall back to Primary. With 2D10 a tie can never involve a critical: 20 only comes
// from two tens and 2 only from two ones, so there is no other combination to tie with.
func pickAttempt(a action.RollAttempts, bias int) []int {
	if bias == 0 || len(a.Secondary) == 0 {
		return a.Primary
	}
	if len(a.Primary) == 0 {
		return a.Secondary
	}
	if bias > 0 {
		if sumDice(a.Secondary) > sumDice(a.Primary) {
			return a.Secondary
		}
		return a.Primary
	}
	if sumDice(a.Secondary) < sumDice(a.Primary) {
		return a.Secondary
	}
	return a.Primary
}

func sumDice(dice []int) int {
	total := 0
	for _, d := range dice {
		total += d
	}
	return total
}

// allDiceShow reports whether every die of the set landed on face. A critical is read on
// the individual dice, never on the sum.
func allDiceShow(dice []int, face int) bool {
	if len(dice) == 0 {
		return false
	}
	for _, d := range dice {
		if d != face {
			return false
		}
	}
	return true
}
