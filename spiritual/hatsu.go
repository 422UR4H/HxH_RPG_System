package spiritual

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
	"github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type Hatsu struct {
	exp        experience.Exp
	abilityExp experience.ICascadeUpgrade
	categories map[enum.CategoryName]NenCategory
}

func NewHatsu(
	exp experience.Exp,
	abilityExp experience.ICascadeUpgrade,
	categories map[enum.CategoryName]NenCategory,
) *Hatsu {
	return &Hatsu{
		exp:        exp,
		abilityExp: abilityExp,
		categories: make(map[enum.CategoryName]NenCategory),
	}
}

func (h *Hatsu) Init(categories map[enum.CategoryName]NenCategory) {
	if len(h.categories) > 0 {
		fmt.Println("hatsu already initialized with categories")
		return
	}
	h.categories = categories
}

func (h *Hatsu) CascadeUpgrade(exp int) int {
	diff := h.exp.IncreasePoints(exp)
	h.abilityExp.CascadeUpgrade(exp)
	return diff
}

func (h *Hatsu) IncreaseExp(points int, name enum.CategoryName) error {
	category, err := h.Get(name)
	if err != nil {
		return fmt.Errorf("%w: %s", err, "failed to increase exp")
	}
	category.CascadeUpgradeTrigger(points)
	return nil
}

func (h *Hatsu) Get(name enum.CategoryName) (NenCategory, error) {
	if category, ok := h.categories[name]; ok {
		return category, nil
	}
	return NenCategory{}, errors.New("category not found")
}

func (h *Hatsu) GetExpPointsOf(name enum.CategoryName) (int, error) {
	category, err := h.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s %s", err, "failed to get exp of", name.String())
	}
	return category.GetExpPoints(), nil
}

func (h *Hatsu) GetExpPoints() int {
	return h.exp.GetPoints()
}

func (h *Hatsu) GetLvlOf(name enum.CategoryName) (int, error) {
	category, err := h.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s %s", err, "failed to get level of", name.String())
	}
	return category.GetLvl(), nil
}

func (h *Hatsu) GetLvl() int {
	return h.exp.GetLevel()
}
