package game_test

// This file is B5, B6 and B10 (design spec §4.3 "B5, B6 e B10"): the origin a Move is
// checked against is never the client's — it is the actor's own piece position on the
// server's board (B6) — and the wall check itself reads cell CENTERS via
// mapservice.SlotCenterToWorld with the session's own grid (B5), with the right conversion
// for a hexagonal grid too (B10). Before this task the check multiplied col×cellSize (a
// cell's top-left CORNER, wrong on both grids) and skipped validation whenever the client's
// declared "from" happened to be the zero value [0,0,0] — indistinguishable, under that old
// sentinel, from "not provided", even when a piece was genuinely standing at (0,0).

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// squareGeomGrid is the grid every square-grid case in this file shares: 20x20 cells of 64px.
var squareGeomGrid = mapentity.GridShape{
	Kind: mapentity.GridKindSquare, Cols: 20, Rows: 20, CellSize: 64, SkewRatio: 1,
}

// hexGeomGrid is the grid the hex case uses — same shape escape_landing_test.go already
// exercises for hex axial math, kept here so this file does not silently drift from it.
var hexGeomGrid = mapentity.GridShape{
	Kind: mapentity.GridKindHex, Cols: 10, Rows: 10, CellSize: 64, SkewRatio: 1,
}

// dividingWall builds a wall segment perpendicular to a→b, centered on a→b's own midpoint,
// long enough (100 world units, split evenly across the midpoint) to guarantee an
// intersection with the straight a→b path regardless of that path's own length or
// orientation. It exists so every "blocked" case in this file can say exactly which two
// slot centers its wall sits between, in world coordinates produced by
// mapservice.SlotCenterToWorld, rather than hand-picked P1/P2 numbers.
func dividingWall(id string, a, b [2]float64) mapentity.WallSegment {
	mx, my := (a[0]+b[0])/2, (a[1]+b[1])/2
	dx, dy := b[0]-a[0], b[1]-a[1]
	px, py := -dy, dx
	length := math.Hypot(px, py)
	if length == 0 {
		length = 1
	}
	const halfSpan = 50.0
	scale := halfSpan / length
	px, py = px*scale, py*scale
	return mapentity.WallSegment{
		ID: id, P1: [2]float64{mx - px, my - py}, P2: [2]float64{mx + px, my + py},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Move: true, Direction: mapentity.WallDirectionBoth, HP: 100, MaxHP: 100,
	}
}

// bandWall builds a short, purely vertical wall straddling worldX at a height band
// [centerY-grid.CellSize/4, centerY+grid.CellSize/4] — wide enough to catch a horizontal
// path that passes through centerY, narrow enough (half of one row) to clear a horizontal
// path a full half-cell away. It exists so the two tests below can each say exactly which ONE
// of the two candidate y-levels — the true slot center, or the pre-B5 col×cellSize corner —
// their wall brackets, and which one it deliberately clears.
func bandWall(id string, worldX, centerY, cellSize float64) mapentity.WallSegment {
	half := cellSize / 4
	return mapentity.WallSegment{
		ID: id, P1: [2]float64{worldX, centerY - half}, P2: [2]float64{worldX, centerY + half},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Move: true, Direction: mapentity.WallDirectionBoth, HP: 100, MaxHP: 100,
	}
}

// seedGeomBoard puts a board directly into the fixture's fake store, exactly like
// combat_e2e_test.go's seedBoard — but with THIS file's own grid/pieces/walls, since every
// case here needs a different layout. Must run before f.connect: the master's register loads
// the board once, synchronously with the room's birth.
func seedGeomBoard(
	t *testing.T, f *combatFixture, grid mapentity.GridShape,
	pieces []mapentity.Piece, walls []mapentity.WallSegment,
) {
	t.Helper()
	f.boards.seed(f.matchUUID, &matchboard.Board{
		MatchUUID: f.matchUUID, MapUUID: uuid.New(),
		Grid: grid, Pieces: pieces, Walls: walls,
	})
}

func squarePiece(id string, charID uuid.UUID, col, row int) mapentity.Piece {
	return mapentity.Piece{
		ID: id, CharacterID: charID.String(),
		Coord:   mapentity.PieceCoord{Slot: mapentity.SquareCoord{Kind: "square", Col: col, Row: row}},
		Visible: true,
	}
}

func hexPiece(id string, charID uuid.UUID, q, r int) mapentity.Piece {
	return mapentity.Piece{
		ID: id, CharacterID: charID.String(),
		Coord:   mapentity.PieceCoord{Slot: mapentity.HexCoord{Kind: "hex", Q: q, R: r}},
		Visible: true,
	}
}

