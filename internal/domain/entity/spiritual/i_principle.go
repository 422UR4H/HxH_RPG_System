package spiritual

type IPrinciple interface {
	// GetPrinciplePower() int
	GetValueForTest() int
	GetNextLvlAggregateExp() int
	GetNextLvlBaseExp() int
	GetCurrentExp() int
	GetExpPoints() int
	GetLevel() int
}
