package match

// ResourceBar is one of the two clocks a character runs on: actionSpeed (attack, item,
// ability) and moveSpeed (shift, dash, leap, roll). They have independent prices but
// share a single clock, and both live on the same scale — skill + the dice set — so the
// engine compares value against value with no conversion.
//
// Balance is the standing credit or debt, which carries across rounds. Speeds is the
// history of the speeds rolled on this bar during the current round; the round-closing
// formula averages it.
//
// Phase 1 defines only the shape. The arithmetic — average, round price, ceiling,
// carry-over — is Phase 3.
type ResourceBar struct {
	Balance int
	Speeds  []int
}

// RecordSpeed appends a speed rolled on this bar during the current round.
func (b *ResourceBar) RecordSpeed(speed int) {
	b.Speeds = append(b.Speeds, speed)
}

// ResetRound clears the round's speed history. The balance is deliberately untouched:
// it is the carry-over into the next round, as credit or as debt.
func (b *ResourceBar) ResetRound() {
	b.Speeds = nil
}
