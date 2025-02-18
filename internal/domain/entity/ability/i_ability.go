package ability

import (
	exp "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type IAbility interface {
	exp.ICascadeUpgrade

	GetNextLvlAggregateExp() int
	GetNextLvlBaseExp() int
	GetCurrentExp() int
	GetExpPoints() int
	GetBonus() float64
	GetExpReference() exp.ICascadeUpgrade
}
