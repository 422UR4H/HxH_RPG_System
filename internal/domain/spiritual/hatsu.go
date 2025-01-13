package spiritual

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/experience"
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

func (h *Hatsu) CascadeUpgrade(exp int) {
	h.exp.IncreasePoints(exp)
	h.abilityExp.CascadeUpgrade(exp)
}

func (h *Hatsu) IncreaseExp(points int, name enum.CategoryName) (int, error) {
	category, err := h.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to increase exp")
	}
	return category.CascadeUpgradeTrigger(points), nil
}

func (h *Hatsu) Get(name enum.CategoryName) (NenCategory, error) {
	if category, ok := h.categories[name]; ok {
		return category, nil
	}
	return NenCategory{}, fmt.Errorf("category %s not found", name.String())
}

func (h *Hatsu) GetPrinciplePower() int {
	return h.GetLevel() + h.abilityExp.GetLevel()/2
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

func (h *Hatsu) GetLevelOf(name enum.CategoryName) (int, error) {
	category, err := h.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s %s", err, "failed to get level of", name.String())
	}
	return category.GetLevel(), nil
}

func (h *Hatsu) GetLevel() int {
	return h.exp.GetLevel()
}
