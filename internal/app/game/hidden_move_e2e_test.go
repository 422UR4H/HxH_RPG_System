package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
)

// A move never reaches a player who should not see the piece (owner decision, 2026-10-01).
//
// turn_opened and match_full_state.openTurn carry the opened action's move — since B6, `from`
// is the server's REAL piece position, and `position` is where the piece now stands. Copied
// verbatim to every recipient, they told a player where a piece hidden by their fog (or by
// visible:false) stood and went, while the live relay of that very same move (pieceMoveView)
// told them nothing. These tests pin the move fields to the SAME gate, per recipient:
//
//   - destination visible → the whole move;
//   - only the origin visible → `from`, no `position` (the piece_removed half);
//   - neither, or the piece is hidden → neither, `category` alone;
//   - the master and the actor's owner → the whole move, always.
//
// They read the RAW JSON, not the typed payload: "absent from the wire" is the claim, and a
// typed decode cannot tell an absent key from a zero value.

// hiddenMoveBystanderAt is where the bystander's own piece stands in every board below: (20,4)
// → (1312,288), the same seat seedBoard gives them.
var hiddenMoveBystanderAt = [2]int{20, 4}

// originOnlyWall hides cell (14,8) → (928,544) from the bystander at (1312,288) while leaving
// (14,4) → (928,288) in plain sight: the sight line to (928,544) crosses y=450 at x≈1069, inside
// the segment, the one to (928,288) runs along y=288 and never reaches it. The move's own path
// runs down x=928, west of the segment's x=960 end, so enqueue_action's wall check lets it
// through.
var originOnlyWall = mapentity.WallSegment{
	ID: "origin-only", P1: [2]float64{960, 450}, P2: [2]float64{1250, 450},
	WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
	Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
}

// seedMoverBoard seeds a board with the attacker at attackerAt (visible or not) and the
// bystander at hiddenMoveBystanderAt, split by the given walls — seedBoard's shape, with the
// geometry each scenario below needs instead of seedBoard's fixed one.
func (f *combatFixture) seedMoverBoard(t *testing.T, attackerAt [2]int, visible bool, walls ...mapentity.WallSegment) {
	t.Helper()
	f.seedHiddenMoveBoard(t, []mapentity.Piece{{
		ID:          attackerPieceID,
		CharacterID: f.attackerID.String(),
		Coord:       mapentity.PieceCoord{Slot: mapentity.SquareCoord{Kind: "square", Col: attackerAt[0], Row: attackerAt[1]}},
		Visible:     visible,
	}}, walls...)
}

// seedHiddenMoveBoard seeds the bystander's piece plus whatever else the scenario puts down —
// nothing, for the actor with no piece on the board.
func (f *combatFixture) seedHiddenMoveBoard(t *testing.T, pieces []mapentity.Piece, walls ...mapentity.WallSegment) {
	t.Helper()
	pieces = append(pieces, mapentity.Piece{
		ID:          bystanderPieceID,
		CharacterID: f.bystanderID.String(),
		Coord: mapentity.PieceCoord{Slot: mapentity.SquareCoord{
			Kind: "square", Col: hiddenMoveBystanderAt[0], Row: hiddenMoveBystanderAt[1],
		}},
		Visible: true,
	})
	f.boards.seed(f.matchUUID, &matchboard.Board{
		MatchUUID: f.matchUUID,
		MapUUID:   uuid.New(),
		Grid:      moveBoardGrid,
		Pieces:    pieces,
		Walls:     walls,
	})
}

// rawMoveAt walks a raw payload down keys and returns the move object found there, as raw
// fields — the shape "is this key on the wire at all" can be asked of.
func rawMoveAt(t *testing.T, payload json.RawMessage, keys ...string) map[string]json.RawMessage {
	t.Helper()
	cur := payload
	for _, k := range keys {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(cur, &obj); err != nil {
			t.Fatalf("unmarshal at %q: %v (%s)", k, err, cur)
		}
		next, ok := obj[k]
		if !ok || string(next) == "null" {
			t.Fatalf("no %q in %s", k, cur)
		}
		cur = next
	}
	var move map[string]json.RawMessage
	if err := json.Unmarshal(cur, &move); err != nil {
		t.Fatalf("unmarshal move: %v (%s)", err, cur)
	}
	return move
}

