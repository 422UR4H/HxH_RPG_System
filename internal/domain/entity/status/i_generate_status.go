package status

import (
	exp "github.com/422UR4H/HxH_RPG_System/internal/domain/experience"
)

type IGenerateStatusBar interface {
	exp.ITriggerCascadeExp

	GetStatus() Bar
}