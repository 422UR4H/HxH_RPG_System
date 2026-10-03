package game

import (
	"encoding/json"
	"slices"
	"sync"
	"testing"
	"time"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// liveMatchRoom builds a Room in the "match already running" shape the player-facing
// fog depends on: a live session, a grid, one wall, and one piece per player placed on
// the board. It mirrors what the master's map_state_sync seeds in production.
func liveMatchRoom(t *testing.T) (room *Room, masterUUID, playerUUID uuid.UUID, sheetUUID uuid.UUID) {
	t.Helper()

	matchUUID := uuid.New()
	masterUUID = uuid.New()
	playerUUID = uuid.New()
	sheetUUID = uuid.New()

	room = newFogRoom(matchUUID, masterUUID)
	room.grid = mapentity.GridShape{
		Kind: mapentity.GridKindSquare, Cols: 20, Rows: 20, CellSize: 64, SkewRatio: 1,
	}

	participant := &match.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet:     csEntity.Summary{UUID: sheetUUID, PlayerUUID: &playerUUID},
	}
	session := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	wall := mapentity.WallSegment{
		ID: "w1", P1: [2]float64{192, 0}, P2: [2]float64{192, 320},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	room.walls[wall.ID] = wall
	room.pieces["p1"] = PieceMovedPayload{
		PieceID: "p1", CharacterID: sheetUUID.String(), Slot: squareSlot(1, 1),
	}

	room.session = session
	session.SyncMapState([]mapentity.WallSegment{wall}, room.grid)
	session.SetPieceSource(room)
	session.SyncPlayerMemories(nil, fogentity.FogModeExplored)
	room.state = RoomStatePlaying

	return room, masterUUID, playerUUID, sheetUUID
}

// withinTimeout runs fn on a goroutine and fails the test if it has not returned in time.
// A blocked mutex cannot be recovered from, so the goroutine is deliberately leaked and
// the test aborts — this is what makes a lock-ordering bug visible instead of silent.
func withinTimeout(t *testing.T, d time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("DEADLOCK: %s did not return within %s", what, d)
	}
}

// Every production path that refreshes fog (StartMatch, RehydrateSession,
// pushVisibilityUpdates, handlePieceMoved) calls session.RecomputeVisibility while
// holding r.mu for WRITING. RecomputeVisibility reaches back into the Room through
// PiecePositionSource. If that callback takes r.mu.RLock(), Go's RWMutex deadlocks
// permanently on the same goroutine and the whole room freezes.
func TestRecomputeVisibilityUnderRoomWriteLock_DoesNotDeadlock(t *testing.T) {
	room, _, playerUUID, _ := liveMatchRoom(t)

	withinTimeout(t, 3*time.Second, "RecomputeVisibility under r.mu.Lock()", func() {
		room.mu.Lock()
		defer room.mu.Unlock()
		if _, err := room.session.RecomputeVisibility(playerUUID); err != nil {
			t.Errorf("recompute visibility: %v", err)
		}
	})
}

func TestPushVisibilityUpdates_DoesNotDeadlock(t *testing.T) {
	room, masterUUID, playerUUID, _ := liveMatchRoom(t)
	room.clients[masterUUID] = NewClient(masterUUID, nil, "gm")
	room.clients[playerUUID] = NewClient(playerUUID, nil, "p1")

	withinTimeout(t, 3*time.Second, "pushVisibilityUpdates", func() {
		room.pushVisibilityUpdates()
	})
}

// B14 (spec §4.3, "Quem carrega") retires map_state_sync's write side entirely: the server
// owns the board now, loaded from the database at the moments loadBoard's own doc comment
// names, and this arm is kept only so the front that has not dropped the send yet (F13) gets
// an answer instead of silence. TestMapStateSync_WithoutPieceInfoKeepsBoard and
// TestMapStateSync_ExplicitEmptyArrayClearsBoard used to prove the OLD write behavior — the
// "nil pieces keeps the board, an explicit empty array clears it" contract this arm no longer
// has, since it does not write under EITHER shape any more. Repurposed below into
// TestMapStateSync_NeverWritesTheBoard, which proves the new contract covers both shapes at
// once. TestMapStateSync_RefreshesPlayerVisibilityAndRepushesState asserted the recompute-and-
// repush-to-every-client half of the old behavior, which B14 also removes (the answer now
// goes only to the sender) — repurposed into
// TestMapStateSync_AnswersOnlyTheSenderWithoutTouchingTheSession below.

