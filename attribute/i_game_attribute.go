package attribute

import exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"

type IGameAttribute interface {
	exp.ICascadeUpgrade

	GetHalfOfAbilityLvl() float64
	GetExpPoints() int
	GetPoints() int
	GetLevel() int
	GetPower() int
}
