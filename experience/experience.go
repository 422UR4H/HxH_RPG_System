package experience

type Experience struct {
	expTable ExpTable
	Points   int
	Level    int
}

func NewExperience(expTable ExpTable) *Experience {
	return &Experience{expTable: expTable}
}

func (e *Experience) GetBaseExpByLvl(lvl int) int {
	return e.expTable.GetBaseExpByLvl(lvl)
}

func (e *Experience) GetAggregateExpByLvl(lvl int) int {
	return e.expTable.GetAggregateExpByLvl(lvl)
}

func (e *Experience) GetCurrentExp() int {
	return e.Points - e.expTable.GetAggregateExpByLvl(e.Level)
}

func (e *Experience) GetExpToEvolve() int {
	return e.GetAggregateExpByLvl(e.Level+1) - e.GetCurrentExp()
}

func (e *Experience) getLvlByExp() int {
	return e.expTable.GetLvlByExp(e.Points)
}

func (e *Experience) upgrade() {
	e.Level = e.getLvlByExp()
}

// return level diff
func (e *Experience) IncreasePoints(exp int) int {
	e.Points += exp

	diff := e.getLvlByExp() - e.Level
	if diff != 0 {
		e.upgrade()
	}
	return diff
}

func (e *Experience) Clone() *Experience {
	return NewExperience(e.expTable)
}
