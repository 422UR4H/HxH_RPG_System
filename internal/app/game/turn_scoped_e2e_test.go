package game_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Owner decision, 2026-10-01: everything that happens INSIDE an open turn — the opened move,
// the master's piece and wall actions, the turn notes — becomes durable together with that
// turn's close, atomically. The master action is built the instant it is applied (views,
// turnUuid, happenedAt) but only written by PersistTurnClose, in the turn's own transaction;
// the board is only saved by the close. Outside a turn nothing changed: saved and recorded at
// the instant. A restart mid-turn rolls the whole turn back to the last close.

// turnClosers are the three verbs that close a turn — every one of them has to carry the
// turn's master actions into its write, or what the master did would depend on which verb
// they happened to use.
var turnClosers = []struct {
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
			sendWS(t, master, string(game.MsgTypePullAction), game.PullActionPayload{ActionID: uuid.New()})
		},
		barrier: game.MsgTypeError,
	},
}

// awaitPersistedTurn waits until PersistTurnClose was called for turnID.
func (f *combatFixture) awaitPersistedTurn(t *testing.T, turnID uuid.UUID) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !contains(f.roundRepo.persistedTurnIDs(), turnID) {
		if time.Now().After(deadline) {
			t.Fatalf("turn %s was never persisted", turnID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// assertNothingWrittenMidTurn is the "inside a turn" half of the rule: no board save since
// savesBefore, no master_actions row outside a turn's transaction, and no turn written.
func (f *combatFixture) assertNothingWrittenMidTurn(t *testing.T, savesBefore int) {
	t.Helper()
	if got := f.boards.saveCount(); got != savesBefore {
		t.Fatalf("saveCount = %d, want %d — the board was saved while the turn was still open", got, savesBefore)
	}
	if recs := f.masterActions.snapshot(); len(recs) != 0 {
		t.Fatalf("master_actions got %d row(s) while the turn was open, want none before the close", len(recs))
	}
	if ids := f.roundRepo.persistedTurnIDs(); len(ids) != 0 {
		t.Fatalf("turns %v were written, want none — the turn is still open", ids)
	}
}

func storedSquare(t *testing.T, f *combatFixture, pieceID string) (col, row int) {
	t.Helper()
	board, err := f.boards.Load(context.Background(), f.matchUUID)
	if err != nil || board == nil {
		t.Fatalf("Load = %v, %v — no board in the store", board, err)
	}
	return squareColRow(t, findPiece(t, board.Pieces, pieceID))
}

// A master drag inside an open turn: nothing is written until the close; then, whichever verb
// closes it, the board row carries the drag and the master action goes to PersistTurnClose
// with the turn's uuid — never to master_actions on its own.
func TestTurnScoped_AMasterDragInsideATurnIsWrittenWithTheTurnsClose(t *testing.T) {
	for _, c := range turnClosers {
		t.Run(c.name, func(t *testing.T) {
			f := newCombatFixture(t)
			f.seedBoard(t)
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			mc := collectFrom(master)
			turnID := f.openAttackTurn(t, master, player, mc)
			saves := f.boards.saveCount()

			sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
			// The arm saves and records BEFORE it echoes: the echo is the barrier.
			if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
				t.Fatal("the drag was never acknowledged")
			}
			f.assertNothingWrittenMidTurn(t, saves)
			if n := len(f.session.GetActiveRound().CurrentTurn().GetMasterActions()); n != 1 {
				t.Fatalf("the open turn carries %d master action(s), want 1", n)
			}

			c.close(t, master)
			if !mc.await(c.barrier, 3*time.Second) {
				t.Fatalf("the turn never closed; the master received: %v", messageTypes(mc.snapshotMessages()))
			}
			f.awaitPersistedTurn(t, turnID)

			recs := f.roundRepo.masterActionsFor(turnID)
			if len(recs) != 1 {
				t.Fatalf("PersistTurnClose got %d master action(s), want the drag", len(recs))
			}
			rec := recs[0]
			if rec.Kind != masteraction.KindMovePiece || rec.TurnUUID == nil || *rec.TurnUUID != turnID {
				t.Fatalf("record = %s turn %v, want movePiece in turn %s", rec.Kind, rec.TurnUUID, turnID)
			}
			if pc := pieceContentOf(t, rec); !samePos(pc.To, ptrPos([3]int{6, 4, 0})) {
				t.Fatalf("content = %s, want the drag to [6 4 0]", rec.Content)
			}
			if n := len(f.masterActions.snapshot()); n != 0 {
				t.Fatalf("master_actions got %d row(s) on its own — an in-turn master action is written in the turn's transaction", n)
			}
			if got := f.boards.saveCount(); got <= saves {
				t.Fatal("the close never saved the board")
			}
			if col, row := storedSquare(t, f, attackerPieceID); col != 6 || row != 4 {
				t.Fatalf("stored board has the attacker at (%d,%d), want the drag's (6,4)", col, row)
			}
		})
	}
}

// Walls too: an interact and a reveal inside the turn change the table live and are written
// with the close — the board with the door open, one master action each, carrying the turn.
func TestTurnScoped_WallActionsInsideATurnAreWrittenWithTheTurnsClose(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoardWithDoors(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	turnID := f.openAttackTurn(t, master, player, mc)
	saves := f.boards.saveCount()

	sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
		"targetIds": []string{doorID.String()}, "interact": map[string]any{"kind": "open"},
	})
	sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
		"targetIds": []string{secretDoorID.String()}, "interact": map[string]any{"kind": "reveal"},
	})
	if !mc.await(game.MsgTypeWallStateChanged, 2*time.Second) {
		t.Fatal("the door never opened")
	}
	chatBarrier(t, master, mc)
	f.assertNothingWrittenMidTurn(t, saves)

	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed")
	}
	f.awaitPersistedTurn(t, turnID)

	recs := f.roundRepo.masterActionsFor(turnID)
	if len(recs) != 2 || recs[0].Kind != masteraction.KindWallInteract || recs[1].Kind != masteraction.KindRevealWall {
		t.Fatalf("PersistTurnClose got %+v, want [wallInteract revealWall] in the order they happened", recs)
	}
	for _, r := range recs {
		if r.TurnUUID == nil || *r.TurnUUID != turnID {
			t.Fatalf("record %s turn = %v, want %s", r.Kind, r.TurnUUID, turnID)
		}
	}
	if n := len(f.masterActions.snapshot()); n != 0 {
		t.Fatalf("master_actions got %d row(s) on its own, want none", n)
	}
	board, _ := f.boards.Load(context.Background(), f.matchUUID)
	var door mapentity.WallSegment
	for _, w := range board.Walls {
		if w.ID == doorID.String() {
			door = w
		}
	}
	if !door.Open {
		t.Fatal("the stored board's door is closed — the close did not save what the turn changed")
	}
}

