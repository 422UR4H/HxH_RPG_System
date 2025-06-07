package status

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
)

const AP_BASE_VALUE = 10

// TODO: resolve fields
type AuraPoints struct {
	// ability   ability.IAbility
	// attribute attribute.IGameAttribute
	skill skill.ISkill
	*Bar
}

func NewAuraPoints(
	// ability ability.IAbility,
	// attribute attribute.IGameAttribute,
	skill skill.ISkill,
) *AuraPoints {
	bar := &AuraPoints{
		// ability:   ability,
		// attribute: attribute,
		skill: skill,
		Bar:   &Bar{},
	}
	bar.Upgrade()

	return bar
}

func (hp *AuraPoints) Upgrade() {
	// TODO: review formula and update skill variable name
	// TODO: check how the buff interferes here
	skLvl := hp.skill.GetLevel() * 1000
	// bonus := hp.ability.GetBonus()
	maxVal := skLvl + AP_BASE_VALUE
	if hp.curr == hp.max {
		hp.curr = maxVal
	}
	// TODO: Implement else case (ex.: hp.current == hp.max - 1 -> threat % case)
	hp.max = maxVal
}
