package game

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
)

// snapshotSceneAndRound copies what the round repository writes of a scene and a round — ids,
// category, brief, regime, created/finished — into fresh objects. Every write of the pair runs
// after r.mu is released, and the live *Scene/*Round belong to the session, which the next
// message mutates under that lock (a regime switch, a round ending because no action can still pay): handing
// them to the gateway would have it read them unguarded. The caller MUST hold r.mu (read or
// write); either argument may be nil, and its copy is then nil too.
func snapshotSceneAndRound(sc *sceneentity.Scene, rd *roundentity.Round) (*sceneentity.Scene, *roundentity.Round) {
	var scCopy *sceneentity.Scene
	if sc != nil {
		scCopy = sceneentity.ReconstructScene(sc.GetID(), sc.GetCategory(), sc.BriefInitialDescription, sc.GetCreatedAt())
		if f := sc.GetFinishedAt(); f != nil {
			scCopy.Close(*f)
		}
	}
	var rdCopy *roundentity.Round
	if rd != nil {
		rdCopy = roundentity.ReconstructRound(rd.GetID(), rd.GetMode(), rd.GetCreatedAt())
		if f := rd.GetFinishedAt(); f != nil {
			rdCopy.Close(*f)
		}
	}
	return scCopy, rdCopy
}

// ensureSceneAndRoundRows writes sc and rd as rows (idempotent) and, only if they are still the
// session's active pair, tells the session they are rows — the same flag a persisted turn close
// sets, and the one change_scene and a round's close read to decide whether there is a row to
// close. A change_scene or a round close in between already reset the flag for the NEW pair,
// which is not a row yet, so it is left alone then.
//
// why names the moment, for the log. A failure is logged saying what was lost and swallowed —
// the persistClosedTurn policy — and reported as false so a caller writing something that
// references the pair (a master action, an event) can skip that write instead of failing on
// the FK. sc and rd are snapshots (snapshotSceneAndRound), never the session's live objects —
// which is also why "still the active pair" compares ids, not pointers. The caller must NOT
// hold r.mu.
func (r *Room) ensureSceneAndRoundRows(
	sess *matchsession.MatchSession, sc *sceneentity.Scene, rd *roundentity.Round, why string,
) bool {
	if sc == nil || rd == nil {
		return false
	}
	if r.deps.RoundRepo == nil {
		log.Printf("%s: no round repository — scene %s / round %s of match %s NOT written", why, sc.GetID(), rd.GetID(), r.matchUUID)
		return false
	}
	if err := r.deps.RoundRepo.EnsureSceneAndRound(context.Background(), r.matchUUID, sc, rd); err != nil {
		log.Printf("%s FAILED — scene %s / round %s of match %s NOT written: %v", why, sc.GetID(), rd.GetID(), r.matchUUID, err)
		return false
	}
	r.mu.Lock()
	if active := sess.GetActiveRound(); active != nil && active.GetID() == rd.GetID() {
		sess.MarkRoundPersisted()
	}
	r.mu.Unlock()
	return true
}

// ensureActiveSceneAndRound makes the session's ACTIVE scene and round rows the moment they are
// born (B15, spec §4.5): start_match and rehydration (the pair the session starts on) and
// change_scene (the new pair). The round born when another ends is not written here: it goes
// with its predecessor's end, in one transaction (persistClosedTurn's, or writeRoundClose's) —
// written alone, next to a predecessor still open on disk, it would be a second open round in
// the scene. A scene the table spent talking and a round that closed with no turn in it happened
// all the same — before B15 they only became rows when their first turn closed, and so were
// missing from the history.
//
// A pair the session already knows to be rows (rehydrated from them, or ensured already) is
// not written again. The pair is read under r.mu; the write is not — the caller must NOT hold
// r.mu.
func (r *Room) ensureActiveSceneAndRound(why string) {
	r.mu.RLock()
	sess := r.session
	if sess == nil {
		r.mu.RUnlock()
		return
	}
	sc, rd := snapshotSceneAndRound(sess.GetActiveScene(), sess.GetActiveRound())
	persisted := sess.IsRoundPersisted()
	r.mu.RUnlock()
	if persisted {
		return
	}
	r.ensureSceneAndRoundRows(sess, sc, rd, why)
}

