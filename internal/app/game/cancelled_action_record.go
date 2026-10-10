package game

import (
	"context"
	"log"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/cancelledaction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// cancelledWrite is a cancelled action to be written now, outside the lock: the record and what
// writing it needs (the scene/round snapshot it references, and whether those are rows yet).
// nil from holdCancelledActionLocked means there is nothing to write now.
type cancelledWrite struct {
	rec       cancelledaction.Record
	sess      *matchsession.MatchSession
	sc        *sceneentity.Scene
	rd        *roundentity.Round
	persisted bool
}

// holdCancelledActionLocked builds the record of an action just cancelled (spec 2026-10-10
// §8.1 P6) in the SAME critical section as the CancelActionUC.Execute that removed it — the
// scene, round and open turn it names are those of the instant of the cancellation, the way
// recordMasterAction snapshots its own.
//
// WHEN it becomes a row follows the master actions' rule (P7):
//
//   - a turn open: held for that turn (turnWrites) and written by PersistTurnClose, in the
//     turn's own transaction; nil is returned — nothing to write now. A restart mid-turn loses
//     it with the turn, on purpose;
//   - no turn open: returned, for writeCancelledAction to write once the lock is released.
//
// by is the user who cancelled. The caller MUST hold r.mu for writing, and r.session must not
// be nil.
func (r *Room) holdCancelledActionLocked(act *action.Action, by uuid.UUID) *cancelledWrite {
	sess := r.session
	// Copies: the write below runs after the unlock, and the live pair is the session's.
	sc, rd := snapshotSceneAndRound(sess.GetActiveScene(), sess.GetActiveRound())
	if sc == nil || rd == nil {
		// Nothing for the row to reference; a combat session always has both.
		log.Printf("cancel_action: no active scene/round in match %s — cancelled action %s NOT recorded", r.matchUUID, act.GetID())
		return nil
	}
	rec := cancelledaction.Record{
		UUID: act.GetID(), MatchUUID: r.matchUUID, SceneUUID: sc.GetID(), RoundUUID: rd.GetID(),
		ActorUUID: act.GetActorID(), CancelledBy: by, Action: *act,
		CancelledAt: time.Now().UTC(),
	}
	if turnID := r.openTurnIDLocked(); turnID != uuid.Nil {
		rec.TurnUUID = &turnID
		w := r.turnWritesLocked(turnID)
		w.cancelledActions = append(w.cancelledActions, rec)
		return nil
	}
	return &cancelledWrite{rec: rec, sess: sess, sc: sc, rd: rd, persisted: sess.IsRoundPersisted()}
}

// writeCancelledAction writes what holdCancelledActionLocked returned — the no-turn path of
// recordMasterAction, unchanged: the scene/round rows are ensured first if the session says they
// are not rows yet, then the insert. nil w is a no-op, and so is a room with no
// CancelledActionRepo (it gates only this path; a held record is PersistTurnClose's). A failure
// is logged saying what was lost and swallowed — the persistClosedTurn policy.
//
// The caller must NOT hold r.mu.
func (r *Room) writeCancelledAction(w *cancelledWrite) {
	if w == nil || r.deps.CancelledActionRepo == nil {
		return
	}
	if !w.persisted && !r.ensureSceneAndRoundRows(w.sess, w.sc, w.rd, "cancel_action") {
		log.Printf("cancel_action FAILED — scene/round of match %s not written, cancelled action %s NOT recorded", r.matchUUID, w.rec.UUID)
		return
	}
	if err := r.deps.CancelledActionRepo.Insert(context.Background(), w.rec); err != nil {
		log.Printf("cancel_action FAILED — cancelled action %s of match %s was NOT recorded: %v", w.rec.UUID, r.matchUUID, err)
	}
}
