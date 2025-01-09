package skill

import "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"

type ISkill interface {
	experience.ITriggerCascadeExp

	GetLvl() int
	GetValueForTest() int
	GetExpPoints() int
}
