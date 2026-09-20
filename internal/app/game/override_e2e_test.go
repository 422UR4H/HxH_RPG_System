package game_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

// TestE2E_OverrideReachesPersistTurnClose closes the one gap left in the override criterion:
// the capture is unit-tested (ApplyMasterAction / TestOverrideCapture, internal/application/match)
// and the persistence is integration-tested (TestPersistTurnCloseWritesOverridden, internal/gateway/pg/round),
// but nothing had ever driven edit_action → close_turn over a real Room and asserted on what
// room.go's persistClosedTurn actually handed PersistTurnClose. mockRoundRepoHandler used to
// record only the closed turn's ID — extended here (see overridesFor in handler_test.go) to
// also record d.Overrides, which is the only way to notice a Lock() silently reverted to
// RLock() (TakeOverridesFor would then race, or read stale state) or a dropped
// `Overrides: overrides` field in the TurnCloseData literal: neither breaks any other test in
// this package, because none of them look at d.Overrides at all.
//
// Two turns, two fixtures — a fresh newCombatFixture per sub-test keeps them independent, so
// neither has to reason about the other's round/turn state.
func TestE2E_OverrideReachesPersistTurnClose(t *testing.T) {
	t.Run("a master's edit reaches PersistTurnClose with the ORIGINAL value", func(t *testing.T) {
		f := newCombatFixture(t)
		master, player := f.connect(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		masterMsgs := newCollector(master)

		f.enqueueAttack(t, player)
		if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
			t.Fatal("the master was never told an action was queued")
		}

		sendWS(t, master, "open_next_action", map[string]any{})
		if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
			t.Fatal("the master never received turn_opened")
		}
		turnID := lastTurnOpened(t, masterMsgs).TurnID

		newTarget := uuid.New()
		sendWS(t, master, "edit_action", map[string]any{
			"targetIds": []string{newTarget.String()},
		})
		if !masterMsgs.await(game.MsgTypeActionEdited, 2*time.Second) {
			t.Fatal("the master never received action_edited")
		}

		sendWS(t, master, "close_turn", map[string]any{})
		if !masterMsgs.await(game.MsgTypeTurnClosed, 2*time.Second) {
			t.Fatal("the master never received turn_closed")
		}

		overrides := f.roundRepo.overridesFor(turnID)
		if len(overrides) != 1 {
			t.Fatalf("PersistTurnClose got %d overrides for turn %s, want exactly 1: %+v",
				len(overrides), turnID, overrides)
		}
		got := overrides[0]
		if got.Field != "targetIds" {
			t.Fatalf("Field = %q, want targetIds", got.Field)
		}
		original, ok := got.Original.([]uuid.UUID)
		if !ok {
			t.Fatalf("Original = %#v (%T), want []uuid.UUID", got.Original, got.Original)
		}
		if len(original) != 1 || original[0] != f.victimID {
			t.Fatalf("Original = %v, want [%v] — the target BEFORE the edit, not %v",
				original, f.victimID, newTarget)
		}
	})

	t.Run("an edit back to the original reaches PersistTurnClose with no override for that field", func(t *testing.T) {
		f := newCombatFixture(t)
		master, player := f.connect(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		masterMsgs := newCollector(master)

		f.enqueueAttack(t, player)
		if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
			t.Fatal("the master was never told an action was queued")
		}

		sendWS(t, master, "open_next_action", map[string]any{})
		if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
			t.Fatal("the master never received turn_opened")
		}
		turnID := lastTurnOpened(t, masterMsgs).TurnID

		beforeEdited := masterMsgs.count(game.MsgTypeActionEdited)
		newTarget := uuid.New()
		sendWS(t, master, "edit_action", map[string]any{
			"targetIds": []string{newTarget.String()},
		})
		if !awaitCount(masterMsgs, game.MsgTypeActionEdited, beforeEdited+1, 2*time.Second) {
			t.Fatal("the master never received action_edited for the first edit")
		}

		// Editing back to the original — the cancel verb IS this, per ApplyMasterAction's doc.
		sendWS(t, master, "edit_action", map[string]any{
			"targetIds": []string{f.victimID.String()},
		})
		if !awaitCount(masterMsgs, game.MsgTypeActionEdited, beforeEdited+2, 2*time.Second) {
			t.Fatal("the master never received action_edited for the revert")
		}

		sendWS(t, master, "close_turn", map[string]any{})
		if !masterMsgs.await(game.MsgTypeTurnClosed, 2*time.Second) {
			t.Fatal("the master never received turn_closed")
		}

		overrides := f.roundRepo.overridesFor(turnID)
		for _, ov := range overrides {
			if ov.Field == "targetIds" {
				t.Fatalf("PersistTurnClose got a targetIds override after a revert to the "+
					"original: %+v — the revert should have erased the row, not left one behind",
					ov)
			}
		}
	})
}
