package ability

import "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"

type Talent struct {
	exp experience.Exp
}

func (t *Talent) NewTalent(exp experience.Exp) *Talent {
	return &Talent{exp: exp}
}

func (t *Talent) GetExpPoints() int {
	return t.exp.GetPoints()
}

func (t *Talent) GetLevel() int {
	return t.exp.GetLevel()
}