// A sync payload carrying pieces, walls and a grid that all disagree with the real board must
// change NOTHING: no new piece appears, the real one stays, and the grid is not overwritten.
// Covers both of the old contract's shapes (nil Pieces and a present-but-different array) with
// one assertion, because neither one means anything to this arm any more.
func TestMapStateSync_NeverWritesTheBoard(t *testing.T) {
	room, masterUUID, _, _ := liveMatchRoom(t)
	master := NewClient(masterUUID, nil, "gm")
	room.clients[masterUUID] = master

	intruder := []PieceMovedPayload{{
		PieceID: "intruder", CharacterID: uuid.New().String(), Slot: squareSlot(9, 9),
	}}
	grid := toGridShapePayload(mapentity.GridShape{
		Kind: mapentity.GridKindSquare, Cols: 5, Rows: 5, CellSize: 999, SkewRatio: 1,
	})
	raw := mustSyncMessage(t, MapStateSyncPayload{
		Pieces: &intruder,
		Walls:  nil,
		Grid:   &grid,
	})

	withinTimeout(t, 3*time.Second, "handleClientMessage(map_state_sync)", func() {
		room.handleClientMessage(master, raw)
	})

	room.mu.RLock()
	_, hasIntruder := room.pieces["intruder"]
	_, hasReal := room.pieces["p1"]
	_, hasWall := room.walls["w1"]
	gotCellSize := room.grid.CellSize
	room.mu.RUnlock()
	if hasIntruder {
		t.Fatal("map_state_sync wrote a piece from its payload — it must write nothing at all (spec §4.3, B14)")
	}
	if !hasReal {
		t.Fatal("map_state_sync erased the real board's piece — it must write nothing at all")
	}
	if !hasWall {
		t.Fatal("map_state_sync erased the real board's wall (payload sent Walls: nil) — it must write nothing at all")
	}
	if gotCellSize == 999 {
		t.Fatal("map_state_sync overwrote the grid from its payload — it must write nothing at all")
	}
}

// The other half of B14's contract: the answer goes to the SENDER alone, and nothing about
// the session — a player's cached visibility included — is touched along the way. Before
// B14 this same sync would have recomputed and re-pushed to every client; now the recompute
// stays as stale as it was, which is exactly the proof that map_state_sync no longer reaches
// the session at all.
func TestMapStateSync_AnswersOnlyTheSenderWithoutTouchingTheSession(t *testing.T) {
	room, masterUUID, playerUUID, sheetUUID := liveMatchRoom(t)
	master := NewClient(masterUUID, nil, "gm")
	player := NewClient(playerUUID, nil, "p1")
	room.clients[masterUUID] = master
	room.clients[playerUUID] = player

	withinTimeout(t, 3*time.Second, "invalidate visibility cache", func() {
		room.mu.Lock()
		defer room.mu.Unlock()
		room.session.InvalidateVisibilityCache()
	})
	if len(room.visibilityFor(playerUUID)) != 0 {
		t.Fatal("precondition failed: expected an empty (invalidated) visibility cache")
	}

	grid := toGridShapePayload(room.grid)
	raw := mustSyncMessage(t, MapStateSyncPayload{Grid: &grid})

	withinTimeout(t, 3*time.Second, "handleClientMessage(map_state_sync)", func() {
		room.handleClientMessage(master, raw)
	})

	if len(room.visibilityFor(playerUUID)) != 0 {
		t.Fatal("map_state_sync recomputed a player's visibility — it must not touch the session at all")
	}
	select {
	case <-player.send:
		t.Fatal("the player received a message from the MASTER's map_state_sync — it must answer only the sender")
	default:
	}

	full := drainMapFullState(t, master)
	if full == nil {
		t.Fatal("the sender never received a map_full_state answer")
	}
	if !hasPieceForCharacter(full.Pieces, sheetUUID.String()) {
		t.Fatal("the sender's map_full_state answer does not carry the real board's piece")
	}
}

