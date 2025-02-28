package status

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
)

const BASE_VALUE = 20

type Bar struct {
	ability   ability.IAbility
	attribute attribute.IGameAttribute
	skill     skill.ISkill

	min  int
	curr int
	max  int
}

// TODO: implement one for each status bar (hp, sp, ap)
// cada status terá seu próprio construtor por onde serão injetadas
// as referências utilizadas para calcular seu valor máximo
// cada status também terá sua própria implementação de upgrade
// onde as funções de refs injetadas serão utilizadas para o cálculo
func NewStatusBar(
	ability ability.IAbility,
	attribute attribute.IGameAttribute,
	skill skill.ISkill,
) *Bar {
	bar := &Bar{
		ability:   ability,
		attribute: attribute,
		skill:     skill,
	}
	bar.Upgrade()

	return bar
}

func (b *Bar) IncreaseAt(value int) int {
	temp := b.curr + value
	b.curr = min(temp, b.max)
	return b.curr
}

func (b *Bar) DecreaseAt(value int) int {
	temp := b.curr - value
	b.curr = max(temp, b.min)
	return b.curr
}

func (b *Bar) Upgrade() {
	// old formula
	// this.hpMax = (this.getProHp() + this.modCon + this.coefHp) * this.lvl + this.valCon + Ficha.getHP_INICIAL();
	// new formula -> attrBonus * (skLvl + attrLvl + attrPoints)

	// TODO: Implement Min for hit_points
	// Min = generateStatus.GetLvl();
	// TODO: check how the buff interferes here
	coeff := float64(b.skill.GetLevel() + b.attribute.GetValue())
	bonus := b.ability.GetBonus()
	maxVal := int(coeff*bonus) + BASE_VALUE
	if b.curr == b.max {
		b.curr = maxVal
	}
	// TODO: Implement else case (ex.: b.current == b.max - 1 -> threat % case)
	b.max = maxVal
}

func (b *Bar) GetMin() int {
	return b.min
}

func (b *Bar) GetCurrent() int {
	return b.curr
}

func (b *Bar) GetMax() int {
	return b.max
}
