package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// ChainState is what one resolution leaves for the next.
//
// The collision with several targets is NOT f(action, reactions[]). It is a walk:
//
//	ataque₀ → resolve(alvo A) → ataque₁ → resolve(alvo B) → ataque₂ → …
//
// Residual is how much of the blow is left. Stopped means a repel ended it — and note that
// stopping is not cancelling: the reactions after it still resolve and their owners still
// narrate. They simply cannot be hit any more.
type ChainState struct {
	Residual int
	Stopped  bool
}

// Reduce applies one target's outcome to the blow travelling onward.
//
//	dodged            → unchanged: dodging does not spend the blow
//	repelled          → stopped
//	defended          → minus the defense of the weapon they defended with
//	hit               → minus the hit target's armour
//
// ⚠️ There is NO rigid rule here — combat-engine.md is explicit that this is contextual and the
// master may override at any point. What lives in code is the DEFAULT per reaction type. The
// override surface is Phase 5's (SystemData); do not build one here.
//
// ⚠️ Armour does not exist in this codebase — there is no armour entity and no sheet field —
// so the hit row currently subtracts zero. The row is encoded because the shape is what
// matters, exactly as ApplicableDefense encodes the damage-type rows it cannot yet read. Do
// not invent an armour model to fill it.
func (c ChainState) Reduce(out ReactionOutcome, defenseWeaponBonus, armour int) ChainState {
	if c.Stopped {
		return ChainState{Stopped: true}
	}
	if out.StopsAttack {
		return ChainState{Stopped: true}
	}
	// The defended row is checked BEFORE the avoided one, because a parry is both: a repel
	// near miss takes zero damage AND counts as having defended. Reading Avoided first would
	// pass the blow on whole and lose the only job Weapon.defense has.
	if out.Defended {
		return ChainState{Residual: floorZero(c.Residual - defenseWeaponBonus)}
	}
	if out.Avoided {
		return c
	}
	return ChainState{Residual: floorZero(c.Residual - armour)}
}

// ReduceSpread is Reduce with the attack's spread taken into account. A simultaneous attack
// does not diminish — everyone takes the same — while the master still opens one target at a
// time so each can narrate. Reserved: nothing sets SpreadSimultaneous until abilities exist.
func (c ChainState) ReduceSpread(spread action.AttackSpread, out ReactionOutcome, defenseWeaponBonus, armour int) ChainState {
	if spread == action.SpreadSimultaneous {
		// A repel still protects the one who made it — that target's own damage was already
		// zeroed above, in the per-target step, not here — but it does not shield the rest,
		// and an ordinary pass does not diminish either: both leave the same residual for
		// whoever comes next, so one line says it.
		return c
	}
	return c.Reduce(out, defenseWeaponBonus, armour)
}

func floorZero(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// weaponDefenseBonus reads the raw defense stat of the weapon that defended a blow — the
// number Reduce needs for the "defended" row of the chain table. It is a flat lookup, not
// ApplicableDefense's armed-vs-armed logic: Repel's own doc is explicit that on a parry "its
// defense is what reduces the blow travelling on to the next target," not a conditional
// block. nil resolves to bare hands, exactly like ApplicableDefense's passive-defense row.
func weaponDefenseBonus(name *enum.WeaponName, cat *item.WeaponsManager) int {
	w, err := lookupWeapon(name, cat)
	if err != nil {
		return 0
	}
	return w.GetDefense()
}

// chainStep is one stop on the walk: a target and, if one was opened, the reaction that
// answers the attack for them. A nil reaction means nothing was opened for this target, and
// ResolveReaction applies the passive defaults.
type chainStep struct {
	targetID uuid.UUID
	reaction *action.Action
}

// buildChainOrder is the walk order itself: every opened reaction, in the order the master
// opened it, then whatever targets remain, in the order the attack named them.
//
// Turn.reactions is append-ordered by ATTACH, which is not the order the master opened them
// — a target can attach a reaction long before the master gets to it. Walking attach order
// instead of open order would silently destroy the one thing this phase is about: that the
// order the master opens the reactions changes the outcome.
func buildChainOrder(a action.Action, reactions []action.Action, openedIDs []uuid.UUID) []chainStep {
	byID := make(map[uuid.UUID]action.Action, len(reactions))
	for _, r := range reactions {
		byID[r.GetID()] = r
	}

	var steps []chainStep
	covered := make(map[uuid.UUID]bool, len(openedIDs))
	for _, id := range openedIDs {
		r, ok := byID[id]
		if !ok {
			continue
		}
		rc := r
		steps = append(steps, chainStep{targetID: r.GetActorID(), reaction: &rc})
		covered[r.GetActorID()] = true
	}
	for _, targetID := range a.TargetID {
		if covered[targetID] {
			continue
		}
		steps = append(steps, chainStep{targetID: targetID})
	}
	return steps
}
