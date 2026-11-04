package match

import "github.com/google/uuid"

// ResourceBar is one of the two clocks a character runs on: actionSpeed (attack, item,
// ability) and moveSpeed (shift, dash, leap, roll). They have independent prices but
// share a single clock, and both live on the same scale — skill + the dice set — so the
// engine compares value against value with no conversion.
//
// Balance is the standing credit or debt, which carries across rounds. Speeds is the
// history of the speeds rolled on this bar during the current round; the round-closing
// formula averages it.
//
// A speed can be SECRET: a reaction's speed is charged here when the reaction is attached, but
// only becomes public once the master opens that reaction or its turn closes (spec D2/F4).
// Speeds always holds every speed (the economy reads it); PublicSpeeds is the filtered view.
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
	// secret maps a reaction's id to its speed while that speed is not yet public. Nil until the
	// first secret is recorded.
	secret map[uuid.UUID]int
}

// RecordSpeed appends a speed rolled on this bar during the current round.
func (b *ResourceBar) RecordSpeed(speed int) {
	b.Speeds = append(b.Speeds, speed)
}

// RecordSecretSpeed appends a speed like RecordSpeed and remembers it, under id, as not yet public.
func (b *ResourceBar) RecordSecretSpeed(id uuid.UUID, speed int) {
	b.Speeds = append(b.Speeds, speed)
	if b.secret == nil {
		b.secret = map[uuid.UUID]int{}
	}
	b.secret[id] = speed
}

// Reveal makes the speed recorded under id public. Unknown ids are ignored.
func (b *ResourceBar) Reveal(id uuid.UUID) {
	delete(b.secret, id)
}

// RevealAll makes every secret speed public.
func (b *ResourceBar) RevealAll() {
	b.secret = nil
}

// PublicSpeeds is Speeds minus one occurrence of each still-secret speed. Occurrences are removed
// from the END, because secrets are recorded after the opened actions they follow. The result is
// a fresh slice and never aliases Speeds.
func (b *ResourceBar) PublicSpeeds() []int {
	out := make([]int, len(b.Speeds))
	copy(out, b.Speeds)
	for _, speed := range b.secret {
		for i := len(out) - 1; i >= 0; i-- {
			if out[i] == speed {
				out = append(out[:i], out[i+1:]...)
				break
			}
		}
	}
	return out
}

// ResetRound clears the round's speed history and its secrets. The balance is deliberately
// untouched: it is the carry-over into the next round, as credit or as debt.
func (b *ResourceBar) ResetRound() {
	b.Speeds = nil
	b.secret = nil
}
