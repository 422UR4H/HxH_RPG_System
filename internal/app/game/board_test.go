package game

import (
	"context"
	"sync"
	"testing"
	"time"

	matchboarduc "github.com/422UR4H/HxH_RPG_System/internal/application/matchboard"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
)

func intp(i int) *int    { return &i }
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

// recordingSaveUC is a fake matchboarduc.ISaveMatchBoard that records every board handed to
// Save, in the order Save actually RAN. Sleeping on the FIRST call is what makes the order
// deterministic instead of a coin flip: persistBoard's persistMu wraps the snapshot AND the
// write (its own doc comment), so a second, faster caller can only reach Save once the first
// one — slow as it is — is fully done. Without that guarantee, this test would be flaky at
// best and silently wrong at worst: a slow first write finishing AFTER a fast second one would
// let the stale snapshot overwrite the fresher one in the store.
type recordingSaveUC struct {
	mu    sync.Mutex
	saved []*matchboard.Board
	calls int
}

var _ matchboarduc.ISaveMatchBoard = (*recordingSaveUC)(nil)

func (u *recordingSaveUC) Save(_ context.Context, b *matchboard.Board, _ []fogentity.PlayerMemory) error {
	u.mu.Lock()
	u.calls++
	first := u.calls == 1
	u.mu.Unlock()

	if first {
		time.Sleep(150 * time.Millisecond)
	}

	u.mu.Lock()
	defer u.mu.Unlock()
	cp := *b
	u.saved = append(u.saved, &cp)
	return nil
}

func (u *recordingSaveUC) last() *matchboard.Board {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.saved[len(u.saved)-1]
}

// TestPersistBoardWritesSnapshotsInOrder guards persistBoard's own review focus 2: two
// goroutines racing persistBoard, one right after the piece moves to A and the other right
// after it moves to B, must land in the DB in the order their SNAPSHOTS were taken (A then
// B) — not whichever Save call happens to finish first.
func TestPersistBoardWritesSnapshotsInOrder(t *testing.T) {
	matchUUID := uuid.New()
	mapUUID := uuid.New()
	const pieceID = "p1"
	saveUC := &recordingSaveUC{}

	r := NewRoom(matchUUID, uuid.New(), RoomDeps{SaveBoardUC: saveUC})
	r.mu.Lock()
	r.mapUUID = mapUUID
	r.pieces[pieceID] = PieceMovedPayload{
		PieceID: pieceID,
		Slot:    SlotPayload{Kind: "square", Col: intp(0), Row: intp(0)},
	}
	r.mu.Unlock()

	movePiece := func(col int) {
		r.mu.Lock()
		p := r.pieces[pieceID]
		p.Slot = SlotPayload{Kind: "square", Col: intp(col), Row: intp(0)}
		r.pieces[pieceID] = p
		r.mu.Unlock()
	}

	var wg sync.WaitGroup

	movePiece(1) // A
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.persistBoard("A") // the FIRST call to reach Save — the slow one.
	}()

	// Not a correctness requirement — persistMu serializes the two calls regardless of who
	// reaches it first — just what keeps THIS test's intended order (A's snapshot strictly
	// before B's) true, instead of leaving it to luck which goroutine's persistBoard call
	// happens to run first.
	time.Sleep(20 * time.Millisecond)

	movePiece(2) // B
	wg.Add(1)
	go func() {
		defer wg.Done()
		r.persistBoard("B")
	}()

	wg.Wait()

	if got := saveUC.calls; got != 2 {
		t.Fatalf("Save was called %d time(s), want 2", got)
	}
	last := saveUC.last()
	if len(last.Pieces) != 1 {
		t.Fatalf("last saved board has %d piece(s), want 1: %+v", len(last.Pieces), last.Pieces)
	}
	slot, ok := last.Pieces[0].Coord.Slot.(mapentity.SquareCoord)
	if !ok {
		t.Fatalf("Coord.Slot = %#v, want mapentity.SquareCoord", last.Pieces[0].Coord.Slot)
	}
	if slot.Col != 2 {
		t.Fatalf("last saved piece is at col %d, want 2 (position B): persistMu did not keep "+
			"the write order matching the snapshot order", slot.Col)
	}
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