// The end-to-end guarantee the feature exists for: a player in a started match sees a
// lit area around their own piece, and the master sees the board with no fog at all.
func TestPlayerSeesOwnPieceAndMasterHasNoFog(t *testing.T) {
	room, masterUUID, playerUUID, sheetUUID := liveMatchRoom(t)

	withinTimeout(t, 3*time.Second, "RecomputeVisibility", func() {
		room.mu.Lock()
		defer room.mu.Unlock()
		_, _ = room.session.RecomputeVisibility(playerUUID)
	})

	playerView := decodeMapFull(t, room.buildMapFullState(playerUUID, false))
	if len(playerView.VisiblePolygons) == 0 {
		t.Fatal("player received zero visibility polygons — the whole map stays fogged")
	}
	if len(playerView.VisiblePolygons[0]) < 3 {
		t.Fatalf("visibility polygon is degenerate: %d vertices", len(playerView.VisiblePolygons[0]))
	}
	if !hasPieceForCharacter(playerView.Pieces, sheetUUID.String()) {
		t.Fatal("player cannot see their own character's piece")
	}

	masterView := decodeMapFull(t, room.buildMapFullState(masterUUID, true))
	if len(masterView.VisiblePolygons) != 0 {
		t.Fatal("master must not receive visibility polygons (no fog for the master)")
	}
	if len(masterView.Pieces) != 1 {
		t.Fatalf("master must see every piece: got %d, want 1", len(masterView.Pieces))
	}
}

// A visibility polygon must stay inside the board. When it does not, the client cannot
// rasterize the LOS mask correctly — the fog renders as a black smear spilling past the
// background image.
func TestRecomputeVisibility_PolygonStaysInsideTheBoard(t *testing.T) {
	room, _, playerUUID, _ := liveMatchRoom(t)

	var polys []domainservice.VisibilityPolygon
	withinTimeout(t, 3*time.Second, "RecomputeVisibility", func() {
		room.mu.Lock()
		defer room.mu.Unlock()
		polys, _ = room.session.RecomputeVisibility(playerUUID)
	})
	if len(polys) == 0 {
		t.Fatal("expected a visibility polygon")
	}

	grid := room.grid
	boardW := float64(grid.Cols) * grid.CellSize
	boardH := float64(grid.Rows) * grid.CellSize
	const tol = 1.0 // the sweep may land a hair outside from epsilon-rounded corner rays

	for _, poly := range polys {
		for _, v := range poly.Vertices {
			if v.X < -tol || v.X > boardW+tol || v.Y < -tol || v.Y > boardH+tol {
				t.Fatalf("polygon vertex (%.0f, %.0f) is outside the %.0fx%.0f board",
					v.X, v.Y, boardW, boardH)
			}
		}
	}
}

// Per-player dispatch runs on the connection's goroutine while the room may be closing
// on its own. If shutdown closes the client's send channel, that send panics with
// "send on closed channel" and takes the whole game server down. Run under -race.
func TestSendMessageDuringRoomShutdown_DoesNotPanic(t *testing.T) {
	room, masterUUID, playerUUID, _ := liveMatchRoom(t)
	master := NewClient(masterUUID, nil, "gm")
	player := NewClient(playerUUID, nil, "p1")
	room.clients[masterUUID] = master
	room.clients[playerUUID] = player

	go room.Run()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if rec := recover(); rec != nil {
				t.Errorf("SendMessage panicked during shutdown: %v", rec)
			}
		}()
		for range 500 {
			room.pushVisibilityUpdates()
		}
	}()

	time.Sleep(2 * time.Millisecond)
	room.Stop()
	wg.Wait()
}

// A wall that blocks the player's vision must still be sent to them: they have to see
// the wall (and doors) bounding their view in order to interact with it, even though
// they cannot see what lies behind it. Such a wall sits exactly ON the boundary of the
// visibility polygon, so testing its midpoint for containment reports "not visible" and
// the wall silently disappears from the player's screen.
func TestBuildMapFullState_PlayerSeesTheWallThatBlocksTheirVision(t *testing.T) {
	room, _, playerUUID, _ := liveMatchRoom(t)

	withinTimeout(t, 3*time.Second, "RecomputeVisibility", func() {
		room.mu.Lock()
		defer room.mu.Unlock()
		_, _ = room.session.RecomputeVisibility(playerUUID)
	})

	view := decodeMapFull(t, room.buildMapFullState(playerUUID, false))
	found := false
	for _, w := range view.Walls {
		if w.ID == "w1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("player cannot see the wall blocking their line of sight (got %d walls)",
			len(view.Walls))
	}
}

