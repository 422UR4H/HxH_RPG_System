package game_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file is B12 (design spec §4.2, §5 "fila" row): match_full_state.ownQueue hands a
// reconnecting NON-master their own queued actions at actionwire.Declaration, so the client
// can reconcile a queue it can never re-observe otherwise (action_queued is master-only, and
// action_enqueued fires once, at enqueue time, same hole match_full_state.queue closed for
// the master in B1). The reconciliation rule itself, and "the client never re-sends on its
// own", live in the contract (match-combat-ws.md) — this file only proves the wire.

// enqueueMoveAttackFrom sends an attack riding a Dash, exactly like
// TestActionQueuedCarriesTheWholeDeclaration (queue_e2e_test.go) — the shape that exercises
// every branch the cut table (design spec §4.1) treats differently at Declaration: weapon
// (declaration, kept everywhere), speed.rollCheck.result and move.finalSpeed (numbers, kept at
// Full/Opened, CUT at Declaration).
func (f *combatFixture) enqueueMoveAttackFrom(
	t *testing.T, conn *websocket.Conn, actorID, targetID uuid.UUID, weapon string, from, to [3]int,
) {
	t.Helper()
	sendWS(t, conn, "enqueue_action", map[string]any{
		"actorId":  actorID.String(),
		"targetId": []string{targetID.String()},
		"speed":    map[string]any{"bar": 0, "rollCheck": map[string]any{"skillName": enum.Legerity.String()}},
		"move": map[string]any{
			"category": string(enum.Dash),
			"from":     from,
			"position": to,
		},
		"attack": map[string]any{
			"weapon": weapon,
			"hit":    map[string]any{"skillName": enum.Accuracy.String()},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})
}

// actionIDsInOrder pulls every actionId out of the collector's action_enqueued messages, in
// arrival order — the order the two enqueue_action calls below were actually acknowledged in,
// which is what "na ordem da fila" (§4.2, B12) has to be checked against.
func actionIDsInOrder(t *testing.T, c *collector) []uuid.UUID {
	t.Helper()
	var ids []uuid.UUID
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeActionEnqueued {
			continue
		}
		var p game.ActionEnqueuedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal action_enqueued: %v", err)
		}
		ids = append(ids, p.ActionID)
	}
	return ids
}

// restartLosingQueue is f.restart (combat_e2e_test.go, T3) with the ONE difference B12 needs:
// a genuinely FRESH *matchsession.MatchSession, built over the same sheets/participants
// newCombatFixture always uses, instead of f.restart's own SAME session pointer.
//
// f.restart's own doc comment is explicit that reusing the pointer is deliberate for B3
// (board/fog persistence) and that "the ONLY thing actually forgotten is the ROOM's own
// in-memory state" — every restart test written against B3 only ever asserts on the board
// store afterwards, never on session state, so that promise never had to cover the queue.
// B12 does: design spec §5's "fila" row says a real server restart loses it ("perdida"),
// because a real restart rebuilds MatchSession from what InitMatchSessionUC actually
// persists — sheets and participants (the board and fog memory separately, via
// SaveMatchBoardUC/LoadBoardUC) — and the queue was never among that; a fresh
// matchsession.NewMatchSession starts with an empty activeQueue, exactly like a real
// restart's would. Kept local to this file rather than widening f.restart's own promise,
// which the B3 tests already pass without it.
func (f *combatFixture) restartLosingQueue(t *testing.T) {
	t.Helper()
	f.server.Close()

	victimPlayer := f.playerUUID
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		f.attackerID: newCombatSheet(t),
		f.victimID:   f.victim,
	}
	participants := []*match.Participant{
		{
			UUID: uuid.New(), MatchUUID: f.matchUUID,
			Sheet: csEntity.Summary{UUID: f.attackerID, PlayerUUID: &f.playerUUID},
		},
		{
			UUID: uuid.New(), MatchUUID: f.matchUUID,
			Sheet: csEntity.Summary{UUID: f.victimID, PlayerUUID: &victimPlayer},
		},
	}
	if f.bystanderUUID != uuid.Nil {
		sheets[f.bystanderID] = newCombatSheet(t)
		participants = append(participants, &match.Participant{
			UUID: uuid.New(), MatchUUID: f.matchUUID,
			Sheet: csEntity.Summary{UUID: f.bystanderID, PlayerUUID: &f.bystanderUUID},
		})
	}

	session := matchsession.NewMatchSession(f.matchUUID, sheets, participants)
	session.SetRollSource(topFaceSource{})
	f.session = session

	hub := game.NewHub()
	go hub.Run()

	roundRepo := &mockRoundRepoHandler{}
	f.roundRepo = roundRepo
	handler := game.NewHandler(
		hub,
		&fogMatchRepo{masterUUID: f.masterUUID, started: !f.lobby},
		&mockEnrollmentChecker{enrolled: true},
		f.roomDeps(session, roundRepo),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)
	f.server = httptest.NewServer(mux)
	t.Cleanup(func() {
		f.server.Close()
		hub.Stop()
	})
}

