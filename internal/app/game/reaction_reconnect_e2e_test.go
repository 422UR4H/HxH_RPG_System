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

// This file is the Phase 7 back package's reconnect half (spec §4.1, §4.2, §5, decisions D5/D6):
// match_full_state hands back what reaction_opened and reaction_attached said live, to a client
// that was not there to hear it. openTurn.reactions is the table's — the OPENED reactions, in
// opening order, each cut as the live reaction_opened cut it. ownReactions is the recipient's
// own — the master's through their NPCs — opened or not, with the true kind and what it consumed.

// fullStateFrom reads the match_full_state a collector holds, both decoded and as raw top-level
// keys — "is this key on the wire at all" is asked of the raw (D6: omitempty).
func fullStateFrom(t *testing.T, who string, c *collector) (game.MatchFullStatePayload, map[string]json.RawMessage) {
	t.Helper()
	if !c.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatalf("%s never received match_full_state", who)
	}
	msg := findMessage(t, c.snapshotMessages(), game.MsgTypeMatchFullState)
	var full game.MatchFullStatePayload
	if err := json.Unmarshal(msg.Payload, &full); err != nil {
		t.Fatalf("%s: unmarshal match_full_state: %v", who, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(msg.Payload, &raw); err != nil {
		t.Fatalf("%s: unmarshal match_full_state keys: %v", who, err)
	}
	return full, raw
}

// reconnectFullState drops a connection and opens a fresh one for the same user, returning it
// with the match_full_state it was greeted with.
func reconnectFullState(
	t *testing.T, who string, old *websocket.Conn, serverURL string, userUUID, matchUUID uuid.UUID,
) (*websocket.Conn, game.MatchFullStatePayload, map[string]json.RawMessage) {
	t.Helper()
	old.Close() //nolint:errcheck
	conn := connectWS(t, serverURL, userUUID, matchUUID)
	readMessage(t, conn) // room_state
	full, raw := fullStateFrom(t, who, collectFrom(conn))
	return conn, full, raw
}

func TestE2E_ReconnectingReactorGetsOwnReactions(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	// The Task 3 shape: the victim has an action waiting in the queue, and the charged repel
	// consumes it.
	actionID := f.openAttackOn(t, player, master, mc)
	queued := mc.count(game.MsgTypeActionQueued)
	f.enqueueAttackFrom(t, player, f.victimID)
	if !awaitCount(mc, game.MsgTypeActionQueued, queued+1, 2*time.Second) {
		t.Fatal("the victim's action never reached the queue")
	}
	victimActionID := lastActionQueuedID(t, mc)
	sword := "Sword"
	sendWS(t, player, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.victimID, ReactToID: actionID, ReactionKind: "repel",
		Repel: &game.RepelPayload{Weapon: &sword},
	})
	if !pc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatalf("the reactor never got reaction_attached; got %v", messageTypes(pc.snapshotMessages()))
	}
	reactionID := lastReactionAttached(t, pc).ReactionID

	// requireOwn checks the one entry ownReactions must carry, with the given opened flag.
	requireOwn := func(t *testing.T, full game.MatchFullStatePayload, wantOpened bool) {
		t.Helper()
		if len(full.OwnReactions) != 1 {
			t.Fatalf("ownReactions = %+v, want exactly the repel", full.OwnReactions)
		}
		got := full.OwnReactions[0]
		if got.ReactionID != reactionID || got.ActorID != f.victimID || got.ReactionKind != "repel" {
			t.Fatalf("ownReactions[0] = %+v, want the repel %s by %s", got, reactionID, f.victimID)
		}
		if got.Opened != wantOpened {
			t.Fatalf("ownReactions[0].opened = %v, want %v", got.Opened, wantOpened)
		}
		if len(got.ConsumedActionIDs) != 1 || got.ConsumedActionIDs[0] != victimActionID {
			t.Fatalf("ownReactions[0].consumedActionIds = %v, want [%v]", got.ConsumedActionIDs, victimActionID)
		}
	}

	t.Run("attached, not opened: the owner gets it back, and the table has nothing to show", func(t *testing.T) {
		conn, full, _ := reconnectFullState(t, "the reactor", player, f.server.URL, f.playerUUID, f.matchUUID)
		player = conn
		requireOwn(t, full, false)
		if full.OpenTurn == nil {
			t.Fatal("openTurn is absent with the attack still open")
		}
		if len(full.OpenTurn.Reactions) != 0 {
			t.Fatalf("openTurn.reactions = %+v, want none — the repel was never announced", full.OpenTurn.Reactions)
		}
	})

	sendWS(t, master, string(game.MsgTypeOpenReaction), game.OpenReactionPayload{ReactionID: reactionID})
	if !mc.await(game.MsgTypeReactionOpened, 2*time.Second) {
		t.Fatalf("the reaction never opened; the master received: %v", messageTypes(mc.snapshotMessages()))
	}

	t.Run("once opened: opened is true, and the table's openTurn.reactions names it", func(t *testing.T) {
		conn, full, _ := reconnectFullState(t, "the reactor", player, f.server.URL, f.playerUUID, f.matchUUID)
		defer conn.Close() //nolint:errcheck
		requireOwn(t, full, true)
		if full.OpenTurn == nil || len(full.OpenTurn.Reactions) != 1 || full.OpenTurn.Reactions[0].UUID != reactionID {
			t.Fatalf("openTurn = %+v, want reactions = [the repel %s]", full.OpenTurn, reactionID)
		}
	})
}

