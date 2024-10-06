package status

import (
	exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type IGenerateStatus interface {
	exp.ITriggerCascadeExp
	GetStatus() IStatus
}
