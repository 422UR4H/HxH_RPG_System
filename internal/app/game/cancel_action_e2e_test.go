package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

// This file is the cancel_action verb (spec 2026-10-10-combat-cancel-action-back-design.md)
// against the real Room: the wire is what is under test, so every assertion reads what a
// client actually received.

// cancelledIDs lists every action_cancelled actionId the collector received.
func cancelledIDs(t *testing.T, c *collector) []uuid.UUID {
	t.Helper()
	var ids []uuid.UUID
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeActionCancelled {
			continue
		}
		var p game.ActionCancelledPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal action_cancelled: %v", err)
		}
		ids = append(ids, p.ActionID)
	}
	return ids
}

// lastError returns the most recent error payload the collector received.
func lastError(t *testing.T, c *collector) game.ErrorPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeError {
			continue
		}
		var p game.ErrorPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		return p
	}
	t.Fatal("no error message was received")
	return game.ErrorPayload{}
}

func orderHasActor(p game.BarsUpdatedPayload, actor uuid.UUID) bool {
	for _, slot := range p.Order {
		if slot.ActorID == actor {
			return true
		}
	}
	return false
}

func TestE2E_PlayerCancelsOwnQueuedAction(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	defer bystander.Close()   //nolint:errcheck
	readMessage(t, bystander) // room_state

	masterMsgs, playerMsgs, bystanderMsgs := collectFrom(master), collectFrom(player), collectFrom(bystander)

	f.enqueueAttack(t, player)
	if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, 1, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}
	actionID := actionIDsInOrder(t, playerMsgs)[0]
	if !awaitCount(bystanderMsgs, game.MsgTypeBarsUpdated, 1, 2*time.Second) {
		t.Fatal("the table never saw the order after the enqueue")
	}
	// Non-vacuous: the actor really was in the public order before the cancel.
	if !orderHasActor(lastBarsUpdated(t, bystanderMsgs), f.attackerID) {
		t.Fatal("the attacker is not in the public order before cancelling — the fixture proves nothing")
	}
	barsBefore := bystanderMsgs.count(game.MsgTypeBarsUpdated)

	sendWS(t, player, "cancel_action", map[string]any{"actionId": actionID.String()})

	if !masterMsgs.await(game.MsgTypeActionCancelled, 2*time.Second) {
		t.Fatal("the master never heard about the cancellation")
	}
	if !playerMsgs.await(game.MsgTypeActionCancelled, 2*time.Second) {
		t.Fatal("the sender never got their action_cancelled")
	}
	if got := cancelledIDs(t, masterMsgs); len(got) != 1 || got[0] != actionID {
		t.Errorf("master action_cancelled ids = %v, want [%v]", got, actionID)
	}
	if got := cancelledIDs(t, playerMsgs); len(got) != 1 || got[0] != actionID {
		t.Errorf("sender action_cancelled ids = %v, want [%v]", got, actionID)
	}

	// The bystander's bars_updated is the barrier: broadcastBars runs after the targeted
	// sends, so once it lands any leaked action_cancelled would have landed too.
	if !awaitCount(bystanderMsgs, game.MsgTypeBarsUpdated, barsBefore+1, 2*time.Second) {
		t.Fatal("the table never saw the order change after the cancel")
	}
	if n := bystanderMsgs.count(game.MsgTypeActionCancelled); n != 0 {
		t.Errorf("the other player received %d action_cancelled — the queue is secret", n)
	}
	if orderHasActor(lastBarsUpdated(t, bystanderMsgs), f.attackerID) {
		t.Error("the last bars_updated still carries the cancelled actor in order")
	}

	t.Run("open_next_action finds nothing to open", func(t *testing.T) {
		sendWS(t, master, "open_next_action", map[string]any{})
		// Wait for the master's answer to this command, whichever form the regime gives it.
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if masterMsgs.count(game.MsgTypeError) > 0 || masterMsgs.count(game.MsgTypeRoundClosed) > 0 {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if n := masterMsgs.count(game.MsgTypeTurnOpened); n != 0 {
			t.Fatalf("the cancelled action opened a turn (%d turn_opened)", n)
		}
		if masterMsgs.count(game.MsgTypeError) == 0 && masterMsgs.count(game.MsgTypeRoundClosed) == 0 {
			t.Fatal("open_next_action got no answer at all")
		}
	})
}

