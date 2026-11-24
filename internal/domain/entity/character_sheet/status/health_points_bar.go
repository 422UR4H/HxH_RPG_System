package status

import (
	"math"

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

	// O mínimo da barra é negativo e por ficha: -⌊10 + nível de Vitality + valor de
	// Resistance + bônus de Físicos⌋ (spec §7 / §16.4 item 29).
	hp.min = -hp.NegativeMax()
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

// NegativeMax é o quanto o HP pode descer abaixo de zero antes da morte:
// ⌊10 + nível de Vitality + valor de Resistance + bônus de Físicos⌋ (spec §7).
func (hp *HealthPoints) NegativeMax() int {
	return int(math.Floor(10 +
		float64(hp.vitality.GetLevel()+hp.resistance.GetValue()) +
		hp.physicals.GetBonus()))
}

// DecreaseAt não tem piso: o personagem morto guarda o HP em que morreu (-40 continua -40).
// A morte (HP abaixo do mínimo) é decidida pela regra da partida, não pela barra.
func (hp *HealthPoints) DecreaseAt(value int) int {
	hp.curr -= value
	return hp.curr
}

// SetCurrent aceita qualquer valor até o máximo: o HP pode estar abaixo do mínimo.
func (hp *HealthPoints) SetCurrent(value int) error {
	if value > hp.max {
		return ErrInvalidValue
	}
	hp.curr = value
	return nil
}