// reconnect drops one of the area fixture's connections and dials a fresh one for the same user.
func (f *areaFixture) reconnect(t *testing.T, c *wsConn) *wsConn {
	t.Helper()
	c.conn.Close() //nolint:errcheck
	return dialAreaConn(t, f.server.URL, c.user, f.matchUUID)
}

func TestE2E_ReconnectingTableGetsOpenedReactionsInOpeningOrder(t *testing.T) {
	faces := []int{
		6, 4, 1, 1, // Attack.Hit
		7, 3, // Sword damage
		7, 4, 1, 1, // A's repel
	}
	for range 20 { // whatever C's dodge rolls, and room to spare: no roll may overrun
		faces = append(faces, 5)
	}
	f := newAreaFixture(t, faces)
	defer f.server.Close()

	sword := "Sword"
	f.attacker.send(t, game.MsgTypeEnqueueAction, game.ActionPayload{
		ActorID:  f.attackerID,
		TargetID: []uuid.UUID{f.a, f.b, f.c},
		Attack: &game.AttackPayload{
			Weapon: &sword,
			Hit:    game.RollCheckPayload{SkillName: "Accuracy"},
			Damage: game.RollCheckPayload{},
		},
	})
	if !f.attacker.msgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the attack was never acknowledged as enqueued")
	}
	f.master.send(t, game.MsgTypeOpenNextAction, struct{}{})
	_, actionID := f.awaitTurnOpened(t)

	repelID := f.attachReaction(t, f.pA, game.ActionPayload{
		ActorID: f.a, ReactToID: actionID, ReactionKind: "repel",
		Repel: &game.RepelPayload{RollCheck: game.RollCheckPayload{SkillName: "Repel"}},
	})
	nothingID := f.attachReaction(t, f.pB, game.ActionPayload{
		ActorID: f.b, ReactToID: actionID, ReactionKind: "nothing",
	})
	// B first, then A: the opposite of arrival order, so opening order is what is checked.
	f.openReaction(t, nothingID)
	f.openReaction(t, repelID)
	if !awaitCount(f.pC.msgs, game.MsgTypeReactionOpened, 2, 2*time.Second) {
		t.Fatal("C never saw both reactions open live")
	}
	// What C was told live, per reaction — the reconnect must hand back the very same cut.
	live := map[uuid.UUID]any{}
	for _, m := range f.pC.msgs.snapshotMessages() {
		if m.Type != game.MsgTypeReactionOpened {
			continue
		}
		var p struct {
			ReactionID uuid.UUID `json:"reactionId"`
			Reaction   any       `json:"reaction"`
		}
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal reaction_opened: %v", err)
		}
		live[p.ReactionID] = p.Reaction
	}
	// C reacts too, and the master does not open it.
	dodgeID := f.attachReaction(t, f.pC, game.ActionPayload{
		ActorID: f.c, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	if f.source.overran {
		t.Fatal("the scripted source ran out: a roll happened that this test did not account for")
	}

	f.pC = f.reconnect(t, f.pC)
	defer f.pC.conn.Close() //nolint:errcheck
	full, raw := fullStateFrom(t, "C", f.pC.msgs)

	t.Run("openTurn.reactions holds the opened ones, in opening order: B, then A", func(t *testing.T) {
		if full.OpenTurn == nil {
			t.Fatal("openTurn is absent with the attack still open")
		}
		got := full.OpenTurn.Reactions
		if len(got) != 2 || got[0].UUID != nothingID || got[1].UUID != repelID {
			ids := make([]uuid.UUID, 0, len(got))
			for _, re := range got {
				ids = append(ids, re.UUID)
			}
			t.Fatalf("openTurn.reactions = %v, want [B %s, A %s] — and never C's unopened dodge %s",
				ids, nothingID, repelID, dodgeID)
		}
	})

	t.Run("each is cut exactly as the live reaction_opened cut it for C", func(t *testing.T) {
		var openTurn struct {
			Reactions []any `json:"reactions"`
		}
		if err := json.Unmarshal(raw["openTurn"], &openTurn); err != nil {
			t.Fatalf("unmarshal openTurn: %v", err)
		}
		for i, id := range []uuid.UUID{nothingID, repelID} {
			if i >= len(openTurn.Reactions) {
				t.Fatalf("openTurn.reactions has %d entries", len(openTurn.Reactions))
			}
			if !reflect.DeepEqual(openTurn.Reactions[i], live[id]) {
				t.Errorf("reaction %s on reconnect:\n%v\nlive reaction_opened:\n%v", id, openTurn.Reactions[i], live[id])
			}
		}
	})

	t.Run("C's own unopened dodge is in ownReactions, opened false, and nobody else's is", func(t *testing.T) {
		if len(full.OwnReactions) != 1 {
			t.Fatalf("ownReactions = %+v, want only C's dodge", full.OwnReactions)
		}
		got := full.OwnReactions[0]
		if got.ReactionID != dodgeID || got.ActorID != f.c || got.ReactionKind != "dodge" || got.Opened {
			t.Fatalf("ownReactions[0] = %+v, want C's dodge %s, unopened", got, dodgeID)
		}
		// A free reaction consumed nothing — and says so with [], never null.
		var own []map[string]json.RawMessage
		if err := json.Unmarshal(raw["ownReactions"], &own); err != nil {
			t.Fatalf("unmarshal ownReactions: %v", err)
		}
		if string(own[0]["consumedActionIds"]) != "[]" {
			t.Fatalf("consumedActionIds = %s, want []", own[0]["consumedActionIds"])
		}
	})
}

// D5: the master's ownReactions are their NPCs' — never a player character's, even though the
// master sees every reaction whole elsewhere (resolution.pendingReactions).
func TestE2E_MasterOwnReactionsCarryOnlyTheNPCs(t *testing.T) {
	f := newCombatFixture(t, withNPCTarget)
	master, player := f.connect(t)
	mc, pc := collectFrom(master), collectFrom(player)

	// One attack, two targets: the player's victim and the master's NPC.
	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId":  f.attackerID.String(),
		"targetId": []string{f.victimID.String(), f.npcID.String()},
		"speed":    map[string]any{"bar": 0, "rollCheck": map[string]any{"skillName": enum.Legerity.String()}},
		"attack": map[string]any{
			"weapon": "Sword",
			"hit":    map[string]any{"skillName": enum.Accuracy.String()},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the attack never reached the queue")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeTurnOpened, 3*time.Second) {
		t.Fatal("the attack never opened")
	}
	turn := f.session.GetActiveRound().CurrentTurn()
	if turn == nil {
		t.Fatal("no current turn on the session right after turn_opened")
	}
	act := turn.GetAction()
	actionID := act.GetID()

	sendWS(t, player, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.victimID, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	if !pc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatalf("the player never got reaction_attached; got %v", messageTypes(pc.snapshotMessages()))
	}
	playerReactionID := lastReactionAttached(t, pc).ReactionID
	sendWS(t, master, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.npcID, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	// The master got the player's reaction_attached too, so the NPC's is the second.
	if !awaitCount(mc, game.MsgTypeReactionAttached, 2, 2*time.Second) {
		t.Fatalf("the master never got the NPC's reaction_attached; got %v", messageTypes(mc.snapshotMessages()))
	}
	npcReactionID := lastReactionAttached(t, mc).ReactionID

	// Every fresh connection stays open to the end: the room closes itself once nobody is in
	// it, and the second reconnect must find the same session the first one left.
	var fresh []*websocket.Conn
	defer func() {
		for _, c := range fresh {
			c.Close() //nolint:errcheck
		}
	}()
	for _, tc := range []struct {
		who     string
		old     *websocket.Conn
		user    uuid.UUID
		reactor uuid.UUID
		id      uuid.UUID
	}{
		{"the master", master, f.masterUUID, f.npcID, npcReactionID},
		{"the player", player, f.playerUUID, f.victimID, playerReactionID},
	} {
		t.Run(tc.who+" gets only their own", func(t *testing.T) {
			conn, full, _ := reconnectFullState(t, tc.who, tc.old, f.server.URL, tc.user, f.matchUUID)
			fresh = append(fresh, conn)
			if len(full.OwnReactions) != 1 {
				t.Fatalf("ownReactions = %+v, want exactly one", full.OwnReactions)
			}
			got := full.OwnReactions[0]
			if got.ReactionID != tc.id || got.ActorID != tc.reactor || got.Opened {
				t.Fatalf("ownReactions[0] = %+v, want %s by %s, unopened", got, tc.id, tc.reactor)
			}
		})
	}
}

// D6: with no turn open there is nothing to reconnect to — no openTurn, no ownReactions, not
// even empty ones. The turn here DID carry an opened reaction until it closed, so this is the
// field going away, not a field that was never filled.
func TestE2E_NoOpenTurnMeansNoReactionFields(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	actionID := f.openAttackOn(t, player, master, mc)
	sendWS(t, player, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.victimID, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	if !pc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatalf("the reactor never got reaction_attached; got %v", messageTypes(pc.snapshotMessages()))
	}
	sendWS(t, master, string(game.MsgTypeOpenReaction),
		game.OpenReactionPayload{ReactionID: lastReactionAttached(t, pc).ReactionID})
	if !mc.await(game.MsgTypeReactionOpened, 2*time.Second) {
		t.Fatal("the reaction never opened")
	}
	// turn_closed, not the writer: the dodge may well clear the hit, and then no sheet is written.
	sendWS(t, master, "close_turn", map[string]any{"confirm": true})
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatalf("the turn never closed; the master received: %v", messageTypes(mc.snapshotMessages()))
	}

	conn, _, raw := reconnectFullState(t, "the reactor", player, f.server.URL, f.playerUUID, f.matchUUID)
	defer conn.Close() //nolint:errcheck
	for _, key := range []string{"openTurn", "ownReactions"} {
		if v, ok := raw[key]; ok {
			t.Errorf("match_full_state carries %q = %s with no turn open, want the key absent", key, v)
		}
	}
}
