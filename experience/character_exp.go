package experience

type CharacterExp struct {
	IEndCascadeUpgrade
	exp Exp
}

func NewCharacterExp(exp Exp) *CharacterExp {
	return &CharacterExp{exp: exp}
}

func (ce *CharacterExp) EndCascadeUpgrade(exp int) {
	ce.exp.IncreasePoints(exp)
}
