package service

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// RoundOrchestrator is a stateless domain service that manages the Round/Turn
// lifecycle: extracting actions from the queue, creating and closing Turns,
// attaching reactions, and closing Rounds.
// It holds no state. All state lives in the Round entity passed as a parameter.
type RoundOrchestrator struct{}

// NextAction closes any open Turn, extracts the highest-priority Action from
// the queue, creates a new Turn, appends it to the Round, and returns it.
func (ro RoundOrchestrator) NextAction(r *round.Round, q *action.PriorityQueue) (*turn.Turn, error) {
	if r.HasOpenTurn() {
		r.CurrentTurn().Close(time.Now())
	}
	next := q.ExtractMax()
	if next == nil {
		return nil, ErrQueueEmpty
	}
	t := turn.NewTurn(*next)
	r.AppendTurn(t)
	return t, nil
}

// PullAction closes any open Turn, extracts the Action with the given UUID from
// the queue, creates a new Turn, and appends it to the Round.
func (ro RoundOrchestrator) PullAction(r *round.Round, q *action.PriorityQueue, id uuid.UUID) (*turn.Turn, error) {
	if r.HasOpenTurn() {
		r.CurrentTurn().Close(time.Now())
	}
	next := q.ExtractByID(id)
	if next == nil {
		return nil, ErrActionNotFound
	}
	t := turn.NewTurn(*next)
	r.AppendTurn(t)
	return t, nil
}

// CloseTurn sets finishedAt on the current Turn and returns it.
// Returns nil if the Round has no turns.
func (ro RoundOrchestrator) CloseTurn(r *round.Round, at time.Time) *turn.Turn {
	t := r.CurrentTurn()
	if t == nil {
		return nil
	}
	t.Close(at)
	return t
}

// CloseTurnErr is identical to CloseTurn but returns ErrNoCurrentTurn when the
// Round has no turns yet.
func (ro RoundOrchestrator) CloseTurnErr(r *round.Round, at time.Time) (*turn.Turn, error) {
	t := r.CurrentTurn()
	if t == nil {
		return nil, ErrNoCurrentTurn
	}
	t.Close(at)
	return t, nil
}

// CloseRound sets finishedAt on the Round and returns it.
func (ro RoundOrchestrator) CloseRound(r *round.Round, at time.Time) *round.Round {
	r.Close(at)
	return r
}

// AttachReaction validates that the reaction targets the current Turn's Action,
// then appends it.
func (ro RoundOrchestrator) AttachReaction(r *round.Round, reaction *action.Action) error {
	t := r.CurrentTurn()
	if t == nil {
		return ErrNoCurrentTurn
	}
	act := t.GetAction()
	if act.GetID() != reaction.ReactToID {
		return ErrReactionNotCompatible
	}
	t.AddReaction(reaction)
	return nil
}

// ChangeMode toggles the Round mode between Free and Race.
// TODO: create and finish Initiative to continue here
func (ro RoundOrchestrator) ChangeMode(r *round.Round, initiative *action.Initiative) {
	r.ToggleMode()
}