// moveShape is what a recipient's move must carry: which of from/position are on the wire, and
// their values when they are.
type moveShape struct {
	from, position bool
}

func assertMoveShape(t *testing.T, who string, move map[string]json.RawMessage, want moveShape, from, to [3]int) {
	t.Helper()
	if _, ok := move["category"]; !ok {
		t.Errorf("%s: move.category is missing — that the actor moves is public mechanics", who)
	}
	check := func(key string, want bool, val [3]int) {
		raw, ok := move[key]
		if ok != want {
			t.Errorf("%s: move.%s on the wire = %v, want %v (move: %s)", who, key, ok, want, mustJSON(move))
			return
		}
		if !ok {
			return
		}
		var got [3]int
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("%s: unmarshal move.%s: %v", who, key, err)
		}
		if got != val {
			t.Errorf("%s: move.%s = %v, want %v", who, key, got, val)
		}
	}
	check("from", want.from, from)
	check("position", want.position, to)
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestE2E_TheOpenedMoveIsGatedByTheRecipientsSight(t *testing.T) {
	full := moveShape{from: true, position: true}
	tests := []struct {
		name    string
		at      [2]int
		to      [3]int
		visible bool
		walls   []mapentity.WallSegment
		// bystander is the move the bystander gets; master and owner always get full.
		bystander moveShape
		// relay is what the LIVE relay must have told the bystander of the same move — the
		// gate the move fields mirror. "" means nothing at all.
		relay game.MessageType
	}{
		{
			name: "sees neither end: no from, no position",
			at:   [2]int{4, 4}, to: [3]int{6, 4, 0}, visible: true,
			walls:     []mapentity.WallSegment{moveBoardWall},
			bystander: moveShape{},
		},
		{
			name: "sees only the origin: from, no position",
			at:   [2]int{14, 4}, to: [3]int{14, 8, 0}, visible: true,
			walls:     []mapentity.WallSegment{originOnlyWall},
			bystander: moveShape{from: true},
			relay:     game.MsgTypePieceRemoved,
		},
		{
			name: "sees the destination: the whole move",
			at:   [2]int{14, 4}, to: [3]int{16, 4, 0}, visible: true,
			walls:     []mapentity.WallSegment{moveBoardWall},
			bystander: full,
			relay:     game.MsgTypePieceMoved,
		},
		{
			name: "a visible:false piece in plain sight: no from, no position",
			at:   [2]int{14, 4}, to: [3]int{16, 4, 0}, visible: false,
			walls:     []mapentity.WallSegment{moveBoardWall},
			bystander: moveShape{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCombatFixture(t, withBystander)
			f.seedMoverBoard(t, tt.at, tt.visible, tt.walls...)
			from := [3]int{tt.at[0], tt.at[1], 0}

			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
			defer bystander.Close()   //nolint:errcheck
			readMessage(t, bystander) // room_state

			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)
			bystanderMsgs := collectFrom(bystander)
			if !bystanderMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the bystander never got a board — they are not really at the table")
			}

			f.enqueueDash(t, player, from, tt.to)
			if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
				t.Fatalf("the dash was never enqueued; the player received: %v",
					messageTypes(playerMsgs.snapshotMessages()))
			}
			sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
			for who, c := range map[string]*collector{"master": masterMsgs, "owner": playerMsgs, "bystander": bystanderMsgs} {
				if !c.await(game.MsgTypeTurnOpened, 2*time.Second) {
					t.Fatalf("the turn never opened for the %s", who)
				}
			}

			// The premise: the live relay decided what the move fields must mirror. Without it
			// a broken scenario (a wall that does not hide what it should) would pass as a gate.
			// turn_opened is the barrier — the relay is queued before it on the same lane.
			for _, typ := range []game.MessageType{game.MsgTypePieceMoved, game.MsgTypePieceRemoved} {
				n := bystanderMsgs.count(typ)
				if want := typ == tt.relay; (n == 1) != want {
					t.Fatalf("premise: the live relay sent the bystander %d %s, want relay %q",
						n, typ, tt.relay)
				}
			}

			opened := func(c *collector) map[string]json.RawMessage {
				return rawMoveAt(t, findMessage(t, c.snapshotMessages(), game.MsgTypeTurnOpened).Payload,
					"action", "move")
			}
			assertMoveShape(t, "master", opened(masterMsgs), full, from, tt.to)
			assertMoveShape(t, "owner", opened(playerMsgs), full, from, tt.to)
			assertMoveShape(t, "bystander", opened(bystanderMsgs), tt.bystander, from, tt.to)

			// The reconnect snapshot runs the same gate: a bystander who reconnects is not told
			// more than the live turn_opened told them.
			bystander.Close() //nolint:errcheck
			late := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
			defer late.Close()   //nolint:errcheck
			readMessage(t, late) // room_state
			lateMsgs := collectFrom(late)
			if !lateMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
				t.Fatal("the reconnecting bystander never received match_full_state")
			}
			snap := rawMoveAt(t, findMessage(t, lateMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload,
				"openTurn", "action", "move")
			assertMoveShape(t, "reconnecting bystander", snap, tt.bystander, from, tt.to)

			// And the master's snapshot keeps it all — the gate is for players only.
			master.Close() //nolint:errcheck
			lateMaster := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
			defer lateMaster.Close() //nolint:errcheck
			lateMasterMsgs := collectFrom(lateMaster)
			if !lateMasterMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
				t.Fatal("the reconnecting master never received match_full_state")
			}
			snap = rawMoveAt(t, findMessage(t, lateMasterMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload,
				"openTurn", "action", "move")
			assertMoveShape(t, "reconnecting master", snap, full, from, tt.to)
		})
	}
}