// A wall the player has no line of sight to must stay hidden — the fix above must not
// turn into "players see every wall on the map".
func TestBuildMapFullState_PlayerDoesNotSeeWallBehindAnother(t *testing.T) {
	room, _, playerUUID, _ := liveMatchRoom(t)

	// Directly in w1's shadow: same span, just past it. The nudge that makes a blocking
	// wall visible must not leak the wall immediately behind it.
	hidden := mapentity.WallSegment{
		ID: "w-far", P1: [2]float64{208, 32}, P2: [2]float64{208, 288},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	room.walls[hidden.ID] = hidden
	room.session.SyncMapState([]mapentity.WallSegment{room.walls["w1"], hidden}, room.grid)

	withinTimeout(t, 3*time.Second, "RecomputeVisibility", func() {
		room.mu.Lock()
		defer room.mu.Unlock()
		_, _ = room.session.RecomputeVisibility(playerUUID)
	})

	view := decodeMapFull(t, room.buildMapFullState(playerUUID, false))
	for _, w := range view.Walls {
		if w.ID == "w-far" {
			t.Fatal("player must not see a wall they have no line of sight to")
		}
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

// toGridShapePayload is the test-only mirror of message.go's toWallSegmentPayload:
// production code never sends a GridShape back out over the wire (map_state_sync is
// master→server only), so no entity→payload conversion exists there. Tests that need to
// simulate that inbound sync build the payload directly instead.
func toGridShapePayload(g mapentity.GridShape) GridShapePayload {
	return GridShapePayload{
		Kind:      string(g.Kind),
		Cols:      g.Cols,
		Rows:      g.Rows,
		CellSize:  g.CellSize,
		SkewRatio: g.SkewRatio,
		Rotation:  g.Rotation,
		Color:     g.Color,
		Opacity:   g.Opacity,
		LineStyle: string(g.LineStyle),
	}
}

func mustSyncMessage(t *testing.T, p MapStateSyncPayload) []byte {
	t.Helper()
	payload, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	raw, err := json.Marshal(map[string]any{
		"type":    string(MsgTypeMapStateSync),
		"payload": json.RawMessage(payload),
	})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	return raw
}

// drainMapFullState returns the LAST map_full_state buffered for the client, or nil.
func drainMapFullState(t *testing.T, c *Client) *MapFullStatePayload {
	t.Helper()
	var last *MapFullStatePayload
	for {
		select {
		case data := <-c.send:
			var m Message
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatalf("unmarshal message: %v", err)
			}
			if m.Type != MsgTypeMapFullState {
				continue
			}
			var p MapFullStatePayload
			if err := json.Unmarshal(m.Payload, &p); err != nil {
				t.Fatalf("unmarshal map_full_state: %v", err)
			}
			last = &p
		default:
			return last
		}
	}
}

func hasPieceForCharacter(pieces []PieceMovedPayload, characterID string) bool {
	for _, p := range pieces {
		if p.CharacterID == characterID {
			return true
		}
	}
	return false
}

// A piece standing behind a wall, outside the player's line of sight, must never reach
// that player — not the piece payload, not its position. Seeing where an enemy stands
// through the fog is the whole thing the feature exists to prevent.
func TestBuildMapFullState_PlayerDoesNotSeePieceBehindWall(t *testing.T) {
	room, masterUUID, playerUUID, _ := liveMatchRoom(t)

	// liveMatchRoom puts w1 at x=192 spanning y 0..320, and the player's piece at
	// slot (1,1) → world (96,96). Slot (4,1) → world (288,96) is on the far side.
	room.pieces["enemy"] = PieceMovedPayload{
		PieceID: "enemy", CharacterID: uuid.New().String(), Slot: squareSlot(4, 1),
	}

	withinTimeout(t, 3*time.Second, "RecomputeVisibility", func() {
		room.mu.Lock()
		defer room.mu.Unlock()
		_, _ = room.session.RecomputeVisibility(playerUUID)
	})

	playerView := decodeMapFull(t, room.buildMapFullState(playerUUID, false))
	for _, p := range playerView.Pieces {
		if p.PieceID == "enemy" {
			t.Fatal("a piece behind a wall was sent to the player — position leak through the fog")
		}
	}

	// The master still sees it: the filtering is per-viewer, not a global drop.
	masterView := decodeMapFull(t, room.buildMapFullState(masterUUID, true))
	if !slices.ContainsFunc(masterView.Pieces, func(p PieceMovedPayload) bool {
		return p.PieceID == "enemy"
	}) {
		t.Fatal("master must see every piece")
	}
}
