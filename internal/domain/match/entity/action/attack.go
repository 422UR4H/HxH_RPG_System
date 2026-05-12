package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type Attack struct {
	Weapon *enum.WeaponName
	Hit    RollCheck
	Damage RollCheck
	Charge *RollCheck

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
