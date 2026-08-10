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

// A wall-only sync (no "pieces" field at all) must leave the board alone. Wiping
// r.pieces removes every LOS origin, which empties the player's visibility polygons
// and makes the fog cover the whole map — the exact production symptom.
func TestMapStateSync_WithoutPieceInfoKeepsBoard(t *testing.T) {
	room, masterUUID, _, _ := liveMatchRoom(t)
	master := NewClient(masterUUID, nil, "gm")
	room.clients[masterUUID] = master

	wall := room.walls["w1"]
	grid := toGridShapePayload(room.grid)
	raw := mustSyncMessage(t, MapStateSyncPayload{
		Pieces: nil, // field absent → "this sync carries no piece information"
		Walls:  []WallSegmentPayload{toWallSegmentPayload(wall)},
		Grid:   &grid,
	})

	withinTimeout(t, 3*time.Second, "handleClientMessage(map_state_sync)", func() {
		room.handleClientMessage(master, raw)
	})

	room.mu.RLock()
	got := len(room.pieces)
	room.mu.RUnlock()
	if got != 1 {
		t.Fatalf("wall-only map_state_sync wiped the board: got %d pieces, want 1", got)
	}
}

// The other half of the contract: an explicitly present (even empty) array is
// authoritative and does replace the board.
func TestMapStateSync_ExplicitEmptyArrayClearsBoard(t *testing.T) {
	room, masterUUID, _, _ := liveMatchRoom(t)
	master := NewClient(masterUUID, nil, "gm")
	room.clients[masterUUID] = master

	empty := []PieceMovedPayload{}
	grid := toGridShapePayload(room.grid)
	raw := mustSyncMessage(t, MapStateSyncPayload{
		Pieces: &empty,
		Walls:  []WallSegmentPayload{toWallSegmentPayload(room.walls["w1"])},
		Grid:   &grid,
	})

	withinTimeout(t, 3*time.Second, "handleClientMessage(map_state_sync)", func() {
		room.handleClientMessage(master, raw)
	})

	room.mu.RLock()
	got := len(room.pieces)
	room.mu.RUnlock()
	if got != 0 {
		t.Fatalf("explicit empty pieces array must clear the board: got %d pieces, want 0", got)
	}
}

// After the board is (re)seeded, the player's cached visibility must be refreshed and
// re-pushed. buildMapFullState reads the CACHE, so a sync that does not recompute leaves
// the player with the polygons computed when the board was still empty — total fog.
func TestMapStateSync_RefreshesPlayerVisibilityAndRepushesState(t *testing.T) {
	room, masterUUID, playerUUID, sheetUUID := liveMatchRoom(t)
	master := NewClient(masterUUID, nil, "gm")
	player := NewClient(playerUUID, nil, "p1")
	room.clients[masterUUID] = master
	room.clients[playerUUID] = player

	// Simulate the real ordering: visibility was computed while the board was empty.
	saved := room.pieces
	withinTimeout(t, 3*time.Second, "seed stale visibility", func() {
		room.mu.Lock()
		defer room.mu.Unlock()
		room.pieces = make(map[string]PieceMovedPayload)
		_, _ = room.session.RecomputeVisibility(playerUUID)
		room.pieces = saved
	})
	if len(room.visibilityFor(playerUUID)) != 0 {
		t.Fatal("precondition failed: expected an empty (stale) visibility cache")
	}

	pieces := []PieceMovedPayload{saved["p1"]}
	grid := toGridShapePayload(room.grid)
	raw := mustSyncMessage(t, MapStateSyncPayload{
		Pieces: &pieces,
		Walls:  []WallSegmentPayload{toWallSegmentPayload(room.walls["w1"])},
		Grid:   &grid,
	})

	withinTimeout(t, 3*time.Second, "handleClientMessage(map_state_sync)", func() {
		room.handleClientMessage(master, raw)
	})

	polys := room.visibilityFor(playerUUID)
	if len(polys) == 0 {
		t.Fatal("player has no visibility polygon after map_state_sync — fog would cover everything")
	}

	full := drainMapFullState(t, player)
	if full == nil {
		t.Fatal("player did not receive a refreshed map_full_state after map_state_sync")
	}
	if len(full.VisiblePolygons) == 0 {
		t.Fatal("refreshed map_full_state carries no visible_polygons")
	}
	if !hasPieceForCharacter(full.Pieces, sheetUUID.String()) {
		t.Fatal("player cannot see their own character's piece in the refreshed map_full_state")
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
