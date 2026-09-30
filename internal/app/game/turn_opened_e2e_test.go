package game_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file is B2 (design spec §4.2, §4.1): turn_opened stops being a bare "whose turn is it"
// announcement and starts carrying the action's public mechanics, projected PER RECIPIENT —
// the master gets Full, everyone else (the owner included) gets Opened after
// service.ProjectAction's deny-list has already run. match_full_state.openTurn gets the same
// two fields (actionId, action) so a reconnecting client's snapshot can never disagree with
// the live turn_opened they already received.
//
// It also pins the delivery order the design spec now promises: piece_moved → turn_closed →
// resolution_updated → turn_opened, all on the direct per-client lane, from the same
// goroutine — see TestE2E_TurnClosedNeverArrivesAfterTheNextTurnOpened below.

// enqueueFeintingAttackFrom sends an attack riding a feint — the one shape that exercises
// every branch this file cares about: attack.weapon (declaration, always kept), speed and
// attack numbers (kept at Opened/Full, cut to skill name alone at neither — the cut table's
// two different rows), and feint itself, which is not cut by Level at all but by
// service.ProjectAction's TEMPORAL deny-list (hidden from a third party while the turn is
// open, kept — numbers and all — for the master and the owner).
func (f *combatFixture) enqueueFeintingAttackFrom(t *testing.T, conn *websocket.Conn, actorID, targetID uuid.UUID) {
	t.Helper()
	sendWS(t, conn, "enqueue_action", map[string]any{
		"actorId":  actorID.String(),
		"targetId": []string{targetID.String()},
		"speed":    map[string]any{"bar": 0, "rollCheck": map[string]any{"skillName": enum.Legerity.String()}},
		"feint":    map[string]any{"skillName": enum.Feint.String()},
		"attack": map[string]any{
			"weapon": "Sword",
			"hit":    map[string]any{"skillName": enum.Accuracy.String()},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})
}

// TestE2E_TurnOpenedProjectsTheActionPerRecipient is B2's core assertion (design spec §4.2,
// cut table §4.1): the SAME turn_opened event carries a different cut of the SAME action to
// each of the three viewer classes, and match_full_state.openTurn agrees with whichever cut
// its recipient already got live.
func TestE2E_TurnOpenedProjectsTheActionPerRecipient(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	defer bystander.Close()   //nolint:errcheck
	readMessage(t, bystander) // room_state

	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)
	bystanderMsgs := newCollector(bystander)

	// f.playerUUID owns BOTH the attacker and the victim (see newCombatFixture) — so the
	// player connection below IS the actor's owner, exactly the "dono" class the brief and
	// the cut table name explicitly (as opposed to the master, and as opposed to the
	// bystander, who owns neither character at this table).
	f.enqueueFeintingAttackFrom(t, player, f.attackerID, f.victimID)

	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master never saw the action land in the queue")
	}

	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the turn never opened for the master")
	}
	if !playerMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the turn never opened for the owner")
	}
	if !bystanderMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the turn never opened for the bystander")
	}

	masterOpened := unmarshalTurnOpened(t, masterMsgs)
	ownerOpened := unmarshalTurnOpened(t, playerMsgs)
	bystanderOpened := unmarshalTurnOpened(t, bystanderMsgs)

	t.Run("the master gets Full — every number, including the closed ones", func(t *testing.T) {
		a := masterOpened.Action
		if a.Attack == nil || a.Attack.Hit.Result == nil {
			t.Error("action.attack.hit.result is nil — the master must see Full's numbers")
		}
		if a.Attack.Damage.Result == nil {
			t.Error("action.attack.damage.result is nil — the master must see Full's numbers")
		}
		if a.Feint == nil || a.Feint.Result == nil {
			t.Error("action.feint(.result) is nil — the master sees the feint's numbers too")
		}
	})

	t.Run("the owner gets Opened — mechanics and speed, no attack numbers, feint with no numbers", func(t *testing.T) {
		a := ownerOpened.Action
		if a.Attack == nil || a.Attack.Weapon == nil || *a.Attack.Weapon != "Sword" {
			t.Fatalf("action.attack.weapon = %+v, want \"Sword\"", a.Attack)
		}
		if len(a.TargetID) != 1 || a.TargetID[0] != f.victimID {
			t.Errorf("action.targetId = %v, want [%v]", a.TargetID, f.victimID)
		}
		if a.Speed.RollCheck.Result == nil {
			t.Error("action.speed.rollCheck.result is nil — Opened must keep the speed number")
		}
		if a.Attack.Hit.Result != nil {
			t.Error("action.attack.hit.result is non-nil — Opened must cut it")
		}
		if a.Attack.Damage.Result != nil {
			t.Error("action.attack.damage.result is non-nil — Opened must cut it")
		}
		if a.Feint == nil {
			t.Fatal("action.feint is nil — the owner sees all of the actor, ProjectAction " +
				"must not hide it from them even with the turn open")
		}
		if a.Feint.SkillName != enum.Feint.String() {
			t.Errorf("action.feint.skillName = %q, want %q", a.Feint.SkillName, enum.Feint.String())
		}
		if a.Feint.Result != nil {
			t.Error("action.feint.result is non-nil — Opened must cut the feint's own numbers " +
				"even for the owner")
		}
	})

	t.Run("the bystander gets the same Opened cut, minus the feint entirely", func(t *testing.T) {
		a := bystanderOpened.Action
		if a.Attack == nil || a.Attack.Weapon == nil || *a.Attack.Weapon != "Sword" {
			t.Fatalf("action.attack.weapon = %+v, want \"Sword\"", a.Attack)
		}
		if len(a.TargetID) != 1 || a.TargetID[0] != f.victimID {
			t.Errorf("action.targetId = %v, want [%v]", a.TargetID, f.victimID)
		}
		if a.Speed.RollCheck.Result == nil {
			t.Error("action.speed.rollCheck.result is nil — Opened must keep the speed number " +
				"for a bystander too")
		}
		if a.Attack.Hit.Result != nil {
			t.Error("action.attack.hit.result is non-nil — Opened must cut it")
		}
		if a.Attack.Damage.Result != nil {
			t.Error("action.attack.damage.result is non-nil — Opened must cut it")
		}
		if a.Feint != nil {
			t.Errorf("action.feint = %+v, want nil — the deny-list hides a feint from a third "+
				"party while the turn is open", a.Feint)
		}
	})

	t.Run("a reconnecting owner's match_full_state.openTurn matches their own live turn_opened", func(t *testing.T) {
		player.Close() //nolint:errcheck
		lateOwner := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
		defer lateOwner.Close()   //nolint:errcheck
		readMessage(t, lateOwner) // room_state
		lateMsgs := collectFrom(lateOwner)
		if !lateMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatal("the reconnecting owner never received match_full_state")
		}
		var full game.MatchFullStatePayload
		if err := json.Unmarshal(
			findMessage(t, lateMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &full,
		); err != nil {
			t.Fatalf("unmarshal match_full_state: %v", err)
		}
		if full.OpenTurn == nil {
			t.Fatal("match_full_state.openTurn is nil — the turn is still open")
		}
		if full.OpenTurn.ActionID != ownerOpened.ActionID {
			t.Errorf("openTurn.actionId = %v, want the live turn_opened's %v",
				full.OpenTurn.ActionID, ownerOpened.ActionID)
		}
		assertSameActionJSON(t, ownerOpened.Action, full.OpenTurn.Action,
			"match_full_state.openTurn.action", "the owner's live turn_opened.action")
	})
}

