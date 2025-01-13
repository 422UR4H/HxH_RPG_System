package ability

import exp "github.com/422UR4H/HxH_RPG_System/internal/domain/experience"

type IAbility interface {
	exp.ICascadeUpgrade
	GetHalfLvl() float64
}
