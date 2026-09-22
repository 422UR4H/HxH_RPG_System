package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file covers the two halves of a turn's public life that the table could not follow:
// WHICH action a turn_opened opened, and THAT a turn ended when the master never said so
// out loud.
//
// Both are debts the front's Phase 6 collected against the back. They live together here
// because they are the same two arms of room.go — open_next_action and pull_action — seen
// from the two ends of the same transition.

// ─── helpers ────────────────────────────────────────────────────────────────

// collectedActionIDs lists the actionId of every message of the given type a collector has
// gathered, in arrival order. action_enqueued (to whoever enqueued) and action_queued (to the
// master) both carry one under the same key, so one reader serves both.
//
// Arrival order is send order here: each connection is drained by a single read pump that
// handles one message at a time, so the first ack belongs to the first enqueue.
func collectedActionIDs(t *testing.T, c *collector, want game.MessageType) []uuid.UUID {
	t.Helper()
	var ids []uuid.UUID
	for _, m := range c.snapshotMessages() {
		if m.Type != want {
			continue
		}
		var p struct {
			ActionID uuid.UUID `json:"actionId"`
		}
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal %s: %v", want, err)
		}
		ids = append(ids, p.ActionID)
	}
	return ids
}

// indexOfTurnMessage returns the arrival position of the first message of that type naming
// that turn, or -1. The collector appends in arrival order, so comparing indices compares
// arrival order — the same trick indexOfMessage plays, narrowed to one turn because a
// transition puts two turn_openeds on the wire and only one of them is the new one.
func indexOfTurnMessage(t *testing.T, msgs []game.Message, want game.MessageType, turnID uuid.UUID) int {
	t.Helper()
	for i, m := range msgs {
		if m.Type != want {
			continue
		}
		var p struct {
			TurnID uuid.UUID `json:"turnId"`
		}
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal %s: %v", want, err)
		}
		if p.TurnID == turnID {
			return i
		}
	}
	return -1
}

// enqueueTwoFromTheSameActor puts two identical attacks from the fixture's attacker in the
// queue and returns their IDs as the MASTER learned them, in arrival order.
//
// Two actions of the SAME character is the shape that makes the debt visible: with one action
// each, actorId alone already tells the table which one opened.
func enqueueTwoFromTheSameActor(
	t *testing.T, f *combatFixture, player *websocket.Conn, masterMsgs *collector,
) []uuid.UUID {
	t.Helper()
	f.enqueueAttack(t, player)
	f.enqueueAttack(t, player)
	if !awaitCount(masterMsgs, game.MsgTypeActionQueued, 2, 2*time.Second) {
		t.Fatalf("the master did not see both actions enter the queue; it received: %v",
			messageTypes(masterMsgs.snapshotMessages()))
	}
	ids := collectedActionIDs(t, masterMsgs, game.MsgTypeActionQueued)
	if len(ids) != 2 || ids[0] == ids[1] || ids[0] == uuid.Nil {
		t.Fatalf("expected two distinct queued action IDs, got %v", ids)
	}
	return ids
}

// ─── B1: turn_opened carries actionId ───────────────────────────────────────

// TestE2E_TurnOpenedNamesTheActionItOpened is B1 over the open_next_action arm.
//
// With two actions of the same character queued, actorId is the same in both turn_openeds and
// the player has nothing left to tell them apart. The IDs the two opens announce must be the
// two the queue acknowledged, one each — an actionId that repeated, or that named something
// nobody enqueued, would be worse than none.
func TestE2E_TurnOpenedNamesTheActionItOpened(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	queued := enqueueTwoFromTheSameActor(t, f, player, masterMsgs)

	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the first action never opened")
	}
	first := lastTurnOpened(t, masterMsgs)

	// Free mode, no economy: the second open closes the first turn and opens the other action.
	sendWS(t, master, "open_next_action", map[string]any{})
	if !awaitCount(masterMsgs, game.MsgTypeTurnOpened, 2, 2*time.Second) {
		t.Fatal("the second action never opened")
	}
	second := lastTurnOpened(t, masterMsgs)

	if first.ActionID == uuid.Nil {
		t.Fatal("turn_opened carried no actionId — with two actions of the same actor queued, " +
			"nothing on the wire says which one opened")
	}
	if first.ActionID == second.ActionID {
		t.Fatalf("both turns announced actionId %s — the field does not follow the turn",
			first.ActionID)
	}
	announced := map[uuid.UUID]bool{first.ActionID: true, second.ActionID: true}
	for _, id := range queued {
		if !announced[id] {
			t.Errorf("actionId %s was acknowledged into the queue and no turn_opened named it; "+
				"the two opens announced %s and %s", id, first.ActionID, second.ActionID)
		}
	}

	t.Run("the table gets the id too, not just the master", func(t *testing.T) {
		if !awaitCount(playerMsgs, game.MsgTypeTurnOpened, 2, 2*time.Second) {
			t.Fatal("the player never saw both turns open — turn_opened is table state")
		}
		// Same message, same field: turn_opened is broadcast, so the player's copy is the
		// master's copy. Checking it here is what proves the id is not master-only.
		p := lastTurnOpened(t, playerMsgs)
		if p.ActionID != second.ActionID {
			t.Errorf("the player's turn_opened named actionId %s, the master's %s",
				p.ActionID, second.ActionID)
		}
	})
}

