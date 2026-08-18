package round

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

type Round struct {
	id            uuid.UUID
	mode          enum.RoundMode
	turns         []*turn.Turn
	masterActions []action.MasterAction //nolint:unused // WIP: match system under development // think more about this field and its usage
	events        []GameEvent
	// prices are the frozen round prices, one per bar. An absent entry means that bar has
	// not priced yet — which is also what a Free round looks like all the way through, since
	// Free has no price at all.
	prices     map[action.Bar]int
	createdAt  time.Time
	finishedAt *time.Time
}

func NewRound(mode enum.RoundMode) *Round {
	return &Round{
		id:        uuid.New(),
		mode:      mode,
		turns:     []*turn.Turn{},
		events:    []GameEvent{},
		prices:    make(map[action.Bar]int),
		createdAt: time.Now(),
	}
}

func ReconstructRound(id uuid.UUID, mode enum.RoundMode, createdAt time.Time) *Round {
	return &Round{
		id:        id,
		mode:      mode,
		turns:     []*turn.Turn{},
		events:    []GameEvent{},
		prices:    make(map[action.Bar]int),
		createdAt: createdAt,
	}
}

func (r *Round) GetID() uuid.UUID {
	return r.id
}

func (r *Round) GetCreatedAt() time.Time {
	return r.createdAt
}

func (r *Round) AppendTurn(t *turn.Turn) {
	r.turns = append(r.turns, t)
}

func (r *Round) CurrentTurn() *turn.Turn {
	if len(r.turns) == 0 {
		return nil
	}
	return r.turns[len(r.turns)-1]
}

func (r *Round) HasOpenTurn() bool {
	t := r.CurrentTurn()
	return t != nil && t.GetFinishedAt() == nil
}

func (r *Round) Close(at time.Time) {
	r.finishedAt = &at
}

func (r *Round) GetFinishedAt() *time.Time {
	return r.finishedAt
}

func (r *Round) ToggleMode() {
	if r.mode == enum.Race {
		r.mode = enum.Free
	} else {
		r.mode = enum.Race
	}
}

func (r *Round) GetMode() enum.RoundMode {
	return r.mode
}

// Price returns the frozen price of a bar, and whether it froze at all.
func (r *Round) Price(bar action.Bar) (int, bool) {
	p, ok := r.prices[bar]
	return p, ok
}

// FreezePrice fixes a bar's price for the rest of the round. The first call wins; every
// later one is ignored.
//
// The price is fixed when the first action on that bar is chosen to open — the smallest speed
// among the actions pending at that moment — and it does not move again. A slower action
// arriving later does NOT re-price the round: that would make the same round charge different
// people different amounts, or reorder what has already been played. Such an action simply
// does not reach the price, sits the round out, and carries its full roll into the next one.
func (r *Round) FreezePrice(bar action.Bar, price int) {
	if r.prices == nil {
		r.prices = make(map[action.Bar]int)
	}
	if _, frozen := r.prices[bar]; frozen {
		return
	}
	r.prices[bar] = price
}

// HasOpenedAction reports whether an action with this ID has already been given a turn in
// this round. The dependency edge of a combined action reads it: the attack half waits for
// the move half to open.
func (r *Round) HasOpenedAction(id uuid.UUID) bool {
	for _, t := range r.turns {
		act := t.GetAction()
		if (&act).GetID() == id {
			return true
		}
	}
	return false
}

// SetMode sets the round regime outright. ToggleMode stays for the paths that flip blindly;
// this one is what an explicit master request needs, and it is idempotent.
func (r *Round) SetMode(mode enum.RoundMode) {
	r.mode = mode
}