// A restart mid-turn rolls the WHOLE turn back to the last close: the opened move AND the
// master's drag are both absent from the board row, and no master action was ever written.
func TestTurnScoped_ARestartMidTurnLosesTheMasterDragWithTheTurn(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	master, player := f.connect(t)
	mc, pc := collectFrom(master), collectFrom(player)
	if !mc.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board")
	}

	f.enqueueDash(t, player, [3]int{4, 4, 0}, [3]int{5, 4, 0})
	if !pc.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the dash was never enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn_opened")
	}
	sendMasterMove(t, master, f.attackerID, [3]int{7, 4, 0})
	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatal("the drag was never acknowledged")
	}
	master.Close() //nolint:errcheck
	player.Close() //nolint:errcheck

	f.restart(t)
	master2, _ := f.connect(t)
	defer master2.Close() //nolint:errcheck
	m2 := collectFrom(master2)
	if !m2.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got a board from the new room")
	}
	p := findPayloadPiece(t, lastMapFullState(t, m2).Pieces, attackerPieceID)
	if p.Slot.Col == nil || p.Slot.Row == nil || *p.Slot.Col != 4 || *p.Slot.Row != 4 {
		t.Fatalf("after the restart the attacker is at %+v, want the last closed (4,4) — neither the "+
			"opened Dash nor the master's drag inside the lost turn may reach the store", p.Slot)
	}
	if recs := f.masterActions.snapshot(); len(recs) != 0 {
		t.Fatalf("master_actions has %d row(s), want none — the drag belonged to a turn that never closed", len(recs))
	}
}

