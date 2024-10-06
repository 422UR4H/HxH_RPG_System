package ability

import exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"

type Talent struct {
	exp exp.Experience
}

func (t *Talent) NewTalent(exp exp.Experience) *Talent {
	return &Talent{exp: exp}
}

func (t *Talent) GetExpPoints() int {
	return t.exp.GetPoints()
}

func (t *Talent) GetLevel() int {
	return t.exp.GetLevel()
}
