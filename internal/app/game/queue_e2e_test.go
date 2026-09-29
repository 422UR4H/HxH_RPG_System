package game_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// TestActionQueuedCarriesTheWholeDeclaration is B1 (design spec §4.2): action_queued and
// match_full_state.queue now carry the WHOLE declaration — the actionwire.Action, at Full —
// not just the scheduling triple (actionId/actorId/bars) they carried before. Both surfaces
// are fed by the SAME newActionQueuedPayload (room.go), so what a reconnecting master learns
// from match_full_state.queue must be byte-for-byte identical to what action_queued already
// told them live; that identity is this test's real payload, not a side note.
//
// The player enqueues an attack riding a Dash — one action carrying both Move and Attack — so
// the assertions below exercise every branch of the design spec's cut table that Full keeps:
// attack.weapon (declaration, always kept), attack.hit.result (a number, Full-only — Opened
// would nil it), speed.rollCheck.result (a number, Full-only), move.finalSpeed (a number, kept
// at Full AND Opened — nil only at Declaration), move.position (declaration) and targetId
// (declaration). attack.hit.result together with move.finalSpeed is what proves Full
// specifically, not merely Opened: Opened alone would still leave move.finalSpeed non-nil but
// would nil attack.hit.result.
func TestActionQueuedCarriesTheWholeDeclaration(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	from, to := [3]int{4, 4, 0}, [3]int{6, 4, 0}
	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId":  f.attackerID.String(),
		"targetId": []string{f.victimID.String()},
		"speed":    map[string]any{"bar": 0, "rollCheck": map[string]any{"skillName": enum.Legerity.String()}},
		"move": map[string]any{
			"category": string(enum.Dash),
			"from":     from,
			"position": to,
		},
		"attack": map[string]any{
			"weapon": "Sword",
			"hit":    map[string]any{"skillName": enum.Accuracy.String()},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})

	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}
	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master never saw the action land in the queue")
	}

	var queued game.ActionQueuedPayload
	if err := json.Unmarshal(
		findMessage(t, masterMsgs.snapshotMessages(), game.MsgTypeActionQueued).Payload, &queued,
	); err != nil {
		t.Fatalf("unmarshal action_queued: %v", err)
	}

	t.Run("the master's live action_queued carries the whole declaration", func(t *testing.T) {
		if queued.Action.Attack == nil || queued.Action.Attack.Weapon == nil ||
			*queued.Action.Attack.Weapon != "Sword" {
			t.Fatalf("action.attack.weapon = %+v, want \"Sword\"", queued.Action.Attack)
		}
		if queued.Action.Attack.Hit.Result == nil {
			t.Error("action.attack.hit.result is nil — Full must keep the number")
		}
		if queued.Action.Speed.RollCheck.Result == nil {
			t.Error("action.speed.rollCheck.result is nil — Full must keep the number")
		}
		if queued.Action.Move == nil || queued.Action.Move.FinalSpeed == nil {
			t.Error("action.move.finalSpeed is nil — Full/Opened must keep the derived speed")
		}
		if queued.Action.Move == nil || queued.Action.Move.Position != to {
			t.Errorf("action.move.position = %+v, want %v", queued.Action.Move, to)
		}
		if len(queued.Action.TargetID) != 1 || queued.Action.TargetID[0] != f.victimID {
			t.Errorf("action.targetId = %v, want [%v]", queued.Action.TargetID, f.victimID)
		}
	})

	// B1 ⚠️ guard: this rule already existed (the queue is secret — combat-engine.md § As
	// barras são públicas), and this task must not weaken it just because the payload grew.
	t.Run("the player who enqueued it never receives action_queued — the queue stays secret", func(t *testing.T) {
		if n := playerMsgs.count(game.MsgTypeActionQueued); n != 0 {
			t.Errorf("the enqueuing player received %d action_queued messages, want 0", n)
		}
	})

	t.Run("a reconnecting master's match_full_state.queue[0].action equals the live action_queued.action", func(t *testing.T) {
		master.Close() //nolint:errcheck
		lateMaster := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
		defer lateMaster.Close() //nolint:errcheck
		lateMasterMsgs := collectFrom(lateMaster)
		if !lateMasterMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatal("the reconnecting master never received match_full_state")
		}

		var full game.MatchFullStatePayload
		if err := json.Unmarshal(
			findMessage(t, lateMasterMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &full,
		); err != nil {
			t.Fatalf("unmarshal match_full_state: %v", err)
		}
		if len(full.Queue) != 1 {
			t.Fatalf("match_full_state.queue has %d entries, want 1: %+v", len(full.Queue), full.Queue)
		}

		// Compare the JSON, not the decoded Go structs: two *int fields pointing at equal
		// values are already equal to reflect.DeepEqual on the structs, so decoding straight
		// into actionwire.Action would prove less than this test claims to. Round-tripping
		// through a generic map is what actually asserts the WIRE shape — same keys, same
		// omissions — is identical between the two surfaces, which is B1's actual promise.
		liveJSON, err := json.Marshal(queued.Action)
		if err != nil {
			t.Fatalf("marshal the live action: %v", err)
		}
		snapshotJSON, err := json.Marshal(full.Queue[0].Action)
		if err != nil {
			t.Fatalf("marshal the snapshot's action: %v", err)
		}
		var liveGeneric, snapshotGeneric any
		if err := json.Unmarshal(liveJSON, &liveGeneric); err != nil {
			t.Fatalf("unmarshal live JSON into generic: %v", err)
		}
		if err := json.Unmarshal(snapshotJSON, &snapshotGeneric); err != nil {
			t.Fatalf("unmarshal snapshot JSON into generic: %v", err)
		}
		if !reflect.DeepEqual(liveGeneric, snapshotGeneric) {
			t.Errorf(
				"match_full_state.queue[0].action != the live action_queued.action\nlive:     %s\nsnapshot: %s",
				liveJSON, snapshotJSON,
			)
		}
	})
}