// Rule 4: no verb that leaves the room alive can drop an open turn without its close.
// change_scene is refused while a turn is open; change_round_mode switches the regime of the
// SAME round and leaves the turn open; a kick takes a player off the table, not the turn. Each
// keeps the master action pending, and the eventual close writes it with the turn.
func TestTurnScoped_VerbsThatDoNotCloseTheTurnKeepItsMasterActionsPending(t *testing.T) {
	cases := []struct {
		name    string
		act     func(t *testing.T, f *combatFixture, master *websocket.Conn)
		barrier game.MessageType
	}{
		{
			name: "change_scene is refused with a turn open",
			act: func(t *testing.T, _ *combatFixture, master *websocket.Conn) {
				sendWS(t, master, string(game.MsgTypeChangeScene), game.ChangeScenePayload{
					Category: string(enum.Roleplay), BriefInitialDescription: "Taverna",
				})
			},
			barrier: game.MsgTypeError,
		},
		{
			name: "change_round_mode keeps the turn open",
			act: func(t *testing.T, _ *combatFixture, master *websocket.Conn) {
				sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{Mode: string(enum.Race)})
			},
			barrier: game.MsgTypeRoundModeChanged,
		},
		{
			name: "kick_player keeps the turn open",
			act: func(t *testing.T, f *combatFixture, master *websocket.Conn) {
				sendWS(t, master, string(game.MsgTypeKickPlayer), game.KickPlayerPayload{PlayerUUID: f.playerUUID})
			},
			barrier: game.MsgTypePlayerKicked,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newCombatFixture(t)
			f.seedBoard(t)
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			mc := collectFrom(master)
			turnID := f.openAttackTurn(t, master, player, mc)
			saves := f.boards.saveCount()

			sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
			if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
				t.Fatal("the drag was never acknowledged")
			}
			c.act(t, f, master)
			if !mc.await(c.barrier, 2*time.Second) {
				t.Fatalf("no %s; the master received: %v", c.barrier, messageTypes(mc.snapshotMessages()))
			}
			f.assertNothingWrittenMidTurn(t, saves)

			sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
			if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
				t.Fatalf("the turn never closed; the master received: %v", messageTypes(mc.snapshotMessages()))
			}
			f.awaitPersistedTurn(t, turnID)
			if recs := f.roundRepo.masterActionsFor(turnID); len(recs) != 1 || recs[0].Kind != masteraction.KindMovePiece {
				t.Fatalf("PersistTurnClose got %+v, want the drag — it was dropped on the way", recs)
			}
		})
	}
}

