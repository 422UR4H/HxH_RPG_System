//go:build smoke

// Live smoke test for fog of war. Unlike the unit and E2E tests, this one talks to the
// REAL game server on :8081 with a REAL match, driving exactly the sequence the browser
// drives: the master seeds the board it loaded from the map, then a player connects and
// must receive line of sight.
//
// The board comes from a JSON file ({"pieces":…,"walls":…,"grid":…}) so the test needs
// no database driver and no login session. Dump it from the dev DB with:
//
//	psql "$DB_URL" -Atc "select json_build_object('pieces',pieces,'walls',walls,'grid',grid)
//	  from maps where uuid='<map-uuid>';" > /tmp/board.json
//
// Then, with the servers up (./dev-checkout.sh <branch>):
//
//	SMOKE_MATCH=<match-uuid> SMOKE_BOARD=/tmp/board.json \
//	SMOKE_MASTER=<master-user-uuid> SMOKE_PLAYER=<player-user-uuid> \
//	SMOKE_SHEET=<player-sheet-uuid> \
//	go test -tags=smoke -count=1 ./internal/app/game/ -run TestSmokeFog -v
package game_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	pkgAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const smokeWsURL = "ws://localhost:8081"

// smokeBoard is the subset of a map the game server needs to seed a room.
type smokeBoard struct {
	Pieces []struct {
		ID    string `json:"id"`
		Coord struct {
			Slot game.SlotPayload `json:"slot"`
		} `json:"coord"`
		Visible     bool   `json:"visible"`
		CharacterID string `json:"character_id"`
	} `json:"pieces"`
	Walls []mapentity.WallSegment `json:"walls"`
	Grid  mapentity.GridShape     `json:"grid"`
}

func smokeEnv(t *testing.T, key string) uuid.UUID {
	t.Helper()
	raw := os.Getenv(key)
	if raw == "" {
		t.Fatalf("%s is required (see the header of this file)", key)
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		t.Fatalf("%s is not a UUID: %v", key, err)
	}
	return id
}

func smokeToken(t *testing.T, userUUID uuid.UUID) string {
	t.Helper()
	token, err := pkgAuth.GenerateToken(userUUID)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func smokeLoadBoard(t *testing.T) smokeBoard {
	t.Helper()
	path := os.Getenv("SMOKE_BOARD")
	if path == "" {
		t.Fatal("SMOKE_BOARD is required (see the header of this file)")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read board %s: %v", path, err)
	}
	var b smokeBoard
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatalf("decode board: %v", err)
	}
	return b
}

func smokeDial(t *testing.T, userUUID, matchUUID uuid.UUID) *websocket.Conn {
	t.Helper()
	url := fmt.Sprintf("%s/ws?match_uuid=%s&token=%s&nickname=smoke",
		smokeWsURL, matchUUID, smokeToken(t, userUUID))

	type result struct {
		conn *websocket.Conn
		err  error
	}
	done := make(chan result, 1)
	go func() {
		c, resp, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil && resp != nil {
			err = fmt.Errorf("status %d: %w", resp.StatusCode, err)
		}
		done <- result{c, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("dial %s: %v", strings.Split(url, "&token=")[0], r.err)
		}
		return r.conn
	case <-time.After(10 * time.Second):
		t.Fatal("DEADLOCK: the game server never completed the handshake — the room is frozen")
		return nil
	}
}

