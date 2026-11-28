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

		// A fixture tira toda face no topo: o acerto é crítico, e o crítico dobra todo número
		// somado ao teste — o +5 vale 10 no total.
		got := lastResolutionUpdated(t, masterMsgs).Action.Total
		if want := baseline + 2*5; got != want {
			t.Errorf("Total = %d, want %d (critical baseline %d + the +5 modifier, doubled)", got, want, baseline)
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

// TestE2E_EditQueuedAction drives edit_action on an action still in the QUEUE (owner decision,
// 2026-10-12): the master gets action_edited, then the action_queued that re-describes the same
// action with the condition in force, and the public bars; the player who
// declared it gets none of it.
func TestE2E_EditQueuedAction(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs, playerMsgs := newCollector(master), newCollector(player)

	f.enqueueAttack(t, player)
	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master was never told an action was queued")
	}
	queued := lastActionQueued(t, masterMsgs)
	if len(queued.Conditions) != 0 || queued.DamageSkill != "Force" {
		t.Fatalf("fresh action_queued = %+v, want no condition and the default Force", queued)
	}

	t.Run("its targets are not editable on the queue", func(t *testing.T) {
		sendWS(t, master, "edit_action", map[string]any{
			"actionId": queued.ActionID.String(), "targetIds": []string{f.attackerID.String()},
		})
		if !masterMsgs.await(game.MsgTypeError, 2*time.Second) {
			t.Fatal("the master never got the refusal")
		}
	})

	t.Run("a condition lands, and the master sees it", func(t *testing.T) {
		beforeQueued := masterMsgs.count(game.MsgTypeActionQueued)
		beforeBars := masterMsgs.count(game.MsgTypeBarsUpdated)
		sendWS(t, master, "edit_action", map[string]any{
			"actionId":   queued.ActionID.String(),
			"conditions": []map[string]any{{"field": "hit", "modifier": 3, "description": "mira"}},
		})
		if !masterMsgs.await(game.MsgTypeActionEdited, 2*time.Second) {
			t.Fatal("the master never received action_edited")
		}
		if !awaitCount(masterMsgs, game.MsgTypeActionQueued, beforeQueued+1, 2*time.Second) {
			t.Fatal("the master never received the action_queued that re-describes the action")
		}
		if !awaitCount(masterMsgs, game.MsgTypeBarsUpdated, beforeBars+1, 2*time.Second) {
			t.Fatal("the bars never went out after the edit")
		}
		got := lastActionQueued(t, masterMsgs)
		if got.ActionID != queued.ActionID || got.DamageSkill != "Force" {
			t.Fatalf("action_queued = %+v, want the same action, still measured by Force", got)
		}
		if len(got.Conditions) != 1 || got.Conditions[0].Field != "hit" || got.Conditions[0].Modifier != 3 ||
			got.Conditions[0].ActionID != queued.ActionID {
			t.Fatalf("conditions = %+v, want the master's +3 on the hit", got.Conditions)
		}
	})

	t.Run("the player who declared it sees none of it", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond)
		if n := playerMsgs.count(game.MsgTypeActionQueued) + playerMsgs.count(game.MsgTypeActionEdited); n != 0 {
			t.Fatalf("the player got %d queue/edit messages, want 0", n)
		}
	})
}

func lastActionQueued(t *testing.T, c *collector) game.ActionQueuedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeActionQueued {
			continue
		}
		var p game.ActionQueuedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal action_queued: %v", err)
		}
		return p
	}
	t.Fatal("no action_queued in the collected messages")
	return game.ActionQueuedPayload{}
}

// TestEditActionRefusesDamageSkill: Force always measures the damage, so an old client that
// still sends edit_action.damageSkill gets a clear refusal — never a silent half-applied edit.
func TestEditActionRefusesDamageSkill(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := newCollector(master)

	f.enqueueAttack(t, player)
	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master was never told an action was queued")
	}
	queued := lastActionQueued(t, masterMsgs)
	editedBefore := masterMsgs.count(game.MsgTypeActionEdited)

	// the condition rides along: a refusal must not apply it either.
	sendWS(t, master, "edit_action", map[string]any{
		"actionId":    queued.ActionID.String(),
		"conditions":  []map[string]any{{"field": "hit", "modifier": 3}},
		"damageSkill": "Grab",
	})
	if !masterMsgs.await(game.MsgTypeError, 2*time.Second) {
		t.Fatal("the master never got the refusal")
	}
	var code string
	for _, m := range masterMsgs.snapshotMessages() {
		if m.Type != game.MsgTypeError {
			continue
		}
		var p game.ErrorPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal error payload: %v", err)
		}
		code = p.Code
	}
	if code != "invalid_payload" {
		t.Errorf("error code = %q, want invalid_payload", code)
	}
	time.Sleep(100 * time.Millisecond)
	if n := masterMsgs.count(game.MsgTypeActionEdited); n != editedBefore {
		t.Errorf("action_edited arrived %d time(s) after a refused edit", n-editedBefore)
	}
	if got := lastActionQueued(t, masterMsgs); got.DamageSkill != "Force" || len(got.Conditions) != 0 {
		t.Errorf("action_queued = %+v, want Force and no condition", got)
	}
}
