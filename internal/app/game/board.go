// internal/app/game/board.go
package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"sort"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
)

// slotShape is what a piece's Coord.Slot decodes to, whichever of its two real shapes it
// started as. Reading it through json.Marshal/Unmarshal — rather than a type switch on
// mapentity.SquareCoord/HexCoord — is what makes this robust to BOTH forms a persisted board
// can hand back: the concrete entity structs a Go caller builds by hand, and the
// map[string]any (with float64 numbers) a driver decodes a JSONB column into. The round trip
// also normalizes a whole-valued float64 like 2.0 to the JSON literal "2", which is what lets
// it unmarshal into *int without error.
type slotShape struct {
	Kind string `json:"kind"`
	Col  *int   `json:"col"`
	Row  *int   `json:"row"`
	Q    *int   `json:"q"`
	R    *int   `json:"r"`
}

// pieceToPayload converts a persisted board piece (internal/domain/map/entity.Piece) into the
// WS wire shape, for loadBoard to hand the room's in-memory r.pieces. An unknown slot kind is
// an error: the caller is expected to log and skip the piece rather than propagate one bad
// slot into a board-wide failure.
func pieceToPayload(p mapentity.Piece) (PieceMovedPayload, error) {
	raw, err := json.Marshal(p.Coord.Slot)
	if err != nil {
		return PieceMovedPayload{}, fmt.Errorf("piece %s: marshal slot: %w", p.ID, err)
	}
	var s slotShape
	if err := json.Unmarshal(raw, &s); err != nil {
		return PieceMovedPayload{}, fmt.Errorf("piece %s: unmarshal slot: %w", p.ID, err)
	}

	var slot SlotPayload
	switch s.Kind {
	case "hex":
		slot = SlotPayload{Kind: "hex", Q: s.Q, R: s.R}
	case "square":
		slot = SlotPayload{Kind: "square", Col: s.Col, Row: s.Row}
	default:
		return PieceMovedPayload{}, fmt.Errorf("piece %s: unknown slot kind %q", p.ID, s.Kind)
	}

	visible := p.Visible
	return PieceMovedPayload{
		PieceID:     p.ID,
		Slot:        slot,
		CharacterID: p.CharacterID,
		Visible:     &visible,
		Z:           p.Coord.Z,
	}, nil
}

// payloadToPiece converts the WS wire shape back into a persisted board piece, for the
// board's own persistence (T3's persistBoard) to write. It never errors: an unrecognized
// Slot.Kind falls back to the square shape rather than failing a save over one bad piece —
// pieceToPayload is the boundary that refuses that, on the way IN.
func payloadToPiece(p PieceMovedPayload) mapentity.Piece {
	var slot any
	if p.Slot.Kind == "hex" {
		var q, r int
		if p.Slot.Q != nil {
			q = *p.Slot.Q
		}
		if p.Slot.R != nil {
			r = *p.Slot.R
		}
		slot = mapentity.HexCoord{Kind: "hex", Q: q, R: r}
	} else {
		var col, row int
		if p.Slot.Col != nil {
			col = *p.Slot.Col
		}
		if p.Slot.Row != nil {
			row = *p.Slot.Row
		}
		slot = mapentity.SquareCoord{Kind: p.Slot.Kind, Col: col, Row: row}
	}

	// Visible defaults to true when the wire omitted it — the same default
	// applyAndRelayPieceMove's `hidden` check gives an absent pointer, read the other way
	// round: nil never means "hidden".
	visible := true
	if p.Visible != nil {
		visible = *p.Visible
	}

	return mapentity.Piece{
		ID:          p.PieceID,
		CharacterID: p.CharacterID,
		Coord:       mapentity.PieceCoord{Slot: slot, Z: p.Z},
		Visible:     visible,
	}
}

// persistBoard writes the match's board — pieces, walls, and every player's fog memory — as it
// stands NOW. It is the ONE place that does, and every definitive change to the board calls it:
// the lobby's moves, start_match, the three verbs that close a turn, the master's piece actions
// (applyMasterPieceAction — the "move"/"remove" of B9/B14), and the master's wall interactions
// and reveals (spec §4.3, "Quando persiste", B3).
//
// persistMu wraps the snapshot AND the write, so two saves racing from two read pumps land in
// the order their snapshots were taken; r.mu is only held for the snapshot, never across the
// round trip. A failure is logged and swallowed — the table goes on, same policy as
// persistClosedTurn.
//
// r.deps.SaveBoardUC == nil means the room has no persistence capability, the same "field left
// nil is a capability the room does not have" rule RoomDeps documents — every test built
// before T3 leaves it unset, and this is a no-op for them.
//
// The caller must NOT hold r.mu.
func (r *Room) persistBoard(reason string) {
	if r.deps.SaveBoardUC == nil {
		return
	}
	r.persistMu.Lock()
	defer r.persistMu.Unlock()

	r.mu.RLock()
	if r.mapUUID == uuid.Nil {
		// No map attached at all — nothing to persist yet (matches loadBoard's own (nil, nil)
		// no-op for an unattached match).
		r.mu.RUnlock()
		return
	}
	b := &matchboard.Board{MatchUUID: r.matchUUID, MapUUID: r.mapUUID, Grid: r.grid, Bg: r.bg}
	if r.session != nil {
		b.Grid = r.session.GetGrid()
	}
	for _, p := range r.pieces {
		b.Pieces = append(b.Pieces, payloadToPiece(p))
	}
	for _, w := range r.walls {
		b.Walls = append(b.Walls, w)
	}
	var mems []fogentity.PlayerMemory
	if r.session != nil {
		for _, pid := range r.session.PlayerIDs() {
			if m, ok := r.session.GetPlayerMemory(pid); ok && m != nil {
				// A copy, with its own Seen map: the snapshot must not alias state the session
				// keeps mutating after r.mu is released below.
				cp := *m
				cp.Seen = maps.Clone(m.Seen)
				mems = append(mems, cp)
			}
		}
	}
	r.mu.RUnlock()

	// Deterministic order: two saves racing on the same match must not flip which piece/wall
	// lands where in the JSONB array for reasons that have nothing to do with the data itself.
	sort.Slice(b.Pieces, func(i, j int) bool { return b.Pieces[i].ID < b.Pieces[j].ID })
	sort.Slice(b.Walls, func(i, j int) bool { return b.Walls[i].ID < b.Walls[j].ID })
	if err := r.deps.SaveBoardUC.Save(context.Background(), b, mems); err != nil {
		log.Printf("persistBoard(%s) FAILED — board of match %s was NOT saved: %v", reason, r.matchUUID, err)
	}
}
