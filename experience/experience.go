package experience

type Experience struct {
	expTable ExpTable
	points   int
	level    int
}

func NewExperience(expTable ExpTable) *Experience {
	return &Experience{expTable: expTable}
}

func (e *Experience) GetPoints() int {
	return e.points
}

func (e *Experience) GetLevel() int {
	return e.level
}

func (e *Experience) GetBaseExpByLvl(lvl int) int {
	return e.expTable.GetBaseExpByLvl(lvl)
}

func (e *Experience) GetAggregateExpByLvl(lvl int) int {
	return e.expTable.GetAggregateExpByLvl(lvl)
}

func (e *Experience) GetCurrentExp() int {
	return e.points - e.expTable.GetAggregateExpByLvl(e.level)
}

func (e *Experience) GetExpToEvolve() int {
	return e.GetAggregateExpByLvl(e.level+1) - e.GetCurrentExp()
}

func (e *Experience) getLvlByExp() int {
	return e.expTable.GetLvlByExp(e.points)
}

func (e *Experience) upgrade() {
	e.level = e.getLvlByExp()
}

// return level diff
func (e *Experience) IncreasePoints(exp int) int {
	e.points += exp

	diff := e.getLvlByExp() - e.level
	if diff != 0 {
		e.upgrade()
	}
	return diff
}

func (e *Experience) Clone() *Experience {
	return NewExperience(e.expTable)
}
