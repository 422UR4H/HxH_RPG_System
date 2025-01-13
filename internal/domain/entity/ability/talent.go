package ability

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"

type Talent struct {
	exp experience.Exp
}

func NewTalent(exp experience.Exp) *Talent {
	return &Talent{exp: exp}
}

func (t *Talent) GetExpPoints() int {
	return t.exp.GetPoints()
}

func (t *Talent) GetLevel() int {
	return t.exp.GetLevel()
}
