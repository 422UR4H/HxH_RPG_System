package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

// AttackSpread is how an attack reaches several targets.
//
// SpreadSequential travels through them: what leaves one target is what enters the next, so
// the order the master opens the reactions changes the outcome. SpreadSimultaneous hits
// everyone at once and does NOT diminish — the master still opens one at a time, because that
// is the table gesture, but only the narration is sequential; the arithmetic is not.
//
// Reserved, not exercised: this will be a configuration of the ability type, and special
// abilities do not exist until post-MVP. The axis is here now because retrofitting it later
// would mean threading an `if` through a chain already written to assume one shape.
type AttackSpread string

const (
	SpreadSequential   AttackSpread = "" // the zero value: today's only reachable behaviour
	SpreadSimultaneous AttackSpread = "simultaneous"
)

type Attack struct {
	Weapon *enum.WeaponName
	Hit    RollCheck
	Damage RollCheck
	// DamageSkill is the skill that measures this attack's damage. The player does not choose
	// it — the weapon deals the damage, and Push measures it (front-combat-phases.md §4.6) —
	// but the master may swap it (edit_action damageSkill: Grab, or another). Zero means Push:
	// every attack, and every row persisted before the field existed.
	//
	// Deliberately NOT Damage.SkillName. That is the field the player sends and the server
	// discards, and the action wire carries it to everyone: storing the master's choice there
	// would let a player choose, and would show a player who reconnects mid-turn the master's
	// edit before the turn closes. No wire maps this field.
	DamageSkill enum.SkillName
	Charge      *RollCheck
	// Spread is how this attack reaches several targets. See AttackSpread: reserved until
	// abilities exist, and today's only reachable value is the zero value, SpreadSequential.
	Spread AttackSpread

	// I was wondering where the damage plus speed should be placed
	// and I realized that the hit also has a speed bonus,
	// so I decided to link it to Attack and have the system resolve it in other local.
	// ActorSpeed  float64
	// TargetSpeed float64
	RelativeVelocity float64
	// --> decidi que esse cálculo será feito em outro local
	// 		- algum objeto de battle, action.engine, ou até a própria move resolverá isso
	// 		- ActorSpeed e TargetSpeed são da action move e serão resolvidas lá
}

// EffectiveDamageSkill is DamageSkill with its zero value read as Push.
func (a Attack) EffectiveDamageSkill() enum.SkillName {
	if a.DamageSkill == "" {
		return enum.Push
	}
	return a.DamageSkill
}
