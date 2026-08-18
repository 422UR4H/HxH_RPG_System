package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

// unarmed is what "no weapon" resolves to. The catalogue already carries a Fist entry
// (D6 + D6 + D4, flat damage 0), so bare hands are a weapon like any other instead of a
// branch everything downstream has to remember.
const unarmed = enum.Fist

// WeaponDice returns the dice a weapon rolls for damage.
//
// This is the second family of roll in the system, and it is NOT MatchRules.DiceSet. The
// test set (hit, skill, actionSpeed) is 2 D10 for everyone; damage comes from the weapon —
// a Sword is D10 + D4, a Rifle is D12 + D10.
func WeaponDice(name *enum.WeaponName, cat *item.WeaponsManager) ([]enum.DieSides, error) {
	w, err := lookupWeapon(name, cat)
	if err != nil {
		return nil, err
	}
	raw := w.GetDice()
	sides := make([]enum.DieSides, 0, len(raw))
	for _, d := range raw {
		sides = append(sides, enum.DieSides(d))
	}
	return sides, nil
}

// RawDamage is the weapon's rolled dice plus its flat damage bonus.
//
// The hit margin deliberately does NOT enter here. Product owner: "it will not add into the
// damage, at least not for now, because this system is already very punishing." It is the
// one place in the system where a margin does not circulate, and that is on purpose —
// revisiting it is a post-MVP playtest question, not a TODO.
func RawDamage(dice []int, name *enum.WeaponName, cat *item.WeaponsManager) (int, error) {
	w, err := lookupWeapon(name, cat)
	if err != nil {
		return 0, err
	}
	total := w.GetDamage()
	for _, d := range dice {
		total += d
	}
	return total, nil
}

// DefenseInput is everything the applicable-defense rules read.
type DefenseInput struct {
	AttackWeapon  *enum.WeaponName // nil = unarmed attack
	DefenseWeapon *enum.WeaponName // nil = bare-handed defense
	Defended      bool             // whether the target's defense actually succeeded
	Catalogue     *item.WeaponsManager
}

// DefenseOutcome is what a defense does to the raw damage.
type DefenseOutcome struct {
	Amount         int  // subtracted from the raw damage
	BlocksEntirely bool // nothing passes through at all
}

// ApplicableDefense decides what a successful defense takes off the raw damage.
//
// The subtraction is CONDITIONAL: defense only counts when the target actually managed to
// defend. It is not automatic damage reduction.
//
//	weapon against weapon    → nothing passes through
//	unarmed attack           → the defending weapon's defense bonus subtracts
//	armed attack, bare hands → the defense has no efficacy against piercing or cutting, and
//	                           works only against concussive
//
// This is the INITIAL form of parrying, not a temporary one — the mechanic exists from the
// start, and what comes after the MVP is its complexity and detail, not a replacement.
// What is missing today is the surrounding entities: damage types (concussive, cutting,
// piercing, ultra-piercing), armour (which subtracts too, with the master deciding whether
// a blow lands on it at all), and Nen (which will reduce the final damage). Because damage
// types do not exist, the armed-attack-versus-bare-hands row subtracts nothing rather than
// guessing which type a weapon deals. No rungs are invented here.
//
// Worth knowing while reading the numbers: every weapon in the catalogue today carries
// defense 0, so the subtraction is currently inert. The shape is what matters.
func ApplicableDefense(in DefenseInput) DefenseOutcome {
	if !in.Defended {
		return DefenseOutcome{}
	}
	attackArmed := in.AttackWeapon != nil
	defenseArmed := in.DefenseWeapon != nil

	if attackArmed && defenseArmed {
		return DefenseOutcome{BlocksEntirely: true}
	}
	if attackArmed && !defenseArmed {
		// No damage types yet: cannot tell concussive from cutting, so nothing is subtracted.
		return DefenseOutcome{}
	}
	// Unarmed attack: the only row that passes reduced damage through.
	w, err := lookupWeapon(in.DefenseWeapon, in.Catalogue)
	if err != nil {
		return DefenseOutcome{}
	}
	return DefenseOutcome{Amount: w.GetDefense()}
}

// EffectiveDamage applies a defense outcome to raw damage. Never negative.
func EffectiveDamage(raw int, d DefenseOutcome) int {
	if d.BlocksEntirely {
		return 0
	}
	if v := raw - d.Amount; v > 0 {
		return v
	}
	return 0
}

func lookupWeapon(name *enum.WeaponName, cat *item.WeaponsManager) (item.Weapon, error) {
	n := unarmed
	if name != nil {
		n = *name
	}
	return cat.Get(n)
}
