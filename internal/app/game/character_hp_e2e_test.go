package game_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
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

// ─── one master command, one transaction (owner decision, 2026-10-02) ───────

// TestE2E_ARestartKeepsTheHPTheTurnCloseWrote: the HP a close applied is written in the turn's
// own transaction (TurnCloseData.StatusBars), so a restart after the close starts the victim at
// exactly that HP — not at the full bar a fresh sheet is built with.
func TestE2E_ARestartKeepsTheHPTheTurnCloseWrote(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	mc := collectFrom(master)
	hpBefore, _ := victimHealthBar(t, f)

	turnID := f.openAttackTurn(t, master, player, mc)
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	f.awaitPersistedTurn(t, turnID)
	// Safe now: persisting is behind the close's last mutation of the sheet.
	hpAfter := f.victimHP(t)
	if hpAfter >= hpBefore {
		t.Fatalf("victim HP %d -> %d: the close applied no damage, so this test proves nothing", hpBefore, hpAfter)
	}

	var bars []appmatch.SheetStatusBars
	for _, d := range f.roundRepo.closeData() {
		if d.Turn.GetID() == turnID {
			bars = d.StatusBars
		}
	}
	if len(bars) != 1 || bars[0].CharacterID != f.victimID || bars[0].Health.GetCurrent() != hpAfter {
		t.Fatalf("PersistTurnClose got status bars %+v, want exactly the victim %s at %d HP", bars, f.victimID, hpAfter)
	}

	master.Close() //nolint:errcheck
	player.Close() //nolint:errcheck
	f.restart(t)
	if got := f.victimHP(t); got != hpAfter {
		t.Fatalf("after the restart the victim has %d HP, want the %d the close wrote", got, hpAfter)
	}
}

// TestE2E_AFailedTurnCloseLeavesTheSheetAsItWas: the HP is durable with its turn or not at all.
// A PersistTurnClose that fails writes no sheet — the table still saw the damage live (it is in
// memory), but a restart starts the victim where the last written close left it.
func TestE2E_AFailedTurnCloseLeavesTheSheetAsItWas(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	mc := collectFrom(master)
	hpBefore, _ := victimHealthBar(t, f)

	turnID := f.openAttackTurn(t, master, player, mc)
	f.roundRepo.setFailPersist(errors.New("the transaction rolled back"))
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	// turn_closed goes out after the write attempt: by then the close is over, written or not.
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed")
	}
	if !mc.await(game.MsgTypeCharacterHpChanged, 2*time.Second) {
		t.Fatal("the table was never told the HP moved — the damage must still apply in memory")
	}
	if contains(f.roundRepo.persistedTurnIDs(), turnID) {
		t.Fatal("the failing repository reports the turn persisted — the fake is wrong")
	}
	if rows, hps := f.writer.snapshot(); len(rows) != 0 {
		t.Fatalf("character_sheets got %v (HP %v) although the turn's transaction failed", rows, hps)
	}

	master.Close() //nolint:errcheck
	player.Close() //nolint:errcheck
	f.restart(t)
	if got := f.victimHP(t); got != hpBefore {
		t.Fatalf("after the restart the victim has %d HP, want the untouched %d", got, hpBefore)
	}
}

// statusBarsOf returns the status bars the successful PersistTurnClose of one turn carried.
func statusBarsOf(f *combatFixture, turnID uuid.UUID) []appmatch.SheetStatusBars {
	for _, d := range f.roundRepo.closeData() {
		if d.Turn.GetID() == turnID {
			return d.StatusBars
		}
	}
	return nil
}

