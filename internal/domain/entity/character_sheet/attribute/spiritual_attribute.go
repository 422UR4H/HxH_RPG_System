package attribute

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type SpiritualAttribute struct {
	name    enum.AttributeName
	exp     experience.Exp
	ability ability.IAbility
	buff    *int
}

func NewSpiritualAttribute(
	name enum.AttributeName,
	exp experience.Exp,
	ability ability.IAbility,
	buff *int,
) *SpiritualAttribute {
	return &SpiritualAttribute{
		name: name, exp: exp, ability: ability, buff: buff,
	}
}

func (pa *SpiritualAttribute) CascadeUpgrade(values *experience.UpgradeCascade) {
	pa.exp.IncreasePoints(values.GetExp())
	pa.ability.CascadeUpgrade(values)

	values.Attributes[pa.name] = experience.AttributeCascade{
		Exp:   pa.GetExpPoints(),
		Lvl:   pa.GetLevel(),
		Power: pa.GetPower(),
	}
}

// TODO: validate if buff is added here
func (pa *SpiritualAttribute) GetPower() int {
	return pa.GetLevel() + int(pa.GetAbilityBonus()) + *pa.buff
}

func (pa *SpiritualAttribute) GetAbilityBonus() float64 {
	return pa.ability.GetBonus()
}

func (pa *SpiritualAttribute) GetNextLvlAggregateExp() int {
	return pa.exp.GetNextLvlAggregateExp()
}

func (pa *SpiritualAttribute) GetNextLvlBaseExp() int {
	return pa.exp.GetNextLvlBaseExp()
}

func (pa *SpiritualAttribute) GetCurrentExp() int {
	return pa.exp.GetCurrentExp()
}

func (pa *SpiritualAttribute) GetExpPoints() int {
	return pa.exp.GetPoints()
}

func (pa *SpiritualAttribute) GetLevel() int {
	return pa.exp.GetLevel()
}

func (pa *SpiritualAttribute) GetName() enum.AttributeName {
	return pa.name
}

func (pa *SpiritualAttribute) Clone(name enum.AttributeName, buff *int) *SpiritualAttribute {
	return NewSpiritualAttribute(name, *pa.exp.Clone(), pa.ability, buff)
}
