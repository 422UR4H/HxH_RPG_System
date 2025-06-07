package sheet

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type TalentByCategorySet struct {
	categories      map[enum.CategoryName]bool
	initialHexValue *int
}

func NewTalentByCategorySet(
	categories map[enum.CategoryName]bool,
	initialHexValue *int,
) (*TalentByCategorySet, error) {

	activeCategoryCount := getActiveCategoryCount(categories)
	if activeCategoryCount == 0 {
		return nil, ErrInvalidActiveCategoryCount
	}
	return &TalentByCategorySet{
		categories:      categories,
		initialHexValue: initialHexValue,
	}, nil
}

const BASE_TALENT_LVL = 20

func (t *TalentByCategorySet) GetTalentLvl() int {
	activeCategoryCount := getActiveCategoryCount(t.categories)

	bonus := activeCategoryCount - 1
	if t.initialHexValue == nil {
		bonus *= 2

		if bonus == 0 {
			bonus = 1
		}
	}
	return BASE_TALENT_LVL + bonus
}

func getActiveCategoryCount(categories map[enum.CategoryName]bool) int {
	var activeCategoryCount int
	for _, isActive := range categories {
		if isActive {
			activeCategoryCount++
		}
	}
	return activeCategoryCount
}

func (t *TalentByCategorySet) GetCategories() map[enum.CategoryName]bool {
	return t.categories
}

func (t *TalentByCategorySet) GetInitialHexValue() *int {
	return t.initialHexValue
}
