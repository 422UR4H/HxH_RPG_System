package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type Attack struct {
	Weapon *enum.WeaponName
	Hit    RollContext
	Damage RollContext
	Charge *RollContext

	// I was wondering where the damage plus speed should be placed
	// and I realized that the hit also has a speed bonus,
	// so I decided to link it to Attack and have the system resolve it internally.
	// Maybe it's better to change it to ActorVelocity and TargetVelocity
	// and have the system resolve the result internally.
	// Consider this in v0.0 or v0.1.
	ActorSpeed  float64
	TargetSpeed float64
}
