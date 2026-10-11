package game_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/cancelledaction"
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

// awaitBarsAfter waits for a bars_updated whose Seq is above seq. Counting messages is not a
// barrier: the enqueue's own bars_updated can land after a count snapshot and satisfy "one
// more", so the count would run before the cancel's messages.
func awaitBarsAfter(t *testing.T, c *collector, seq uint64, d time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if c.count(game.MsgTypeBarsUpdated) > 0 && lastBarsUpdated(t, c).Seq > seq {
			return true
		}
		time.Sleep(10 * time.Millisecond)
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
	seqBefore := lastBarsUpdated(t, bystanderMsgs).Seq

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
	if !awaitBarsAfter(t, bystanderMsgs, seqBefore, 2*time.Second) {
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

// The master cancelling an NPC's action is both the master and the sender: the arm must send
// ONE action_cancelled, not two, and nothing to the players.
func TestE2E_MasterCancelsAnNPCActionOnce(t *testing.T) {
	f := newCombatFixture(t, withNPCTarget)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs, playerMsgs := collectFrom(master), collectFrom(player)

	f.enqueueAttackFrom(t, master, f.npcID)
	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatalf("the master never saw their NPC's action queued; errors: %v", errorCodes(t, masterMsgs))
	}
	actionID := lastActionQueuedID(t, masterMsgs)
	// The enqueue's own bars_updated goes out on another goroutine: wait for it, then take its
	// Seq as the line the cancel's bars_updated must cross.
	if !awaitCount(masterMsgs, game.MsgTypeBarsUpdated, 1, 2*time.Second) {
		t.Fatal("the master never saw the order after the enqueue")
	}
	seqBefore := lastBarsUpdated(t, masterMsgs).Seq

	sendWS(t, master, "cancel_action", map[string]any{"actionId": actionID.String()})

	// The master's bars_updated is the barrier: broadcastBars runs after the targeted sends
	// to the master, so once it lands a duplicate action_cancelled would have landed too.
	if !awaitBarsAfter(t, masterMsgs, seqBefore, 2*time.Second) {
		t.Fatalf("the master never saw the order change; errors: %v", errorCodes(t, masterMsgs))
	}
	if got := cancelledIDs(t, masterMsgs); len(got) != 1 || got[0] != actionID {
		t.Errorf("master action_cancelled ids = %v, want exactly [%v]", got, actionID)
	}
	if n := playerMsgs.count(game.MsgTypeActionCancelled); n != 0 {
		t.Errorf("the player received %d action_cancelled for the master's NPC — the queue is secret", n)
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
	// No barrier orders the bystander's refusal against the master's connection (different
	// sockets); these counts only catch a leak that already landed. The reconnection check
	// below is what proves the queue is intact.
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

// fakeCancelledActionStore stands in for pgcancelledaction.Repository: it keeps every Record it
// was handed, in order.
type fakeCancelledActionStore struct {
	mu   sync.Mutex
	recs []cancelledaction.Record
}

func (s *fakeCancelledActionStore) Insert(_ context.Context, r cancelledaction.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recs = append(s.recs, r)
	return nil
}

func (s *fakeCancelledActionStore) snapshot() []cancelledaction.Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]cancelledaction.Record(nil), s.recs...)
}

