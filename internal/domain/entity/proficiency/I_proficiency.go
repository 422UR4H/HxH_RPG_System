package proficiency

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"

type IProficiency interface {
	experience.ITriggerCascadeExp

	// GetValueForTest() int
	GetNextLvlAggregateExp() int
	GetNextLvlBaseExp() int
	GetCurrentExp() int
	GetExpPoints() int
	GetLevel() int
}
