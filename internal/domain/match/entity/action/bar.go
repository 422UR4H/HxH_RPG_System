package action

// Bar names one of the two clocks a character runs on. They have independent prices but
// share a single clock, and both live on the same scale — skill + the dice set — so the
// engine compares value against value with no conversion.
type Bar string

const (
	BarAction Bar = "action" // attack, item, ability
	BarMove   Bar = "move"   // shift, dash, leap, roll
)

// Bars reports which clocks this action is paid from.
//
// A combined action — cait, arremetida, investida — is ONE action with Move and Attack both
// filled in, and it charges BOTH bars. It is not split into two actions and it does not open
// two turns: the master opens it once, it is one turn, and it happens at the time of its
// slower half. See combat-engine.md § Ações compostas.
//
// The action bar is always first, and the result is never empty: an action with nothing in it
// at all still belongs to the action bar, or the scheduler would have nothing to price it by.
//
// The one exception is a reaction — see the gate below.
func (a *Action) Bars() []Bar {
	// A reaction answers with its declared kind. This gate comes first because the three
	// escapes are shape-identical and priced differently — no inspection below could tell
	// them apart. It is also the only path that may answer empty.
	if a.ReactionKind != "" {
		return a.ReactionKind.Bars()
	}
	if a.Move == nil {
		return []Bar{BarAction}
	}
	if a.chargesActionBar() {
		return []Bar{BarAction, BarMove}
	}
	return []Bar{BarMove}
}

// chargesActionBar reports whether the action carries anything paid from the action bar, as
// opposed to being pure movement.
func (a *Action) chargesActionBar() bool {
	return a.Attack != nil || a.Defense != nil || a.Dodge != nil || a.Repel != nil ||
		a.Interact != nil || a.Feint != nil || len(a.Skills) > 0
}

// SpeedOn is the speed this action entered the round with on one bar. Both speeds are derived
// once, when the action arrives, and neither is ever re-rolled — a combined action keeps its
// actionSpeed AND its moveSpeed; what the combination changes is only when it happens.
//
// A bar the action does not charge reads 0, which no caller reaches: the scheduler only ever
// asks about the bars Bars() returned.
func (a *Action) SpeedOn(bar Bar) int {
	if bar == BarMove {
		if a.Move == nil {
			return 0
		}
		return a.Move.FinalSpeed
	}
	if !a.chargesActionBar() && a.Move != nil {
		return 0
	}
	return a.Speed.Result
}
