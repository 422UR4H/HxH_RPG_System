package game

import (
	"log"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/google/uuid"
)

// turnWrites is what happened INSIDE one open turn and becomes durable only together with that
// turn's close, in PersistTurnClose's own transaction — or not at all (owner decision,
// 2026-10-01). The turn itself is only ever written when it closes, so anything written about
// it before that moment could outlive it: a restart mid-turn used to leave master_actions rows
// pointing at a turn that never existed, and a board row already holding the opened move and
// the master's changes. Holding them here instead makes the whole turn roll back together.
//
// Each entry is built the instant it happened — views, timestamps, ids are those of that
// instant — and only its WRITE waits. persistClosedTurn drains it (takeTurnWritesLocked) into
// TurnCloseData; a new kind of in-turn data adds a field here and its drain there.
type turnWrites struct {
	// masterActions are the turn's master actions, in the order they were applied.
	masterActions []masteraction.Record
	// moveViews is what each session player saw of the turn's opened move, recorded at the
	// opening (recordOpenedMoveViews) and written with the action (actions.move_views): the
	// history shows each reader the move as they saw it then. nil when the action has no move.
	moveViews map[uuid.UUID]masteraction.View
	// reactionMoveViews is, per opened reaction that moves (an escape), what each session
	// player saw of its destination when the master opened it (recordOpenedReactionMoveViews) —
	// written with the reaction's own row (actions.move_views): the history shows each reader
	// a reaction's move as they saw it then (Phase 7, item 1). Absent for a reaction never
	// opened: it was never shown.
	reactionMoveViews map[uuid.UUID]map[uuid.UUID]masteraction.View
}

// openTurnIDLocked is the session's open turn, uuid.Nil when there is none (or no session).
// "Open" is HasOpenTurn, not CurrentTurnID alone: the round keeps its last turn as current
// after it closed. The caller must hold r.mu (read or write).
func (r *Room) openTurnIDLocked() uuid.UUID {
	if r.session == nil {
		return uuid.Nil
	}
	rd := r.session.GetActiveRound()
	if rd == nil || !rd.HasOpenTurn() {
		return uuid.Nil
	}
	return r.session.CurrentTurnID()
}

// turnWritesLocked returns the pending writes of turnID, creating them. The caller must hold
// r.mu for writing.
func (r *Room) turnWritesLocked(turnID uuid.UUID) *turnWrites {
	if r.pendingTurns == nil {
		r.pendingTurns = map[uuid.UUID]*turnWrites{}
	}
	w, ok := r.pendingTurns[turnID]
	if !ok {
		w = &turnWrites{}
		r.pendingTurns[turnID] = w
	}
	return w
}

// takeTurnWritesLocked hands over and forgets what is pending for turnID — the turn is closing,
// and it is written with it now or lost with it. Keyed by turn, not a single slot: open_next_action
// closes one turn and opens the next inside the same Execute, so the turn being drained is not
// necessarily the one open by then. The caller must hold r.mu for writing.
func (r *Room) takeTurnWritesLocked(turnID uuid.UUID) turnWrites {
	w, ok := r.pendingTurns[turnID]
	if !ok {
		return turnWrites{}
	}
	delete(r.pendingTurns, turnID)
	return *w
}

// droppedTurn is what a room that goes away with a turn still open loses: the turn (its opened
// move and anything else applied live inside it) and the master actions held for it.
type droppedTurn struct {
	turnID        uuid.UUID
	masterActions int
}

// describeDroppedTurnLocked reads what is lost when the room itself goes away — its last client
// left, or the hub stopped it — with a turn open. That is a restart as far as persistence goes:
// the next room rehydrates from the database, where the turn was never written, so nothing of it
// is written now either (the board is not saved, the held master actions are not inserted) and
// the turn rolls back whole to the last close, exactly like a restart mid-turn. Writing the board
// here would hand the next room an opened move whose turn does not exist. The result is only for
// logDroppedTurn, after the unlock. The caller must hold r.mu.
func (r *Room) describeDroppedTurnLocked() droppedTurn {
	d := droppedTurn{turnID: r.openTurnIDLocked()}
	for _, w := range r.pendingTurns {
		d.masterActions += len(w.masterActions)
	}
	return d
}

// logDroppedTurn says what a room that went away mid-turn did not write — never silently.
func logDroppedTurn(matchUUID uuid.UUID, d droppedTurn) {
	if d.turnID == uuid.Nil && d.masterActions == 0 {
		return
	}
	log.Printf("room of match %s closed with turn %s still open — the turn, its opened move and the %d master action(s) "+
		"applied inside it were NOT persisted; the match resumes from the last closed turn", matchUUID, d.turnID, d.masterActions)
}