// persistRoundClose writes a round that ended with no turn closing in the same command (an
// open_next_action with nothing open and no action that can still pay its price): its end and
// the round CloseRound installed in its place, in one transaction of their own. closed is the
// round the use case returned; the scene and the successor are read here, under r.mu. The caller
// must NOT hold r.mu.
func (r *Room) persistRoundClose(sess *matchsession.MatchSession, closed *roundentity.Round) {
	r.mu.RLock()
	sc, closedCopy := snapshotSceneAndRound(sess.GetActiveScene(), closed)
	_, next := snapshotSceneAndRound(nil, sess.GetActiveRound())
	r.mu.RUnlock()
	r.writeRoundClose(sess, sc, closedCopy, next, "round_closed")
}

// writeRoundClose writes a round's end and its successor's birth together
// (RoundRepo.PersistRoundClose) and, on success, tells the session the successor is a row — if
// it is still the active round. Never one write without the other: a successor born open next
// to a predecessor still open on disk is two open rounds in one scene. sc, closed and next are
// snapshots (snapshotSceneAndRound). why names the moment, for the log; a failure is logged
// saying what was lost and swallowed — the persistClosedTurn policy. The caller must NOT hold
// r.mu.
func (r *Room) writeRoundClose(
	sess *matchsession.MatchSession, sc *sceneentity.Scene, closed, next *roundentity.Round, why string,
) {
	if sc == nil || closed == nil || next == nil {
		return
	}
	if r.deps.RoundRepo == nil {
		log.Printf("%s: no round repository — the end of round %s and the birth of round %s (match %s) NOT written",
			why, closed.GetID(), next.GetID(), r.matchUUID)
		return
	}
	if err := r.deps.RoundRepo.PersistRoundClose(context.Background(), r.matchUUID, sc, closed, next); err != nil {
		log.Printf("%s FAILED — the end of round %s and the birth of round %s (scene %s, match %s) were NOT written: %v",
			why, closed.GetID(), next.GetID(), sc.GetID(), r.matchUUID, err)
		return
	}
	r.mu.Lock()
	if active := sess.GetActiveRound(); active != nil && active.GetID() == next.GetID() {
		sess.MarkRoundPersisted()
	}
	r.mu.Unlock()
}

// recordRoundModeChanged writes the round's regime change to match_events (B15, spec §4.5) —
// the regime a round passed through is history, and the round row only keeps the last one.
//
// sc/rd are snapshots of the pair the change was applied to, taken under the lock that applied
// it, after the switch (snapshotSceneAndRound). The pair
// is ensured first, unconditionally: besides making sure the event's FKs exist, it is what
// refreshes the round row's mode (EnsureSceneAndRound updates it on conflict). A switch to the
// regime the round was already in changed nothing and records nothing.
//
// A failure is logged saying what was lost and swallowed. The caller must NOT hold r.mu.
func (r *Room) recordRoundModeChanged(
	sess *matchsession.MatchSession, sc *sceneentity.Scene, rd *roundentity.Round, from, to enum.RoundMode,
) {
	if !r.ensureSceneAndRoundRows(sess, sc, rd, "change_round_mode") {
		return
	}
	if from == to || r.deps.EventRepo == nil {
		return
	}
	payload, err := json.Marshal(map[string]string{"from": string(from), "to": string(to)})
	if err != nil {
		log.Printf("change_round_mode FAILED — event payload could not be marshaled, the %s -> %s change of round %s (match %s) was NOT recorded: %v",
			from, to, rd.GetID(), r.matchUUID, err)
		return
	}
	ev := matchevent.Event{
		UUID: uuid.New(), MatchUUID: r.matchUUID, SceneUUID: sc.GetID(), RoundUUID: rd.GetID(),
		Kind: matchevent.KindRoundModeChanged, Payload: payload, CreatedAt: time.Now().UTC(),
	}
	if err := r.deps.EventRepo.Insert(context.Background(), ev); err != nil {
		log.Printf("change_round_mode FAILED — the %s -> %s change of round %s (match %s) was NOT recorded: %v",
			from, to, rd.GetID(), r.matchUUID, err)
	}
}