// await waits for at least n records. Like fakeMasterActionStore.await, reading under the
// store's mutex is the barrier that makes reading the session afterwards race-free.
func (s *fakeCancelledActionStore) await(t *testing.T, n int, d time.Duration) []cancelledaction.Record {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if recs := s.snapshot(); len(recs) >= n {
			return recs
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("cancelled_actions has %d record(s), want at least %d", len(s.snapshot()), n)
	return nil
}

// With no turn open, a cancelled action is written the instant it is cancelled (spec 2026-10-10
// §8.1 P7): the action whole, who cancelled it, where it would have happened, and no turn.
func TestE2E_ACancelWithNoOpenTurnIsRecordedAtOnce(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	playerMsgs := collectFrom(player)

	f.enqueueAttack(t, player)
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}
	actionID := actionIDsInOrder(t, playerMsgs)[0]
	sendWS(t, player, "cancel_action", map[string]any{"actionId": actionID.String()})

	recs := f.cancelled.await(t, 1, 2*time.Second)
	if len(recs) != 1 {
		t.Fatalf("cancelled_actions = %d record(s), want 1", len(recs))
	}
	rec := recs[0]
	if rec.UUID != actionID || rec.Action.GetID() != actionID {
		t.Errorf("record uuid = %v, action id = %v, want both %v", rec.UUID, rec.Action.GetID(), actionID)
	}
	if rec.MatchUUID != f.matchUUID || rec.ActorUUID != f.attackerID || rec.CancelledBy != f.playerUUID {
		t.Errorf("record = match %v actor %v by %v, want %v %v %v",
			rec.MatchUUID, rec.ActorUUID, rec.CancelledBy, f.matchUUID, f.attackerID, f.playerUUID)
	}
	if rec.TurnUUID != nil {
		t.Errorf("turnUuid = %v, want nil — no turn was open", *rec.TurnUUID)
	}
	if rec.Action.Attack == nil {
		t.Error("the recorded action lost its attack")
	}
	if rec.CancelledAt.IsZero() {
		t.Error("cancelledAt is zero")
	}
	if rd := f.session.GetActiveRound(); rd == nil || rec.RoundUUID != rd.GetID() {
		t.Errorf("round = %v, want the active round", rec.RoundUUID)
	}
	if sc := f.session.GetActiveScene(); sc == nil || rec.SceneUUID != sc.GetID() {
		t.Errorf("scene = %v, want the active scene", rec.SceneUUID)
	}
}

// With a turn open, the record is built at the instant of the cancellation but written only by
// that turn's close, in its transaction (spec 2026-10-10 §8.1 P7) — nothing before it.
func TestE2E_ACancelInsideAnOpenTurnIsRecordedWithItsClose(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)
	turnID := f.openAttackTurn(t, master, player, mc)

	// A second character of the same player queues while the turn is open, then takes it back.
	f.enqueueAttackAt(t, player, f.victimID, f.attackerID)
	if !awaitCount(mc, game.MsgTypeActionQueued, 2, 2*time.Second) {
		t.Fatalf("the second action was never queued; errors: %v", errorCodes(t, pc))
	}
	actionID := lastActionQueuedID(t, mc)
	sendWS(t, player, "cancel_action", map[string]any{"actionId": actionID.String()})
	if !pc.await(game.MsgTypeActionCancelled, 2*time.Second) {
		t.Fatalf("the cancellation was never acknowledged; errors: %v", errorCodes(t, pc))
	}
	// The ack is sent after the arm's write would have run: anything written at once is here.
	if recs := f.cancelled.snapshot(); len(recs) != 0 {
		t.Fatalf("cancelled_actions got %d record(s) mid-turn, want none before the close", len(recs))
	}
	if n := len(f.roundRepo.closeData()); n != 0 {
		t.Fatalf("PersistTurnClose ran %d time(s) before the close", n)
	}

	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed")
	}
	f.awaitPersistedTurn(t, turnID)

	closes := f.roundRepo.closeData()
	if len(closes) != 1 {
		t.Fatalf("PersistTurnClose ran %d time(s), want 1", len(closes))
	}
	got := closes[0].CancelledActions
	if len(got) != 1 || got[0].UUID != actionID || got[0].ActorUUID != f.victimID {
		t.Fatalf("PersistTurnClose got cancelled actions %+v, want the victim's %v", got, actionID)
	}
	if got[0].TurnUUID == nil || *got[0].TurnUUID != turnID {
		t.Errorf("turnUuid = %v, want the open turn %v", got[0].TurnUUID, turnID)
	}
	if recs := f.cancelled.snapshot(); len(recs) != 0 {
		t.Errorf("the room also inserted %d record(s) on its own — a held one is the close's alone", len(recs))
	}
}
