package game

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// applyMasterPieceAction is enqueue_master_action's piece branch (spec §4.3, "Master action de
// peça" — B14, the `move` of B9, and B11 with the room alive): during a match, the master drags
// a piece, puts one on the board, or takes one off, for the character in targetIds[0].
//
//   - drag: the character has a piece → it goes to move.position, keeping its slot shape and
//     its Z, exactly the write applyMove does for a game movement.
//   - place: the character has no piece → a new one (new id, slot shape from the grid,
//     visible, z 0). A player's character that is not a participant is refused
//     (not_participant); the master's NPC that is not one yet is enrolled first, through the
//     SAME enrollLiveNPC add_npc uses, so the table hears npc_added either way.
//   - remove: the piece leaves the board. Nothing else: no un-enrolling, no death, no history
//     erased.
//
// It applies at once, with or without an open turn; with one, the master action also hangs on
// it. Each emits piece_moved/piece_removed fog-projected for everyone, the master included —
// the master's screen waits for that confirmation — and master_action_enqueued to the master
// ALONE: the echo carries the position, which for a hidden piece would leak to the table.
// Then the board is saved and the master action recorded with what each player saw (§4.8).
//
// In the lobby there is no session: the master moves pieces with piece_moved there.
//
// The caller must NOT hold r.mu.
func (r *Room) applyMasterPieceAction(client *Client, ma *action.MasterAction, remove bool) {
	r.mu.RLock()
	session := r.session
	r.mu.RUnlock()
	if session == nil {
		client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
		return
	}
	if len(ma.TargetID) != 1 {
		client.SendMessage(NewErrorMessage("invalid_action",
			"a piece master action names exactly one character in targetIds"))
		return
	}
	if remove && ma.Move != nil {
		client.SendMessage(NewErrorMessage("invalid_action", "move and remove are exclusive"))
		return
	}
	charID := ma.TargetID[0]

	var (
		kind    masteraction.Kind
		content masteraction.PieceContent
		views   map[uuid.UUID]masteraction.View
		ok      bool
	)
	if remove {
		kind, content, views, ok = r.masterRemovePiece(client, charID)
	} else {
		kind, content, views, ok = r.masterMoveOrPlacePiece(client, session, charID, ma.Move.Position)
	}
	if !ok {
		return // already answered
	}

	// Hung on the open turn too, when there is one; with none, ErrNoActiveTurn stops nothing —
	// the piece already moved (spec §4.3).
	r.mu.Lock()
	err := session.EnqueueMasterAction(ma)
	r.mu.Unlock()
	if err != nil && !errors.Is(err, matchsession.ErrNoActiveTurn) {
		log.Printf("master piece action: hanging it on the open turn: %v", err)
	}

	r.persistBoard("master_piece")
	r.recordMasterAction(kind, content, views)

	echo := MasterActionEnqueuedPayload{TargetIDs: ma.TargetID}
	if remove {
		echo.Remove = &RemovePiecePayload{}
	} else {
		echo.Move = &MovePayload{Position: ma.Move.Position}
	}
	client.SendMessage(NewServerMessage(MsgTypeMasterActionEnqueued, echo))
}

// masterRemovePiece takes the character's piece off the board — the same piece applyMove would
// move (pieceOfLocked). ok false: the character has no piece, and the master was told.
func (r *Room) masterRemovePiece(
	client *Client, charID uuid.UUID,
) (masteraction.Kind, masteraction.PieceContent, map[uuid.UUID]masteraction.View, bool) {
	r.mu.Lock()
	pieceID := r.pieceOfLocked(charID.String())
	if pieceID == "" {
		r.mu.Unlock()
		client.SendMessage(NewErrorMessage("invalid_action", "character has no piece"))
		return "", masteraction.PieceContent{}, nil, false
	}
	old := r.pieces[pieceID]
	delete(r.pieces, pieceID)
	r.mu.Unlock()

	views := r.relayPieceRemoved(old, true, uuid.Nil)
	from := slotPosition(old.Slot)
	return masteraction.KindRemovePiece,
		masteraction.PieceContent{CharacterID: old.CharacterID, PieceID: pieceID, From: &from},
		views, true
}