// The settled resolution reaches the whole table, escape included — but where a failed
// escape's piece landed is the same news as a move's destination: a bystander who cannot see
// the landing cell is not told it, one who can is. The master and the escaping character's
// owner keep it either way.
func TestE2E_TheSettledEscapeLandingIsGatedByTheRecipientsSight(t *testing.T) {
	tests := []struct {
		name  string
		walls []mapentity.WallSegment
		// sees is whether the bystander sees the landing cell — and so keeps escape.landing.
		sees bool
	}{
		// The divider hides the whole west half — the landing included — from the bystander.
		{name: "landing out of sight: no landing", walls: []mapentity.WallSegment{moveBoardWall}, sees: false},
		// No wall: the landing (7,6) is in plain sight of the bystander at (20,4).
		{name: "landing in sight: the landing", walls: nil, sees: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCombatFixture(t, withVictimPiece, withBystander)
			f.seedBoardWith(t, tt.walls...)
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
			defer bystander.Close()   //nolint:errcheck
			readMessage(t, bystander) // room_state
			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)
			bystanderMsgs := collectFrom(bystander)
			if !bystanderMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the bystander never got a board — they are not really at the table")
			}

			// A failed escape (the movement does not pass) whose landing the master chooses.
			reactionID := f.escapeStage(t, master, player, masterMsgs, playerMsgs, false, false, true)
			pos := escapeLanding
			f.chooseLanding(t, master, masterMsgs, reactionID, &pos)
			sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})

			for who, c := range map[string]*collector{"master": masterMsgs, "owner": playerMsgs, "bystander": bystanderMsgs} {
				if !awaitSettled(c, 3*time.Second) {
					t.Fatalf("the %s never received the settled resolution; got: %v",
						who, messageTypes(c.snapshotMessages()))
				}
			}

			// The premise: the live relay of the landing told the bystander exactly what the
			// landing field must mirror. The settled resolution is sent after the move.
			if n := bystanderMsgs.count(game.MsgTypePieceMoved); (n == 1) != tt.sees {
				t.Fatalf("premise: the live relay sent the bystander %d piece_moved, want sees=%v", n, tt.sees)
			}

			for who, c := range map[string]*collector{"master": masterMsgs, "owner": playerMsgs} {
				esc := f.escapeOf(t, findSettledResolution(t, c))
				if esc.Landing == nil || *esc.Landing != escapeLanding {
					t.Errorf("%s: settled escape.landing = %v, want %v", who, esc.Landing, escapeLanding)
				}
			}
			esc := f.escapeOf(t, findSettledResolution(t, bystanderMsgs))
			switch {
			case tt.sees && (esc.Landing == nil || *esc.Landing != escapeLanding):
				t.Errorf("bystander: settled escape.landing = %v, want %v — they see the landing cell",
					esc.Landing, escapeLanding)
			case !tt.sees && esc.Landing != nil:
				t.Errorf("bystander: settled escape.landing = %v, want absent — the landing cell is "+
					"out of their sight", *esc.Landing)
			}
			// The rest of the verdict is public once settled: only WHERE is gated.
			if esc.Escaped || esc.MovePassed || !esc.DodgePassed {
				t.Errorf("bystander: escape verdict = %+v, want the public verdict (failed on the movement)", *esc)
			}
		})
	}
}