// TestE2E_ReconnectingOwnerGetsOwnQueuedActionsAtDeclaration is B12's core assertion: the
// owner of two still-queued actions reconnects and finds BOTH of them in match_full_state,
// in the queue's own order, cut to actionwire.Declaration — weapon kept, speed/move numbers
// gone.
func TestE2E_ReconnectingOwnerGetsOwnQueuedActionsAtDeclaration(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck

	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	// Both actors belong to f.playerUUID (see newCombatFixture) — two DIFFERENT characters so
	// the queue holds two genuinely distinct actions, exactly the "duas ações" the brief asks
	// for, without needing a second enqueue for the very same actor.
	f.enqueueMoveAttackFrom(t, player, f.attackerID, f.victimID, "Sword", [3]int{4, 4, 0}, [3]int{6, 4, 0})
	if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, 1, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 1, game.MsgTypeActionEnqueued)
	}
	f.enqueueMoveAttackFrom(t, player, f.victimID, f.attackerID, "Dagger", [3]int{8, 8, 0}, [3]int{7, 8, 0})
	if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, 2, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 2, game.MsgTypeActionEnqueued)
	}
	if !awaitCount(masterMsgs, game.MsgTypeActionQueued, 2, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 2, game.MsgTypeActionQueued)
	}

	wantOrder := actionIDsInOrder(t, playerMsgs)
	if len(wantOrder) != 2 {
		t.Fatalf("got %d action_enqueued acks, want 2: %v", len(wantOrder), wantOrder)
	}
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
	if full.OwnQueue == nil {
		t.Fatal("match_full_state.ownQueue is nil for a non-master — it must always be present")
	}
	own := *full.OwnQueue
	if len(own) != 2 {
		t.Fatalf("match_full_state.ownQueue has %d entries, want 2: %+v", len(own), own)
	}

	t.Run("the order matches the queue's own insertion order", func(t *testing.T) {
		if own[0].ActionID != wantOrder[0] || own[1].ActionID != wantOrder[1] {
			t.Errorf("ownQueue actionIds = [%v, %v], want [%v, %v]",
				own[0].ActionID, own[1].ActionID, wantOrder[0], wantOrder[1])
		}
	})

	t.Run("action.attack.weapon travels — declaration always does", func(t *testing.T) {
		if own[0].Action.Attack == nil || own[0].Action.Attack.Weapon == nil ||
			*own[0].Action.Attack.Weapon != "Sword" {
			t.Errorf("ownQueue[0].action.attack.weapon = %+v, want \"Sword\"", own[0].Action.Attack)
		}
		if own[1].Action.Attack == nil || own[1].Action.Attack.Weapon == nil ||
			*own[1].Action.Attack.Weapon != "Dagger" {
			t.Errorf("ownQueue[1].action.attack.weapon = %+v, want \"Dagger\"", own[1].Action.Attack)
		}
	})

	t.Run("no numbers at all — Declaration cuts speed.rollCheck.result and move.finalSpeed", func(t *testing.T) {
		for i, entry := range own {
			if entry.Action.Speed.RollCheck.Result != nil {
				t.Errorf("ownQueue[%d].action.speed.rollCheck.result = %v, want nil at Declaration",
					i, *entry.Action.Speed.RollCheck.Result)
			}
			if entry.Action.Move == nil {
				t.Fatalf("ownQueue[%d].action.move is nil — declaration (category/from/position) must survive", i)
			}
			if entry.Action.Move.FinalSpeed != nil {
				t.Errorf("ownQueue[%d].action.move.finalSpeed = %v, want nil at Declaration",
					i, *entry.Action.Move.FinalSpeed)
			}
			// attack.hit/damage numbers are cut at Declaration too (they are cut at Opened
			// already) — the skill name still travels.
			if entry.Action.Attack.Hit.Result != nil {
				t.Errorf("ownQueue[%d].action.attack.hit.result = %v, want nil at Declaration",
					i, *entry.Action.Attack.Hit.Result)
			}
		}
	})
}

