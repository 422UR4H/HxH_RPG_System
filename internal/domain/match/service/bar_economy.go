package service

// BarEconomy is the arithmetic of one resource bar, for one character, in one round.
// Stateless: every number it needs arrives as a parameter, so it is trivially testable and
// the state stays where it belongs, in match.CharacterStatus.
//
// Integers in, exact fractions out. A rolled speed is a skill value plus dice and a price is
// one of those speeds, so both are whole; everything DERIVED — the mean, the key, the balance,
// the carry — is float64 and keeps its fraction. Rounding would be a policy the rules never
// asked for, and the error would compound across rounds through the carry.
//
// Vocabulary, fixed by docs/dev/match/combat-engine.md:
//
//	carry — the balance that crossed over from the previous round, credit or debt
//	acted — the speeds of this character's actions on this bar that have ALREADY OPENED,
//	        in order. An action still sitting in the queue is not in here.
//	speed — the speed a pending action entered with, on this bar
//	price — the round price of this bar: the smallest speed among the actions pending on it
//	        when it froze. Equal for everyone, and also the ceiling on the carry-over.
type BarEconomy struct{}

// Mean is the round's average speed on this bar, exact.
func (BarEconomy) Mean(acted []int) float64 {
	if len(acted) == 0 {
		return 0
	}
	sum := 0
	for _, s := range acted {
		sum += s
	}
	return float64(sum) / float64(len(acted))
}

// Balance is what is left after paying for the actions that have already acted.
//
//	carry + mean(acted) − len(acted) × price
func (e BarEconomy) Balance(carry float64, acted []int, price int) float64 {
	return carry + e.Mean(acted) - float64(len(acted)*price)
}

// Key is where a pending action sits in the queue. It ORDERS ONLY — it never decides whether
// the action happens. See IsEligible.
//
//	carry + mean(acted ++ [speed]) − len(acted) × price
//
// The pending action's own speed is inside the average, which is why sending a second, slower
// action delays it within the round: the average drops and the key drops with it.
func (e BarEconomy) Key(carry float64, acted []int, speed, price int) float64 {
	withPending := make([]int, 0, len(acted)+1)
	withPending = append(withPending, acted...)
	withPending = append(withPending, speed)
	return carry + e.Mean(withPending) - float64(len(acted)*price)
}

// IsEligible is the gate: whether this pending action happens in this round at all.
//
// THERE ARE TWO GATES, AND NEITHER OF THEM IS THE KEY:
//
//   - First action of the round — the BAR reaches the price: carry + speed >= price.
//     Measured on the bar and not on the raw roll, so standing credit can rescue a bad roll.
//     With no carry (round 0) the two readings coincide.
//   - Second action onward — the LEFTOVER of the ones that already acted reaches the price.
//     Note what this means: the right to act again is decided BEFORE the new die falls, and a
//     bad roll does not revoke it. It only makes the action cost more afterwards, by dragging
//     the average down. In the canonical example p2's second action enters at key 9, below the
//     price of 11, and happens anyway.
//
// Using the key as the gate loses exactly that action and breaks the round's published order.
func (e BarEconomy) IsEligible(carry float64, acted []int, speed, price int) bool {
	if len(acted) == 0 {
		return carry+float64(speed) >= float64(price)
	}
	return e.Balance(carry, acted, price) >= float64(price)
}

// CloseBalance is what this bar carries into the next round, ceiling applied.
//
// The ceiling is the round price — never configurable, because it is what makes standing
// still stop paying after one round instead of compounding forever.
//
// A character who sent nothing carries the floor, which on this bar is the same number as the
// ceiling. That is deliberate, not a coincidence: standing still trades an action for time,
// and the trade is worth exactly one round's price. A character in debt recovers toward it
// rather than jumping to it.
func (e BarEconomy) CloseBalance(carry float64, acted []int, price int) float64 {
	balance := carry + float64(price)
	if len(acted) > 0 {
		balance = e.Balance(carry, acted, price)
	}
	if balance > float64(price) {
		return float64(price)
	}
	return balance
}
