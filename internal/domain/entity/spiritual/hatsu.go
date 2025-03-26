package spiritual

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type Hatsu struct {
	exp              experience.Exp
	ability          ability.IAbility
	categories       map[enum.CategoryName]NenCategory
	categoryPercents map[enum.CategoryName]float64
}

func NewHatsu(
	exp experience.Exp,
	ability ability.IAbility,
	categories map[enum.CategoryName]NenCategory,
	categoryPercents map[enum.CategoryName]float64,
) *Hatsu {
	return &Hatsu{
		exp:              exp,
		ability:          ability,
		categories:       make(map[enum.CategoryName]NenCategory),
		categoryPercents: categoryPercents,
	}
}

// TODO: handle error here
func (h *Hatsu) Init(categories map[enum.CategoryName]NenCategory) {
	if len(h.categories) > 0 {
		fmt.Println("hatsu already initialized with categories")
		return
	}
	h.categories = categories
}

func (h *Hatsu) SetCategoryPercents(
	categoryPercents map[enum.CategoryName]float64,
) error {

	if len(categoryPercents) != 6 {
		return ErrInvalidCategoryPercents
	}
	h.categoryPercents = categoryPercents
	return nil
}

func (h *Hatsu) CascadeUpgrade(values *experience.UpgradeCascade) {
	h.exp.IncreasePoints(values.GetExp())
	h.ability.CascadeUpgrade(values)

	values.Principles[enum.Hatsu] = experience.PrincipleCascade{
		Lvl:     h.GetLevel(),
		Exp:     h.GetCurrentExp(),
		TestVal: h.GetValueForTest(),
	}
}

func (h *Hatsu) IncreaseExp(
	values *experience.UpgradeCascade,
	name enum.CategoryName,
) error {
	category, err := h.Get(name)
	if err != nil {
		return fmt.Errorf("%w: %s", err, "failed to increase exp")
	}
	category.CascadeUpgradeTrigger(values)
	return nil
}

func (h *Hatsu) Get(name enum.CategoryName) (ICategory, error) {
	if category, ok := h.categories[name]; ok {
		return &category, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrCategoryNotFound, name.String())
}

func (h *Hatsu) GetValueForTest() int {
	return h.GetLevel() + int(h.ability.GetBonus())
}

func (h *Hatsu) GetNextLvlAggregateExpOf(name enum.CategoryName) (int, error) {
	category, err := h.Get(name)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s", err, "failed to get aggregate exp of next lvl")
	}
	return category.GetNextLvlAggregateExp(), nil
}

func (h *Hatsu) GetNextLvlBaseExpOf(name enum.CategoryName) (int, error) {
	category, err := h.Get(name)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s", err, "failed to get base exp of next lvl")
	}
	return category.GetNextLvlBaseExp(), nil
}

func (h *Hatsu) GetCurrentExpOf(name enum.CategoryName) (int, error) {
	category, err := h.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get current exp of")
	}
	return category.GetCurrentExp(), nil
}

func (h *Hatsu) GetExpPointsOf(name enum.CategoryName) (int, error) {
	category, err := h.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get exp of")
	}
	return category.GetExpPoints(), nil
}

func (h *Hatsu) GetLevelOf(name enum.CategoryName) (int, error) {
	category, err := h.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level of")
	}
	return category.GetLevel(), nil
}

func (h *Hatsu) GetCategoriesNextLvlAggregateExp() map[enum.CategoryName]int {
	expList := make(map[enum.CategoryName]int)
	for name, category := range h.categories {
		expList[name] = category.GetNextLvlAggregateExp()
	}
	return expList
}

func (h *Hatsu) GetCategoriesNextLvlBaseExp() map[enum.CategoryName]int {
	expList := make(map[enum.CategoryName]int)
	for name, category := range h.categories {
		expList[name] = category.GetNextLvlBaseExp()
	}
	return expList
}

func (h *Hatsu) GetCategoriesCurrentExp() map[enum.CategoryName]int {
	expList := make(map[enum.CategoryName]int)
	for name, category := range h.categories {
		expList[name] = category.GetCurrentExp()
	}
	return expList
}

func (h *Hatsu) GetCategoriesExpPoints() map[enum.CategoryName]int {
	expList := make(map[enum.CategoryName]int)
	for name, category := range h.categories {
		expList[name] = category.GetExpPoints()
	}
	return expList
}

func (h *Hatsu) GetCategoriesLevel() map[enum.CategoryName]int {
	lvlList := make(map[enum.CategoryName]int)
	for name, category := range h.categories {
		lvlList[name] = category.GetLevel()
	}
	return lvlList
}

func (h *Hatsu) GetCategoriesTestValue() map[enum.CategoryName]int {
	lvlList := make(map[enum.CategoryName]int)
	for name, category := range h.categories {
		lvlList[name] = category.GetValueForTest()
	}
	return lvlList
}

func (h *Hatsu) GetNextLvlAggregateExp() int {
	return h.exp.GetNextLvlAggregateExp()
}

func (h *Hatsu) GetNextLvlBaseExp() int {
	return h.exp.GetNextLvlBaseExp()
}

func (h *Hatsu) GetCurrentExp() int {
	return h.exp.GetCurrentExp()
}

func (h *Hatsu) GetExpPoints() int {
	return h.exp.GetPoints()
}

func (h *Hatsu) GetLevel() int {
	return h.exp.GetLevel()
}

func (h *Hatsu) GetCategoryPercents() map[enum.CategoryName]float64 {
	return h.categoryPercents
}

func (h *Hatsu) GetPercentOf(category enum.CategoryName) float64 {
	return h.categoryPercents[category]
}

func (h *Hatsu) GetAllCategories() map[enum.CategoryName]ICategory {
	categories := make(map[enum.CategoryName]ICategory)
	for name, category := range h.categories {
		categories[name] = &category
	}
	return categories
}
