package spiritual

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type NenHexagon struct {
	hexagonRange    int
	nenCategoryName enum.CategoryName
}

func NewNenHexagon(hexagonRange int) *NenHexagon {
	hexagonRange %= 600

	return &NenHexagon{
		hexagonRange:    hexagonRange,
		nenCategoryName: getCategoryByHexagon(hexagonRange),
	}
}

func (nh *NenHexagon) IncreaseRange() (int, enum.CategoryName) {
	nh.hexagonRange++
	nh.hexagonRange %= 600
	nh.nenCategoryName = getCategoryByHexagon(nh.hexagonRange)

	return nh.hexagonRange, nh.nenCategoryName
}

func (nh *NenHexagon) DecreaseRange() (int, enum.CategoryName) {
	nh.hexagonRange--
	nh.hexagonRange %= 600

	if nh.hexagonRange < 0 {
		nh.hexagonRange += 600
	}
	nh.nenCategoryName = getCategoryByHexagon(nh.hexagonRange)
	return nh.hexagonRange, nh.nenCategoryName
}

func getCategoryByHexagon(hexagonRange int) enum.CategoryName {
	if hexagonRange < 50 || hexagonRange >= 550 {
		return enum.Reinforcement

	} else if hexagonRange < 150 {
		return enum.Transmutation

	} else if hexagonRange < 250 {
		return enum.Materialization

	} else if hexagonRange < 350 {
		return enum.Specialization

	} else if hexagonRange < 450 {
		return enum.Manipulation

	} else { // hexagonRange < 550
		return enum.Emission
	}
}

// ResetCategory resets the category to the default value
// in the same way that happened to Gon after the events
// of the end of the Chimera Ants arc.
func (nh *NenHexagon) ResetCategory() (int, enum.CategoryName) {
	if nh.nenCategoryName == enum.Reinforcement {
		nh.hexagonRange = 0
	} else if nh.nenCategoryName == enum.Transmutation {
		nh.hexagonRange = 100
	} else if nh.nenCategoryName == enum.Materialization {
		nh.hexagonRange = 200
	} else if nh.nenCategoryName == enum.Specialization {
		nh.hexagonRange = 300
	} else if nh.nenCategoryName == enum.Manipulation {
		nh.hexagonRange = 400
	} else { // nh.nenCategoryName == enum.Emission
		nh.hexagonRange = 500
	}
	return nh.hexagonRange, nh.nenCategoryName
}

func (nh *NenHexagon) GetCategoryName() enum.CategoryName {
	return nh.nenCategoryName
}

func (nh *NenHexagon) GetRange() int {
	return nh.hexagonRange
}
