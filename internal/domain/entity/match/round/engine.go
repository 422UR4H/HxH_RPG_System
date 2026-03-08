package round

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/turn"
	"github.com/google/uuid"
)

type Engine struct {
	mode                enum.TurnMode
	actionQueue         ActionPriorityQueue
	preparedActions     map[uuid.UUID]*action.Action
	rounds              []*Round
	currentTurn         *turn.Turn
	currentRound        *Round
	currentAction       *action.Action
	closeRoundTriggered *bool // flag used to client persist the turn
}

// TODO: refatorar trocando closeRoundTriggered por um método que chame CloseRound e faça o trigger, ou algo do tipo, para evitar que o cliente tenha acesso a esse flag
// inclusive, talvez seja melhor persistir cada fechamento de Turno mesmo, ao invés de round
func NewEngine(actionQueue *[]*action.Action, rounds *[]*Round, closeRoundTriggered *bool) (*Engine, error) {
	if closeRoundTriggered == nil {
		return nil, ErrCloseRoundTriggeredCantBeNil
	}
	if rounds == nil {
		rounds = &[]*Round{NewRound(enum.Free)}
	}
	aq := NewActionPriorityQueue(actionQueue)

	mode := enum.Free
	lenRounds := len(*rounds)
	currentRound := (*rounds)[0]
	if lenRounds > 0 {
		currentRound = (*rounds)[lenRounds-1]
		mode = currentRound.GetMode()
	}

	return &Engine{
		mode:                mode,
		actionQueue:         aq,
		rounds:              *rounds,
		currentRound:        currentRound,
		currentAction:       nil,
		closeRoundTriggered: closeRoundTriggered,
	}, nil
}

func (e *Engine) Add(action *action.Action) {
	e.actionQueue.Insert(action)
}

// NextAction move the action with the highest ActionSpeed in ActionQueue to
// currentTurn, set it as currentAction and return it
func (e *Engine) NextAction() *action.Action {
	nextAction := e.actionQueue.ExtractMax()
	if nextAction == nil {
		return nil
	}
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
	// o fechamento de round será chamado no usecase/matchEngine do fechamento de turno
	// a rodada é diferente do turno e esse closeTurn hoje, na verdade, é o closeRound
	// refatorar, de forma que o turno (semântica) seja composto pela action e reactions
	// o round que é composto por turnos
	if *e.closeRoundTriggered {
		e.CloseRound()
	}
	e.currentTurn.AddAction(nextAction)
	e.currentAction = nextAction
	return nextAction
}

// AttachReaction adds a reaction to the current turn below the last action
func (e *Engine) AttachReaction(reaction *action.Action) error {
	action, err := e.currentTurn.GetLastAction()
	if err != nil {
		return err
	}
	if action.GetID() != reaction.ReactToID {
		return turn.ErrReactionNotCompatible
	}

	e.currentTurn.AddAction(reaction)
	return nil
}

// CloseRound finalizes the current round and starts a new one with the same mode,
// returning the closed round.
// Called by match.engine when the match ends OR before action switching (act open)
func (e *Engine) CloseRound() *Round {
	targetRound := e.currentRound

	newRound := NewRound(e.mode)
	e.rounds = append(e.rounds, newRound)
	e.currentRound = newRound
	*e.closeRoundTriggered = false

	return targetRound
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
	*e.closeRoundTriggered = true
	// return turn?
}

func (e *Engine) GetCurrentRound() *Round {
	return e.currentRound
}

func (e *Engine) GetCurrentAction() *action.Action {
	return e.currentAction
}
