package service

import (
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// TargetKind identifies the entity type a UUID refers to in an active match.
type TargetKind string

const (
	TargetKindCharacter   TargetKind = "character" // checked first in CategorizeTarget
	TargetKindWallSegment TargetKind = "wall_segment"
	TargetKindUnknown     TargetKind = "unknown"
	// TODO: TargetKindFloorTile, TargetKindItem — future phases
)

// TargetReader allows TurnResolver to categorize and read action targets
// without importing matchsession (prevents circular imports).
// *matchsession.MatchSession implements this interface implicitly.
type TargetReader interface {
	CategorizeTarget(id uuid.UUID) TargetKind
	GetWall(id string) (mapentity.WallSegment, bool)
}

// WallResultKind discriminates attack vs interact outcomes in WallResult.
type WallResultKind string

const (
	WallResultKindAttack   WallResultKind = "attack"
	WallResultKindInteract WallResultKind = "interact"
)

// WallResult is the computed outcome of one player action targeting a wall.
type WallResult struct {
	UpdatedWall     mapentity.WallSegment
	EffectiveDamage int
	ReboundDamage   int // melee rebound candidate; TODO: apply to actor if melee, subtract actor Defense
	Kind            WallResultKind
}

// ResolveInput is everything a resolution reads. It is a struct rather than a parameter
// list because the list has already grown twice and will grow again — the match rules and
// the weapon catalogue arrived with the collision, and reactions arrive after it.
type ResolveInput struct {
	Turn *turn.Turn
	// Sheets is keyed by sheet UUID — the same ID the board pieces carry as CharacterID,
	// and the same ID Action.actorID and Action.TargetID carry.
	Sheets  map[uuid.UUID]*csSheet.CharacterSheet
	Targets TargetReader // nil disables target routing
	Rules   match.MatchRules
	Weapons *item.WeaponsManager
}

// TurnResolution is the snapshot of a Turn's result — character combat, wall
// interactions, or any mix thereof.
//
// It is a DRY RUN. Nothing here has been applied to a sheet. The master sees the projected
// HP reduction from the first resolution onward, and the real application happens once, at
// turn close. That is what lets the collision be recomputed on every reaction without
// applying damage several times.
type TurnResolution struct {
	ActionResult     RollResult
	ReactionResults  []ReactionResult
	CharacterResults []CharacterResult
	Blows            []*battle.Blow
	WallResults      []WallResult
	IsSettled        bool
}

// RollResult holds the outcome of a single dice roll check.
type RollResult struct {
	SkillName  string
	SkillValue int
	DiceRolled []int
	Total      int
	// The critical flags travel untouched. No rule consumes them yet — the design notes
	// point at a narrative consequence rather than a multiplier — and the resolver must not
	// invent one.
	IsCritical        bool
	IsCriticalFailure bool
	// Margin is nil until an opposed roll gives this test a CD.
	Margin *int
}

// ReactionResult holds the outcome of one reaction within the Turn.
type ReactionResult struct {
	ReactorID uuid.UUID
	Roll      RollResult
}

// CharacterResult is the computed outcome of one attack against one character.
type CharacterResult struct {
	TargetID uuid.UUID
	Hit      RollOutcome
	Dodge    RollOutcome
	// Defense is the zero value when the dodge already stopped the attack.
	Defense  RollOutcome
	Dodged   bool
	Defended bool

	DamageDice      []int
	RawDamage       int
	DefenseApplied  int
	EffectiveDamage int

	Blow *battle.Blow
}

// TurnResolver is a stateless domain service that calculates Turn resolution
// for any action type: character combat, wall attacks, door interactions, etc.
type TurnResolver struct{}

// Resolve calculates the current resolution snapshot for the given Turn.
//
// It is PURE: every die it needs has already fallen and lives in the action's RollCheck
// Attempts, put there by MatchSession the moment the action arrived. Resolve derives, it
// never rolls. That is what makes it safe to call again on every reaction and on every
// master edit — the master never re-rolls a player's die.
func (tr TurnResolver) Resolve(in ResolveInput) *TurnResolution {
	res := &TurnResolution{}
	if in.Turn == nil {
		return res
	}
	res.IsSettled = in.Turn.GetFinishedAt() != nil
	a := in.Turn.GetAction()

	if in.Targets != nil {
		for _, targetID := range a.TargetID {
			switch in.Targets.CategorizeTarget(targetID) {
			case TargetKindCharacter:
				cr, ok := tr.resolveCharacter(in, a, targetID)
				if !ok {
					continue
				}
				res.CharacterResults = append(res.CharacterResults, cr)
				res.Blows = append(res.Blows, cr.Blow)
				dodgeTotal := cr.Dodge.Total
				res.ActionResult = rollResultOf(cr.Hit, &dodgeTotal)

			case TargetKindWallSegment:
				wall, found := in.Targets.GetWall(targetID.String())
				if !found {
					continue
				}
				if a.Attack != nil {
					raw, err := RawDamage(a.Attack.Damage.Attempts.Primary, a.Attack.Weapon, in.Weapons)
					if err != nil {
						raw = 0
					}
					sdr := ApplyStructuralDamage(wall, raw)
					res.WallResults = append(res.WallResults, WallResult{
						UpdatedWall:     sdr.UpdatedWall,
						EffectiveDamage: sdr.EffectiveDamage,
						ReboundDamage:   sdr.ReboundDamage,
						Kind:            WallResultKindAttack,
					})
				}
				if a.Interact != nil {
					updated, ok := ApplyWallInteract(wall, a.Interact)
					if ok {
						res.WallResults = append(res.WallResults, WallResult{
							UpdatedWall: updated,
							Kind:        WallResultKindInteract,
						})
					}
				}

			case TargetKindUnknown:
				// TODO: record unknown-target error in resolution for caller to surface
			}
		}
	}

	reactions := in.Turn.GetReactions()
	res.ReactionResults = make([]ReactionResult, len(reactions))
	for i, r := range reactions {
		// TODO: implement per-reaction resolution
		res.ReactionResults[i] = ReactionResult{ReactorID: r.ReactToID}
	}
	return res
}

// resolveCharacter runs one attack against one character: the hit, then the two passive
// reactions in the order the rules give them, then damage.
//
// The passives are free and automatic. The reflex dodge takes the dice set's average
// instead of rolling — rolling has exactly zero expected gain, so the player only gambles
// when they need luck above the average — and the defense is one ladder step easier than
// the attack, because it should be easier to parry than to land a blow.
func (tr TurnResolver) resolveCharacter(
	in ResolveInput, a action.Action, targetID uuid.UUID,
) (CharacterResult, bool) {
	if a.Attack == nil {
		return CharacterResult{}, false
	}
	actorSheet, okActor := in.Sheets[a.GetActorID()]
	targetSheet, okTarget := in.Sheets[targetID]
	if !okActor || !okTarget || actorSheet == nil || targetSheet == nil {
		// TODO: surface a missing-sheet error in the resolution, with the rest of the error
		// reporting the caller needs
		return CharacterResult{}, false
	}

	calc := RollCalculator{}
	cr := CharacterResult{TargetID: targetID}

	// 1. The hit — the attacker's active test.
	//    Ledger is deliberately nil: the accumulated difference a character carries is
	//    always an actionSpeed adjustment, never a hit adjustment.
	cr.Hit = calc.Derive(in.Rules, a.Attack.Hit.Attempts, RollInput{
		SkillName:  a.Attack.Hit.SkillName,
		SkillValue: skillValueOf(actorSheet, a.Attack.Hit.SkillName),
		Condition:  a.Attack.Hit.Context.Condition,
		AgainstID:  &targetID,
	})

	// 2. Reflex dodge — passive, free, automatic. Ties favour the defender.
	cr.Dodge = calc.Derive(in.Rules, action.RollAttempts{}, RollInput{
		SkillName:  enum.Reflex.String(),
		SkillValue: skillValueOf(targetSheet, enum.Reflex.String()),
		Passive:    true,
	})
	cr.Dodged = cr.Dodge.Total >= cr.Hit.Total

	// 3. Defense — only if the dodge failed, at a CD one ladder step lower.
	if !cr.Dodged {
		cr.Defense = calc.Derive(in.Rules, action.RollAttempts{}, RollInput{
			SkillName:  enum.Defense.String(),
			SkillValue: skillValueOf(targetSheet, enum.Defense.String()),
			Passive:    true,
		})
		cr.Defended = cr.Defense.Total >= cr.Hit.Total-in.Rules.LadderStep
	}

	// 4. Damage — the weapon's own dice, already rolled on arrival.
	if !cr.Dodged {
		cr.DamageDice = append([]int(nil), a.Attack.Damage.Attempts.Primary...)
		raw, err := RawDamage(cr.DamageDice, a.Attack.Weapon, in.Weapons)
		if err == nil {
			cr.RawDamage = raw
			def := ApplicableDefense(DefenseInput{
				AttackWeapon: a.Attack.Weapon,
				// The passive defense is always bare-handed. An armed parry needs the target
				// to declare one, which is an active reaction.
				DefenseWeapon: nil,
				Defended:      cr.Defended,
				Catalogue:     in.Weapons,
			})
			cr.DefenseApplied = def.Amount
			cr.EffectiveDamage = EffectiveDamage(raw, def)
		}
	}

	cr.Blow = battle.NewBlow(a.GetActorID(), targetID, *a.Attack, nil, nil, nil)
	return cr, true
}

// skillValueOf reads a skill off the sheet, crossing the string→enum boundary. A name the
// sheet does not know contributes 0 rather than failing the whole resolution: the mapper at
// the WS boundary already rejects unknown skill names, so reaching here means an internal
// name, and a resolution that silently drops one test is better than one that drops the turn.
func skillValueOf(cs *csSheet.CharacterSheet, name string) int {
	skillName, err := enum.SkillNameFrom(name)
	if err != nil {
		return 0
	}
	v, err := cs.GetValueForTestOfSkill(skillName)
	if err != nil {
		return 0
	}
	return v
}

// rollResultOf flattens an outcome into the wire-facing shape, deriving the margin now that
// a CD exists.
func rollResultOf(o RollOutcome, cd *int) RollResult {
	r := RollResult{
		SkillName:         o.SkillName,
		SkillValue:        o.SkillValue,
		DiceRolled:        append([]int(nil), o.Dice...),
		Total:             o.Total,
		IsCritical:        o.IsCritical,
		IsCriticalFailure: o.IsCriticalFailure,
	}
	if cd != nil {
		m := o.Margin(*cd)
		r.Margin = &m
	}
	return r
}
