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
	Charge *RollCheck
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
