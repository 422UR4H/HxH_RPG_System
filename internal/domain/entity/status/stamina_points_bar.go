package status

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
)

const SP_BASE_VALUE = 30

type StaminaPoints struct {
	spirituals ability.IAbility
	resistance attribute.IGameAttribute
	energy     skill.ISkill
	*Bar
}

func NewStaminaPoints(
	spirituals ability.IAbility,
	resistance attribute.IGameAttribute,
	energy skill.ISkill,
) *StaminaPoints {
	bar := &StaminaPoints{
		spirituals: spirituals,
		resistance: resistance,
		energy:     energy,
		Bar:        &Bar{},
	}
	bar.Upgrade()

	return bar
}

func (sp *StaminaPoints) Upgrade() {
	// TODO: check how the buff interferes here
	energyLvl := sp.energy.GetLevel()
	if energyLvl == 0 {
		energyLvl = 1
	}
	resistance := sp.resistance.GetValue()
	if resistance == 0 {
		resistance = 1
	}
	// TODO: Implement Min for stamina_points
	coeff := float64(energyLvl * resistance)
	bonus := sp.spirituals.GetBonus()
	maxVal := int(coeff*bonus) * SP_BASE_VALUE
	if sp.curr == sp.max {
		sp.curr = maxVal
	}
	// TODO: Implement else case (ex.: sp.current == sp.max - 1 -> threat % case)
	sp.max = maxVal
}
