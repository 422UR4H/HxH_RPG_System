package game_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

// TestE2E_Regency drives the master's edit over a real socket, against a real Room: the
// conditions in force and the damage skill reach the master — live and after a reload — and
// nothing of the edit reaches a player before the turn closes (spec 2026-10-09 §4, §5).
func TestE2E_Regency(t *testing.T) {
	f := newCombatFixture(t)

	master := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	readMessage(t, master) // room_state
	masterMsgs := newCollector(master)
	player := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	defer func() { player.Close() }() //nolint:errcheck // player is swapped by the reconnect below
	readMessage(t, player)            // room_state
	playerMsgs := newCollector(player)

	f.enqueueAttack(t, player)
	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master was never told an action was queued")
	}
	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("no resolution_updated for the opened turn")
	}
	playerResolutionsBefore := playerMsgs.count(game.MsgTypeResolutionUpdate)

	before := masterMsgs.count(game.MsgTypeResolutionUpdate)
	sendWS(t, master, "edit_action", map[string]any{
		"conditions":  []map[string]any{{"field": "hit", "bias": -1, "modifier": -2, "description": "escuridao"}},
		"damageSkill": "Grab",
	})
	if !masterMsgs.await(game.MsgTypeActionEdited, 2*time.Second) {
		t.Fatal("the master never received action_edited")
	}
	if !awaitCount(masterMsgs, game.MsgTypeResolutionUpdate, before+1, 2*time.Second) {
		t.Fatal("the master never received the recomputed resolution_updated")
	}

	assertRegency := func(t *testing.T, p game.ResolutionUpdatedPayload, where string) {
		t.Helper()
		if p.DamageSkill != "Grab" {
			t.Errorf("%s: damageSkill = %q, want Grab", where, p.DamageSkill)
		}
		if len(p.Conditions) != 1 {
			t.Fatalf("%s: conditions = %+v, want the one hit condition", where, p.Conditions)
		}
		c := p.Conditions[0]
		if c.ActionID == uuid.Nil || c.Field != "hit" || c.Bias != -1 || c.Modifier != -2 || c.Description != "escuridao" {
			t.Errorf("%s: conditions[0] = %+v", where, c)
		}
	}

	t.Run("the master sees what is in force", func(t *testing.T) {
		assertRegency(t, lastResolutionUpdated(t, masterMsgs), "resolution_updated")
	})

	t.Run("a player receives nothing of the edit while the turn is open", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond) // let anything misrouted arrive
		if n := playerMsgs.count(game.MsgTypeResolutionUpdate); n != playerResolutionsBefore {
			t.Errorf("the player received %d resolution_updated mid-turn", n-playerResolutionsBefore)
		}
		if n := playerMsgs.count(game.MsgTypeActionEdited); n != 0 {
			t.Errorf("the player received action_edited")
		}
	})

	t.Run("a player who reconnects mid-turn sees nothing of the edit", func(t *testing.T) {
		// The master holds the room while the player reconnects. The new socket REPLACES the old
		// one for the rest of the test: a second socket of the same user takes over its slot, so
		// keeping the first one around would leave the room with no player once either closes.
		player.Close() //nolint:errcheck
		player = connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
		playerMsgs = newCollector(player)
		if !playerMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatal("the reconnecting player never received match_full_state")
		}
		raw := string(findMessage(t, playerMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload)
		for _, leak := range []string{"Grab", "escuridao", "damageSkill", "conditions", "defaultDefense"} {
			if strings.Contains(raw, leak) {
				t.Errorf("match_full_state for a player carries %q: %s", leak, raw)
			}
		}
	})

	t.Run("the master who reloads finds what they had set", func(t *testing.T) {
		master.Close() //nolint:errcheck // the player keeps the room alive
		reloaded := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
		defer reloaded.Close() //nolint:errcheck
		reloadedMsgs := newCollector(reloaded)
		if !reloadedMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatal("the reloading master never received match_full_state")
		}
		var full game.MatchFullStatePayload
		if err := json.Unmarshal(
			findMessage(t, reloadedMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &full,
		); err != nil {
			t.Fatalf("unmarshal match_full_state: %v", err)
		}
		if full.Resolution == nil {
			t.Fatal("the master reloaded into an open turn and lost the calculation")
		}
		assertRegency(t, *full.Resolution, "match_full_state.resolution")

		// Closing the turn: the settled resolution reaches the player with the damage skill
		// and without the master's conditions.
		settledBefore := playerMsgs.count(game.MsgTypeResolutionUpdate)
		sendWS(t, reloaded, "close_turn", map[string]any{})
		if !awaitCount(playerMsgs, game.MsgTypeResolutionUpdate, settledBefore+1, 3*time.Second) {
			t.Fatal("the player never received the settled resolution")
		}
		settled := lastResolutionUpdated(t, playerMsgs)
		if !settled.IsSettled {
			t.Fatal("the resolution the player received is not the settled one")
		}
		if settled.DamageSkill != "Grab" {
			t.Errorf("settled damageSkill = %q, want Grab", settled.DamageSkill)
		}
		if settled.Conditions != nil {
			t.Errorf("settled conditions reached a player: %+v", settled.Conditions)
		}
	})
}