// unmarshalTurnOpened pulls the most recent turn_opened out of a collector.
func unmarshalTurnOpened(t *testing.T, c *collector) game.TurnOpenedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeTurnOpened {
			continue
		}
		var p game.TurnOpenedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal turn_opened: %v", err)
		}
		return p
	}
	t.Fatal("no turn_opened in the collected messages")
	return game.TurnOpenedPayload{}
}

// assertSameActionJSON compares two actionwire.Action values by their WIRE shape, not by the
// decoded Go structs — two *int fields pointing at equal values are already DeepEqual on the
// structs, which would prove less than this helper claims to. Round-tripping through a
// generic map is what actually asserts the same keys and the same omissions, the same way
// queue_e2e_test.go's B1 test already does for action_queued vs match_full_state.queue.
func assertSameActionJSON(t *testing.T, live, snapshot any, snapshotLabel, liveLabel string) {
	t.Helper()
	liveJSON, err := json.Marshal(live)
	if err != nil {
		t.Fatalf("marshal %s: %v", liveLabel, err)
	}
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal %s: %v", snapshotLabel, err)
	}
	var liveGeneric, snapshotGeneric any
	if err := json.Unmarshal(liveJSON, &liveGeneric); err != nil {
		t.Fatalf("unmarshal live JSON into generic: %v", err)
	}
	if err := json.Unmarshal(snapshotJSON, &snapshotGeneric); err != nil {
		t.Fatalf("unmarshal snapshot JSON into generic: %v", err)
	}
	if !reflect.DeepEqual(liveGeneric, snapshotGeneric) {
		t.Errorf("%s != %s\nlive:     %s\nsnapshot: %s", snapshotLabel, liveLabel, liveJSON, snapshotJSON)
	}
}

