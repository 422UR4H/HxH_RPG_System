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
	// Balance is the standing credit or debt, and it is a float64 on purpose: it is DERIVED
	// from an average that rarely divides evenly, and it crosses rounds, so truncating it
	// would compound an error the rules never asked for.
	Balance float64
	// Speeds is the ordered list of the speeds that ACTUALLY ACTED on this bar during the current
	// round — a speed is appended when the master opens that action, never when it is enqueued. An
	// action still waiting in the queue is not in here, which is exactly what lets one that never
	// reaches the price roll over to the next round with its full roll and nothing to unwind.
	//
	// A combined action appends to BOTH bars, because it charges both.
	Speeds []int
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
