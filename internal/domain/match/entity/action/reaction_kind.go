package action

import "fmt"

// ReactionKind is what the target chose to do about an attack.
//
// It is DECLARED by the sender and never inferred from which components the payload filled
// in. The three escapes carry exactly the same shape — a Dodge and a Move — and cost three
// different things; no inspection of the shape can tell them apart, because the information
// that separates them is the player's intent.
//
// The reflex dodge and the default defense are NOT kinds. Nobody sends them: they are what
// the engine applies when nothing arrives. ReactNothing exists precisely because refusing
// them is a send, and the only way to tell "sent nothing" from "sent nothing on purpose" is
// to have received something.
type ReactionKind string

const (
	ReactNothing      ReactionKind = "nothing"      // refuses even the passives — takes the blow raw
	ReactDodge        ReactionKind = "dodge"        // active dodge: gamble the roll instead of the average
	ReactClosedDodge  ReactionKind = "closedDodge"  // closed dodge, with Evasion folded in
	ReactEscape       ReactionKind = "escape"       // standard escape: gives up the default defense
	ReactEscapeGuard  ReactionKind = "escapeGuard"  // defensive escape: keeps the safety net
	ReactClosedEscape ReactionKind = "closedEscape" // closed escape
	ReactRepel        ReactionKind = "repel"        // attacks the attack
)

// Bars is what this reaction costs, by the table in combat-engine.md § Reações.
//
// nil is a real answer, not an omission: the closed variants pay for themselves. Done in the
// exact instant, without opening the guard, they give the action back.
//
// ⚠️ Note the asymmetry with Action.Bars(), which is never empty. That invariant exists
// because the scheduler has to price every action it schedules by some bar — and a reaction is
// NOT scheduled: it never enters the queue and is never chosen by RoundScheduler. Empty is the
// correct answer for a free reaction.
func (k ReactionKind) Bars() []Bar {
	switch k {
	case ReactRepel:
		return []Bar{BarAction}
	case ReactClosedEscape:
		return []Bar{BarMove}
	case ReactEscape, ReactEscapeGuard:
		return []Bar{BarAction, BarMove}
	default:
		return nil
	}
}

// IsFree reports whether this reaction charges no bar — and therefore consumes no pending
// action and rolls at no disadvantage.
func (k ReactionKind) IsFree() bool { return len(k.Bars()) == 0 }

// Displaces reports whether this reaction moves the character. All three escapes do, by
// definition — an escape that does not displace is just a dodge — so the WS boundary refuses
// one that arrives without a Move.
func (k ReactionKind) Displaces() bool {
	switch k {
	case ReactEscape, ReactEscapeGuard, ReactClosedEscape:
		return true
	default:
		return false
	}
}

// KeepsDefault reports whether the default defense still stands behind this reaction when it
// fails.
//
// Escaping gives up the safety net: force the dodge and miss, and the automatic defense does
// not come in — full damage. Only the defensive escape keeps it. Repelling gives it up too:
// you committed the weapon to the incoming blow, you were not also ducking.
func (k ReactionKind) KeepsDefault() bool {
	switch k {
	case ReactEscape, ReactClosedEscape, ReactRepel, ReactNothing:
		return false
	default:
		return true
	}
}

// ReactionComponent names one piece of an Action a ReactionKind may require to be well-formed.
//
// The mapper enforces these; the kind only declares them. A reaction accepted without a
// component its kind requires does not fail loudly downstream — it derives against an empty
// RollCheck, which silently becomes the worst possible outcome for whoever sent it (Total = 0,
// worse than the passive it was supposed to replace, or a guaranteed RungFailure on a repel).
type ReactionComponent string

const (
	ComponentDodge ReactionComponent = "Dodge"
	ComponentMove  ReactionComponent = "Move"
	ComponentRepel ReactionComponent = "Repel"
)

// RequiredComponents lists what this kind needs on the Action to be well-formed.
//
//	nothing                        → none
//	dodge, closedDodge              → Dodge
//	escape, escapeGuard, closedEscape → Dodge AND Move — they force the dodge BY displacing, so
//	                                    Move alone (Displaces()) is necessary but not sufficient
//	repel                          → Repel
//
// Move is folded in via Displaces() rather than re-listed per kind, so there is exactly one
// place that knows which kinds move.
func (k ReactionKind) RequiredComponents() []ReactionComponent {
	var out []ReactionComponent
	switch k {
	case ReactRepel:
		return []ReactionComponent{ComponentRepel}
	case ReactDodge, ReactClosedDodge, ReactEscape, ReactEscapeGuard, ReactClosedEscape:
		out = append(out, ComponentDodge)
	default:
		return nil
	}
	if k.Displaces() {
		out = append(out, ComponentMove)
	}
	return out
}

// RequiresEvasionSkill reports whether this kind needs a named Evasion entry in Skills to be
// well-formed — the closed variants' whole point, and the one requirement RequiredComponents
// cannot express: a Skills entry is not shaped like an Action sub-struct the way Dodge, Move
// and Repel are, so it is not a ReactionComponent. Bending it into a fake component would hide
// that it is a different kind of check; a separate method says so honestly. Without this, a
// closed dodge accepted with no Evasion entry derives against an empty RollCheck (skillValue +
// 0, Passive: false) — worse than the passive it was meant to replace, and dodgeAndReserve
// still takes it as "the dodge" and banks a reserve off the bogus gap. Enforced at the WS
// boundary, refused the same way a missing Dodge/Move/Repel is.
func (k ReactionKind) RequiresEvasionSkill() bool {
	return k == ReactClosedDodge || k == ReactClosedEscape
}

func (k ReactionKind) IsValid() bool {
	switch k {
	case ReactNothing, ReactDodge, ReactClosedDodge,
		ReactEscape, ReactEscapeGuard, ReactClosedEscape, ReactRepel:
		return true
	}
	return false
}

func (k ReactionKind) String() string { return string(k) }

// ReactionKindFrom crosses the string→enum boundary. An unknown or empty kind is an error
// here, where it can still be answered with a WS error: the server never infers what a
// reaction costs from the shape of what arrived.
func ReactionKindFrom(s string) (ReactionKind, error) {
	k := ReactionKind(s)
	if !k.IsValid() {
		return "", fmt.Errorf("reaction kind %q is not in the catalogue", s)
	}
	return k, nil
}
