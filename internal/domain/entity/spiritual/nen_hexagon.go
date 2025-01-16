package spiritual

import (
	"math"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

const (
	percentageJumpByCategory = 20
	maxHexRange              = 600
	categoryRange            = maxHexRange / 6 // 100
)

var nenHexagon = map[enum.CategoryName]int{
	enum.Reinforcement:   categoryRange * 0, // 0
	enum.Transmutation:   categoryRange * 1, // 100
	enum.Materialization: categoryRange * 2, // 200
	enum.Specialization:  categoryRange * 3, // 300
	enum.Manipulation:    categoryRange * 4, // 400
	enum.Emission:        categoryRange * 5, // 500
}

type NenHexagon struct {
	currHexValue    int
	nenCategoryName enum.CategoryName
}

func NewNenHexagon(currHexValue int) *NenHexagon {
	currHexValue %= maxHexRange

	return &NenHexagon{
		currHexValue:    currHexValue,
		nenCategoryName: getCategoryByHexagon(currHexValue),
	}
}

func (nh *NenHexagon) IncreaseCurrHexValue() (int, enum.CategoryName) {
	nh.currHexValue++
	nh.currHexValue %= maxHexRange
	nh.nenCategoryName = getCategoryByHexagon(nh.currHexValue)

	return nh.currHexValue, nh.nenCategoryName
}

func (nh *NenHexagon) DecreaseCurrHexValue() (int, enum.CategoryName) {
	nh.currHexValue--
	nh.currHexValue %= maxHexRange

	if nh.currHexValue < 0 {
		nh.currHexValue += maxHexRange
	}
	nh.nenCategoryName = getCategoryByHexagon(nh.currHexValue)
	return nh.currHexValue, nh.nenCategoryName
}

func getCategoryByHexagon(currHexValue int) enum.CategoryName {
	currHexValue %= maxHexRange
	halfCategoryRange := categoryRange / 2

	for key, val := range nenHexagon {
		if currHexValue < (val + halfCategoryRange) {
			return key
		}
	}
	return enum.Reinforcement

	// TODO: remove after tests
	// if currHexValue < halfCategoryRange || currHexValue >= 550 {
	// 	return enum.Reinforcement

	// } else if currHexValue < 150 {
	// 	return enum.Transmutation

	// } else if currHexValue < 250 {
	// 	return enum.Materialization

	// } else if currHexValue < 350 {
	// 	return enum.Specialization

	// } else if currHexValue < 450 {
	// 	return enum.Manipulation

	// } else { // currHexValue < 550
	// 	return enum.Emission
	// }
}

func (nh *NenHexagon) GetPercentOf(category enum.CategoryName) float64 {
	// absHexDiff is absolute hexagonal difference
	// it will always be positive (absolute) and symmetrical to the hexagon
	absHexDiff := math.Abs(float64(nenHexagon[category] - nh.currHexValue))
	if absHexDiff > maxHexRange/2 {
		absHexDiff = maxHexRange - absHexDiff
	}
	divisor := categoryRange / percentageJumpByCategory
	absHexDiff /= float64(divisor)
	percent := 100.0 - absHexDiff

	return percent
}

func (nh *NenHexagon) GetCategoryPercents() map[enum.CategoryName]float64 {
	modifiers := make(map[enum.CategoryName]float64)

	for key := range nenHexagon {
		modifiers[key] = nh.GetPercentOf(key)
	}
	return modifiers
}

// ResetCategory resets the category to the default value
// in the same way that happened to Gon after the events
// of the end of the Chimera Ants arc.
func (nh *NenHexagon) ResetCategory() (int, enum.CategoryName) {
	nh.currHexValue = nenHexagon[nh.nenCategoryName]
	return nh.currHexValue, nh.nenCategoryName
}

func (nh *NenHexagon) GetCategoryName() enum.CategoryName {
	return nh.nenCategoryName
}

func (nh *NenHexagon) GetCurrHexValue() int {
	return nh.currHexValue
}