// TestSmokeFogAgainstLiveServers is the acceptance check: against the real servers and
// the real match, a player must receive a visibility polygon and see their own piece,
// and the master must receive no fog at all.
func TestSmokeFogAgainstLiveServers(t *testing.T) {
	matchUUID := smokeEnv(t, "SMOKE_MATCH")
	masterUUID := smokeEnv(t, "SMOKE_MASTER")
	playerUUID := smokeEnv(t, "SMOKE_PLAYER")
	sheetUUID := smokeEnv(t, "SMOKE_SHEET")

	board := smokeLoadBoard(t)
	t.Logf("board: %d pieces, %d walls, grid %dx%d cell=%.0f",
		len(board.Pieces), len(board.Walls), board.Grid.Cols, board.Grid.Rows, board.Grid.CellSize)
	if len(board.Pieces) == 0 {
		t.Fatal("the map has no pieces — nothing can light up the fog")
	}

	master := smokeDial(t, masterUUID, matchUUID)
	defer master.Close()
	t.Log("master connected (session rehydrated without deadlock)")

	// Exactly what the fixed useMatchWs sends once the REST map has loaded.
	pieces := make([]game.PieceMovedPayload, 0, len(board.Pieces))
	for _, p := range board.Pieces {
		visible := p.Visible
		pieces = append(pieces, game.PieceMovedPayload{
			PieceID:     p.ID,
			Slot:        p.Coord.Slot,
			CharacterID: p.CharacterID,
			Visible:     &visible,
		})
	}
	payload, err := json.Marshal(game.MapStateSyncPayload{
		Pieces: &pieces, Walls: board.Walls, Grid: &board.Grid,
	})
	if err != nil {
		t.Fatalf("marshal sync: %v", err)
	}
	raw, err := json.Marshal(map[string]any{
		"type": "map_state_sync", "payload": json.RawMessage(payload),
	})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}

	player := smokeDial(t, playerUUID, matchUUID)
	defer player.Close()
	t.Log("player connected")

	if err := master.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("master send map_state_sync: %v", err)
	}
	t.Logf("master synced %d pieces and %d walls", len(pieces), len(board.Walls))

	playerView := smokeAwaitMapFullState(t, player, 10*time.Second)
	if playerView == nil {
		t.Fatal("FAIL: player never received map_full_state — the screen stays fully fogged")
	}
	if len(playerView.VisiblePolygons) == 0 {
		t.Fatal("FAIL: player received zero visibility polygons — the fog covers the whole map")
	}
	if len(playerView.VisiblePolygons[0]) < 3 {
		t.Fatalf("FAIL: visibility polygon is degenerate (%d vertices)", len(playerView.VisiblePolygons[0]))
	}
	sawOwn := false
	for _, p := range playerView.Pieces {
		if p.CharacterID == sheetUUID.String() {
			sawOwn = true
		}
	}
	if !sawOwn {
		t.Fatal("FAIL: player cannot see their own character's piece")
	}
	t.Logf("PLAYER OK: %d polygon(s), first has %d vertices, %d piece(s) visible, %d wall(s), fog_mode=%s",
		len(playerView.VisiblePolygons), len(playerView.VisiblePolygons[0]),
		len(playerView.Pieces), len(playerView.Walls), playerView.FogMode)

	// The client rasterizes this polygon into a render texture; a polygon far larger
	// than the board makes that texture enormous.
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range playerView.VisiblePolygons {
		for _, pt := range poly {
			minX, maxX = math.Min(minX, pt.X), math.Max(maxX, pt.X)
			minY, maxY = math.Min(minY, pt.Y), math.Max(maxY, pt.Y)
		}
	}
	boardW := float64(board.Grid.Cols) * board.Grid.CellSize
	boardH := float64(board.Grid.Rows) * board.Grid.CellSize
	t.Logf("POLYGON EXTENT: x[%.0f..%.0f] y[%.0f..%.0f]  (board is %.0fx%.0f)",
		minX, maxX, minY, maxY, boardW, boardH)
	t.Logf("EXPLORED CELLS: %d", len(playerView.ExploredCells))

	// Optional: persist the player's real payload so the frontend can be tested
	// against production data instead of a hand-written fixture.
	if dump := os.Getenv("SMOKE_DUMP"); dump != "" {
		out, err := json.Marshal(map[string]any{
			"visible_polygons": playerView.VisiblePolygons,
			"explored_cells":   playerView.ExploredCells,
			"fog_mode":         playerView.FogMode,
			"grid":             board.Grid,
		})
		if err != nil {
			t.Fatalf("marshal dump: %v", err)
		}
		if err := os.WriteFile(dump, out, 0o600); err != nil {
			t.Fatalf("write dump: %v", err)
		}
		t.Logf("dumped player fog payload to %s", dump)
	}

	masterView := smokeAwaitMapFullState(t, master, 10*time.Second)
	if masterView == nil {
		t.Fatal("FAIL: master never received map_full_state")
	}
	if len(masterView.VisiblePolygons) != 0 {
		t.Fatalf("FAIL: master is fogged (%d polygons)", len(masterView.VisiblePolygons))
	}
	if len(masterView.Pieces) != len(board.Pieces) {
		t.Fatalf("FAIL: master sees %d pieces, want all %d", len(masterView.Pieces), len(board.Pieces))
	}
	t.Logf("MASTER OK: no fog, sees all %d pieces and %d walls",
		len(masterView.Pieces), len(masterView.Walls))
}

func smokeAwaitMapFullState(t *testing.T, conn *websocket.Conn, d time.Duration) *game.MapFullStatePayload {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, data, err := conn.ReadMessage()
		if err != nil {
			return nil
		}
		var msg game.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type != game.MsgTypeMapFullState {
			t.Logf("  ← %s %s", msg.Type, truncate(string(msg.Payload), 200))
			continue
		}
		var p game.MapFullStatePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			t.Fatalf("decode map_full_state: %v", err)
		}
		return &p
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
