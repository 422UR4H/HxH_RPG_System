package status

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
)

const HP_BASE_VALUE = 20

type HealthPoints struct {
	physicals  ability.IAbility
	resistance attribute.IGameAttribute
	vitality   skill.ISkill
	*Bar
}

// TODO: implement one for each status bar (hp, sp, ap)
// cada status terá seu próprio construtor por onde serão injetadas
// as referências utilizadas para calcular seu valor máximo
// cada status também terá sua própria implementação de upgrade
// onde as funções de refs injetadas serão utilizadas para o cálculo
func NewHealthPoints(
	physicals ability.IAbility,
	resistance attribute.IGameAttribute,
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
	// Min = generateStatus.GetLvl();
	// TODO: check how the buff interferes here
	coeff := float64(hp.vitality.GetLevel() + hp.resistance.GetValue())
	bonus := hp.physicals.GetBonus()
	maxVal := int(coeff*bonus) + HP_BASE_VALUE
	if hp.curr == hp.max {
		hp.curr = maxVal
	}
	// TODO: Implement else case (ex.: hp.current == hp.max - 1 -> threat % case)
	hp.max = maxVal
}
