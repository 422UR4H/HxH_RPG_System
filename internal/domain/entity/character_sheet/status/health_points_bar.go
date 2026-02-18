package status

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
)

const HP_BASE_VALUE = 20

type HealthPoints struct {
	physicals  ability.IAbility
	resistance attribute.IDistributableAttribute
	vitality   skill.ISkill
	*Bar
}

func NewHealthPoints(
	physicals ability.IAbility,
	resistance attribute.IDistributableAttribute,
	vitality skill.ISkill,
) *HealthPoints {
	bar := &HealthPoints{
		physicals:  physicals,
		resistance: resistance,
		vitality:   vitality,
		Bar:        &Bar{},
	}
	bar.Upgrade()

	return bar
}

func (hp *HealthPoints) Upgrade() {
	// old formula
	// this.hpMax = (this.getProHp() + this.modCon + this.coefHp) * this.lvl + this.valCon + Ficha.getHP_INICIAL();
	// new formula -> attrBonus * (skLvl + attrLvl + attrPoints)

	// TODO: Implement Min for hit_points
	// Min = -generateStatus.GetLvl();
	// TODO: check how the buff interferes here
	coeff := float64(hp.vitality.GetLevel() + hp.resistance.GetValue())
	bonus := hp.physicals.GetBonus()
	maxVal := HP_BASE_VALUE + int(coeff*bonus)

	// if character is fully healed, upgrade current hp to new max
	if hp.curr == hp.max {
		hp.curr = maxVal
	}
	// TODO: Implement else case (ex.: hp.current == hp.max - 1 -> threat % case)
	hp.max = maxVal
}