// TestE2E_TurnClosedNeverArrivesAfterTheNextTurnOpened is B2's ordering guarantee (design spec
// §4.2 ⚠️): piece_moved → turn_closed → resolution_updated → turn_opened, all on the direct
// per-client lane, from the same goroutine, so send order IS arrival order — no channel hop
// in the middle to let a fast direct message overtake a slow broadcast one.
//
// The bystander is the recipient that matters here: the master and the owner both already
// exercise this window in other tests (assertClosedBeforeOpened, turn_lifecycle_e2e_test.go),
// but a genuine THIRD viewer, with no ownership shortcut anywhere in the path, is what proves
// the guarantee holds for "the table", not just for a recipient the dispatch might special-case.
//
// 50 repetitions, a fresh fixture (and a fresh goroutine schedule) each time: a single run
// cannot tell a correct fix (deterministic — same lane, same goroutine, no race possible)
// apart from a half-fix that merely usually wins the race (turn_opened on the direct lane,
// turn_closed left on the slower channel lane) — see this task's report for the RED run that
// demonstrates the half-fix failing here.
func TestE2E_TurnClosedNeverArrivesAfterTheNextTurnOpened(t *testing.T) {
	for i := 0; i < 50; i++ {
		f := newCombatFixture(t, withBystander)
		master, player := f.connect(t)
		bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
		readMessage(t, bystander) // room_state

		masterMsgs := newCollector(master)
		bystanderMsgs := newCollector(bystander)

		queued := enqueueTwoFromTheSameActor(t, f, player, masterMsgs)
		_ = queued

		sendWS(t, master, "open_next_action", map[string]any{})
		if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
			t.Fatalf("iteration %d: the first action never opened", i)
		}
		firstTurn := lastTurnOpened(t, masterMsgs).TurnID

		sendWS(t, master, "open_next_action", map[string]any{})
		if !awaitCount(masterMsgs, game.MsgTypeTurnOpened, 2, 2*time.Second) {
			t.Fatalf("iteration %d: the second action never opened", i)
		}
		secondTurn := lastTurnOpened(t, masterMsgs).TurnID

		if !awaitCount(bystanderMsgs, game.MsgTypeTurnOpened, 2, 2*time.Second) {
			t.Fatalf("iteration %d: the bystander never saw both turns open; it received: %v",
				i, messageTypes(bystanderMsgs.snapshotMessages()))
		}

		msgs := bystanderMsgs.snapshotMessages()
		closedAt := indexOfTurnMessage(t, msgs, game.MsgTypeTurnClosed, firstTurn)
		if closedAt < 0 {
			t.Fatalf("iteration %d: the bystander never got turn_closed for the first turn; "+
				"it received: %v", i, messageTypes(msgs))
		}
		openedAt := indexOfTurnMessage(t, msgs, game.MsgTypeTurnOpened, secondTurn)
		if openedAt < 0 {
			t.Fatalf("iteration %d: the bystander never got turn_opened for the second turn", i)
		}
		if closedAt > openedAt {
			t.Fatalf("iteration %d: the bystander saw the second turn_opened (position %d) "+
				"BEFORE the first turn_closed (position %d) — the two lanes raced",
				i, openedAt, closedAt)
		}

		master.Close()    //nolint:errcheck
		player.Close()    //nolint:errcheck
		bystander.Close() //nolint:errcheck
	}
}
