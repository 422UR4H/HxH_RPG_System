package game

import (
	"log"
	"sort"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// What a failed write left behind, and that the NEXT write of the same kind carries in its own
// transaction — so a failure heals itself on the next success instead of leaving the database
// wrong until a restart (controller ruling on Fix C, 2026-10-02). Both live on the Room, guarded
// by r.mu, and die with the process: after a restart the session is rebuilt from what IS on disk,
// which is then consistent with itself.
//
//   - unwrittenSheets: sheets whose HP a turn's close applied but whose PersistTurnClose failed.
//     The next close writes them with its own damaged sheets. Only these, never every sheet of
//     the session: writing a sheet the match did not touch would clobber an edit made to it over
//     REST meanwhile.
//   - unwrittenRoundEnds: rounds that ended at the table but whose end was not written (a failed
//     PersistRoundClose, a failed turn close plus a failed salvage, a change_scene whose old pair
//     was never a row or whose CloseSceneAndRound failed). The next write that could
//     give birth to the round the session is in now — a turn close, an ensure (master action,
//     regime change, change_scene), another round's end — writes them closed first, in its own
//     transaction: no round is ever born open next to a predecessor still open on disk.

// sheetsToWriteLocked is what a turn's close writes: the sheets it damaged plus the ones an earlier
// failed close left unwritten. The caller must hold r.mu.
func (r *Room) sheetsToWriteLocked(damaged []matchsession.DamagedCharacter) map[uuid.UUID]*csSheet.CharacterSheet {
	out := make(map[uuid.UUID]*csSheet.CharacterSheet, len(damaged)+len(r.unwrittenSheets))
	for id, sh := range r.unwrittenSheets {
		out[id] = sh
	}
	for _, d := range damaged {
		if d.Sheet != nil {
			out[d.CharacterID] = d.Sheet
		}
	}
	return out
}

// markSheetsUnwrittenLocked remembers the sheets a failed close damaged, for the next close. The
// caller must hold r.mu.
func (r *Room) markSheetsUnwrittenLocked(damaged []matchsession.DamagedCharacter) {
	for _, d := range damaged {
		if d.Sheet == nil {
			continue
		}
		if r.unwrittenSheets == nil {
			r.unwrittenSheets = make(map[uuid.UUID]*csSheet.CharacterSheet)
		}
		r.unwrittenSheets[d.CharacterID] = d.Sheet
	}
}

// forgetSheetsWrittenLocked drops the sheets a successful close just wrote. The caller must hold
// r.mu.
func (r *Room) forgetSheetsWrittenLocked(written map[uuid.UUID]*csSheet.CharacterSheet) {
	for id := range written {
		delete(r.unwrittenSheets, id)
	}
}

// statusBarsLocked copies the three bars of each sheet, one entry per sheet in id order, for the
// turn's own transaction (TurnCloseData.StatusBars). The caller must hold r.mu: the sheets are the
// session's. A sheet missing one of its bars is left out and logged — writing a zero in its place
// would wipe the row.
func statusBarsLocked(sheets map[uuid.UUID]*csSheet.CharacterSheet) []appmatch.SheetStatusBars {
	if len(sheets) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(sheets))
	for id := range sheets {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	out := make([]appmatch.SheetStatusBars, 0, len(ids))
	for _, id := range ids {
		bars := sheets[id].GetAllStatusBar()
		health, stamina, aura := bars[enum.Health], bars[enum.Stamina], bars[enum.Aura]
		if health == nil || stamina == nil || aura == nil {
			log.Printf("sheet %s has no %s/%s/%s bar — its HP after the close is NOT written",
				id, enum.Health, enum.Stamina, enum.Aura)
			continue
		}
		out = append(out, appmatch.SheetStatusBars{
			CharacterID: id,
			Health:      status.ReconstructBar(health.GetMin(), health.GetCurrent(), health.GetMax()),
			Stamina:     status.ReconstructBar(stamina.GetMin(), stamina.GetCurrent(), stamina.GetMax()),
			Aura:        status.ReconstructBar(aura.GetMin(), aura.GetCurrent(), aura.GetMax()),
		})
	}
	return out
}

// unwrittenRoundEndsLocked copies the round ends still waiting for a write. The caller must hold
// r.mu (read or write).
func (r *Room) unwrittenRoundEndsLocked() []appmatch.RoundEnd {
	return append([]appmatch.RoundEnd(nil), r.unwrittenRoundEnds...)
}

// markRoundEndUnwrittenLocked remembers a round end whose write failed (snapshots), once. The
// caller must hold r.mu.
func (r *Room) markRoundEndUnwrittenLocked(end appmatch.RoundEnd) {
	for _, e := range r.unwrittenRoundEnds {
		if e.Round.GetID() == end.Round.GetID() {
			return
		}
	}
	r.unwrittenRoundEnds = append(r.unwrittenRoundEnds, end)
}

// forgetRoundEndsWrittenLocked drops the round ends a successful write just carried. Compared by
// round id: an end marked while that write was in flight stays. The caller must hold r.mu.
func (r *Room) forgetRoundEndsWrittenLocked(written []appmatch.RoundEnd) {
	if len(written) == 0 {
		return
	}
	done := make(map[uuid.UUID]bool, len(written))
	for _, e := range written {
		done[e.Round.GetID()] = true
	}
	kept := r.unwrittenRoundEnds[:0]
	for _, e := range r.unwrittenRoundEnds {
		if !done[e.Round.GetID()] {
			kept = append(kept, e)
		}
	}
	r.unwrittenRoundEnds = kept
}

// markRoundPersistedIfActiveLocked tells the session its active round is a row, if the round a
// write just made a row is still the active one — a later command may already have moved on, and
// then the new active round is not a row yet. The caller must hold r.mu for writing.
func markRoundPersistedIfActiveLocked(sess *matchsession.MatchSession, written uuid.UUID) {
	if active := sess.GetActiveRound(); active != nil && active.GetID() == written {
		sess.MarkRoundPersisted()
	}
}
