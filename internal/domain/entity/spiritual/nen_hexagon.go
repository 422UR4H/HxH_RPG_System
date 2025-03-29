package spiritual

import (
	"math"
	"sort"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

const (
	percentageJumpByCategory = 20
	maxHexRange              = 600
	categoryRange            = maxHexRange / 6 // 100
)

type CategoryPair struct {
	Category enum.CategoryName
	Value    int
}

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

func (nh *NenHexagon) IncreaseCurrHexValue() (
	map[enum.CategoryName]float64, enum.CategoryName) {

	nh.currHexValue++
	nh.currHexValue %= maxHexRange
	nh.nenCategoryName = nh.UpdateCategoryByHexagon()

	return nh.GetCategoryPercents(), nh.nenCategoryName
}

func (nh *NenHexagon) DecreaseCurrHexValue() (
	map[enum.CategoryName]float64, enum.CategoryName) {

	nh.currHexValue--
	nh.currHexValue %= maxHexRange

	if nh.currHexValue < 0 {
		nh.currHexValue += maxHexRange
	}
	nh.nenCategoryName = nh.UpdateCategoryByHexagon()
	return nh.GetCategoryPercents(), nh.nenCategoryName
}

// UpdateCategoryByHexagon updates the category for increase and decrease hexValue
// this allows an intermediate value between the categories
func (nh *NenHexagon) UpdateCategoryByHexagon() enum.CategoryName {
	halfCategoryRange := categoryRange / 2
	sortedNenHexagon := sortNenHexagon()

	for _, hex := range sortedNenHexagon {
		if nh.currHexValue < (hex.Value + halfCategoryRange) {
			return hex.Category
		} else if nh.currHexValue == (hex.Value + halfCategoryRange) {
			return nh.nenCategoryName
		}
	}
	return enum.Reinforcement
}

// getCategoryByHexagon returns the initial category of the hexagon
func getCategoryByHexagon(currHexValue int) enum.CategoryName {
	currHexValue %= maxHexRange
	halfCategoryRange := categoryRange / 2

	sortedNenHexagon := sortNenHexagon()

	for _, hex := range sortedNenHexagon {
		if currHexValue < (hex.Value + halfCategoryRange) {
			return hex.Category
		}
	}
	return enum.Reinforcement
}

func sortNenHexagon() []CategoryPair {
	var pairs []CategoryPair
	for key, val := range nenHexagon {
		pairs = append(pairs, CategoryPair{Category: key, Value: val})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value < pairs[j].Value
	})
	return pairs
}

func (nh *NenHexagon) GetPercentOf(category enum.CategoryName) float64 {
	if category == enum.Specialization && nh.nenCategoryName != enum.Specialization {
		return 0.0
	}

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

	for category := range nenHexagon {
		modifiers[category] = nh.GetPercentOf(category)
	}
	return modifiers
}

// ResetCategory resets the category to the default value
// in the same way that happened to Gon after the events
// of the end of the Chimera Ants arc.
func (nh *NenHexagon) ResetCategory() int {
	nh.currHexValue = nenHexagon[nh.nenCategoryName]
	return nh.currHexValue
}

func (nh *NenHexagon) GetCategoryName() enum.CategoryName {
	return nh.nenCategoryName
}

func (nh *NenHexagon) GetCurrHexValue() int {
	return nh.currHexValue
}
