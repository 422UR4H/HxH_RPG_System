package turn

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/google/uuid"
)

type Engine struct {
	mode               enum.TurnMode
	actionQueue        ActionPriorityQueue
	turns              []*Turn
	currentTurn        *Turn
	currentAction      *action.Action
	closeTurnTriggered *bool // flag used to client persist the turn
}

// TODO: refatorar trocando closeTurnTriggered por um método que chame CloseTurn e faça o trigger, ou algo do tipo, para evitar que o cliente tenha acesso a esse flag
func NewEngine(actionQueue *[]*action.Action, turns *[]*Turn, closeTurnTriggered *bool) *Engine {
	if turns == nil {
		turns = &[]*Turn{NewTurn(enum.Free)}
	}
	aq := NewActionPriorityQueue(actionQueue)

	mode := enum.Free
	lenTurns := len(*turns)
	currentTurn := (*turns)[0]
	if lenTurns > 0 {
		currentTurn = (*turns)[lenTurns-1]
		mode = currentTurn.mode
	}

	return &Engine{
		mode:               mode,
		actionQueue:        aq,
		turns:              *turns,
		currentTurn:        currentTurn,
		currentAction:      nil,
		closeTurnTriggered: closeTurnTriggered,
	}
}

func (e *Engine) Add(action *action.Action) {
	e.actionQueue.Insert(action)
}

// NextAction move the action with the highest ActionSpeed in ActionQueue to
// currentTurn, set it as currentAction and return it
func (e *Engine) NextAction() *action.Action {
	nextAction := e.actionQueue.ExtractMax()
	return e.switchAction(nextAction)
}

// PullAction returns the action with the specified ID
func (e *Engine) PullAction(id uuid.UUID) *action.Action {
	nextAction := e.actionQueue.ExtractByID(id)
	if nextAction == nil {
		return nil
	}
	return e.switchAction(nextAction)
}

func (e *Engine) switchAction(nextAction *action.Action) *action.Action {
	if *e.closeTurnTriggered {
		e.CloseTurn()
	}
	e.currentTurn.AddAction(nextAction)
	e.currentAction = nextAction
	return nextAction
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
// Called by match.engine when the match ends OR before action switching (act open)
func (e *Engine) CloseTurn() *Turn {
	targetTurn := e.currentTurn

	newTurn := NewTurn(e.mode)
	e.turns = append(e.turns, newTurn)
	e.currentTurn = newTurn
	*e.closeTurnTriggered = false

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

func (e *Engine) GetCurrentTurn() *Turn {
	return e.currentTurn
}

func (e *Engine) GetCurrentAction() *action.Action {
	return e.currentAction
}
