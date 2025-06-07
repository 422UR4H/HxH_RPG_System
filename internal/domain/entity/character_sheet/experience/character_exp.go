package experience

type CharacterExp struct {
	IEndCascadeUpgrade
	points int
	exp    Exp
}

func NewCharacterExp(exp Exp) *CharacterExp {
	return &CharacterExp{exp: exp}
}

func (ce *CharacterExp) EndCascadeUpgrade(values *UpgradeCascade) {
	ce.exp.IncreasePoints(values.GetExp())
	values.CharacterExp = ce
}

func (ce *CharacterExp) IncreaseCharacterPoints(pts int) {
	ce.points += pts
}

func (ce *CharacterExp) GetCharacterPoints() int {
	return ce.points
}

func (ce *CharacterExp) GetNextLvlBaseExp() int {
	return ce.exp.GetNextLvlBaseExp()
}

func (ce *CharacterExp) GetNextLvlAggregateExp() int {
	return ce.exp.GetNextLvlAggregateExp()
}

func (ce *CharacterExp) GetCurrentExp() int {
	return ce.exp.GetCurrentExp()
}

func (ce *CharacterExp) GetExpPoints() int {
	return ce.exp.GetPoints()
}

func (ce *CharacterExp) GetLevel() int {
	return ce.exp.GetLevel()
}
