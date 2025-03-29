package spiritual

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type NenHexagonUpdateResult struct {
	PercentList   map[enum.CategoryName]float64
	CategoryName  enum.CategoryName
	CurrentHexVal int
}
