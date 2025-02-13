package attribute

import exp "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"

type IGameAttribute interface {
	exp.ICascadeUpgrade

	GetNextLvlAggregateExp() int
	GetNextLvlBaseExp() int
	GetCurrentExp() int
	GetExpPoints() int
	GetPoints() int
	GetPower() int
	GetBonus() float64
}
