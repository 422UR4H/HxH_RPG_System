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
	// escapeLandings is where the MASTER put the piece of an escape, keyed by the reaction's
	// own ID — chosen over edit_action while the turn is open (front-combat-phases.md §6A.5,
	// B13). It is only a standing choice: the resolver copies it into the resolution only
	// for an escape that FAILED, and an escape that passes goes to its own slot whatever is
	// stored here. nil until the first choice.
	escapeLandings map[uuid.UUID][3]int
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

// SetEscapeLanding records where the master decided a failed escape leaves the piece. Setting
// it again replaces the previous choice. The turn does not judge the reaction or the position
// — MatchSession.SetEscapeLanding does, before calling this.
func (t *Turn) SetEscapeLanding(reactionID uuid.UUID, pos [3]int) {
	if t.escapeLandings == nil {
		t.escapeLandings = make(map[uuid.UUID][3]int)
	}
	t.escapeLandings[reactionID] = pos
}

// ClearEscapeLanding takes the master's choice back: a failed escape goes back to staying
// where it stood. Clearing what was never set is a no-op.
func (t *Turn) ClearEscapeLanding(reactionID uuid.UUID) {
	delete(t.escapeLandings, reactionID)
}

// EscapeLandings returns a copy of every landing the master has chosen, by reaction ID.
func (t *Turn) EscapeLandings() map[uuid.UUID][3]int {
	out := make(map[uuid.UUID][3]int, len(t.escapeLandings))
	for id, pos := range t.escapeLandings {
		out[id] = pos
	}
	return out
}
