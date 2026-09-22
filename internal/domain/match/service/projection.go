package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// Viewer is one recipient's standing, and there are exactly THREE classes, not four: the
// master, the owner of a given action or reaction, and everyone else.
//
// The target is deliberately NOT a class. A feint against you does not tell you it was a
// feint — privileging the target would delete the feint from the game.
type Viewer struct {
	IsMaster bool
	// Owns is the set of character sheet UUIDs this viewer controls.
	Owns map[uuid.UUID]bool
}

// SeesAllOf reports whether this viewer is entitled to the unprojected truth about one
// character's action or reaction.
func (v Viewer) SeesAllOf(charID uuid.UUID) bool {
	return v.IsMaster || v.Owns[charID]
}

// publicKind demotes the closed variants to the open ones they must be indistinguishable
// from.
//
// The LABEL is the leak. If closedDodge arrives public, nobody has to deduce anything — the
// name already said there was an Evasion folded in. Deducing from the public bar is
// legitimate (a closed escape charges one bar where the standard charges two, and
// bars_updated is public); being told is not.
func publicKind(k string) string {
	switch action.ReactionKind(k) {
	case action.ReactClosedDodge:
		return string(action.ReactDodge)
	case action.ReactClosedEscape:
		return string(action.ReactEscape)
	default:
		return k
	}
}

// ProjectResolution returns the copy of res this viewer is entitled to.
//
// Public by omission: the numbers travel, because "the opponent has to deduce from the
// numbers" is impossible without them. What is withheld is a closed list.
//
// It NEVER mutates res: the master holds the same pointer, and a projection that edited in
// place would turn their copy into a lie one recipient at a time.
func ProjectResolution(res *TurnResolution, v Viewer) *TurnResolution {
	if res == nil {
		return nil
	}
	out := *res
	out.CharacterResults = make([]CharacterResult, 0, len(res.CharacterResults))
	for _, cr := range res.CharacterResults {
		if !v.SeesAllOf(cr.TargetID) {
			public := publicKind(cr.ReactionKind)
			// The DEMOTION is the discriminator, not the field. A payout is secret only when
			// the reaction that earned it is: the closed dodge's reserve is the other half of
			// the same secret, because the size of the dodge that was not spent says how much
			// Evasion was folded in — so it goes out with the label.
			//
			// Everything else stays. reacoes.md is explicit about the one that matters:
			// "a penalidade de quem aparou vale contra todo mundo — qualquer um pode
			// aproveitar", and a repel is never demoted, so its penalty travels. Zeroing
			// Payouts for every third party took that one with it and left the table to
			// reconstruct it by algebra off Ladder.Difference — the exact reconstruction
			// ReactionTotal exists to spare a client.
			if public != cr.ReactionKind {
				cr.Payouts = nil
			}
			cr.ReactionKind = public
		}
		out.CharacterResults = append(out.CharacterResults, cr)
	}
	// PendingReactions is the master's own to-do list — attached, not yet given the floor.
	// Nobody else is owed the knowledge that an answer is waiting.
	if !v.IsMaster {
		out.PendingReactions = nil
		// Errors is on the same side of the fence, for a different reason: these are
		// diagnostics about the ENGINE, not facts about the fiction, and the master is the
		// only recipient who can act on one. A target the engine could not classify is also,
		// in practice, a name the table was never told about.
		out.Errors = nil
	}
	// Blows carry no numbers (see battle.Blow) and are not projected. There is no separate
	// per-reaction list to project either — every reaction outcome is already on the
	// CharacterResult of the target that sent it, projected above.
	return &out
}

// ProjectAction returns the copy of an action this viewer is entitled to. Used by the Action
// History, where the same policy applies to the DECLARATION rather than to the arithmetic.
//
// isSettled is the TIME axis, the same one ProjectResolution reads. A feint is secret while
// the turn is open and public once it closes: the target who fell for it finds out inside that
// same turn's resolution, because their success was against a false attack and the real one
// follows. Hiding it after the turn closed hides it forever, which is not the rule.
func ProjectAction(a action.Action, v Viewer, isSettled bool) action.Action {
	if v.SeesAllOf(a.GetActorID()) {
		return a
	}
	out := a
	if !isSettled {
		// A revealed feint is not a feint — while the swing is still in the air.
		out.Feint = nil
	}
	// Trigger stays hidden on both axes. The feint's rule was decided; the trigger's has not
	// been written, and action.Trigger is an empty object today. Do not widen the decision by
	// symmetry.
	out.Trigger = nil
	out.ReactionKind = action.ReactionKind(publicKind(string(a.ReactionKind)))
	if len(a.Skills) > 0 {
		// The Evasion entry is NOT the same rule as the feint: it is the other half of the
		// closed dodge's secret, tied to the label demotion above, and that demotion is
		// permanent by design.
		kept := make([]action.Skill, 0, len(a.Skills))
		for _, s := range a.Skills {
			if s.SkillName == enum.Evasion.String() {
				continue
			}
			kept = append(kept, s)
		}
		out.Skills = kept
	}
	return out
}
