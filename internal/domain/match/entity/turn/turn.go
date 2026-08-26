package turn

import (
	"slices"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

type Turn struct {
	id        uuid.UUID
	action    action.Action
	reactions []action.Action
	// openedReactions is the order the MASTER opened the reactions, which is not the order
	// they arrived — reactions land whenever their players send them, and the master picks.
	// The chain walks this order, and walking it in reverse produces a different outcome; that
	// is game power, not a matter of pacing.
	openedReactions []uuid.UUID
	masterActions   []action.MasterAction
	finishedAt      *time.Time
}

func NewTurn(action action.Action) *Turn {
	return &Turn{
		id:     uuid.New(),
		action: action,
	}
}

func (t *Turn) GetID() uuid.UUID {
	return t.id
}

func (t *Turn) AddMasterAction(ma action.MasterAction) {
	t.masterActions = append(t.masterActions, ma)
}

func (t *Turn) AddReaction(action *action.Action) {
	t.reactions = append(t.reactions, *action)
}

func (t *Turn) Close(finishedAt time.Time) {
	t.finishedAt = &finishedAt
}

func (t *Turn) GetAction() action.Action {
	return t.action
}

// ActionRef and ReactionRef hand out POINTERS, unlike GetAction/GetReactions which copy.
//
// The copies exist so a reader cannot mutate the turn by accident. The master's edit is the
// one caller that must mutate it — the edited action IS the action — so it gets the real
// thing, deliberately and narrowly, rather than by turning the safe accessors unsafe.
func (t *Turn) ActionRef() *action.Action { return &t.action }

func (t *Turn) ReactionRef(id uuid.UUID) *action.Action {
	for i := range t.reactions {
		if t.reactions[i].GetID() == id {
			return &t.reactions[i]
		}
	}
	return nil
}

func (t *Turn) GetReactions() []action.Action {
	reactionsCp := make([]action.Action, len(t.reactions))
	copy(reactionsCp, t.reactions)
	return reactionsCp
}

func (t *Turn) GetMasterActions() []action.MasterAction {
	masterActionsCp := make([]action.MasterAction, len(t.masterActions))
	copy(masterActionsCp, t.masterActions)
	return masterActionsCp
}

func (t *Turn) GetFinishedAt() *time.Time {
	return t.finishedAt
}

// OpenReaction records that the master passed the microphone to one of the attached
// reactions, and reports whether it found it. Idempotent: opening the same one twice does not
// give it a second slot in the order.
func (t *Turn) OpenReaction(id uuid.UUID) bool {
	found := false
	for i := range t.reactions {
		if t.reactions[i].GetID() == id {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	if slices.Contains(t.openedReactions, id) {
		return true
	}
	t.openedReactions = append(t.openedReactions, id)
	return true
}

// OpenedReactionIDs returns a copy of the opening order.
func (t *Turn) OpenedReactionIDs() []uuid.UUID {
	out := make([]uuid.UUID, len(t.openedReactions))
	copy(out, t.openedReactions)
	return out
}

// UnopenedReactions is every attached reaction the master has not yet given the floor to.
//
// These are exactly the ones close_turn warns about: they ARE in the calculation — the chain
// walks the opened ones first and then everyone left over — so nobody is punished
// mechanically. What they lose is the moment to narrate, and that is what the master is being
// asked to confirm away.
func (t *Turn) UnopenedReactions() []action.Action {
	out := make([]action.Action, 0, len(t.reactions))
	for i := range t.reactions {
		if !slices.Contains(t.openedReactions, t.reactions[i].GetID()) {
			out = append(out, t.reactions[i])
		}
	}
	return out
}
