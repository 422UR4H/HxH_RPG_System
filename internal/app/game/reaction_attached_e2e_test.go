package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

// This file is the Phase 7 back package's reaction_attached (spec §4.2, §4.3, §4.4): the
// answer to whoever attached a reaction — and to the master, who is told which queued actions
// a charged reaction consumed. The table hears nothing: that someone reacted is not table news
// until the master opens it.

func lastReactionAttached(t *testing.T, c *collector) game.ReactionAttachedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeReactionAttached {
			continue
		}
		var p game.ReactionAttachedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal reaction_attached: %v", err)
		}
		return p
	}
	t.Fatal("no reaction_attached in the collected messages")
	return game.ReactionAttachedPayload{}
}

// lastActionQueuedID reads the actionId off the newest action_queued the collector holds.
func lastActionQueuedID(t *testing.T, c *collector) uuid.UUID {
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
		return p.ActionID
	}
	t.Fatal("no action_queued in the collected messages")
	return uuid.Nil
}

func TestE2E_ReactionAttachedAnswersTheReactorAndTheMasterOnly(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	f.seedBoard(t) // connectTable waits for every client to hold a board
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	actionID := f.openAttackOn(t, player, master, mc)
	sendWS(t, player, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.victimID, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	if !pc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatalf("the reactor never got reaction_attached; got %v", messageTypes(pc.snapshotMessages()))
	}
	if !mc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatal("the master never got reaction_attached")
	}
	got := lastReactionAttached(t, pc)
	if got.ActorID != f.victimID || got.ReactionID == uuid.Nil || got.TurnID == uuid.Nil {
		t.Fatalf("reaction_attached = %+v", got)
	}
	toMaster := lastReactionAttached(t, mc)
	if toMaster.ActorID != got.ActorID || toMaster.ReactionID != got.ReactionID || toMaster.TurnID != got.TurnID {
		t.Fatalf("the master's copy differs: %+v vs %+v", toMaster, got)
	}
	// consumedActionIds is ALWAYS a list — [] on a free reaction, never null or absent.
	var raw map[string]json.RawMessage
	for _, m := range pc.snapshotMessages() {
		if m.Type == game.MsgTypeReactionAttached {
			if err := json.Unmarshal(m.Payload, &raw); err != nil {
				t.Fatal(err)
			}
		}
	}
	if string(raw["consumedActionIds"]) != "[]" {
		t.Fatalf("consumedActionIds = %s, want []", raw["consumedActionIds"])
	}
	// The table hears nothing.
	time.Sleep(150 * time.Millisecond)
	if bc.count(game.MsgTypeReactionAttached) != 0 {
		t.Fatal("a bystander was told that someone reacted")
	}
}

func TestE2E_ReactionAttachedNamesTheConsumedAction(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	actionID := f.openAttackOn(t, player, master, mc)
	// The victim (the player's too) queues something AFTER the attack opened, so the
	// attack is the one under the baton and this one waits in the queue.
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
	if !pc.await(game.MsgTypeReactionAttached, 2*time.Second) || !mc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatal("reaction_attached never arrived")
	}
	for name, c := range map[string]*collector{"reactor": pc, "master": mc} {
		got := lastReactionAttached(t, c)
		if len(got.ConsumedActionIDs) != 1 || got.ConsumedActionIDs[0] != victimActionID {
			t.Fatalf("%s: consumedActionIds = %v, want [%v]", name, got.ConsumedActionIDs, victimActionID)
		}
	}
}

// The NPC is the target: the player's character attacks it, and the master attaches a dodge
// with actorId = the NPC sheet. The master is reactor and master at once, so he is answered
// exactly ONCE; the player, who is neither, hears nothing.
func TestE2E_TheMasterReactsThroughAnNPCAndIsAnsweredOnce(t *testing.T) {
	f := newCombatFixture(t, withNPCTarget)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	f.enqueueAttackAt(t, player, f.attackerID, f.npcID)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the attack on the NPC never reached the queue")
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

	sendWS(t, master, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.npcID, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	if !mc.await(game.MsgTypeReactionAttached, 2*time.Second) {
		t.Fatalf("the master never got reaction_attached; got %v", messageTypes(mc.snapshotMessages()))
	}
	// Let a second copy, if the room wrongly sent one, arrive.
	time.Sleep(150 * time.Millisecond)
	if n := mc.count(game.MsgTypeReactionAttached); n != 1 {
		t.Fatalf("the master got %d reaction_attached, want exactly 1", n)
	}
	if got := lastReactionAttached(t, mc); got.ActorID != f.npcID || got.ReactionID == uuid.Nil {
		t.Fatalf("reaction_attached = %+v, want actorId = the NPC", got)
	}
	if pc.count(game.MsgTypeReactionAttached) != 0 {
		t.Fatal("the player was told the master reacted")
	}
}