// An actor with no piece at the opening moves nothing on the board — applyMove has no piece to
// walk, so the relay tells nobody anything. The move fields must say no more than that, even
// with the destination in plain sight: no piece is treated as a hidden one.
func TestE2E_TheOpenedMoveOfAnActorWithNoPieceTellsTheTableNowhere(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	f.seedHiddenMoveBoard(t, nil) // no attacker piece, no wall: (16,4) is in plain sight
	to := [3]int{16, 4, 0}

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	defer bystander.Close()   //nolint:errcheck
	readMessage(t, bystander) // room_state
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	bystanderMsgs := collectFrom(bystander)
	if !bystanderMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the bystander never got a board — they are not really at the table")
	}

	f.enqueueDash(t, player, [3]int{14, 4, 0}, to)
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatalf("the dash was never enqueued; the player received: %v",
			messageTypes(playerMsgs.snapshotMessages()))
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	for who, c := range map[string]*collector{"master": masterMsgs, "owner": playerMsgs, "bystander": bystanderMsgs} {
		if !c.await(game.MsgTypeTurnOpened, 2*time.Second) {
			t.Fatalf("the turn never opened for the %s", who)
		}
	}
	if n := bystanderMsgs.count(game.MsgTypePieceMoved) + bystanderMsgs.count(game.MsgTypePieceRemoved); n != 0 {
		t.Fatalf("premise: the relay told the bystander about a piece that is not there (%d message(s))", n)
	}

	opened := rawMoveAt(t, findMessage(t, bystanderMsgs.snapshotMessages(), game.MsgTypeTurnOpened).Payload,
		"action", "move")
	// No from either way: there was no piece to read one off at enqueue (B6).
	assertMoveShape(t, "bystander", opened, moveShape{}, [3]int{}, to)
	ownerOpened := rawMoveAt(t, findMessage(t, playerMsgs.snapshotMessages(), game.MsgTypeTurnOpened).Payload,
		"action", "move")
	assertMoveShape(t, "owner", ownerOpened, moveShape{position: true}, [3]int{}, to)
}

