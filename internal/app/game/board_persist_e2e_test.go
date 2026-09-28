package game_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file is the end-to-end guarantee for B3: the match board and every player's fog
// memory persist per match, so a server restart brings them back (spec §4.3 "Quando
// persiste" and §5). It reuses combatFixture (combat_e2e_test.go) — its fakeBoardStore and
// fakeMemoryStore ARE the "database" a restart must survive, and its enqueueDash/seedBoard
// helpers are what put a piece somewhere worth asserting on.

// TestE2E_AClosedTurnPersistsTheBoard proves the write half of B3: once a turn that moved a
// piece CLOSES — by any of the three verbs room.go's own persistBoard call sites name (spec
// §4.3, "Quando persiste") — the board in the STORE (not just in the room's own memory)
// shows the piece where the close left it.
func TestE2E_AClosedTurnPersistsTheBoard(t *testing.T) {
	cases := []struct {
		name    string
		close   func(t *testing.T, master *websocket.Conn)
		barrier game.MessageType
	}{
		{
			name: "close_turn",
			close: func(t *testing.T, master *websocket.Conn) {
				sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
			},
			barrier: game.MsgTypeTurnClosed,
		},
		{
			name: "open_next_action on an empty queue",
			close: func(t *testing.T, master *websocket.Conn) {
				sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
			},
			barrier: game.MsgTypeError,
		},
		{
			name: "pull_action for an action that is not queued",
			close: func(t *testing.T, master *websocket.Conn) {
				sendWS(t, master, string(game.MsgTypePullAction),
					game.PullActionPayload{ActionID: uuid.New()})
			},
			barrier: game.MsgTypeError,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newCombatFixture(t)
			f.seedBoard(t)

			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)

			if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the master never got the board — the fixture never started")
			}

			f.enqueueDash(t, player, [3]int{4, 4, 0}, [3]int{5, 4, 0})
			if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
				t.Fatal("the dash was never enqueued")
			}
			sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
			if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
				t.Fatal("no piece_moved at the open: the Dash never applied")
			}
			if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
				t.Fatal("no turn_opened: the Dash never opened, so there is nothing to close")
			}

			c.close(t, master)
			if !masterMsgs.await(c.barrier, 3*time.Second) {
				t.Fatalf("the turn never closed; the master received: %v",
					messageTypes(masterMsgs.snapshotMessages()))
			}

			board, err := f.boards.Load(context.Background(), f.matchUUID)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if board == nil {
				t.Fatal("persistBoard never saved a row: the store still has nothing for this match")
			}
			p := findPiece(t, board.Pieces, attackerPieceID)
			col, row := squareColRow(t, p)
			if col != 5 || row != 4 {
				t.Fatalf("the PERSISTED board has the attacker at (%d,%d), want (5,4) — the close "+
					"applied the move in memory but never wrote it to the store", col, row)
			}
		})
	}
}

// TestE2E_AServerRestartBringsTheBoardBack proves the read half of B3: a brand-new Room,
// stood up over the SAME store after the close above, loads the persisted board instead of
// starting from the seed.
func TestE2E_AServerRestartBringsTheBoardBack(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)

	master, player := f.connect(t)
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board")
	}

	f.enqueueDash(t, player, [3]int{4, 4, 0}, [3]int{5, 4, 0})
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the dash was never enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("no piece_moved at the open")
	}
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn_opened")
	}
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !masterMsgs.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed — nothing was persisted to restart from")
	}
	master.Close() //nolint:errcheck
	player.Close() //nolint:errcheck

	f.restart(t)

	master2, _ := f.connect(t)
	defer master2.Close() //nolint:errcheck
	master2Msgs := collectFrom(master2)
	if !master2Msgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got a board from the NEW room — the restart lost it")
	}

	var board game.MapFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, master2Msgs.snapshotMessages(), game.MsgTypeMapFullState).Payload, &board,
	); err != nil {
		t.Fatalf("unmarshal map_full_state: %v", err)
	}
	p := findPayloadPiece(t, board.Pieces, attackerPieceID)
	if p.Slot.Col == nil || p.Slot.Row == nil || *p.Slot.Col != 5 || *p.Slot.Row != 4 {
		t.Fatalf("after the restart the master's board has the attacker at %+v, want (5,4) — "+
			"the new room did not load what the old one persisted", p.Slot)
	}
}

// TestE2E_ARestartMidTurnLosesTheTurnAndTheMove is the mirror of the test above, and the
// reason spec §4.3 opens a whole section with a star: the piece walks on OPEN, but the board
// is only written on CLOSE. A server that dies mid-turn loses the turn — and the piece that
// walked with it, back to wherever the last close actually left it.
func TestE2E_ARestartMidTurnLosesTheTurnAndTheMove(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)

	master, player := f.connect(t)
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board")
	}

	f.enqueueDash(t, player, [3]int{4, 4, 0}, [3]int{5, 4, 0})
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the dash was never enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("no piece_moved at the open — the Dash never applied, so this test would prove nothing")
	}
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn_opened")
	}
	// Deliberately NOT closed: the turn stays open when the server "dies".
	master.Close() //nolint:errcheck
	player.Close() //nolint:errcheck

	f.restart(t)

	master2, _ := f.connect(t)
	defer master2.Close() //nolint:errcheck
	master2Msgs := collectFrom(master2)
	if !master2Msgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got a board from the NEW room")
	}

	var board game.MapFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, master2Msgs.snapshotMessages(), game.MsgTypeMapFullState).Payload, &board,
	); err != nil {
		t.Fatalf("unmarshal map_full_state: %v", err)
	}
	p := findPayloadPiece(t, board.Pieces, attackerPieceID)
	if p.Slot.Col == nil || p.Slot.Row == nil || *p.Slot.Col != 4 || *p.Slot.Row != 4 {
		t.Fatalf("after the restart the attacker is at %+v, want back at the SEEDED (4,4) — the "+
			"open-but-unclosed Dash should never have reached the store", p.Slot)
	}
}