// masterMoveOrPlacePiece drags the character's piece to pos, or — when it has none — puts a new
// one there, enrolling the master's NPC first when it is not a participant yet. ok false: it
// was refused, and the master was told.
func (r *Room) masterMoveOrPlacePiece(
	client *Client, session *matchsession.MatchSession, charID uuid.UUID, pos [3]int,
) (masteraction.Kind, masteraction.PieceContent, map[uuid.UUID]masteraction.View, bool) {
	to := pos
	// Drag: the same single critical section applyMove uses to find the piece and write it back.
	r.mu.Lock()
	if pieceID := r.pieceOfLocked(charID.String()); pieceID != "" {
		old := r.pieces[pieceID]
		moved := old
		moved.Slot = slotKeepingKind(old.Slot, pos)
		// Z is not touched, for the reason applyMove gives: it is metres, pos[2] a grid index.
		r.pieces[pieceID] = moved
		r.mu.Unlock()

		views := r.relayPieceMove(moved, old, true, uuid.Nil)
		from := slotPosition(old.Slot)
		return masteraction.KindMovePiece,
			masteraction.PieceContent{CharacterID: old.CharacterID, PieceID: pieceID, From: &from, To: &to},
			views, true
	}
	_, participant := session.GetCharToPlayer()[charID.String()]
	r.mu.Unlock()

	// Place. A character that is not a participant has to become one first, and only the
	// master's NPC can: a player's character enters a match by enrollment, not by the master
	// dropping a token. The ownership read is I/O, so it runs unlocked.
	if !participant {
		if r.deps.SheetOwnership != nil {
			rel, err := r.deps.SheetOwnership.GetCharacterSheetRelationshipUUIDs(context.Background(), charID)
			if err == nil && rel.PlayerUUID != nil {
				client.SendMessage(NewErrorMessage("not_participant",
					"a player's character that is not in the match cannot be put on the board"))
				return "", masteraction.PieceContent{}, nil, false
			}
			// A read error falls through on purpose: enrollLiveNPC reads the sheet again through
			// AddMatchNPCUC, whose guards answer it (not_found, invalid_npc) exactly as add_npc does.
		}
		if !r.enrollLiveNPC(client, charID) {
			return "", masteraction.PieceContent{}, nil, false // already answered
		}
	}

	visible := true
	placed := PieceMovedPayload{
		PieceID:     uuid.NewString(),
		CharacterID: charID.String(),
		Slot:        slotFor(r.gridShape(), pos),
		Visible:     &visible,
	}
	r.mu.Lock()
	// Re-checked in the SAME critical section as the write: the lookup above released r.mu for
	// the ownership and enrollment I/O, and a second master socket could have placed this
	// character meanwhile. A second piece for one character is the undecided situation
	// applyMove's TODO describes — never created here.
	if r.pieceOfLocked(charID.String()) != "" {
		r.mu.Unlock()
		client.SendMessage(NewErrorMessage("invalid_action", "character already has a piece"))
		return "", masteraction.PieceContent{}, nil, false
	}
	r.pieces[placed.PieceID] = placed
	r.mu.Unlock()

	views := r.relayPieceMove(placed, PieceMovedPayload{}, false, uuid.Nil)
	return masteraction.KindPlacePiece,
		masteraction.PieceContent{CharacterID: placed.CharacterID, PieceID: placed.PieceID, To: &to},
		views, true
}

// slotFor is the slot a NEW piece gets at grid position pos: its shape follows the grid, since
// there is no previous slot to keep the shape of.
func slotFor(g mapentity.GridShape, pos [3]int) SlotPayload {
	a, b := pos[0], pos[1]
	if g.Kind == mapentity.GridKindHex {
		return SlotPayload{Kind: "hex", Q: &a, R: &b}
	}
	return SlotPayload{Kind: "square", Col: &a, Row: &b}
}

