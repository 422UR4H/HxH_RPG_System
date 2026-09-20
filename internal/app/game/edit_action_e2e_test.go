package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
)

// TestE2E_EditAction drives edit_action over a real socket, against a real Room, using the
// real EditActionUC — the route that, until this test, had never executed once: a non-master
// is refused, the master's edit produces action_edited, and the resolution_updated that
// follows reflects the edit. Shaped like TestE2E_CloseTurn and TestActionQueuedReachesOnlyTheMaster
// exercise their own routes.
func TestE2E_EditAction(t *testing.T) {
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
	if !masterMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("the master never received a resolution_updated for the opened turn")
	}
	baseline := lastResolutionUpdated(t, masterMsgs).Action.Total

	t.Run("a non-master is refused", func(t *testing.T) {
		sendWS(t, player, "edit_action", map[string]any{
			"conditions": []map[string]any{{"field": "hit", "modifier": 5}},
		})
		if code := awaitErrorCode(t, player, 2*time.Second); code != "forbidden" {
			t.Errorf("error code = %q, want forbidden", code)
		}
	})

	t.Run("the master's edit produces action_edited and a recomputed resolution_updated", func(t *testing.T) {
		beforeResolutions := masterMsgs.count(game.MsgTypeResolutionUpdate)

		sendWS(t, master, "edit_action", map[string]any{
			"conditions": []map[string]any{{"field": "hit", "modifier": 5}},
		})
		if !masterMsgs.await(game.MsgTypeActionEdited, 2*time.Second) {
			t.Fatal("the master never received action_edited")
		}
		if !awaitCount(masterMsgs, game.MsgTypeResolutionUpdate, beforeResolutions+1, 2*time.Second) {
			t.Fatal("the master never received a recomputed resolution_updated")
		}

		got := lastResolutionUpdated(t, masterMsgs).Action.Total
		if want := baseline + 5; got != want {
			t.Errorf("Total = %d, want %d (baseline %d + the +5 modifier)", got, want, baseline)
		}
	})
}

// lastResolutionUpdated pulls the most recent resolution_updated payload out of what the
// collector has gathered so far — resolution_updated is republished after every master edit,
// so callers want the latest snapshot, not the first one. Same shape as combat_e2e_test.go's
// lastBarsUpdated/lastTurnOpened.
func lastResolutionUpdated(t *testing.T, c *collector) game.ResolutionUpdatedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeResolutionUpdate {
			continue
		}
		var p game.ResolutionUpdatedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal resolution_updated: %v", err)
		}
		return p
	}
	t.Fatal("no resolution_updated in the collected messages")
	return game.ResolutionUpdatedPayload{}
}
