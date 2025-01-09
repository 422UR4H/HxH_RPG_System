package status

import (
	exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type IGenerateStatusBar interface {
	exp.ITriggerCascadeExp

	GetStatus() Bar
}
