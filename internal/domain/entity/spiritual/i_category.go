package spiritual

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type ICategory interface {
	experience.ITriggerCascadeExp

	GetNextLvlAggregateExp() int
	GetNextLvlBaseExp() int
	GetCurrentExp() int
	GetExpPoints() int
	GetLevel() int
}
