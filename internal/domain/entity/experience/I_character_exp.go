package experience

type ICharacterExp interface {
	IEndCascadeUpgrade
	IncreaseCharacterPoints(int)
	GetCharacterPoints() int
}
