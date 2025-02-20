package status

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"

type IStatusBar interface {
	IncreaseAt(value int) int
	DecreaseAt(value int) int
	Upgrade(level int, attr attribute.IGameAttribute)
	GetMin() int
	GetCurrent() int
	GetMax() int
}
