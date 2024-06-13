package experience

type CharacterExp struct {
	ITriggerCascadeExp
	exp Experience
}

func NewCharacterExp(exp Experience) *CharacterExp {
	return &CharacterExp{exp: exp}
}

func (ce *CharacterExp) TriggerEndUpgrade(exp int) {
	ce.exp.IncreasePoints(exp)
}