// ─── memória de fog sobrevive a um reinício ─────────────────────────────────

// memWallGrid is a small 10x10 board built just for TestE2E_PlayerMemorySurvivesARestart: a
// full-width divider at row 5 splits it into a north half and a south half that cannot see
// each other — the same "wall fully divides the board" shape moveBoardWall already proves
// elsewhere, just rotated, plus a SECOND, short wall (the one being remembered) sitting only
// in the north half.
var memWallGrid = mapentity.GridShape{
	Kind: mapentity.GridKindSquare, Cols: 10, Rows: 10, CellSize: 64, SkewRatio: 1,
}

// memDivider is what makes the SOUTH position genuinely blind to memWallTarget — without it
// "far away" would not actually occlude anything on an otherwise-empty board (the visibility
// radius comfortably covers this whole 10x10 grid).
var memDivider = mapentity.WallSegment{
	ID: "mem-divider", P1: [2]float64{0, 320}, P2: [2]float64{640, 320},
	WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
	Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
}

// memWallTarget is the wall the player must REMEMBER once they can no longer see it.
var memWallTarget = mapentity.WallSegment{
	ID: "mem-target", P1: [2]float64{128, 0}, P2: [2]float64{128, 128},
	WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
	Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
}

// TestE2E_PlayerMemorySurvivesARestart proves the OTHER half of B3: fog memory (player_memories)
// persists too, and comes back on a restart even for a player who is nowhere near what they
// once saw.
func TestE2E_PlayerMemorySurvivesARestart(t *testing.T) {
	f := newCombatFixture(t)
	f.boards.seed(f.matchUUID, &matchboard.Board{
		MatchUUID: f.matchUUID,
		MapUUID:   uuid.New(),
		Grid:      memWallGrid,
		Pieces: []mapentity.Piece{{
			ID:          attackerPieceID,
			CharacterID: f.attackerID.String(),
			// (2,1): north of the divider, a couple of cells from memWallTarget — close enough
			// that connecting here already sees it, with nothing in the way.
			Coord:   mapentity.PieceCoord{Slot: mapentity.SquareCoord{Kind: "square", Col: 2, Row: 1}},
			Visible: true,
		}},
		Walls: []mapentity.WallSegment{memDivider, memWallTarget},
	})

	master, player := f.connect(t)
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board")
	}
	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board")
	}

	// The premise: from (2,1) the player already sees memWallTarget — otherwise there would
	// be nothing to remember, and the assertion after the restart would be vacuous.
	var initial game.MapFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, playerMsgs.snapshotMessages(), game.MsgTypeMapFullState).Payload, &initial,
	); err != nil {
		t.Fatalf("unmarshal map_full_state: %v", err)
	}
	if !hasWall(initial.Walls, memWallTarget.ID) {
		t.Fatal("the player cannot even see memWallTarget from the seeded position — this test " +
			"would prove nothing about memory")
	}

	// from is the zero value ON PURPOSE: it tells the server's move-blocking check "no origin
	// declared", skipping path validation (room.go's own `if from != ([3]int{})` gate) — the
	// straight line from (2,1) to (7,8) crosses memDivider, which a real client would never be
	// asked to walk through. What this test needs is the character to END UP south of the
	// divider, not a lawful path there.
	f.enqueueDash(t, player, [3]int{0, 0, 0}, [3]int{7, 8, 0})
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the dash was never enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("no piece_moved at the open")
	}
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn_opened")
	}
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !masterMsgs.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed — nothing was persisted")
	}
	master.Close() //nolint:errcheck
	player.Close() //nolint:errcheck

	f.restart(t)

	master2, player2 := f.connect(t)
	defer master2.Close() //nolint:errcheck
	defer player2.Close() //nolint:errcheck
	player2Msgs := collectFrom(player2)
	if !player2Msgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got a board from the NEW room")
	}

	var after game.MapFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, player2Msgs.snapshotMessages(), game.MsgTypeMapFullState).Payload, &after,
	); err != nil {
		t.Fatalf("unmarshal map_full_state: %v", err)
	}
	if !hasWall(after.Walls, memWallTarget.ID) {
		t.Fatalf("after the restart, the player — now south of memDivider and unable to see "+
			"memWallTarget directly — no longer has it in map_full_state at all: %+v", after.Walls)
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func findPiece(t *testing.T, pieces []mapentity.Piece, id string) mapentity.Piece {
	t.Helper()
	for _, p := range pieces {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("no piece %q in %+v", id, pieces)
	return mapentity.Piece{}
}

func squareColRow(t *testing.T, p mapentity.Piece) (col, row int) {
	t.Helper()
	switch slot := p.Coord.Slot.(type) {
	case mapentity.SquareCoord:
		return slot.Col, slot.Row
	default:
		t.Fatalf("piece %q slot = %#v, want mapentity.SquareCoord", p.ID, p.Coord.Slot)
		return 0, 0
	}
}

func findPayloadPiece(t *testing.T, pieces []game.PieceMovedPayload, id string) game.PieceMovedPayload {
	t.Helper()
	for _, p := range pieces {
		if p.PieceID == id {
			return p
		}
	}
	t.Fatalf("no piece %q in %+v", id, pieces)
	return game.PieceMovedPayload{}
}

func hasWall(walls []game.WallSegmentPayload, id string) bool {
	for _, w := range walls {
		if w.ID == id {
			return true
		}
	}
	return false
}