func TestE2E_CancelledActionIsGoneAfterReconnect(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	playerMsgs := newCollector(player)

	f.enqueueAttack(t, player)
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}
	actionID := actionIDsInOrder(t, playerMsgs)[0]
	sendWS(t, player, "cancel_action", map[string]any{"actionId": actionID.String()})
	if !playerMsgs.await(game.MsgTypeActionCancelled, 2*time.Second) {
		t.Fatal("the cancellation was never acknowledged")
	}
	master.Close() //nolint:errcheck
	player.Close() //nolint:errcheck

	lateMaster := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	defer lateMaster.Close()   //nolint:errcheck
	readMessage(t, lateMaster) // room_state
	lateMasterMsgs := collectFrom(lateMaster)
	if !lateMasterMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the reconnecting master never received match_full_state")
	}
	var masterFull game.MatchFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, lateMasterMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &masterFull,
	); err != nil {
		t.Fatalf("unmarshal master match_full_state: %v", err)
	}
	for _, q := range masterFull.Queue {
		if q.ActionID == actionID {
			t.Errorf("the master's queue still holds the cancelled action %v", actionID)
		}
	}

	latePlayer := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	defer latePlayer.Close()   //nolint:errcheck
	readMessage(t, latePlayer) // room_state
	latePlayerMsgs := collectFrom(latePlayer)
	if !latePlayerMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the reconnecting player never received match_full_state")
	}
	var playerFull game.MatchFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, latePlayerMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &playerFull,
	); err != nil {
		t.Fatalf("unmarshal player match_full_state: %v", err)
	}
	if playerFull.OwnQueue == nil || len(*playerFull.OwnQueue) != 0 {
		t.Errorf("ownQueue = %+v, want a present, empty list", playerFull.OwnQueue)
	}
}

func TestE2E_CannotCancelSomeoneElsesAction(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	defer bystander.Close()   //nolint:errcheck
	readMessage(t, bystander) // room_state

	masterMsgs, playerMsgs, bystanderMsgs := collectFrom(master), collectFrom(player), collectFrom(bystander)

	f.enqueueAttack(t, player)
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}
	actionID := actionIDsInOrder(t, playerMsgs)[0]

	sendWS(t, bystander, "cancel_action", map[string]any{"actionId": actionID.String()})
	if !bystanderMsgs.await(game.MsgTypeError, 2*time.Second) {
		t.Fatal("the intruder was not refused")
	}
	if e := lastError(t, bystanderMsgs); e.Code != "game_error" || e.Message != "action actor does not match player" {
		t.Errorf("error = %+v, want game_error / action actor does not match player", e)
	}
	// The refusal is the barrier for the master: the arm answers the sender before any
	// targeted send could have happened.
	if n := masterMsgs.count(game.MsgTypeActionCancelled); n != 0 {
		t.Errorf("the master received %d action_cancelled for a refused cancel", n)
	}
	if n := playerMsgs.count(game.MsgTypeActionCancelled); n != 0 {
		t.Errorf("the owner received %d action_cancelled for a refused cancel", n)
	}

	master.Close() //nolint:errcheck
	lateMaster := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	defer lateMaster.Close()   //nolint:errcheck
	readMessage(t, lateMaster) // room_state
	lateMsgs := collectFrom(lateMaster)
	if !lateMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the reconnecting master never received match_full_state")
	}
	var full game.MatchFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, lateMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &full,
	); err != nil {
		t.Fatalf("unmarshal match_full_state: %v", err)
	}
	found := false
	for _, q := range full.Queue {
		if q.ActionID == actionID {
			found = true
		}
	}
	if !found {
		t.Error("the refused cancel removed the action from the master's queue")
	}
}

func TestE2E_CancelActionRejectsBadPayload(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	playerMsgs := collectFrom(player)

	sendWS(t, player, "cancel_action", map[string]any{"actionId": 123})
	if !playerMsgs.await(game.MsgTypeError, 2*time.Second) {
		t.Fatal("a malformed cancel_action got no error")
	}
	if e := lastError(t, playerMsgs); e.Code != "invalid_payload" || e.Message != "invalid cancel_action payload" {
		t.Errorf("error = %+v, want invalid_payload / invalid cancel_action payload", e)
	}
}
