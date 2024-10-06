package status

import (
	experience "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type IGenerateStatus interface {
	experience.ITriggerCascadeExp
	GetStatus() IStatus
}
