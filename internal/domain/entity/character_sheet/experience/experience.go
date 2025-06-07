package experience

type Exp struct {
	expTable *ExpTable
	points   int
	level    int
}

func NewExperience(expTable *ExpTable) *Exp {
	return &Exp{expTable: expTable, points: 0, level: 0}
}

func (e *Exp) GetPoints() int {
	return e.points
}

func (e *Exp) GetLevel() int {
	return e.level
}

func (e *Exp) GetBaseExpByLvl(lvl int) int {
	return e.expTable.GetBaseExpByLvl(lvl)
}

func (e *Exp) GetAggregateExpByLvl(lvl int) int {
	return e.expTable.GetAggregateExpByLvl(lvl)
}

func (e *Exp) GetCurrentExp() int {
	return e.points - e.GetAggregateExpByLvl(e.level)
}

func (e *Exp) GetExpToEvolve() int {
	return e.GetAggregateExpByLvl(e.level+1) - e.GetCurrentExp()
}

func (e *Exp) GetNextLvlBaseExp() int {
	return e.GetBaseExpByLvl(e.level + 1)
}

func (e *Exp) GetNextLvlAggregateExp() int {
	return e.GetAggregateExpByLvl(e.level + 1)
}

func (e *Exp) getLvlByExp() int {
	return e.expTable.GetLvlByExp(e.points)
}

func (e *Exp) upgrade() {
	e.level = e.getLvlByExp()
}

// return level diff
func (e *Exp) IncreasePoints(exp int) int {
	e.points += exp

	diff := e.getLvlByExp() - e.level
	if diff != 0 {
		e.upgrade()
	}
	return diff
}

func (e *Exp) Clone() *Exp {
	return NewExperience(e.expTable)
}