// TestE2E_PulledTurnOpenedNamesThePulledAction is B1 over the pull_action arm — the sharp
// half of the debt, because here the test KNOWS which of the two the master asked for.
func TestE2E_PulledTurnOpenedNamesThePulledAction(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := newCollector(master)

	queued := enqueueTwoFromTheSameActor(t, f, player, masterMsgs)

	// The SECOND one: pulling out of order is the whole point of pull_action, and it is also
	// what keeps the assertion honest — the first is what open_next_action would have taken.
	wanted := queued[1]
	sendWS(t, master, "pull_action", map[string]any{"actionId": wanted.String()})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatalf("pull_action opened nothing; the master received: %v",
			messageTypes(masterMsgs.snapshotMessages()))
	}

	got := lastTurnOpened(t, masterMsgs).ActionID
	if got != wanted {
		t.Errorf("turn_opened named actionId %s, want the pulled %s (the other queued one is %s)",
			got, wanted, queued[0])
	}
}

// ─── B3: turn_closed on the implicit close ──────────────────────────────────

// TestE2E_OpenNextActionAnnouncesTheTurnItClosed is B3 over the open_next_action arm.
//
// Opening the next action closes the open turn on the way through. Until this test, that
// close was silent: only the close_turn verb ever emitted turn_closed, so a table watching
// the event list saw turns begin and never end.
func TestE2E_OpenNextActionAnnouncesTheTurnItClosed(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	enqueueTwoFromTheSameActor(t, f, player, masterMsgs)

	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the first action never opened")
	}
	firstTurn := lastTurnOpened(t, masterMsgs).TurnID

	sendWS(t, master, "open_next_action", map[string]any{})
	if !awaitCount(masterMsgs, game.MsgTypeTurnOpened, 2, 2*time.Second) {
		t.Fatal("the second action never opened")
	}
	secondTurn := lastTurnOpened(t, masterMsgs).TurnID

	assertClosedBeforeOpened(t, playerMsgs, firstTurn, secondTurn, "the player")
	assertClosedBeforeOpened(t, masterMsgs, firstTurn, secondTurn, "the master")
}

// TestE2E_PullActionAnnouncesTheTurnItClosed is B3 over the pull_action arm. Same close, same
// announcement: which verb the master reached for must not change what the table is told.
func TestE2E_PullActionAnnouncesTheTurnItClosed(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	queued := enqueueTwoFromTheSameActor(t, f, player, masterMsgs)

	sendWS(t, master, "pull_action", map[string]any{"actionId": queued[0].String()})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the first pull opened nothing")
	}
	firstTurn := lastTurnOpened(t, masterMsgs).TurnID

	sendWS(t, master, "pull_action", map[string]any{"actionId": queued[1].String()})
	if !awaitCount(masterMsgs, game.MsgTypeTurnOpened, 2, 2*time.Second) {
		t.Fatal("the second pull opened nothing")
	}
	secondTurn := lastTurnOpened(t, masterMsgs).TurnID

	assertClosedBeforeOpened(t, playerMsgs, firstTurn, secondTurn, "the player")
	assertClosedBeforeOpened(t, masterMsgs, firstTurn, secondTurn, "the master")
}

// assertClosedBeforeOpened is the shared assertion of both B3 tests: the old turn's
// turn_closed reached this client, and it reached it BEFORE the new turn's turn_opened.
//
// The order is the point, not a detail — a table that is told the next turn started before it
// is told the last one ended has to reorder the two itself to draw a correct event list.
func assertClosedBeforeOpened(t *testing.T, c *collector, closed, opened uuid.UUID, who string) {
	t.Helper()
	if !c.await(game.MsgTypeTurnClosed, 2*time.Second) {
		t.Fatalf("%s was never told turn %s ended; it received: %v",
			who, closed, messageTypes(c.snapshotMessages()))
	}
	msgs := c.snapshotMessages()
	closedAt := indexOfTurnMessage(t, msgs, game.MsgTypeTurnClosed, closed)
	if closedAt < 0 {
		t.Fatalf("%s got a turn_closed, but never for turn %s — the implicit close named "+
			"the wrong turn", who, closed)
	}
	openedAt := indexOfTurnMessage(t, msgs, game.MsgTypeTurnOpened, opened)
	if openedAt < 0 {
		t.Fatalf("%s never saw turn %s open", who, opened)
	}
	if closedAt > openedAt {
		t.Errorf("%s saw the next turn open (position %d) before the last one closed "+
			"(position %d)", who, openedAt, closedAt)
	}
}
