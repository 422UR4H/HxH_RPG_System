package ability

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type Ability struct {
	exp     experience.Exp
	charExp experience.IEndCascadeUpgrade
}

func NewAbility(
	exp experience.Exp, charExp experience.IEndCascadeUpgrade,
) *Ability {
	return &Ability{exp: exp, charExp: charExp}
}

func (a *Ability) GetHalfLvl() float64 {
	return float64(a.exp.GetLevel()) / 2.0
}

// talvez eu deva subir a exp apenas metrica,
// mas subir o lvl para o characterPower que desce pras skills
// melhorando os treinos e testes
func (a *Ability) CascadeUpgrade(exp int) {
	a.exp.IncreasePoints(exp)
	a.charExp.EndCascadeUpgrade(exp)
}

func (a *Ability) GetExpPoints() int {
	return a.exp.GetPoints()
}

func (a *Ability) GetLevel() int {
	return a.exp.GetLevel()
}