// Rule 4, the one path that drops an open turn: the room itself goes away (its last client
// leaves; the hub stops). That is a restart as far as persistence goes — the next room
// rehydrates from the database — so the turn rolls back whole, the master's drag inside it
// included: no board save, no master action.
func TestTurnScoped_ARoomThatEmptiesMidTurnWritesNothingOfIt(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	master, player := f.connect(t)
	mc := collectFrom(master)
	f.openAttackTurn(t, master, player, mc)
	saves := f.boards.saveCount()

	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatal("the drag was never acknowledged")
	}
	// Read before the sockets close: every board read from here on is a new room's birth (the
	// test's own storedSquare below reads the store too, after the loop).
	loads := f.boards.loadCount()
	player.Close() //nolint:errcheck
	master.Close() //nolint:errcheck

	// The old room closes once it empties; until then a reconnecting master lands back in it.
	// A NEW room is the one that loads the board from the store at birth — and what it shows is
	// what the old room wrote.
	deadline := time.Now().Add(3 * time.Second)
	for {
		m2 := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
		readMessage(t, m2) // room_state
		c2 := collectFrom(m2)
		if !c2.await(game.MsgTypeMapFullState, 2*time.Second) {
			t.Fatal("the master never got a board on reconnect")
		}
		fresh := f.boards.loadCount() > loads
		p := findPayloadPiece(t, lastMapFullState(t, c2).Pieces, attackerPieceID)
		m2.Close() //nolint:errcheck
		if fresh {
			if p.Slot.Col == nil || *p.Slot.Col != 4 || *p.Slot.Row != 4 {
				t.Fatalf("the new room shows the attacker at %+v, want the last closed (4,4) — the drag "+
					"inside the dropped turn reached the store", p.Slot)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the old room never closed after its last client left")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := f.boards.saveCount(); got != saves {
		t.Fatalf("saveCount = %d, want %d — nothing of the dropped turn may be saved", got, saves)
	}
	if recs := f.masterActions.snapshot(); len(recs) != 0 {
		t.Fatalf("master_actions has %d row(s), want none", len(recs))
	}
	if recs := f.roundRepo.allMasterActions(); len(recs) != 0 {
		t.Fatalf("PersistTurnClose got %d master action(s), want none — the turn never closed", len(recs))
	}
}

// ─── the board goes in the turn's transaction (fix round 1) ──────────────────

// The board the close leaves is handed to PersistTurnClose itself — written in the turn's own
// transaction — by every closing verb.
func TestTurnScoped_TheCloseHandsTheBoardToTheTurnsTransaction(t *testing.T) {
	for _, c := range turnClosers {
		t.Run(c.name, func(t *testing.T) {
			f := newCombatFixture(t)
			f.seedBoard(t)
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			mc := collectFrom(master)
			turnID := f.openAttackTurn(t, master, player, mc)

			sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
			if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
				t.Fatal("the drag was never acknowledged")
			}
			c.close(t, master)
			if !mc.await(c.barrier, 3*time.Second) {
				t.Fatalf("the turn never closed; the master received: %v", messageTypes(mc.snapshotMessages()))
			}
			f.awaitPersistedTurn(t, turnID)

			b, ok := f.roundRepo.boardFor(turnID)
			if !ok {
				t.Fatal("PersistTurnClose got no board — the close wrote it outside the turn's transaction")
			}
			if col, row := squareColRow(t, findPiece(t, b.Pieces, attackerPieceID)); col != 6 || row != 4 {
				t.Fatalf("the turn's board has the attacker at (%d,%d), want the drag's (6,4)", col, row)
			}
		})
	}
}

// All or nothing: a close whose transaction fails writes no board either — the store keeps the
// last closed state, with neither the turn's move nor the master's drag.
func TestTurnScoped_AFailedCloseWritesNoBoard(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	f.openAttackTurn(t, master, player, mc)
	saves := f.boards.saveCount()
	f.roundRepo.setFailPersist(errors.New("transaction rolled back"))

	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatal("the drag was never acknowledged")
	}
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed")
	}
	if got := f.boards.saveCount(); got != saves {
		t.Fatalf("saveCount = %d, want %d — the board was written although the turn's transaction failed", got, saves)
	}
	if col, row := storedSquare(t, f, attackerPieceID); col != 4 || row != 4 {
		t.Fatalf("stored attacker at (%d,%d), want the seeded (4,4)", col, row)
	}
	if n := len(f.masterActions.snapshot()); n != 0 {
		t.Fatalf("master_actions got %d row(s) on its own, want none", n)
	}
}

