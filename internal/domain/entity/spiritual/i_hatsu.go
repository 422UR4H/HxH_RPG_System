package spiritual

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type IHatsu interface {
	experience.ICascadeUpgrade
	GetPercentOf(category enum.CategoryName) float64
}
