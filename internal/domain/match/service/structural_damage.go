package service

import mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"

// StructuralDamageResult is the outcome of one attack on a wall segment.
type StructuralDamageResult struct {
	UpdatedWall     mapentity.WallSegment
	EffectiveDamage int // damage applied to the wall (≥ 0)
	ReboundDamage   int // = min(rawDamage, Resistance) — melee rebound candidate
	// TODO: apply ReboundDamage to actor only if attack is melee (check Attack.Category)
	// TODO: subtract actor Defense from ReboundDamage before applying
	// TODO: include ReboundDamage in broadcast (enrich wall_hp_changed or separate event)
}

// ApplyStructuralDamage applies raw attack damage to a WallSegment, respecting
// material resistance. MaxHP==0 signals an indestructible wall (no HP system).
func ApplyStructuralDamage(w mapentity.WallSegment, rawDamage int) StructuralDamageResult {
	if w.MaxHP == 0 {
		// Indestructible — no HP system; full rebound regardless of attack type.
		// TODO: if range attack (Attack.Category == "range"), ReboundDamage = 0
		return StructuralDamageResult{UpdatedWall: w, EffectiveDamage: 0, ReboundDamage: rawDamage}
	}
	effective := rawDamage - w.Resistance
	if effective < 0 {
		effective = 0
	}
	rebound := rawDamage - effective // = min(rawDamage, Resistance)
	w.HP -= effective
	if w.HP < 0 {
		w.HP = 0
	}
	if w.HP == 0 && w.MaxHP > 0 {
		w.Destroyed = true
	}
	// TODO: persist new HP state in map snapshot on turn close (see PersistTurnClose).
	return StructuralDamageResult{UpdatedWall: w, EffectiveDamage: effective, ReboundDamage: rebound}
}