// actionIDOf returns the actionId of the n-th action_enqueued (0-based) the collector has.
func actionIDOf(t *testing.T, c *collector, n int) uuid.UUID {
	t.Helper()
	i := 0
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeActionEnqueued {
			continue
		}
		if i == n {
			var p game.ActionEnqueuedPayload
			if err := json.Unmarshal(m.Payload, &p); err != nil {
				t.Fatalf("unmarshal action_enqueued: %v", err)
			}
			return p.ActionID
		}
		i++
	}
	t.Fatalf("no action_enqueued #%d", n)
	return uuid.Nil
}

// open_next_action closes T1 and opens T2 in one Execute. Each turn's in-turn drag goes with
// its own turn: T1's into T1's PersistTurnClose, T2's held until T2 closes. And T1's board is
// T1's: it carries T1's drag but NOT T2's opened move, which only walks after T1 was written.
func TestTurnScoped_OpenNextActionKeepsEachTurnsWritesWithItsTurn(t *testing.T) {
	f := newCombatFixture(t, withVictimPiece)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	f.enqueueAttack(t, player)
	if !awaitCount(pc, game.MsgTypeActionEnqueued, 1, 2*time.Second) {
		t.Fatal("the attack was never enqueued")
	}
	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId": f.victimID.String(),
		"move":    map[string]any{"category": string(enum.Dash), "from": [3]int{6, 6, 0}, "position": [3]int{6, 7, 0}},
	})
	if !awaitCount(pc, game.MsgTypeActionEnqueued, 2, 2*time.Second) {
		t.Fatal("the victim's dash was never enqueued")
	}

	// T1 is the attack, pulled explicitly so the order does not hang on the dice.
	sendWS(t, master, string(game.MsgTypePullAction), game.PullActionPayload{ActionID: actionIDOf(t, pc, 0)})
	if !awaitCount(mc, game.MsgTypeTurnOpened, 1, 2*time.Second) {
		t.Fatal("T1 never opened")
	}
	t1 := lastTurnOpened(t, mc).TurnID
	sendMasterMove(t, master, f.attackerID, [3]int{4, 2, 0})
	if !awaitCount(mc, game.MsgTypeMasterActionEnqueued, 1, 2*time.Second) {
		t.Fatal("T1's drag was never acknowledged")
	}

	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !awaitCount(mc, game.MsgTypeTurnOpened, 2, 2*time.Second) {
		t.Fatal("T2 never opened")
	}
	t2 := lastTurnOpened(t, mc).TurnID
	f.awaitPersistedTurn(t, t1)
	sendMasterMove(t, master, f.attackerID, [3]int{3, 2, 0})
	if !awaitCount(mc, game.MsgTypeMasterActionEnqueued, 2, 2*time.Second) {
		t.Fatal("T2's drag was never acknowledged")
	}

	r1 := f.roundRepo.masterActionsFor(t1)
	if len(r1) != 1 || !samePos(pieceContentOf(t, r1[0]).To, ptrPos([3]int{4, 2, 0})) {
		t.Fatalf("T1's PersistTurnClose got %+v, want only T1's drag", r1)
	}
	b1, ok := f.roundRepo.boardFor(t1)
	if !ok {
		t.Fatal("T1's close handed no board")
	}
	if col, row := squareColRow(t, findPiece(t, b1.Pieces, victimPieceID)); col != 6 || row != 6 {
		t.Fatalf("T1's board has the victim at (%d,%d), want (6,6) — T2's opened Dash leaked into T1's write", col, row)
	}
	if contains(f.roundRepo.persistedTurnIDs(), t2) {
		t.Fatal("T2 was written while still open")
	}
	if n := len(f.masterActions.snapshot()); n != 0 {
		t.Fatalf("master_actions got %d row(s) on its own — T2's drag must be held", n)
	}

	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("T2 never closed")
	}
	f.awaitPersistedTurn(t, t2)
	r2 := f.roundRepo.masterActionsFor(t2)
	if len(r2) != 1 || r2[0].TurnUUID == nil || *r2[0].TurnUUID != t2 ||
		!samePos(pieceContentOf(t, r2[0]).To, ptrPos([3]int{3, 2, 0})) {
		t.Fatalf("T2's PersistTurnClose got %+v, want only T2's drag", r2)
	}
	b2, _ := f.roundRepo.boardFor(t2)
	if b2 == nil {
		t.Fatal("T2's close handed no board")
	}
	if col, row := squareColRow(t, findPiece(t, b2.Pieces, victimPieceID)); col != 6 || row != 7 {
		t.Fatalf("T2's board has the victim at (%d,%d), want its Dash's (6,7)", col, row)
	}
}