// sendGeomDash sends a bare Dash — no attack, no target — exactly like combat_e2e_test.go's
// own enqueueDash, but for an arbitrary actor rather than always the fixture's attacker
// (several cases below move the victim instead, on purpose, to keep the piece the movement
// check exercises unambiguous within each sub-test).
func sendGeomDash(t *testing.T, conn *websocket.Conn, actorID uuid.UUID, from, to [3]int) {
	t.Helper()
	sendWS(t, conn, "enqueue_action", map[string]any{
		"actorId": actorID.String(),
		"move": map[string]any{
			"category": string(enum.Dash),
			"from":     from,
			"position": to,
		},
	})
}

// awaitErrorWithCode waits for an "error" message with the given code and returns it, or
// fails. Named apart from lobby_board_e2e_test.go's own awaitError (no code filter, same
// package) rather than reusing it, since every case here needs to wait for a SPECIFIC code.
func awaitErrorWithCode(t *testing.T, c *collector, code string, d time.Duration) game.ErrorPayload {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		for _, m := range c.snapshotMessages() {
			if m.Type != game.MsgTypeError {
				continue
			}
			var ep game.ErrorPayload
			if err := json.Unmarshal(m.Payload, &ep); err != nil {
				continue
			}
			if ep.Code == code {
				return ep
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("never saw an error with code %q", code)
	return game.ErrorPayload{}
}

// refutesError asserts NO error with this code ever arrived, after waiting long enough for
// the positive signal (want) to have landed first — so a slow server cannot produce a false
// pass by the assertion running before the error would have.
func refutesErrorAfter(t *testing.T, c *collector, code string, want game.MessageType, d time.Duration) {
	t.Helper()
	if !c.await(want, d) {
		t.Fatalf("never saw %s — cannot tell whether %q was correctly absent", want, code)
	}
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeError {
			continue
		}
		var ep game.ErrorPayload
		if err := json.Unmarshal(m.Payload, &ep); err == nil && ep.Code == code {
			t.Fatalf("got an unexpected %q error: %s", code, ep.Message)
		}
	}
}

// TestE2E_MoveWallCheck_CenterCrossesAWallTheOldCornerFormulaWouldHaveMissed is B5's positive
// case, built to actually DISCRIMINATE the fix rather than just exercise a wall that happens
// to block under both the old and the new geometry.
//
// Before B5, the server computed a piece's world position as `col × cellSize, row × cellSize`
// — a cell's own top-left CORNER, one half-cell short (in both x and y) of its true center
// (`(col+0.5) × cellSize`, `(row+0.5) × cellSize`). For a Dash from (4,4) to (5,4) on a
// 64px grid that puts the two candidate horizontal paths a half-cell apart:
//
//	old (corner) path: y = 256, x ∈ [256, 320]
//	new (center) path: y = 288, x ∈ [288, 352]
//
// The wall below is a short vertical band straddling x=320 (the midpoint between the two
// centers) from y=272 to y=304 — it brackets the CENTER path's y=288 with room to spare, and
// sits clear of the CORNER path's y=256 by 16 world units. So: the real fix (centers) must
// report this Dash blocked; the old, buggy formula (corners) would have missed this wall
// entirely and reported it clear. Verified against both formulas directly (not just this
// e2e path) in the fix report for this task.
func TestE2E_MoveWallCheck_CenterCrossesAWallTheOldCornerFormulaWouldHaveMissed(t *testing.T) {
	f := newCombatFixture(t)
	cellSize := squareGeomGrid.CellSize
	fx, fy := mapservice.SlotCenterToWorld(4, 4, squareGeomGrid)
	tx, ty := mapservice.SlotCenterToWorld(5, 4, squareGeomGrid)
	wall := bandWall("band", (fx+tx)/2, (fy+ty)/2, cellSize)
	seedGeomBoard(t, f, squareGeomGrid,
		[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 4, 4)},
		[]mapentity.WallSegment{wall})

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	playerMsgs := collectFrom(player)
	if !collectFrom(master).await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}

	// The payload's own "from" is garbage on purpose (checked properly, separately, below) —
	// this test only cares that (4,4)→(5,4) is blocked.
	sendGeomDash(t, player, f.attackerID, [3]int{9, 9, 0}, [3]int{5, 4, 0})
	awaitErrorWithCode(t, playerMsgs, "move_blocked", 2*time.Second)
}

