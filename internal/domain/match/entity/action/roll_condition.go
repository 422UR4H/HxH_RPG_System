package action

type RollCondition struct {
	// Bias: 1 = advantage, -1 = disadvantage, can accumulate (0 = neutral)
	Bias        int
	Modifier    int
	Description string
}