// A Race round that runs out closes its last turn AND itself in one open_next_action (the
// closedIn path): the master action held for that turn is written with it, under the round it
// happened in — not the successor.
func TestTurnScoped_AnExhaustedRoundWritesTheHeldMasterActionWithItsLastTurn(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	round1 := f.session.GetActiveRound().GetID()
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{Mode: string(enum.Race)})
	if !mc.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("the regime switch was never announced")
	}
	f.enqueueAttack(t, player)
	if !pc.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the action never opened")
	}
	turnID := lastTurnOpened(t, mc).TurnID
	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatal("the drag was never acknowledged")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeRoundClosed, 2*time.Second) {
		t.Fatal("the round never ran out")
	}
	f.awaitPersistedTurn(t, turnID)

	if rd, ok := f.roundRepo.roundOfPersistedTurn(turnID); !ok || rd != round1 {
		t.Fatalf("the turn was written under round %s, want the exhausted %s", rd, round1)
	}
	recs := f.roundRepo.masterActionsFor(turnID)
	if len(recs) != 1 || recs[0].RoundUUID != round1 || recs[0].TurnUUID == nil || *recs[0].TurnUUID != turnID {
		t.Fatalf("PersistTurnClose got %+v, want the drag, in round %s and turn %s", recs, round1, turnID)
	}
	if _, ok := f.roundRepo.boardFor(turnID); !ok {
		t.Fatal("the exhausting close handed no board")
	}
}

// After a close the round still names its last turn as current, but no turn is OPEN: a drag
// then is between turns — saved and inserted at once, with no turnUuid.
func TestTurnScoped_ADragAfterACloseIsWrittenAtOnce(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	turnID := f.openAttackTurn(t, master, player, mc)
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed")
	}
	f.awaitPersistedTurn(t, turnID)
	saves := f.boards.saveCount()

	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatal("the drag was never acknowledged")
	}
	recs := f.masterActions.await(t, 1, 2*time.Second)
	if recs[0].TurnUUID != nil {
		t.Fatalf("turnUuid = %v, want nil — the turn had already closed", *recs[0].TurnUUID)
	}
	if got := f.boards.saveCount(); got != saves+1 {
		t.Fatalf("saveCount = %d, want %d — a drag between turns saves at once", got, saves+1)
	}
	if col, row := storedSquare(t, f, attackerPieceID); col != 6 || row != 4 {
		t.Fatalf("stored attacker at (%d,%d), want the drag's (6,4)", col, row)
	}
}

// The room needs no master-action repository of its own to hold one for a turn: the held path
// is written by PersistTurnClose.
func TestTurnScoped_AHeldMasterActionNeedsNoMasterActionRepo(t *testing.T) {
	f := newCombatFixture(t, withoutMasterActionRepo)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	turnID := f.openAttackTurn(t, master, player, mc)

	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatal("the drag was never acknowledged")
	}
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed")
	}
	f.awaitPersistedTurn(t, turnID)
	if recs := f.roundRepo.masterActionsFor(turnID); len(recs) != 1 {
		t.Fatalf("PersistTurnClose got %d master action(s), want the drag", len(recs))
	}
}
