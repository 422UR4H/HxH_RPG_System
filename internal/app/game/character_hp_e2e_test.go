package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// This file covers the live HP path: until it existed, the damage a close applied only
// reached the table by REST, on the next time somebody re-fetched a sheet. The front's
// Phase 6 needs the bar to move while the turn is being watched, which is why the news
// travels over the socket now.
//
// It is a message of its own rather than a field of resolution_updated on purpose — see the
// type's comment in message.go. The tests here therefore never read a resolution.

// ─── helpers ────────────────────────────────────────────────────────────────

// victimHealthBar reads the fixture victim's health bar directly.
//
// Same rule as victimHP: the room mutates this sheet from its own goroutine, so a caller
// reads it only when nothing is in flight — before the first send, or after the writer has
// confirmed the close persisted. The maximum is what the tests here need and it never moves,
// so reading it up front is always safe.
func victimHealthBar(t *testing.T, f *combatFixture) (current, max int) {
	t.Helper()
	bar, ok := f.victim.GetAllStatusBar()[enum.Health]
	if !ok {
		t.Fatal("the victim's sheet has no health bar")
	}
	return bar.GetCurrent(), bar.GetMax()
}

// collectedHpChanges lists every character_hp_changed a collector has gathered, in arrival
// order.
func collectedHpChanges(t *testing.T, c *collector) []game.CharacterHpChangedPayload {
	t.Helper()
	var out []game.CharacterHpChangedPayload
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeCharacterHpChanged {
			continue
		}
		var p game.CharacterHpChangedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal character_hp_changed: %v", err)
		}
		out = append(out, p)
	}
	return out
}

// assertVictimHpChange checks the one change this client was entitled to: the right character,
// the HP the sheet actually holds now, the bar's real maximum, and the damage that got it
// there.
func assertVictimHpChange(
	t *testing.T, c *collector, f *combatFixture, hpBefore, maxHP int, who string,
) {
	t.Helper()
	changes := collectedHpChanges(t, c)
	if len(changes) != 1 {
		t.Fatalf("%s received %d character_hp_changed, want exactly 1; they received: %v",
			who, len(changes), messageTypes(c.snapshotMessages()))
	}
	got := changes[0]
	if got.CharacterID != f.victimID {
		t.Errorf("%s was told %s changed, want the victim %s", who, got.CharacterID, f.victimID)
	}
	if got.Damage <= 0 {
		t.Fatalf("%s got damage %d — nothing was applied, so this test proves nothing",
			who, got.Damage)
	}
	if got.HP != hpBefore-got.Damage {
		t.Errorf("%s got hp %d with damage %d, want %d (%d - %d)",
			who, got.HP, got.Damage, hpBefore-got.Damage, hpBefore, got.Damage)
	}
	if got.MaxHP != maxHP {
		t.Errorf("%s got maxHp %d, want the bar's real maximum %d", who, got.MaxHP, maxHP)
	}
}

// ─── B2: character_hp_changed ───────────────────────────────────────────────

// TestE2E_ClosingATurnTellsTheMasterAndTheOwnerTheNewHP is the projection axis of B2.
//
// Three clients, because two cannot prove a gate: the master sees everything and the victim's
// owner is entitled to their own character's HP, so only the third player's copy decides
// whether the projection is real.
func TestE2E_ClosingATurnTellsTheMasterAndTheOwnerTheNewHP(t *testing.T) {
	f := newCombatFixture(t, withBystander)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	outsider := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	defer outsider.Close()   //nolint:errcheck
	readMessage(t, outsider) // room_state

	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	outsiderMsgs := collectFrom(outsider)

	// Nothing is in flight yet, so the sheet is safe to read.
	hpBefore, maxHP := victimHealthBar(t, f)
	if hpBefore <= 0 || maxHP <= 0 {
		t.Fatalf("the victim starts at %d/%d HP — the fixture is wrong", hpBefore, maxHP)
	}

	f.enqueueAttack(t, player)
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}
	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatalf("the attack never opened; the master received: %v",
			messageTypes(masterMsgs.snapshotMessages()))
	}

	sendWS(t, master, "close_turn", map[string]any{"confirm": true})
	if !f.writer.awaitPersisted(3 * time.Second) {
		t.Fatal("the closing turn never persisted the damage")
	}

	if !masterMsgs.await(game.MsgTypeCharacterHpChanged, 2*time.Second) {
		t.Fatalf("the master was never told the HP moved; they received: %v",
			messageTypes(masterMsgs.snapshotMessages()))
	}
	assertVictimHpChange(t, masterMsgs, f, hpBefore, maxHP, "the master")

	if !playerMsgs.await(game.MsgTypeCharacterHpChanged, 2*time.Second) {
		t.Fatalf("the victim's owner was never told their own HP moved; they received: %v",
			messageTypes(playerMsgs.snapshotMessages()))
	}
	assertVictimHpChange(t, playerMsgs, f, hpBefore, maxHP, "the victim's owner")

	t.Run("a third player at the table hears nothing", func(t *testing.T) {
		// turn_closed is the barrier, not a sleep: the room dispatches character_hp_changed
		// straight into each client's queue BEFORE handing turn_closed to the room broadcast,
		// on the same goroutine — so by the time turn_closed lands here, anything this player
		// was entitled to is already queued ahead of it.
		if !outsiderMsgs.await(game.MsgTypeTurnClosed, 2*time.Second) {
			t.Fatal("the third player received nothing at all — this connection is dead, not gated")
		}
		if n := outsiderMsgs.count(game.MsgTypeCharacterHpChanged); n != 0 {
			t.Fatalf("a player who owns neither the target nor the table was told its HP %d "+
				"time(s); they received: %v", n, messageTypes(outsiderMsgs.snapshotMessages()))
		}
	})
}

// TestE2E_TheImplicitCloseAlsoTellsTheNewHP is B2 over the implicit close.
//
// open_next_action closes the open turn on its way through, and that close applies damage
// exactly like close_turn's. A live HP path that only worked when the master said "close"
// out loud would leave the bar stale for the whole rest of the round.
func TestE2E_TheImplicitCloseAlsoTellsTheNewHP(t *testing.T) {
	f := newCombatFixture(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	hpBefore, maxHP := victimHealthBar(t, f)

	enqueueTwoFromTheSameActor(t, f, player, masterMsgs)

	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the first action never opened")
	}
	// Free mode, no economy: the second open closes the first turn — damage and all — and
	// opens the other action.
	sendWS(t, master, "open_next_action", map[string]any{})
	if !awaitCount(masterMsgs, game.MsgTypeTurnOpened, 2, 2*time.Second) {
		t.Fatal("the second action never opened")
	}

	if !masterMsgs.await(game.MsgTypeCharacterHpChanged, 2*time.Second) {
		t.Fatalf("the implicit close applied damage silently; the master received: %v",
			messageTypes(masterMsgs.snapshotMessages()))
	}
	assertVictimHpChange(t, masterMsgs, f, hpBefore, maxHP, "the master")

	if !playerMsgs.await(game.MsgTypeCharacterHpChanged, 2*time.Second) {
		t.Fatalf("the victim's owner was never told; they received: %v",
			messageTypes(playerMsgs.snapshotMessages()))
	}
	assertVictimHpChange(t, playerMsgs, f, hpBefore, maxHP, "the victim's owner")
}
