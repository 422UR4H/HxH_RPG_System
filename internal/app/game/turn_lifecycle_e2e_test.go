package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file covers the part of a turn's public life the table could not follow: WHICH action
// a turn_opened opened.
//
// It is a debt the front's Phase 6 collected against the back, and it is one debt seen from
// the two arms that pay it — open_next_action and pull_action both end in announceOpenedTurn.

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