// TestE2E_MoveWallCheck_OldCornerFormulaFalselyBlockedThisCenterFormulaDoesNot is B5's
// negative case — the mirror of the one above, proving the fix also REMOVES a false
// positive, not just adds a true one.
//
// Same Dash, (4,4)→(5,4), same x=320. This wall instead brackets the OLD CORNER path's
// y=256 (band y ∈ [240, 272]) and sits clear of the NEW CENTER path's y=288 by 16 world
// units. So: the old, buggy formula (corners) would have reported this Dash wrongly
// blocked — the piece's "position" sat exactly on this wall's own corner at (320, 256). The
// real fix (centers) correctly clears it: the center-to-center path at y=288 never comes
// near this band. Verified against both formulas directly in the fix report for this task.
func TestE2E_MoveWallCheck_OldCornerFormulaFalselyBlockedThisCenterFormulaDoesNot(t *testing.T) {
	f := newCombatFixture(t)
	cellSize := squareGeomGrid.CellSize
	fx, fy := mapservice.SlotCenterToWorld(4, 4, squareGeomGrid)
	tx, _ := mapservice.SlotCenterToWorld(5, 4, squareGeomGrid)
	oldCornerY := fy - cellSize/2 // the pre-B5 col×cellSize formula's y for row 4
	wall := bandWall("band", (fx+tx)/2, oldCornerY, cellSize)
	seedGeomBoard(t, f, squareGeomGrid,
		[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 4, 4)},
		[]mapentity.WallSegment{wall})

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	playerMsgs := collectFrom(player)
	if !collectFrom(master).await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}

	sendGeomDash(t, player, f.attackerID, [3]int{9, 9, 0}, [3]int{5, 4, 0})
	refutesErrorAfter(t, playerMsgs, "move_blocked", game.MsgTypeActionEnqueued, 2*time.Second)
}

// TestE2E_MoveWallCheck_ZeroZeroIsNotASentinelAnymore is B6's other half: before this task,
// a piece genuinely sitting at grid (0,0) got its Move.From compared against the payload's
// zero-value SENTINEL for "not provided" and skipped the wall check outright, since [0,0,0]
// could not be told apart from "the client sent nothing". A *[3]int has no such ambiguity —
// nil means "no piece"; a non-nil pointer to [0,0,0] means exactly what it says.
func TestE2E_MoveWallCheck_ZeroZeroIsNotASentinelAnymore(t *testing.T) {
	f := newCombatFixture(t)
	fromX, fromY := mapservice.SlotCenterToWorld(0, 0, squareGeomGrid)
	toX, toY := mapservice.SlotCenterToWorld(0, 2, squareGeomGrid)
	wall := dividingWall("divider", [2]float64{fromX, fromY}, [2]float64{toX, toY})
	seedGeomBoard(t, f, squareGeomGrid,
		[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 0, 0)},
		[]mapentity.WallSegment{wall})

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	playerMsgs := collectFrom(player)
	if !collectFrom(master).await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}

	sendGeomDash(t, player, f.attackerID, [3]int{0, 0, 0}, [3]int{0, 2, 0})
	awaitErrorWithCode(t, playerMsgs, "move_blocked", 2*time.Second)
}

// TestE2E_MoveFromIsDerivedFromThePieceNeverThePayload is B6's core promise: the client's
// "from" is parsed and discarded. The payload here lies about it ([9,9,0]) while the
// attacker's real piece sits at (4,4) — the check must use (4,4) (there is no wall in this
// board to fail against), and the master's own queue ack must report (4,4), not the lie.
func TestE2E_MoveFromIsDerivedFromThePieceNeverThePayload(t *testing.T) {
	f := newCombatFixture(t)
	seedGeomBoard(t, f, squareGeomGrid,
		[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 4, 4)}, nil)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}

	sendGeomDash(t, player, f.attackerID, [3]int{9, 9, 0}, [3]int{6, 4, 0})
	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master never saw action_queued")
	}

	var queued game.ActionQueuedPayload
	if err := json.Unmarshal(
		findMessage(t, masterMsgs.snapshotMessages(), game.MsgTypeActionQueued).Payload, &queued,
	); err != nil {
		t.Fatalf("unmarshal action_queued: %v", err)
	}
	if queued.Action.Move == nil {
		t.Fatal("action_queued carried no move at all")
	}
	if queued.Action.Move.From == nil {
		t.Fatal("action_queued.action.move.from is absent, want the piece's own position")
	}
	if got := *queued.Action.Move.From; got != [3]int{4, 4, 0} {
		t.Fatalf("action_queued.action.move.from = %v, want [4 4 0] (the piece's real slot, "+
			"not the payload's [9 9 0])", got)
	}
}