// slotPosition reads a slot back as a grid position [col, row, 0] (or [q, r, 0]) for a master
// action's record. The third coordinate is 0: the piece's Z is a height in metres, not the grid
// index a position carries, and the two have never been reconciled (see applyMove).
func slotPosition(s SlotPayload) [3]int {
	var a, b int
	if s.Kind == "hex" {
		if s.Q != nil {
			a = *s.Q
		}
		if s.R != nil {
			b = *s.R
		}
	} else {
		if s.Col != nil {
			a = *s.Col
		}
		if s.Row != nil {
			b = *s.Row
		}
	}
	return [3]int{a, b, 0}
}

// enrollLiveNPC puts the master's NPC into the match: the SAME AddMatchNPCUC the REST POST
// /npcs runs (same guards, same match_participants write), then — with a live session — the
// sheet goes into it, and the table hears npc_added and a new bars_updated. Both add_npc and
// the "place" of a piece master action call it, so the two announce an NPC the same way (spec
// §4.3, "B11 … Com a sala viva").
//
// It returns true once the NPC is enrolled — in the live session when there is one; in the
// lobby, in the roster the session will load. false means it answered the master with the
// error itself.
//
// The caller must NOT hold r.mu.
func (r *Room) enrollLiveNPC(client *Client, sheetUUID uuid.UUID) bool {
	if r.deps.AddLiveNPCUC == nil {
		client.SendMessage(NewErrorMessage("game_error", "this room cannot enroll NPCs"))
		return false
	}
	// DB I/O, so no r.mu here. The use case already swallows ErrNPCAlreadyInMatch: every
	// guard runs BEFORE the INSERT that reports the duplicate, so "already in the database"
	// means "passed everything, only the session is behind" — exactly the NPC the master
	// added over REST mid-match. Re-sending add_npc is how the session catches up; the
	// duplicate this verb does refuse is the SESSION's, below.
	sheet, err := r.deps.AddLiveNPCUC.Execute(context.Background(), &appmatch.AddMatchNPCInput{
		RequesterUUID: client.userUUID,
		MatchUUID:     r.matchUUID,
		SheetUUID:     sheetUUID,
	})
	if err != nil {
		code := "game_error"
		switch {
		case errors.Is(err, appmatch.ErrNotMatchMaster):
			code = "forbidden"
		case errors.Is(err, appmatch.ErrMatchNotFound),
			errors.Is(err, appmatch.ErrCharacterSheetNotFound):
			code = "not_found"
		case errors.Is(err, appmatch.ErrSheetNotNPC),
			errors.Is(err, appmatch.ErrSheetNotOwnedByMaster),
			errors.Is(err, appmatch.ErrMatchAlreadyFinished):
			code = "invalid_npc"
		}
		client.SendMessage(NewErrorMessage(code, err.Error()))
		return false
	}
	// Write lock: AddNPC writes the session's sheets, statuses and charToPlayer. With no
	// session (the lobby) there is nothing live to inject into — the row just written is
	// what InitMatchSessionUC will load when the match starts.
	r.mu.Lock()
	session := r.session
	var addErr error
	if session != nil {
		addErr = session.AddNPC(sheetUUID, sheet, r.masterUUID)
	}
	r.mu.Unlock()
	if errors.Is(addErr, matchsession.ErrCharacterAlreadyInSession) {
		client.SendMessage(NewErrorMessage("npc_already_in_match", addErr.Error()))
		return false
	}
	if addErr != nil {
		client.SendMessage(NewErrorMessage("game_error", addErr.Error()))
		return false
	}
	// npc_added is the master's ack and the table's cue to fetch the sheet over REST — the
	// same shape as scene_changed. The combat state that changed is only the set of
	// characters on the bars, so bars_updated carries it, NOT match_full_state: broadcastBars
	// bumps seq, while match_full_state repeats the CURRENT seq by contract, and a delayed
	// bars_updated of that same seq, without the NPC, would be applied over it and wipe the
	// NPC off the screen. match_full_state stays what its name says: the snapshot of whoever
	// connects.
	out := NewServerMessage(MsgTypeNPCAdded, NPCAddedPayload{CharacterID: sheetUUID})
	data, _ := json.Marshal(out)
	go func() { r.broadcast <- data }()
	if session != nil {
		r.broadcastBars(session)
	}
	return true
}
