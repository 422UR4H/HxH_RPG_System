package game

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func intp(i int) *int { return &i }
func boolp(b bool) *bool { return &b }

// TestPieceToPayload_HexAndSquare covers the two shapes the board can carry, and both the
// canonical struct form (mapentity.HexCoord/SquareCoord) and the map[string]any form a JSON
// round-trip through the database driver produces — board.go's whole reason to exist is being
// robust to both.
func TestPieceToPayload_HexAndSquare(t *testing.T) {
	t.Run("hex, slot as map[string]any (the DB round-trip shape)", func(t *testing.T) {
		p := mapentity.Piece{
			ID:          "p1",
			CharacterID: "c1",
			Coord: mapentity.PieceCoord{
				Slot: map[string]any{"kind": "hex", "q": 2.0, "r": -1.0},
				Z:    1.5,
			},
			Visible: false,
		}
		got, err := pieceToPayload(p)
		if err != nil {
			t.Fatalf("pieceToPayload: %v", err)
		}
		want := PieceMovedPayload{
			PieceID: "p1", CharacterID: "c1",
			Slot: SlotPayload{Kind: "hex", Q: intp(2), R: intp(-1)},
			Z:    1.5, Visible: boolp(false),
		}
		assertPayloadEqual(t, got, want)
	})

	t.Run("square, slot as the concrete entity struct", func(t *testing.T) {
		p := mapentity.Piece{
			ID:          "p2",
			CharacterID: "c2",
			Coord: mapentity.PieceCoord{
				Slot: mapentity.SquareCoord{Kind: "square", Col: 3, Row: 4},
				Z:    0,
			},
			Visible: true,
		}
		got, err := pieceToPayload(p)
		if err != nil {
			t.Fatalf("pieceToPayload: %v", err)
		}
		want := PieceMovedPayload{
			PieceID: "p2", CharacterID: "c2",
			Slot: SlotPayload{Kind: "square", Col: intp(3), Row: intp(4)},
			Z:    0, Visible: boolp(true),
		}
		assertPayloadEqual(t, got, want)
	})

	t.Run("unknown slot kind is an error", func(t *testing.T) {
		p := mapentity.Piece{
			ID: "p3",
			Coord: mapentity.PieceCoord{
				Slot: map[string]any{"kind": "triangle"},
			},
		}
		if _, err := pieceToPayload(p); err == nil {
			t.Fatal("pieceToPayload: expected an error for an unknown slot kind, got nil")
		}
	})
}

func TestPayloadToPiece_HexAndSquare(t *testing.T) {
	t.Run("hex", func(t *testing.T) {
		payload := PieceMovedPayload{
			PieceID: "p1", CharacterID: "c1",
			Slot: SlotPayload{Kind: "hex", Q: intp(2), R: intp(-1)},
			Z:    1.5, Visible: boolp(false),
		}
		got := payloadToPiece(payload)
		want := mapentity.Piece{
			ID:          "p1",
			CharacterID: "c1",
			Coord: mapentity.PieceCoord{
				Slot: mapentity.HexCoord{Kind: "hex", Q: 2, R: -1},
				Z:    1.5,
			},
			Visible: false,
		}
		if got.ID != want.ID || got.CharacterID != want.CharacterID || got.Coord.Z != want.Coord.Z ||
			got.Visible != want.Visible {
			t.Fatalf("payloadToPiece = %+v, want %+v", got, want)
		}
		slot, ok := got.Coord.Slot.(mapentity.HexCoord)
		if !ok {
			t.Fatalf("Coord.Slot = %#v (%T), want mapentity.HexCoord", got.Coord.Slot, got.Coord.Slot)
		}
		if slot != want.Coord.Slot.(mapentity.HexCoord) {
			t.Fatalf("Coord.Slot = %+v, want %+v", slot, want.Coord.Slot)
		}
	})

	t.Run("square", func(t *testing.T) {
		payload := PieceMovedPayload{
			PieceID: "p2", CharacterID: "c2",
			Slot: SlotPayload{Kind: "square", Col: intp(3), Row: intp(4)},
			Z:    0, Visible: boolp(true),
		}
		got := payloadToPiece(payload)
		slot, ok := got.Coord.Slot.(mapentity.SquareCoord)
		if !ok {
			t.Fatalf("Coord.Slot = %#v (%T), want mapentity.SquareCoord", got.Coord.Slot, got.Coord.Slot)
		}
		want := mapentity.SquareCoord{Kind: "square", Col: 3, Row: 4}
		if slot != want {
			t.Fatalf("Coord.Slot = %+v, want %+v", slot, want)
		}
	})
}

func assertPayloadEqual(t *testing.T, got, want PieceMovedPayload) {
	t.Helper()
	if got.PieceID != want.PieceID || got.CharacterID != want.CharacterID || got.Z != want.Z {
		t.Fatalf("payload = %+v, want %+v", got, want)
	}
	if (got.Visible == nil) != (want.Visible == nil) || (got.Visible != nil && *got.Visible != *want.Visible) {
		t.Fatalf("Visible = %v, want %v", got.Visible, want.Visible)
	}
	if got.Slot.Kind != want.Slot.Kind {
		t.Fatalf("Slot.Kind = %q, want %q", got.Slot.Kind, want.Slot.Kind)
	}
	eqp := func(a, b *int) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	if !eqp(got.Slot.Col, want.Slot.Col) || !eqp(got.Slot.Row, want.Slot.Row) ||
		!eqp(got.Slot.Q, want.Slot.Q) || !eqp(got.Slot.R, want.Slot.R) {
		t.Fatalf("Slot = %+v, want %+v", got.Slot, want.Slot)
	}
}