// TestE2E_BystanderReconnectsToAPresentButEmptyOwnQueue is B12's "sempre presente" rule: a
// non-master with NOTHING of their own queued still gets ownQueue on the wire — `[]`, not an
// absent field — so the front can tell "the server says you have nothing" apart from "an old
// server that never sent this field at all". Checked on the RAW JSON, not the decoded Go
// struct, because *[]T decodes an absent key and an empty array into the same Go zero value
// once dereferenced carelessly — the raw bytes are the only thing that actually distinguishes
// them.
func TestE2E_BystanderReconnectsToAPresentButEmptyOwnQueue(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	bystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	defer bystander.Close()   //nolint:errcheck
	readMessage(t, bystander) // room_state

	playerMsgs := newCollector(player)
	// Someone else's action is in the queue too — proves the filter is by OWNER, not merely
	// "the queue happens to be empty".
	f.enqueueAttack(t, player)
	if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, 1, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 1, game.MsgTypeActionEnqueued)
	}

	bystander.Close() //nolint:errcheck
	lateBystander := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	defer lateBystander.Close()   //nolint:errcheck
	readMessage(t, lateBystander) // room_state
	lateMsgs := collectFrom(lateBystander)
	if !lateMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the reconnecting bystander never received match_full_state")
	}

	raw := findMessage(t, lateMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal match_full_state into a field map: %v", err)
	}
	ownQueueRaw, present := fields["ownQueue"]
	if !present {
		t.Fatalf("match_full_state has no \"ownQueue\" key at all, want it present and empty: %s", raw)
	}
	if string(ownQueueRaw) != "[]" {
		t.Errorf("match_full_state.ownQueue raw JSON = %s, want exactly \"[]\"", ownQueueRaw)
	}
}

// TestE2E_MasterNeverGetsOwnQueue is the other half of the same rule: the master's own
// reconnection snapshot carries `queue` (master-only, B1) and has NO `ownQueue` key at all —
// not `null`, ABSENT — which is exactly why the field is a pointer with omitempty rather than
// a bare slice (a bare nil slice would still serialize to `null`, not an absent key).
func TestE2E_MasterNeverGetsOwnQueue(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer player.Close() //nolint:errcheck

	playerMsgs := newCollector(player)
	f.enqueueAttack(t, player)
	if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, 1, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 1, game.MsgTypeActionEnqueued)
	}

	master.Close() //nolint:errcheck
	lateMaster := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	defer lateMaster.Close()   //nolint:errcheck
	readMessage(t, lateMaster) // room_state
	lateMsgs := collectFrom(lateMaster)
	if !lateMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the reconnecting master never received match_full_state")
	}

	raw := findMessage(t, lateMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal match_full_state into a field map: %v", err)
	}
	if _, present := fields["ownQueue"]; present {
		t.Errorf("match_full_state has an \"ownQueue\" key for the MASTER, want it absent: %s", raw)
	}
	if _, present := fields["queue"]; !present {
		t.Errorf("match_full_state has no \"queue\" key for the master, want the master-only queue present: %s", raw)
	}
}