// The piece can move between the enqueue (which reads move.from off the board, B6) and the
// opening — here by a SECOND queued move of the same character opened first. The live gate has
// to judge the move from where the piece actually stood when it opened (the slot the relay
// judged), not from the stale enqueue-time move.from, or a bystander is told an origin the relay
// never showed them.
//
// Both dashes are enqueued with the piece at A, so both carry move.from = A. The first one
// opened walks it A→B; the second then walks it B→C, but still says from: A.
func TestE2E_TheLiveGateJudgesTheMoveFromWhereThePieceStoodAtTheOpening(t *testing.T) {
	tests := []struct {
		name    string
		a, b, c [2]int
		// relay is how many piece_removed the bystander must have got by the second opening —
		// the live gate's own verdict on the second move (piece_moved is checked separately).
		removed, moved int
	}{
		{
			// A in sight, B and C behind originOnlyWall. Judged from the stale A, the second move
			// would look like "only the origin" and hand out from: A; judged from B (where the
			// piece really left from) it is neither end — the relay sent nothing for it.
			name: "the stale origin is in sight, the real one is not: nothing",
			a:    [2]int{14, 4}, b: [2]int{14, 8}, c: [2]int{14, 9},
			removed: 1, moved: 0, // the first move: A in sight, B not → piece_removed
		},
		{
			// A behind the wall, B in sight, C behind the wall. The real origin B is in sight, so
			// the second move is "only the origin" — but move.from still says A, which this
			// bystander never saw the piece on: from is dropped too.
			name: "the real origin is in sight, the stale one is not: no from of a slot never shown",
			a:    [2]int{14, 8}, b: [2]int{14, 4}, c: [2]int{14, 9},
			removed: 1, moved: 1, // first: A hidden, B in sight → piece_moved; second: B in sight, C not → piece_removed
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCombatFixture(t, withBystander)
			f.seedMoverBoard(t, tt.a, true, originOnlyWall)
			from := [3]int{tt.a[0], tt.a[1], 0}
			toB, toC := [3]int{tt.b[0], tt.b[1], 0}, [3]int{tt.c[0], tt.c[1], 0}

			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
			defer bystander.Close()   //nolint:errcheck
			readMessage(t, bystander) // room_state
			masterMsgs := collectFrom(master)
			playerMsgs := collectFrom(player)
			bystanderMsgs := collectFrom(bystander)
			if !bystanderMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
				t.Fatal("the bystander never got a board — they are not really at the table")
			}

			var ids []uuid.UUID
			for i, to := range [][3]int{toB, toC} {
				f.enqueueDash(t, player, from, to)
				if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, i+1, 2*time.Second) {
					t.Fatalf("dash %d was never enqueued; the player received: %v",
						i+1, messageTypes(playerMsgs.snapshotMessages()))
				}
			}
			for _, m := range playerMsgs.snapshotMessages() {
				if m.Type != game.MsgTypeActionEnqueued {
					continue
				}
				var p game.ActionEnqueuedPayload
				if err := json.Unmarshal(m.Payload, &p); err != nil {
					t.Fatalf("unmarshal action_enqueued: %v", err)
				}
				ids = append(ids, p.ActionID)
			}

			// pull_action fixes the order: A→B first, then the one that still says from: A.
			for i, id := range ids {
				sendWS(t, master, string(game.MsgTypePullAction), game.PullActionPayload{ActionID: id})
				if !awaitCount(bystanderMsgs, game.MsgTypeTurnOpened, i+1, 2*time.Second) {
					t.Fatalf("turn %d never opened for the bystander; the master received: %v",
						i+1, messageTypes(masterMsgs.snapshotMessages()))
				}
			}

			if n := bystanderMsgs.count(game.MsgTypePieceRemoved); n != tt.removed {
				t.Fatalf("premise: the bystander got %d piece_removed, want %d", n, tt.removed)
			}
			if n := bystanderMsgs.count(game.MsgTypePieceMoved); n != tt.moved {
				t.Fatalf("premise: the bystander got %d piece_moved, want %d", n, tt.moved)
			}

			var second json.RawMessage
			for _, m := range bystanderMsgs.snapshotMessages() {
				if m.Type == game.MsgTypeTurnOpened {
					second = m.Payload
				}
			}
			move := rawMoveAt(t, second, "action", "move")
			assertMoveShape(t, "bystander, second opening", move, moveShape{}, from, toC)
		})
	}
}
