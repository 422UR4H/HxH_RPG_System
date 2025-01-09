package spiritual

import "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"

type IPrinciple interface {
	experience.ITriggerCascadeExp

	GetExpPoints() int
	GetLvl() int
}
