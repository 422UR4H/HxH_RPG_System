package ability

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"

type Talent struct {
	exp experience.Exp
}

func NewTalent(exp experience.Exp) *Talent {
	return &Talent{exp: exp}
}

func (t *Talent) InitWithLvl(lvl int) {
	aggregateExp := t.exp.GetAggregateExpByLvl(lvl)
	t.IncreaseExp(aggregateExp)
}

func (t *Talent) IncreaseExp(value int) int {
	return t.exp.IncreasePoints(value)
}

func (t *Talent) GetNextLvlAggregateExp() int {
	return t.exp.GetNextLvlAggregateExp()
}

func (t *Talent) GetNextLvlBaseExp() int {
	return t.exp.GetNextLvlBaseExp()
}

func (t *Talent) GetCurrentExp() int {
	return t.exp.GetCurrentExp()
}

func (t *Talent) GetExpPoints() int {
	return t.exp.GetPoints()
}

func (t *Talent) GetLevel() int {
	return t.exp.GetLevel()
}
