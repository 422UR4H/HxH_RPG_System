package action

// RollAttempts holds BOTH dice sets, rolled together the moment the action or reaction
// arrives.
//
// Advantage means rolling the set twice and keeping the better one — but the master can
// grant advantage *after* the dice have already fallen, and the master never re-rolls a
// player's die. Rolling both sets up front is the only shape that satisfies both: a later
// edit changes which set is read, never what was rolled. On a neutral bias, Secondary is
// simply never read.
//
// It lives in this package, not in the domain service, so a RollCheck can hold it:
// service imports action, never the reverse.
type RollAttempts struct {
	Primary   []int
	Secondary []int
}

// IsEmpty reports whether the dice have not fallen yet. It is what keeps the roll-once
// rule honest: whoever rolls checks this first and never overwrites a set that already
// landed.
func (a RollAttempts) IsEmpty() bool {
	return len(a.Primary) == 0 && len(a.Secondary) == 0
}
