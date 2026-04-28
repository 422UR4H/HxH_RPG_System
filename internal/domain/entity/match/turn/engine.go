package turn

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
)

// REFATORAR CONSIDERANDO ROUND != TURN (add coast field)
type Engine struct {
	// *Turn
	mode      enum.TurnMode
	action    action.Action
	reactions action.PriorityQueue
	coast     int
}

func NewEngine(mode enum.TurnMode, action action.Action, coast int) *Engine {
	return &Engine{
		mode:   mode,
		action: action,
		coast:  coast,
	}
}

// AttachReaction adds a reaction to the current turn below the last action
func (e *Engine) AttachReaction(reaction *action.Action) error {
	turn := e.currentTurn
	action, err := turn.GetLastAction()
	if err != nil {
		return err
	}
	if action.GetID() != reaction.ReactToID {
		return ErrReactionNotCompatible
	}

	turn.AddAction(reaction)
	return nil
}

// CloseTurn finalizes the current turn and starts a new one with the same mode,
// returning the closed turn.
// Called by when the turn ends
func (e *Engine) CloseTurn() *Turn {
	// targetTurn := e.currentTurn

	newTurn := NewTurn(e.mode)

	return targetTurn
}

// TODO: create and finish Initiative to continue here
func (e *Engine) ChangeMode(initiative *action.Initiative) {
	if e.mode == enum.Race {
		e.mode = enum.Free
	} else {
		e.mode = enum.Race
	}

	if initiative != nil {
		if e.mode == enum.Free {
			// TODO: throws domain error
			return
		}
		// TODO: thread initiative
	}

	// e.switchTurn()
	*e.closeTurnTriggered = true
	// return turn?
}

func (e *Engine) GetAction() action.Action {
	return e.action
}

func (e *Engine) GetCurrentAction() *action.Action {
	return e.currentAction
}
