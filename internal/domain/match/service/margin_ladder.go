package service

// LadderRung is where a margin landed.
//
// The shape of the ladder — how many rungs and what each one does — stays in code, because
// changing it changes the game. Only the step size is a match rule (MatchRules.LadderStep).
//
// The asymmetry is intentional and comes from the product owner: "it should be easier to
// parry than to hit the target; this system is already very punishing." Hence the near-miss
// rung, where failing by less than a step still costs the attacker the damage.
type LadderRung string

const (
	// RungGreatSuccess: cleared the CD by a full step or more. Zero damage, plus a bonus
	// equal to the difference — and that bonus is specific to the opponent read.
	RungGreatSuccess LadderRung = "great_success"
	// RungSuccess: cleared the CD. Zero damage.
	RungSuccess LadderRung = "success"
	// RungNearMiss: missed by less than a full step. Parried, which is zero damage, not
	// reduced damage — the price is a penalty equal to the difference, and that penalty is
	// general, because being off balance is something anyone can exploit.
	RungNearMiss LadderRung = "near_miss"
	// RungFailure: missed by a full step or more. The attack lands.
	RungFailure LadderRung = "failure"
)

// LadderOutcome is a margin read against the ladder.
type LadderOutcome struct {
	Rung   LadderRung
	Margin int
	// Difference is the size of the bonus on RungGreatSuccess and of the penalty on
	// RungNearMiss — the only two rungs that pay out into the ModifierLedger. It is zero
	// on the other two.
	Difference int
}

// ClimbLadder reads margin against a ladder of the given step.
//
// margin is the defender's total minus the CD, so a margin of 0 — landing exactly on the CD
// — is a success: ties favour the defender, the same way the repel table treats CD itself as
// a defender's row.
//
// Nothing is wired into this yet. Phase 4 hands it the repel reaction; Phase 2 only needs it
// to exist and be right.
func ClimbLadder(margin, step int) LadderOutcome {
	out := LadderOutcome{Margin: margin}
	switch {
	case margin >= step:
		out.Rung = RungGreatSuccess
		out.Difference = margin
	case margin >= 0:
		out.Rung = RungSuccess
	case margin >= -step:
		out.Rung = RungNearMiss
		out.Difference = -margin
	default:
		out.Rung = RungFailure
	}
	return out
}
