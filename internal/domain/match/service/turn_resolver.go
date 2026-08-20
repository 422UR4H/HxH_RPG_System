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
	Sheets map[uuid.UUID]*csSheet.CharacterSheet
	// Statuses is keyed the same way as Sheets. Each character's ModifierLedger is read
	// (never written) while resolving their reaction — a closed dodge's reserve from an
	// earlier turn only does anything if a later resolution can see it. nil, or a missing
	// entry, reads as an empty ledger: Resolve stays pure either way.
	Statuses map[uuid.UUID]*match.CharacterStatus
	Targets  TargetReader // nil disables target routing
	Rules    match.MatchRules
	Weapons  *item.WeaponsManager
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
	Defense RollOutcome
	// Avoided is true when the blow did not land on this target at all — a dodge cleared it,
	// or a repel stopped it here. It is NOT "dodged": a successful repel and a closed escape
	// set it too, and neither is a dodge. Ask ReactionKind if the distinction matters.
	Avoided  bool
	Defended bool

	// ReactionKind is what this target answered with — "" when nothing was opened and the
	// passive defaults applied instead.
	ReactionKind string
	// Ladder is the zero value outside a repel.
	Ladder LadderOutcome
	// AttackStopped is true when a repel earlier in the chain already stopped this attack
	// before it reached this target. Their own reaction above still resolved and still
	// narrates — stopping is not cancelling — it simply could not be hit any more.
	AttackStopped bool
	// ReactionStopsAttack is true when THIS target's OWN reaction stopped the attack — a
	// repel that cleared the ladder (great success or plain success). Do not confuse this
	// with AttackStopped: that one reads the INCOMING chain state, whether someone earlier
	// in the walk already stopped it before it reached this target. This one is the verdict
	// on this target's own answer. A target can have AttackStopped false and
	// ReactionStopsAttack true (their repel is what stopped it going further), or
	// AttackStopped true and ReactionStopsAttack false (an earlier repel stopped it before
	// they even got a swing to answer).
	ReactionStopsAttack bool
	// ReactionTotal is the reaction's own roll total, whichever kind it was — the dodge
	// family's Dodge.Total, or a repel's own total (which CharacterResult otherwise has no
	// place for: ResolveReaction's Repel RollOutcome is not copied onto this struct the way
	// Dodge and Defense are). Computed once here, at the one place that already knows which
	// roll a given kind reads, so a consumer (the WS payload) does not have to know the
	// kind-to-roll mapping — or worse, reconstruct the number by algebra off Ladder.Margin.
	ReactionTotal int
	// ReactionID is the attached reaction's own Action ID — the zero value when nothing was
	// opened (or attached) and the passive defaults applied instead. It is the identifier
	// MatchSession.OpenReaction expects on a later open_reaction; without projecting it here,
	// the wire never tells the master what to send back, and open_reaction becomes
	// unreachable from a real client that only ever sees TargetID and ReactionKind. Read
	// straight off step.reaction, the one place resolveCharacterStep already holds the actual
	// attached Action rather than just its derived kind/total.
	ReactionID uuid.UUID
	// Payouts is what this target's own reaction earned — a closed dodge's reserve, a
	// repel's bonus or penalty. Written into their ledger once, at turn close, alongside the
	// damage (see MatchSession.applyResolution, which reads this field by TargetID directly
	// — there is deliberately no second, resolution-level list to keep in sync with it).
	Payouts []match.Modifier

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
		// The character targets are NOT an independent loop over a.TargetID: they are a
		// walk. What leaves one target is what enters the next, in the order the master
		// opened the reactions — see buildChainOrder and ChainState. Wall and unknown
		// targets have no chain to walk; they stay in the second loop, in the attack's own
		// target order, untouched by this.
		if a.Attack != nil {
			chain := tr.seedChain(in, a)
			for _, step := range buildChainOrder(a, in.Turn.GetReactions(), in.Turn.OpenedReactionIDs()) {
				if in.Targets.CategorizeTarget(step.targetID) != TargetKindCharacter {
					continue
				}
				cr, next, ok := tr.resolveCharacterStep(in, a, step, chain)
				if !ok {
					continue
				}
				chain = next
				res.CharacterResults = append(res.CharacterResults, cr)
				res.Blows = append(res.Blows, cr.Blow)
				dodgeTotal := cr.Dodge.Total
				res.ActionResult = rollResultOf(cr.Hit, &dodgeTotal)
			}
		}

		for _, targetID := range a.TargetID {
			switch in.Targets.CategorizeTarget(targetID) {
			case TargetKindCharacter:
				// Walked above, in chain order rather than a.TargetID order.

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

// seedChain computes ataque₀: the whole attack's raw damage, rolled once when the action
// arrived and never re-rolled. Every target in the walk only ever subtracts from this one
// number — it is not recomputed per target.
func (tr TurnResolver) seedChain(in ResolveInput, a action.Action) ChainState {
	raw, err := RawDamage(a.Attack.Damage.Attempts.Primary, a.Attack.Weapon, in.Weapons)
	if err != nil {
		return ChainState{}
	}
	return ChainState{Residual: raw}
}

// resolveCharacterStep runs one attack against one target in the walk: the hit, the
// target's reaction (or the passive defaults, when none was opened), what THIS target takes,
// and what the chain leaves for whoever is walked next.
//
// chainIn is the residual the walk carries INTO this target — what this target takes is read
// off chainIn, not off a fresh roll: the attack already happened once, and the chain is what
// is left of it by the time it reaches them.
func (tr TurnResolver) resolveCharacterStep(
	in ResolveInput, a action.Action, step chainStep, chainIn ChainState,
) (CharacterResult, ChainState, bool) {
	if a.Attack == nil {
		return CharacterResult{}, chainIn, false
	}
	actorSheet, okActor := in.Sheets[a.GetActorID()]
	targetSheet, okTarget := in.Sheets[step.targetID]
	if !okActor || !okTarget || actorSheet == nil || targetSheet == nil {
		// TODO: surface a missing-sheet error in the resolution, with the rest of the error
		// reporting the caller needs
		return CharacterResult{}, chainIn, false
	}

	calc := RollCalculator{}
	cr := CharacterResult{TargetID: step.targetID}

	// 1. The hit — the attacker's active test, shared by the whole chain: every target in
	//    this attack answers against the same CD, because it is the same swing.
	//    Ledger is deliberately nil here: the duel reserve (repel/parry) modifies
	//    actionSpeed, not the hit — see match.Modifier.Applies — and the closed dodge's
	//    reserve modifies the dodge, which ResolveReaction reads on the target's own side,
	//    not on this roll.
	cr.Hit = calc.Derive(in.Rules, a.Attack.Hit.Attempts, RollInput{
		SkillName:  a.Attack.Hit.SkillName,
		SkillValue: skillValueOf(actorSheet, a.Attack.Hit.SkillName),
		Condition:  a.Attack.Hit.Context.Condition,
		AgainstID:  &step.targetID,
	})

	// 2. The target's answer — an opened reaction if one exists, the passive defaults
	//    (reflex dodge, then defense) otherwise. This is Task 9's ResolveReaction, reused
	//    rather than re-derived: it is the one place that already knows how every kind of
	//    reaction reads against a hit.
	kind := action.ReactionKind("")
	if step.reaction != nil {
		kind = step.reaction.ReactionKind
	}
	var ledger *match.ModifierLedger
	if st, ok := in.Statuses[step.targetID]; ok && st != nil {
		ledger = &st.Ledger
	}
	out := ResolveReaction(ReactionInput{
		Kind:       kind,
		Reaction:   step.reaction,
		Target:     targetSheet,
		Ledger:     ledger,
		AttackerID: a.GetActorID(),
		HitTotal:   cr.Hit.Total,
		Rules:      in.Rules,
	})
	cr.ReactionKind = string(out.Kind)
	cr.Dodge = out.Dodge
	cr.Defense = out.Defense
	cr.Avoided = out.Avoided
	cr.Defended = out.Defended
	cr.Ladder = out.Ladder
	cr.Payouts = out.Payouts
	cr.AttackStopped = chainIn.Stopped
	cr.ReactionStopsAttack = out.StopsAttack
	// The repel branch is the one kind whose own RollOutcome CharacterResult does not
	// otherwise keep (see the field comment on ReactionTotal) — every other kind's total is
	// already sitting on cr.Dodge.Total above.
	cr.ReactionTotal = out.Dodge.Total
	if out.Kind == action.ReactRepel {
		cr.ReactionTotal = out.Repel.Total
	}
	if step.reaction != nil {
		cr.ReactionID = step.reaction.GetID()
	}

	// 3. What THIS target takes: the chain's current residual, reduced exactly the way a
	//    single-target attack always was (ApplicableDefense/EffectiveDamage, Phase 2) —
	//    unless the chain never reached them at all, stopped earlier by someone else's
	//    repel, or they avoided the blow themselves (a dodge, or a repel that stopped it
	//    here). Either way that is zero, and nothing else here changes.
	//
	//    This is deliberately a SEPARATE computation from step 4's onward reduction, not one
	//    number feeding the other: ApplicableDefense already knows the armed-vs-armed rules
	//    (block entirely, or nothing, for lack of damage types) that decide what THIS target
	//    takes, while the chain's Reduce only ever needs a flat weapon-defense number for
	//    what carries on. Collapsing them would change what a bare-handed default defense
	//    does to an armed attack today — see the single-target test that pins EffectiveDamage
	//    at the full raw value even though Defended is true.
	if !chainIn.Stopped && !out.Avoided {
		cr.DamageDice = append([]int(nil), a.Attack.Damage.Attempts.Primary...)
		cr.RawDamage = chainIn.Residual
		def := ApplicableDefense(DefenseInput{
			AttackWeapon: a.Attack.Weapon,
			// The passive defense is always bare-handed. An armed parry needs the target
			// to declare one, which is the repel reaction, handled by ReduceSpread below —
			// not by this per-target damage math.
			DefenseWeapon: nil,
			Defended:      cr.Defended,
			Catalogue:     in.Weapons,
		})
		cr.DefenseApplied = def.Amount
		cr.EffectiveDamage = EffectiveDamage(cr.RawDamage, def)
	}

	// 4. What the chain leaves for whoever is walked next.
	var defenseWeapon *enum.WeaponName
	if out.Kind == action.ReactRepel && step.reaction != nil && step.reaction.Repel != nil {
		defenseWeapon = step.reaction.Repel.Weapon
	}
	// Armour does not exist in this codebase — there is no armour entity and no sheet field
	// — so the hit row currently subtracts zero. Encoded anyway, exactly as
	// ApplicableDefense encodes the damage-type rows it cannot yet read: the shape is what
	// matters, not the number. Do not invent an armour model to fill it.
	const armour = 0
	chainOut := chainIn.ReduceSpread(a.Attack.Spread, out, weaponDefenseBonus(defenseWeapon, in.Weapons), armour)

	cr.Blow = battle.NewBlow(a.GetActorID(), step.targetID, *a.Attack, nil, nil, nil)
	return cr, chainOut, true
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
