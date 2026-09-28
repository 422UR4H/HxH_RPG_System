// internal/app/game/board.go
package game

import (
	"encoding/json"
	"fmt"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
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
