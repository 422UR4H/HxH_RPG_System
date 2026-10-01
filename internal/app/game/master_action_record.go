package game

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// wallActionContent is the Content of the two wall kinds (wallInteract, revealWall): the walls
// the action actually changed — a targetId the server did not know, or whose interact kind does
// not apply to it, was answered or skipped and is not part of what happened — and the kind.
// A wallInteract always carries ONE wall (a batch is recorded once per wall, each with its own
// views); a revealWall carries every wall it revealed, since a reveal reaches every player.
type wallActionContent struct {
	WallIDs  []string `json:"wallIds"`
	Interact string   `json:"interact"`
}

// viewGate decides what ONE player saw of something that just happened on the board, from
// that player's visibility polygons. false means they saw nothing: no entry for them.
type viewGate func(polys []domainservice.VisibilityPolygon) (masteraction.View, bool)

// pieceMoveView is the fog gate of one piece move for ONE player — the same decision the live
// dispatch makes, extracted so the SAME decision can be recorded with a master action and the
// history can show each reader exactly what they saw then (spec §4.8). Pure: reads the cache.
func pieceMoveView(polys []domainservice.VisibilityPolygon, newPt, oldPt domainservice.Point2D, hadOld, hidden bool) (masteraction.View, bool) {
	if hidden {
		return "", false
	}
	switch {
	case domainservice.IsVisible(newPt, polys):
		return masteraction.ViewFull, true
	case hadOld && domainservice.IsVisible(oldPt, polys):
		return masteraction.ViewLeft, true
	default:
		return "", false
	}
}

// pieceRemovedView is the fog gate of one piece removal for ONE player, extracted from the
// removal's dispatch for the same reason as pieceMoveView. Only whoever could see the piece at
// its last position is told — and a removal with no known last position (hadOld false) has
// nothing to gate on, so it goes to everyone, as it always has. Pure.
func pieceRemovedView(polys []domainservice.VisibilityPolygon, oldPt domainservice.Point2D, hadOld, hidden bool) (masteraction.View, bool) {
	if hidden {
		return "", false
	}
	if hadOld && !domainservice.IsVisible(oldPt, polys) {
		return "", false
	}
	return masteraction.ViewFull, true
}

// seenByAll is the gate of what reaches every player regardless of fog (a wall changing that is
// not an unrevealed secret door, a reveal).
func seenByAll([]domainservice.VisibilityPolygon) (masteraction.View, bool) {
	return masteraction.ViewFull, true
}

// seenByNone is the gate of what reaches the master alone (an unrevealed secret door changing).
func seenByNone([]domainservice.VisibilityPolygon) (masteraction.View, bool) { return "", false }

// sessionViews runs gate over EVERY player of the session — connected or not: what counts is
// their fog at this instant, read from the visibility cache the session keeps for all of them
// (spec §4.8). The master is not a player of the session and has no entry; the master sees
// every master action. nil when there is no session (the lobby records nothing).
//
// The caller must NOT hold r.mu.
func (r *Room) sessionViews(gate viewGate) map[uuid.UUID]masteraction.View {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.session == nil {
		return nil
	}
	views := map[uuid.UUID]masteraction.View{}
	for _, pid := range r.session.PlayerIDs() {
		if pid == r.masterUUID {
			continue
		}
		if v, ok := gate(r.session.GetVisibility(pid)); ok {
			views[pid] = v
		}
	}
	return views
}

// recordMasterAction records one master action the instant it was applied (spec §4.8). It is
// the ONLY writer of master_actions, and it is called from the enqueue_master_action arm alone:
// edit_action also hangs a MasterAction on the turn, but an edit is not a master action (it
// lives in overridden_action_values), which is why this never reads Turn.GetMasterActions().
//
// views is what each player of the session saw of it live; the history shows every reader
// exactly that. The caller must NOT hold r.mu.
//
// WHEN it becomes a row depends on whether a turn is open (owner decision, 2026-10-01):
//
//   - no turn open: written now, as it always was;
//   - a turn open: the record is built now — views, turnUuid, happenedAt are this instant's —
//     but held for that turn (turnWrites) and written by PersistTurnClose, in the turn's own
//     transaction. A turn is only ever written when it closes; a row written before that could
//     outlive it, and a restart mid-turn would leave history pointing at a turn that never was.
//
// Which of the two is decided in ONE critical section with the hold itself, so a close running
// on another read pump can never drain the turn between "a turn is open" and the append.
//
// master_actions references the ACTIVE scene and round. Since B15 they are rows from birth
// (ensureActiveSceneAndRound), so this is normally a no-op; if that write failed, they are
// ensured here first, and the session is told they are rows now (ensureSceneAndRoundRows) —
// without that flag, change_scene and the round's close would skip closing rows that do exist,
// and a restart would rehydrate onto a scene that had already ended. A held record needs none
// of that: PersistTurnClose ensures the same pair in its own transaction, and neither can change
// while a turn is open (change_scene and the round's close are refused then).
//
// A failure is logged saying what was lost and swallowed — the persistClosedTurn policy.
func (r *Room) recordMasterAction(kind masteraction.Kind, content any, views map[uuid.UUID]masteraction.View) {
	if r.deps.MasterActionRepo == nil {
		return
	}
	// Before the lock: pure, and the lock below is a write lock.
	raw, err := json.Marshal(content)
	if err != nil {
		log.Printf("recordMasterAction(%s) FAILED — content could not be marshaled, master action of match %s was NOT recorded: %v", kind, r.matchUUID, err)
		return
	}

	r.mu.Lock()
	sess := r.session
	if sess == nil {
		r.mu.Unlock()
		return
	}
	// Copies: the writes below run after the unlock, and the live pair is the session's.
	sc, rd := snapshotSceneAndRound(sess.GetActiveScene(), sess.GetActiveRound())
	rec := masteraction.Record{
		UUID: uuid.New(), MatchUUID: r.matchUUID, SceneUUID: sc.GetID(), RoundUUID: rd.GetID(),
		MasterUUID: r.masterUUID, Kind: kind, Content: raw, Views: views,
		HappenedAt: time.Now().UTC(),
	}
	if turnID := r.openTurnIDLocked(); turnID != uuid.Nil {
		rec.TurnUUID = &turnID
		w := r.turnWritesLocked(turnID)
		w.masterActions = append(w.masterActions, rec)
		r.mu.Unlock()
		return
	}
	persisted := sess.IsRoundPersisted()
	r.mu.Unlock()

	if !persisted && !r.ensureSceneAndRoundRows(sess, sc, rd, "recordMasterAction("+string(kind)+")") {
		log.Printf("recordMasterAction(%s) FAILED — scene/round of match %s not written, action NOT recorded", kind, r.matchUUID)
		return
	}
	if err := r.deps.MasterActionRepo.Insert(context.Background(), rec); err != nil {
		log.Printf("recordMasterAction(%s) FAILED — master action of match %s was NOT recorded: %v", kind, r.matchUUID, err)
	}
}