// TestE2E_TheNextCloseWritesTheSheetAFailedCloseLeft: a failed close's HP is not lost until a
// restart — the sheet is remembered as unwritten, and the NEXT close writes it, even when that
// close damages a different character (controller ruling on Fix C). Only the sheets a close
// touched are ever written, never every sheet: that would clobber a REST edit made meanwhile.
func TestE2E_TheNextCloseWritesTheSheetAFailedCloseLeft(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	pc := collectFrom(player)

	// Close 1: the attacker hits the victim, and the turn's transaction fails.
	turn1 := f.openAttackTurn(t, master, player, mc)
	f.roundRepo.setFailPersist(errors.New("the transaction rolled back"))
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("turn 1 never closed")
	}
	if contains(f.roundRepo.persistedTurnIDs(), turn1) {
		t.Fatal("the failing repository reports turn 1 persisted — the fake is wrong")
	}
	f.roundRepo.setFailPersist(nil)

	// Close 2: the victim hits the attacker — a different character — and the write succeeds.
	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId":  f.victimID.String(),
		"targetId": []string{f.attackerID.String()},
		"speed":    map[string]any{"bar": 0, "rollCheck": map[string]any{"skillName": enum.Legerity.String()}},
		"attack": map[string]any{
			"weapon": "Sword",
			"hit":    map[string]any{"skillName": enum.Accuracy.String()},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})
	// The count, not the type: the first attack's acknowledgement is already in the collector.
	if !awaitCount(pc, game.MsgTypeActionEnqueued, 2, 2*time.Second) {
		t.Fatal("the second attack was never enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !awaitCount(mc, game.MsgTypeTurnOpened, 2, 2*time.Second) {
		t.Fatalf("the second attack never opened; master got %v", messageTypes(mc.snapshotMessages()))
	}
	turn2 := lastTurnOpened(t, mc).TurnID
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	f.awaitPersistedTurn(t, turn2)

	victimHP := f.victimHP(t) // safe: turn 2's write is behind its last mutation of the sheets
	written := map[uuid.UUID]int{}
	for _, sb := range statusBarsOf(f, turn2) {
		written[sb.CharacterID] = sb.Health.GetCurrent()
	}
	if len(written) != 2 {
		t.Fatalf("turn 2's PersistTurnClose wrote %v, want both the attacker it damaged and the victim turn 1 left unwritten", written)
	}
	if hp, ok := written[f.victimID]; !ok || hp != victimHP {
		t.Fatalf("victim written at %d (present %v), want the %d turn 1 applied", hp, ok, victimHP)
	}

	// Close 3: the victim hits the attacker again, so only the attacker is damaged. The victim is
	// not written — turn 2 wrote it, so it is no longer unwritten.
	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId":  f.victimID.String(),
		"targetId": []string{f.attackerID.String()},
		"speed":    map[string]any{"bar": 0, "rollCheck": map[string]any{"skillName": enum.Legerity.String()}},
		"attack": map[string]any{
			"weapon": "Sword",
			"hit":    map[string]any{"skillName": enum.Accuracy.String()},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})
	if !awaitCount(pc, game.MsgTypeActionEnqueued, 3, 2*time.Second) {
		t.Fatal("the third attack was never enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !awaitCount(mc, game.MsgTypeTurnOpened, 3, 2*time.Second) {
		t.Fatal("the third attack never opened")
	}
	turn3 := lastTurnOpened(t, mc).TurnID
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	f.awaitPersistedTurn(t, turn3)
	bars3 := statusBarsOf(f, turn3)
	if len(bars3) != 1 || bars3[0].CharacterID != f.attackerID {
		t.Fatalf("turn 3 wrote %+v, want only the attacker it damaged — the victim was written by turn 2", bars3)
	}
}

// TestE2E_TheHalfSuccessCloseWritesItsHP: an open_next_action that closes a turn and then fails
// to open the next one (empty Free queue) still hands that turn's HP to its PersistTurnClose —
// the use case returns the result alongside the error, and the room persists the closed half.
func TestE2E_TheHalfSuccessCloseWritesItsHP(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	hpBefore, _ := victimHealthBar(t, f)

	turnID := f.openAttackTurn(t, master, player, mc)
	// Nothing else queued: this closes the turn, applies its damage, and fails to open anything.
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	f.awaitPersistedTurn(t, turnID)

	hpAfter := f.victimHP(t)
	if hpAfter >= hpBefore {
		t.Fatalf("victim HP %d -> %d: the close applied no damage, so this test proves nothing", hpBefore, hpAfter)
	}
	bars := statusBarsOf(f, turnID)
	if len(bars) != 1 || bars[0].CharacterID != f.victimID || bars[0].Health.GetCurrent() != hpAfter {
		t.Fatalf("the half-success close's PersistTurnClose got %+v, want the victim at %d HP", bars, hpAfter)
	}
}
