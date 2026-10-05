package actionwire

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// Level is how much of an action's NUMBERS this recipient's surface entitles them to. It is
// the one axis this package owns — see From's own doc for the axis it deliberately does not
// own.
type Level int

const (
	// Full keeps every number: dice, totals, both speeds. The REST Action History uses this
	// level, and only this level, which is why the cut table below always keeps at Full — see
	// get_match_history_golden_test.go for the byte-identical guarantee that depends on it.
	Full Level = iota
	// Opened keeps the mechanics and both speeds (actionSpeed and moveSpeed), but cuts dice
	// and results everywhere else: attack.hit/damage/charge, move.charge, skills[], feint,
	// defense, dodge, repel all come through as skill name only.
	Opened
	// Declaration keeps only what the owner declared: targets, weapon, move (category, from,
	// position), skill NAMES, reactionKind, interact, spread, relativeVelocity, systemBias. No
	// dice, no totals, no speeds at all — not even actionSpeed's or moveSpeed's own skill
	// value, though their skill NAME still travels (see RollCheck's own doc).
	Declaration
)

// From projects a domain action.Action into the shared wire.Action, cut to lvl.
//
// From carries NO visibility deny-list. Feint and Trigger nil-ing, ReactionKind demotion, and
// the stripped Evasion skill entry are service.ProjectAction's job, and ProjectAction ALWAYS
// runs before From — on the REST history path (get_match_history.go, via the use case) and on
// every WebSocket path alike. From receives whatever action.Action already survived that
// deny-list and only ever asks one question of it, field by field: does this recipient's level
// entitle them to the NUMBERS behind an already-visible field? It never decides whether a
// field is visible at all — that axis and this one are deliberately kept apart, each in its
// own place, so that widening one can never accidentally widen the other.
//
// The cut is exactly the table in design spec §4.1: speed.rollCheck and move.speed/finalSpeed
// (the two "speed" rows) are kept whenever lvl != Declaration; every other number
// (attack.hit/damage/charge, move.charge, skills[], feint, defense, dodge, repel) is kept only
// when lvl == Full. Everything that is not a number — targets, weapon, move's category/
// from/position, skill names, reactionKind, interact, spread, relativeVelocity, systemBias —
// is copied unconditionally, at every level. Whether a recipient may see WHERE a piece moves
// (move.from/position) is the fog's question, not a Level's: the game server answers it after
// From, per recipient — From itself never withholds them.
func From(a action.Action, lvl Level) Action {
	speedKeep := lvl != Declaration
	numbersKeep := lvl == Full

	out := Action{
		UUID:         a.GetID(),
		ActorID:      a.GetActorID(),
		TargetID:     a.TargetID,
		ReactionKind: string(a.ReactionKind),
		Speed: ActionSpeed{
			Bar:       a.Speed.Bar,
			RollCheck: rollCheck(a.Speed.RollCheck, speedKeep),
		},
		SystemBias:        a.SystemBias,
		ConsumedActionIDs: a.ConsumedActionIDs,
	}
	if a.ReactToID != uuid.Nil {
		id := a.ReactToID
		out.ReactToID = &id
	}
	for _, s := range a.Skills {
		out.Skills = append(out.Skills, ActionSkill{
			SkillName: s.SkillName, Difficulty: s.Difficulty,
			RollCheck: rollCheck(s.RollCheck, numbersKeep),
		})
	}
	if a.Feint != nil {
		rc := rollCheck(*a.Feint, numbersKeep)
		out.Feint = &rc
	}
	if a.Trigger != nil {
		out.Trigger = &Trigger{}
	}
	if a.Move != nil {
		out.Move = &Move{
			Category: string(a.Move.Category), From: fromPtr(a.Move.From), Position: positionPtr(a.Move.Position),
			Speed:      rollCheckPtr(a.Move.Speed, speedKeep),
			Charge:     rollCheckPtr(a.Move.Charge, numbersKeep),
			FinalSpeed: intPtr(a.Move.FinalSpeed, speedKeep),
		}
	}
	if a.Attack != nil {
		out.Attack = &Attack{
			Weapon: weaponPtr(a.Attack.Weapon),
			Hit:    rollCheck(a.Attack.Hit, numbersKeep), Damage: rollCheck(a.Attack.Damage, numbersKeep),
			Charge: rollCheckPtr(a.Attack.Charge, numbersKeep), Spread: string(a.Attack.Spread),
			RelativeVelocity: a.Attack.RelativeVelocity,
		}
	}
	if a.Defense != nil {
		out.Defense = &Defense{
			Weapon: weaponPtr(a.Defense.Weapon), RollCheck: rollCheck(a.Defense.RollCheck, numbersKeep),
		}
	}
	if a.Dodge != nil {
		out.Dodge = &Dodge{RollCheck: rollCheck(a.Dodge.RollCheck, numbersKeep)}
	}
	if a.Repel != nil {
		out.Repel = &Repel{
			Weapon: weaponPtr(a.Repel.Weapon), RollCheck: rollCheck(a.Repel.RollCheck, numbersKeep),
		}
	}
	if a.Interact != nil {
		out.Interact = &Interact{Kind: string(a.Interact.Kind)}
	}
	return out
}

// rollCheck projects one domain RollCheck. SkillName always travels; the numbers travel only
// when keep is true, and when they do they are filled unconditionally — even a genuine zero
// value — because a nil pointer, not a zero int, is what "cut" means on this wire. That is
// also what keeps Level Full's output identical to what a non-pointer field emitted before
// this package existed: every number is always present there, zero or not.
func rollCheck(rc action.RollCheck, keep bool) RollCheck {
	out := RollCheck{SkillName: rc.SkillName}
	if !keep {
		return out
	}
	skillValue, result := rc.SkillValue, rc.Result
	out.SkillValue = &skillValue
	out.Result = &result
	out.Attempts = &RollAttempts{Primary: rc.Attempts.Primary, Secondary: rc.Attempts.Secondary}
	return out
}

// rollCheckPtr is rollCheck for the optional RollCheck fields (move/attack charge, move
// speed): a nil input (no roll at all, e.g. a Shift that never rolls speed) stays nil
// regardless of keep — there is no number to cut because there was never a roll.
func rollCheckPtr(rc *action.RollCheck, keep bool) *RollCheck {
	if rc == nil {
		return nil
	}
	out := rollCheck(*rc, keep)
	return &out
}

// intPtr fills a pointer to v when keep is true (even when v is the zero value — see
// rollCheck's own doc for why that distinction matters), and nils it out otherwise.
func intPtr(v int, keep bool) *int {
	if !keep {
		return nil
	}
	out := v
	return &out
}

// fromPtr defensively copies the domain Move.From pointer onto the wire, rather than sharing
// it: From/Position are copied unconditionally at every level (see From's own doc), but a
// nil-safe clone keeps a caller that later mutates the domain Action from reaching back
// through a wire struct that outlives it, the same defensive posture weaponPtr takes below.
func fromPtr(v *[3]int) *[3]int {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

// positionPtr lifts the domain's always-present Move.Position onto the wire's pointer. From
// always fills it; the pointer exists so the game server can withhold it per recipient (see
// Move's own doc), not because the domain ever lacks it.
func positionPtr(v [3]int) *[3]int {
	out := v
	return &out
}

// weaponPtr renders an optional domain weapon as the wire's plain *string, nil-safe.
func weaponPtr(w *enum.WeaponName) *string {
	if w == nil {
		return nil
	}
	s := string(*w)
	return &s
}