// TestE2E_MoveFromIsAbsentWhenTheActorHasNoPiece is B6's other edge: an actor with nothing on
// the board gets no wall check (there is no origin to check from) and no "from" on the wire
// at all — actionwire.Move.From is a *[3]int with omitempty specifically so this is "absent",
// not a misleading zero value.
func TestE2E_MoveFromIsAbsentWhenTheActorHasNoPiece(t *testing.T) {
	f := newCombatFixture(t)
	// Only the attacker gets a piece; the victim — who sends this Dash — has none.
	seedGeomBoard(t, f, squareGeomGrid,
		[]mapentity.Piece{squarePiece("p-attacker", f.attackerID, 4, 4)}, nil)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}

	sendGeomDash(t, player, f.victimID, [3]int{1, 1, 0}, [3]int{2, 2, 0})
	refutesErrorAfter(t, playerMsgs, "move_blocked", game.MsgTypeActionEnqueued, 2*time.Second)
	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master never saw action_queued")
	}

	var queued game.ActionQueuedPayload
	if err := json.Unmarshal(
		findMessage(t, masterMsgs.snapshotMessages(), game.MsgTypeActionQueued).Payload, &queued,
	); err != nil {
		t.Fatalf("unmarshal action_queued: %v", err)
	}
	if queued.Action.Move == nil {
		t.Fatal("action_queued carried no move at all")
	}
	if queued.Action.Move.From != nil {
		t.Fatalf("action_queued.action.move.from = %v, want absent — the actor has no piece",
			*queued.Action.Move.From)
	}
}

// TestE2E_HexGridWallBlocksTheStepAndAppliesTheMoveWithQR is B10: the wall check on a hex
// grid must use the hex axial→world conversion (mapservice.SlotCenterToWorld already branches
// on GridKindHex), not the square formula. It also closes the brief's open question — no
// existing e2e test moved a piece across a hex grid and checked that applyMove wrote q/r
// (as opposed to col/row) on the resulting piece_moved.
func TestE2E_HexGridWallBlocksTheStepAndAppliesTheMoveWithQR(t *testing.T) {
	f := newCombatFixture(t)
	fromX, fromY := mapservice.SlotCenterToWorld(2, 1, hexGeomGrid)
	toX, toY := mapservice.SlotCenterToWorld(3, 1, hexGeomGrid)
	wall := dividingWall("hex-divider", [2]float64{fromX, fromY}, [2]float64{toX, toY})
	seedGeomBoard(t, f, hexGeomGrid,
		[]mapentity.Piece{hexPiece("p-attacker", f.attackerID, 2, 1)},
		[]mapentity.WallSegment{wall})

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board — the fixture never started")
	}

	// The step from (2,1) to (3,1) crosses the wall: blocked.
	sendGeomDash(t, player, f.attackerID, [3]int{2, 1, 0}, [3]int{3, 1, 0})
	awaitErrorWithCode(t, playerMsgs, "move_blocked", 2*time.Second)

	// A step away from the wall, in the opposite direction, is NOT blocked — and once opened,
	// applyMove must write it back as a HEX slot (q/r), not a square one (col/row).
	sendGeomDash(t, player, f.attackerID, [3]int{2, 1, 0}, [3]int{1, 1, 0})
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the unobstructed hex dash was never enqueued")
	}

	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("no piece_moved: the resolved hex move was never applied to the board")
	}
	var mp game.PieceMovedPayload
	if err := json.Unmarshal(
		findMessage(t, masterMsgs.snapshotMessages(), game.MsgTypePieceMoved).Payload, &mp,
	); err != nil {
		t.Fatalf("unmarshal piece_moved: %v", err)
	}
	if mp.Slot.Kind != "hex" {
		t.Fatalf("slot kind = %q, want %q — applyMove must keep the piece's own hex kind", mp.Slot.Kind, "hex")
	}
	if mp.Slot.Q == nil || mp.Slot.R == nil {
		t.Fatalf("piece_moved carried no hex q/r: %+v", mp.Slot)
	}
	if *mp.Slot.Q != 1 || *mp.Slot.R != 1 {
		t.Fatalf("piece landed on (q=%d,r=%d), want (q=1,r=1)", *mp.Slot.Q, *mp.Slot.R)
	}
	if mp.Slot.Col != nil || mp.Slot.Row != nil {
		t.Fatalf("piece_moved carried square col/row (%v,%v) on a hex piece", mp.Slot.Col, mp.Slot.Row)
	}
}
