package status

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
)

const SP_COEF_VALUE = 10

type StaminaPoints struct {
	physicals  ability.IAbility
	resistance attribute.IDistributableAttribute
	energy     skill.ISkill
	*Bar
}

func NewStaminaPoints(
	physicals ability.IAbility,
	resistance attribute.IDistributableAttribute,
	energy skill.ISkill,
) *StaminaPoints {
	bar := &StaminaPoints{
		physicals:  physicals,
		resistance: resistance,
		energy:     energy,
		Bar:        &Bar{},
	}
	bar.Upgrade()

	return bar
}

func (sp *StaminaPoints) Upgrade() {
	// TODO: check how the buff interferes here
	// TODO: Implement Min for stamina_points
	coeff := float64(sp.energy.GetLevel() + sp.resistance.GetValue())
	bonus := sp.physicals.GetBonus()
	maxVal := SP_COEF_VALUE * int(coeff*bonus)

	// if character is fully rested, upgrade current sp to new max
	if sp.curr == sp.max {
		sp.curr = maxVal
	}
	// TODO: Implement else case (ex.: sp.current == sp.max - 1 -> threat % case)
	sp.max = maxVal
}
