package service

import (
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// ReactionInput is everything ResolveReaction reads to answer one attack for one target.
//
// Ledger does double duty, and both halves are read-only from Resolve's point of view:
// deriveReflex and deriveEvasion read it (Dimension: DimDodge, AgainstID: the CURRENT
// attacker) to see a reserve banked by an earlier closed dodge — that is what makes
// ScopeAllBut(originalAttacker) actually count for whoever comes at this target next turn.
// The other half is unread here: a reaction's OWN payout (a fresh reserve, a repel's
// bonus/penalty) is collected on ReactionOutcome.Payouts and written into this same ledger
// once, at turn close, by MatchSession.applyResolution — never from inside this pure
// function.
type ReactionInput struct {
	Kind       action.ReactionKind
	Reaction   *action.Action // nil = nothing was sent; the passives apply
	Target     *csSheet.CharacterSheet
	Ledger     *match.ModifierLedger // the target's; nil = empty
	AttackerID uuid.UUID
	HitTotal   int // the CD every defensive test is read against
	Rules      match.MatchRules
}

// ReactionOutcome is one target's answer to one attack, fully derived.
type ReactionOutcome struct {
	Kind        action.ReactionKind
	Dodge       RollOutcome
	Defense     RollOutcome
	Evasion     RollOutcome // closed variants only
	Repel       RollOutcome // repel only
	Ladder      LadderOutcome
	Avoided     bool // the blow did not land on this target
	Defended    bool
	StopsAttack bool             // a successful repel — nothing travels on
	Payouts     []match.Modifier // what this reaction wrote into the target's ledger
	// Escape is how an escape came out — nil on every reaction that does not displace
	// (ReactionKind.Displaces()). See EscapeResult's own doc.
	Escape *EscapeResult
}

// EscapeResult is how an escape came out. nil on every reaction that does not displace.
type EscapeResult struct {
	MovePassed  bool // Move.FinalSpeed >= the attacker's hit
	DodgePassed bool // Dodge.Total >= the attacker's hit
	Escaped     bool // both — the only way an escape avoids the blow and reaches its slot
	// Landing is where the MASTER put the piece of an escape that failed (edit_action,
	// escapeLanding). nil = not chosen; the piece stays where it stood. Filled by the
	// resolver from the turn, never by ResolveReaction.
	Landing *[3]int
}

// ResolveReaction is one target's answer to one attack, as a pure function.
//
// Pure in the strong sense Phase 2 established: every die it reads has already fallen and lives
// in the reaction's RollCheck.Attempts. It derives, it never rolls — which is what lets the
// master recompute on every new reaction and on every edit without re-rolling a player's die.
//
// A nil Reaction means nothing was sent, and the engine applies the defaults: reflex dodge,
// then — only if that fails — the defense, one ladder step easier. Sending ReactNothing is a
// different thing entirely: it refuses even those.
func ResolveReaction(in ReactionInput) ReactionOutcome {
	calc := RollCalculator{}
	out := ReactionOutcome{Kind: in.Kind}

	if in.Kind == action.ReactNothing {
		// Refusing the safety net is a choice, and the engine honours it exactly.
		return out
	}

	if in.Kind == action.ReactRepel {
		return resolveRepel(in, calc)
	}

	dodge, evasion, bonus := dodgeAndReserve(in, calc)
	out.Dodge = dodge
	if isClosedKind(in.Kind) {
		out.Evasion = evasion
	}
	if bonus != nil {
		out.Payouts = append(out.Payouts, *bonus)
	}
	dodgePassed := out.Dodge.Total >= in.HitTotal
	out.Avoided = dodgePassed
	if in.Kind.Displaces() {
		// An escape is a dodge that moves, and the move is its own test against the same CD:
		// the attacker's hit. Both have to clear it (front-combat-phases.md §6A.5, B13) — the
		// user cannot get out of the way if the step fails. Failing either one is not escaping,
		// and the blow is read as if they had stayed: escapeGuard still falls back on the
		// defense below, the others take it whole.
		//
		// TODO(collision): the known rule is that the movement ADDS to the dodge, which makes
		// escaping easier than dodging in place. It is part of the collision design that does
		// not exist yet — when it does, the sum goes into out.Dodge.Total right here, before
		// either comparison.
		movePassed := in.Reaction != nil && in.Reaction.Move != nil && in.Reaction.Move.FinalSpeed >= in.HitTotal
		out.Escape = &EscapeResult{MovePassed: movePassed, DodgePassed: dodgePassed, Escaped: dodgePassed && movePassed}
		out.Avoided = out.Escape.Escaped
	}
	if out.Avoided {
		return out
	}
	if !in.Kind.KeepsDefault() {
		// Escaping and repelling give up the automatic defense. Miss, and the blow lands whole.
		return out
	}
	// The master's condition on the default defense lives on the reaction's DefaultDefense
	// (edit_action, field "defense"). A target that did not react has no reaction to carry it
	// — that passive is not editable yet (front-combat-phases.md, pendências).
	var defenseCond *action.RollCondition
	if in.Reaction != nil && in.Reaction.DefaultDefense != nil {
		defenseCond = in.Reaction.DefaultDefense.Context.Condition
	}
	out.Defense = calc.Derive(in.Rules, action.RollAttempts{}, RollInput{
		SkillName:  enum.Defense.String(),
		SkillValue: skillValueOf(in.Target, enum.Defense.String()),
		Passive:    true,
		Condition:  defenseCond,
	})
	out.Defended = out.Defense.Total >= in.HitTotal-in.Rules.LadderStep
	return out
}

// dodgeAndReserve is the reflex test behind dodge, closedDodge, escape, escapeGuard and
// closedEscape, plus the closed variants' reserve payout — derived together so the two can
// never desynchronize. Reflex and, for the closed variants, Evasion are each derived EXACTLY
// ONCE here: dodgeAndReserve used to be two separate functions that each re-derived both rolls
// independently, and while Derive is pure so they could not disagree today, nothing stopped a
// future edit from adding a Ledger, Dimension or AgainstID to only one call site and silently
// splitting "the dodge" that was compared to the hit from "the dodge" the reserve gap was
// computed against. Threading one pair of outcomes through both uses removes that seam.
//
// A nil Reaction (nothing sent) takes the passive; every kind that arrives with its own dice
// rolls instead — that is the "gamble the roll" the table describes. For the closed variants
// the worse of Reflex and Evasion counts as the dodge: Evasion does not add to it, it enters
// the Disadvantage logic. The reserve banked is |Reflex − Evasion|, for whoever comes at this
// target next turn, excluding the attacker who was just read; a zero gap banks nothing. Every
// other kind gets no Evasion roll and no reserve.
func dodgeAndReserve(in ReactionInput, calc RollCalculator) (dodge, evasion RollOutcome, payout *match.Modifier) {
	reflex := deriveReflex(in, calc)
	if !isClosedKind(in.Kind) {
		return reflex, RollOutcome{}, nil
	}

	evasion = deriveEvasion(in, calc)
	dodge = reflex
	if evasion.Total < reflex.Total {
		dodge = evasion
	}

	gap := reflex.Total - evasion.Total
	if gap < 0 {
		gap = -gap
	}
	if gap == 0 {
		return dodge, evasion, nil
	}
	m := match.Modifier{
		Amount:    gap,
		Applies:   match.DimDodge,
		Source:    match.SourceSystem,
		Against:   match.ScopeAllBut(in.AttackerID),
		ExpiresAt: match.LifetimeNextTurn,
		Reason:    "closed dodge reserve",
	}
	return dodge, evasion, &m
}

// deriveReflex derives the Reflex test alone: passive when nothing was sent, rolled off the
// reaction's own Dodge.RollCheck otherwise.
func deriveReflex(in ReactionInput, calc RollCalculator) RollOutcome {
	passive := in.Reaction == nil
	var attempts action.RollAttempts
	var cond *action.RollCondition
	if !passive && in.Reaction.Dodge != nil {
		attempts = in.Reaction.Dodge.Attempts
		// The master's edit on this reaction's dodge (edit_action, field "dodge"). Nil reads
		// neutral. A passive dodge has no reaction and therefore nowhere to carry one — the
		// passive of a target that did not react is not editable yet (front-combat-phases.md).
		cond = in.Reaction.Dodge.Context.Condition
	}
	return calc.Derive(in.Rules, attempts, RollInput{
		SkillName:  enum.Reflex.String(),
		SkillValue: skillValueOf(in.Target, enum.Reflex.String()),
		Passive:    passive,
		Condition:  cond,
		// A closed dodge banked in an earlier turn lives on this same dimension, scoped
		// AllBut the attacker it was earned against — reading it here, against the CURRENT
		// attacker, is what lets "whoever comes at this target next turn" actually count.
		Ledger:    in.Ledger,
		Dimension: match.DimDodge,
		AgainstID: &in.AttackerID,
	})
}

// deriveEvasion derives the Evasion test off the reaction's Skills list, matched by name — the
// shape the WS mapper already produces for a skill list, and the one the closed variants' tests
// build against. Evasion never has a passive reading: the closed variants always send it rolled.
func deriveEvasion(in ReactionInput, calc RollCalculator) RollOutcome {
	var attempts action.RollAttempts
	var cond *action.RollCondition
	if in.Reaction != nil {
		for _, s := range in.Reaction.Skills {
			if s.SkillName == enum.Evasion.String() {
				attempts = s.Attempts
				// edit_action's skillName "Evasion". The master's panel does not offer it
				// (front-combat-phases.md §8 — the test chain will redesign it), but the engine
				// reads every condition it accepts.
				cond = s.Context.Condition
				break
			}
		}
	}
	return calc.Derive(in.Rules, attempts, RollInput{
		SkillName:  enum.Evasion.String(),
		SkillValue: skillValueOf(in.Target, enum.Evasion.String()),
		Condition:  cond,
		// Same reserve, same dimension: the closed variants take the worse of Reflex and
		// Evasion as "the dodge" (dodgeAndReserve), so a banked reserve has to reach whichever
		// of the two ends up being read as that.
		Ledger:    in.Ledger,
		Dimension: match.DimDodge,
		AgainstID: &in.AttackerID,
	})
}

func isClosedKind(k action.ReactionKind) bool {
	return k == action.ReactClosedDodge || k == action.ReactClosedEscape
}

// resolveRepel reads the repel roll against the ladder built off the attack's hit total, and
// pays out per rung. The bonus on a great success is specific to the attacker read — you
// learned to read THAT opponent — while the penalty on a near miss is general: being left off
// balance is something anyone can exploit.
//
// A near miss is zero damage, not reduced damage — Avoided is true on every rung but
// RungFailure — and it also sets Defended, because it is the "defended" row of the chain
// table: the blow travels on reduced by the repelling weapon's defense. Only the two clearing
// rungs set StopsAttack. On RungFailure the passives do not apply either: repelling commits the
// weapon to the incoming blow instead of also ducking.
func resolveRepel(in ReactionInput, calc RollCalculator) ReactionOutcome {
	out := ReactionOutcome{Kind: in.Kind}

	var attempts action.RollAttempts
	var cond *action.RollCondition
	if in.Reaction != nil && in.Reaction.Repel != nil {
		attempts = in.Reaction.Repel.Attempts
		cond = in.Reaction.Repel.Context.Condition
	}
	out.Repel = calc.Derive(in.Rules, attempts, RollInput{
		SkillName:  enum.Repel.String(),
		SkillValue: skillValueOf(in.Target, enum.Repel.String()),
		Condition:  cond,
		AgainstID:  &in.AttackerID,
	})

	out.Ladder = ClimbLadder(out.Repel.Total-in.HitTotal, in.Rules.LadderStep)

	switch out.Ladder.Rung {
	case RungGreatSuccess:
		out.Avoided = true
		out.StopsAttack = true
		out.Payouts = append(out.Payouts, match.Modifier{
			Amount:    out.Ladder.Difference,
			Applies:   match.DimActionSpeed,
			Source:    match.SourceSystem,
			Against:   match.ScopeOnly(in.AttackerID),
			ExpiresAt: match.LifetimeNextTurn,
			Reason:    "repel: great success bonus",
		})
	case RungSuccess:
		out.Avoided = true
		out.StopsAttack = true
	case RungNearMiss:
		out.Avoided = true
		out.Defended = true
		out.Payouts = append(out.Payouts, match.Modifier{
			Amount:    -out.Ladder.Difference,
			Applies:   match.DimActionSpeed,
			Source:    match.SourceSystem,
			Against:   match.ScopeAnyone(),
			ExpiresAt: match.LifetimeNextTurn,
			Reason:    "repel: near miss penalty",
		})
	case RungFailure:
		// Avoided and Defended stay false: the attack lands whole, and repelling gave up the
		// passives too.
	}
	return out
}