// TestE2E_OwnQueueExcludesTheActionTheMasterAlreadyOpened is the reconciliation rule's other
// leg (design spec §4.2, B12): once the master opens one of the owner's two queued actions,
// that action is no longer in ownQueue — it is openTurn.actionId instead. A client that only
// consulted ownQueue would wrongly conclude the opened action needs to be resubmitted; the
// contract says the open one is "known" too, just by a different field.
func TestE2E_OwnQueueExcludesTheActionTheMasterAlreadyOpened(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck

	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	f.enqueueMoveAttackFrom(t, player, f.attackerID, f.victimID, "Sword", [3]int{4, 4, 0}, [3]int{6, 4, 0})
	if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, 1, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 1, game.MsgTypeActionEnqueued)
	}
	f.enqueueMoveAttackFrom(t, player, f.victimID, f.attackerID, "Dagger", [3]int{8, 8, 0}, [3]int{7, 8, 0})
	if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, 2, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 2, game.MsgTypeActionEnqueued)
	}
	if !awaitCount(masterMsgs, game.MsgTypeActionQueued, 2, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 2, game.MsgTypeActionQueued)
	}

	both := actionIDsInOrder(t, playerMsgs)
	if len(both) != 2 {
		t.Fatalf("got %d action_enqueued acks, want 2: %v", len(both), both)
	}

	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the turn never opened")
	}
	opened := unmarshalTurnOpened(t, masterMsgs)

	var stillQueued uuid.UUID
	for _, id := range both {
		if id != opened.ActionID {
			stillQueued = id
		}
	}
	if stillQueued == uuid.Nil {
		t.Fatalf("open_next_action opened %v, which is not one of the two enqueued actions %v", opened.ActionID, both)
	}

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
	if full.OpenTurn == nil || full.OpenTurn.ActionID != opened.ActionID {
		t.Fatalf("match_full_state.openTurn.actionId = %+v, want %v", full.OpenTurn, opened.ActionID)
	}
	if full.OwnQueue == nil {
		t.Fatal("match_full_state.ownQueue is nil — it must be present for a non-master")
	}
	own := *full.OwnQueue
	if len(own) != 1 {
		t.Fatalf("match_full_state.ownQueue has %d entries, want 1 (the one NOT opened): %+v", len(own), own)
	}
	if own[0].ActionID != stillQueued {
		t.Errorf("ownQueue[0].actionId = %v, want the still-queued %v", own[0].ActionID, stillQueued)
	}
}

// TestE2E_AServerRestartEmptiesOwnQueue is design spec §5's "fila" row, restart column: the
// queue never survives a real server restart — it is charged against in-memory bars with no
// table behind it (§4.2's "Por que só reconciliar"), so a queue persisted over zeroed bars
// would be a state that never existed. ownQueue therefore comes back `[]`, never the two
// actions that were pending when the server went down, and the front is the one that decides
// what to do about a rescinded draft — the server never re-sends it on its own.
func TestE2E_AServerRestartEmptiesOwnQueue(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)

	playerMsgs := newCollector(player)
	f.enqueueMoveAttackFrom(t, player, f.attackerID, f.victimID, "Sword", [3]int{4, 4, 0}, [3]int{6, 4, 0})
	if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, 1, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 1, game.MsgTypeActionEnqueued)
	}
	f.enqueueMoveAttackFrom(t, player, f.victimID, f.attackerID, "Dagger", [3]int{8, 8, 0}, [3]int{7, 8, 0})
	if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, 2, 2*time.Second) {
		t.Fatalf("never saw %d %s messages", 2, game.MsgTypeActionEnqueued)
	}

	master.Close() //nolint:errcheck
	player.Close() //nolint:errcheck

	f.restartLosingQueue(t)

	// The master reconnects FIRST, exactly as f.connect(t) always orders it: a player is
	// refused with lobby_not_open until the master has opened the (new) room. See f.connect's
	// own doc comment.
	lateMaster := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	defer lateMaster.Close()   //nolint:errcheck
	readMessage(t, lateMaster) // room_state

	lateOwner := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	defer lateOwner.Close()   //nolint:errcheck
	readMessage(t, lateOwner) // room_state
	lateMsgs := collectFrom(lateOwner)
	if !lateMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the reconnecting owner never received match_full_state after the restart")
	}

	raw := findMessage(t, lateMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal match_full_state into a field map: %v", err)
	}
	ownQueueRaw, present := fields["ownQueue"]
	if !present {
		t.Fatalf("match_full_state has no \"ownQueue\" key after the restart, want it present and empty: %s", raw)
	}
	if string(ownQueueRaw) != "[]" {
		t.Errorf("match_full_state.ownQueue raw JSON after the restart = %s, want exactly \"[]\" — "+
			"the queue must not have survived", ownQueueRaw)
	}
}
