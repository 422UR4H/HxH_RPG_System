package spiritual

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type IHatsu interface {
	experience.ICascadeUpgrade
	GetPercentOf(category enum.CategoryName) float64
	GetNextLvlAggregateExp() int
	GetNextLvlBaseExp() int
	GetCurrentExp() int
	GetExpPoints() int
}
