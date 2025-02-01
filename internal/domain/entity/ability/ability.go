package ability

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type Ability struct {
	exp     experience.Exp
	charExp experience.ICharacterExp
}

func NewAbility(
	exp experience.Exp, charExp experience.ICharacterExp,
) *Ability {
	return &Ability{exp: exp, charExp: charExp}
}

func (a *Ability) GetBonus() float64 {
	pts := float64(a.charExp.GetCharacterPoints())
	lvl := float64(a.exp.GetLevel())
	return (pts + lvl) / 2.0
}

// talvez eu deva subir a exp apenas para metrica,
// mas subir o lvl para o characterPower que desce pras skills
// melhorando os treinos e testes
func (a *Ability) CascadeUpgrade(exp int) {
	diff := a.exp.IncreasePoints(exp)
	a.charExp.EndCascadeUpgrade(exp)

	if diff > 0 {
		a.charExp.IncreaseCharacterPoints(diff)
	}
}

func (a *Ability) GetExpPoints() int {
	return a.exp.GetPoints()
}

func (a *Ability) GetLevel() int {
	return a.exp.GetLevel()
}
