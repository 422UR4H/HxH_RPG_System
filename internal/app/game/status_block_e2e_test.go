package game_test

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Peças que bloqueiam (Task 11, decisões 8 e 9): a sala recusa o que os status proíbem com
// blocked_by_status, tira da fila o que um status novo recusa (gravado como cancelado pelo
// mestre, com a mesma mensagem do cancel_action) e dá ao mestre as anotações dos status.

func applyStatus(t *testing.T, master *websocket.Conn, mc *collector, charID uuid.UUID, kind string, params map[string]any) {
	t.Helper()
	n := mc.count(game.MsgTypeStatusEffectsUpdated)
	p := map[string]any{"characterIds": []string{charID.String()}, "kind": kind}
	if params != nil {
		p["params"] = params
	}
	sendApplyStatus(t, master, p)
	if !awaitCount(mc, game.MsgTypeStatusEffectsUpdated, n+1, 2*time.Second) {
		t.Fatalf("apply %s never reached the master; errors: %v", kind, errorCodes(t, mc))
	}
}

func awaitBlockedError(t *testing.T, c *collector, code string) game.ErrorPayload {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if slices.Contains(errorCodes(t, c), code) {
			return lastError(t, c)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no %s error; errors: %v", code, errorCodes(t, c))
	return game.ErrorPayload{}
}

func awaitCancelled(t *testing.T, who string, c *collector, id uuid.UUID) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if slices.Contains(cancelledIDs(t, c), id) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s never got action_cancelled for %v; got %v", who, id, cancelledIDs(t, c))
}

func TestStatusBlock_EnqueueRefused(t *testing.T) {
	f := newStatusFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	applyStatus(t, master, mc, f.attackerID, "stunned", map[string]any{"points": 3})
	f.enqueueAttack(t, player)
	e := awaitBlockedError(t, pc, "blocked_by_status")
	if !strings.Contains(e.Message, "agir") || !strings.Contains(e.Message, "stunned") {
		t.Errorf("message = %q, want what is blocked and by which kind", e.Message)
	}
	if n := mc.count(game.MsgTypeActionQueued); n != 0 {
		t.Fatalf("the master got %d action_queued for a refused action", n)
	}

	t.Run("the master's NPC is refused too", func(t *testing.T) {
		applyStatus(t, master, mc, f.npcID, "paralyzed", map[string]any{"points": 3, "domains": []string{"attack"}})
		f.enqueueAttackAt(t, master, f.npcID, f.victimID)
		e := awaitBlockedError(t, mc, "blocked_by_status")
		if !strings.Contains(e.Message, "ataque") || !strings.Contains(e.Message, "paralyzed") {
			t.Errorf("message = %q, want the attack domain and paralyzed", e.Message)
		}
	})

}

func TestStatusBlock_ReactionRefused(t *testing.T) {
	f := newStatusFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)
	turnID := f.openAttackTurn(t, master, player, mc)
	applyStatus(t, master, mc, f.victimID, "openGuard", nil)

	attackID := lastTurnOpened(t, mc).ActionID
	sword := "Sword"
	sendWS(t, player, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: f.victimID, ReactToID: attackID, ReactionKind: "repel",
		Repel: &game.RepelPayload{Weapon: &sword},
	})
	e := awaitBlockedError(t, pc, "blocked_by_status")
	if !strings.Contains(e.Message, "openGuard") {
		t.Errorf("message = %q, want openGuard", e.Message)
	}
	if n := pc.count(game.MsgTypeReactionAttached); n != 0 {
		t.Fatalf("a refused reaction was attached (%d reaction_attached)", n)
	}
	sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
	if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
		t.Fatal("the turn never closed")
	}
	f.awaitPersistedTurn(t, turnID)
}

func TestStatusBlock_ApplyDropsQueue(t *testing.T) {
	t.Run("no turn open: recorded at once, cancelled by the master", func(t *testing.T) {
		f := newStatusFixture(t)
		master, player := f.connect(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		mc, pc := collectFrom(master), collectFrom(player)

		f.enqueueAttack(t, player)
		if !pc.await(game.MsgTypeActionEnqueued, 2*time.Second) {
			t.Fatalf("never enqueued; errors: %v", errorCodes(t, pc))
		}
		actionID := actionIDsInOrder(t, pc)[0]
		if !awaitCount(mc, game.MsgTypeBarsUpdated, 1, 2*time.Second) {
			t.Fatal("no bars after the enqueue")
		}
		seq := lastBarsUpdated(t, mc).Seq

		applyStatus(t, master, mc, f.attackerID, "stunned", map[string]any{"points": 3})
		awaitCancelled(t, "master", mc, actionID)
		awaitCancelled(t, "owner", pc, actionID)

		recs := f.cancelled.await(t, 1, 2*time.Second)
		if len(recs) != 1 || recs[0].UUID != actionID {
			t.Fatalf("cancelled_actions = %+v, want the attack %v", recs, actionID)
		}
		if recs[0].CancelledBy != f.masterUUID || recs[0].TurnUUID != nil {
			t.Errorf("record by %v turn %v, want the master and no turn", recs[0].CancelledBy, recs[0].TurnUUID)
		}
		if !awaitBarsAfter(t, mc, seq, 2*time.Second) {
			t.Fatal("no bars_updated after the queue changed")
		}
		if orderHasActor(lastBarsUpdated(t, mc), f.attackerID) {
			t.Error("the stunned attacker is still in the master's order")
		}
	})

	t.Run("turn open: held for the close", func(t *testing.T) {
		f := newStatusFixture(t)
		master, player := f.connect(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		mc, pc := collectFrom(master), collectFrom(player)
		turnID := f.openAttackTurn(t, master, player, mc)

		f.enqueueAttackAt(t, player, f.victimID, f.attackerID)
		if !awaitCount(mc, game.MsgTypeActionQueued, 2, 2*time.Second) {
			t.Fatalf("the victim's action was never queued; errors: %v", errorCodes(t, pc))
		}
		actionID := lastActionQueuedID(t, mc)

		applyStatus(t, master, mc, f.victimID, "stunned", map[string]any{"points": 3})
		awaitCancelled(t, "master", mc, actionID)
		awaitCancelled(t, "owner", pc, actionID)
		if recs := f.cancelled.snapshot(); len(recs) != 0 {
			t.Fatalf("cancelled_actions got %d record(s) mid-turn", len(recs))
		}

		sendWS(t, master, string(game.MsgTypeCloseTurn), game.CloseTurnPayload{Confirm: true})
		if !mc.await(game.MsgTypeTurnClosed, 3*time.Second) {
			t.Fatalf("the turn never closed; errors: %v", errorCodes(t, mc))
		}
		f.awaitPersistedTurn(t, turnID)
		closes := f.roundRepo.closeData()
		if len(closes) != 1 {
			t.Fatalf("PersistTurnClose ran %d time(s), want 1", len(closes))
		}
		got := closes[0].CancelledActions
		if len(got) != 1 || got[0].UUID != actionID || got[0].CancelledBy != f.masterUUID {
			t.Fatalf("held cancelled actions = %+v, want the victim's %v by the master", got, actionID)
		}
		if got[0].TurnUUID == nil || *got[0].TurnUUID != turnID {
			t.Errorf("turnUuid = %v, want %v", got[0].TurnUUID, turnID)
		}
		// O atordoado recebeu um nothing sentado no lugar da reação padrão.
		res := lastResolutionUpdated(t, mc)
		var seen bool
		for _, cr := range res.Targets {
			if cr.TargetID != f.victimID {
				continue
			}
			seen = true
			if cr.Reaction == nil || cr.Reaction.Kind != "nothing" {
				t.Errorf("the stunned victim answered with %+v, want nothing", cr.Reaction)
			}
		}
		if !seen {
			t.Error("the last resolution has no result for the victim")
		}
	})
}

func TestStatusBlock_NotesToMaster(t *testing.T) {
	f := newStatusFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	applyStatus(t, master, mc, f.attackerID, "injured", map[string]any{"injuries": map[string]string{"attack": "disabled"}})
	applyStatus(t, master, mc, f.attackerID, "compelled", map[string]any{"conduct": "Proteger a vila", "irresistible": true})

	f.enqueueAttack(t, player)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatalf("never queued; errors: %v", errorCodes(t, pc))
	}
	notes := lastActionQueued(t, mc).StatusNotes
	assertNotes(t, "action_queued", notes)

	chatBarrier(t, master, mc, pc)
	for _, m := range pc.snapshotMessages() {
		if bytes.Contains(m.Payload, []byte("statusNotes")) {
			t.Errorf("the player got statusNotes on %s", m.Type)
		}
	}

	t.Run("match_full_state.queue carries them", func(t *testing.T) {
		late := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
		defer late.Close()   //nolint:errcheck
		readMessage(t, late) // room_state
		lc := collectFrom(late)
		if !lc.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatal("no match_full_state")
		}
		var full game.MatchFullStatePayload
		if err := json.Unmarshal(findMessage(t, lc.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &full); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(full.Queue) != 1 {
			t.Fatalf("queue = %d entries, want 1", len(full.Queue))
		}
		assertNotes(t, "match_full_state.queue", full.Queue[0].StatusNotes)
	})
}

func assertNotes(t *testing.T, where string, notes []game.StatusNotePayload) {
	t.Helper()
	var injured, compelled bool
	for _, n := range notes {
		switch n.Kind {
		case "injured":
			injured = n.Domain == "attack"
		case "compelled":
			compelled = n.Conduct == "Proteger a vila" && n.Irresistible
		}
	}
	if !injured || !compelled || len(notes) != 2 {
		t.Fatalf("%s statusNotes = %+v, want the attack warning and the conduct", where, notes)
	}
}

func TestStatusBlock_RemoveUnblocks(t *testing.T) {
	f := newStatusFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	applyStatus(t, master, mc, f.attackerID, "stunned", map[string]any{"points": 3})
	id := lastStatusUpdate(t, mc).StatusEffects[0].ID
	f.enqueueAttack(t, player)
	awaitBlockedError(t, pc, "blocked_by_status")

	n := mc.count(game.MsgTypeStatusEffectsUpdated)
	sendWS(t, master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": id.String()})
	if !awaitCount(mc, game.MsgTypeStatusEffectsUpdated, n+1, 2*time.Second) {
		t.Fatal("the removal never reached the master")
	}
	f.enqueueAttack(t, player)
	if !pc.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatalf("still refused after the removal; errors: %v", errorCodes(t, pc))
	}
}

// §9: uma recusa não revela um status que o remetente não podia ver. O dono tem um Paralisado só
// do mestre ("players: []") e recebe "um efeito"; o mestre, mandando a action do NPC com o mesmo
// status escondido, vê o tipo.
func TestStatusBlock_RefusalHidesWhatTheSenderCannotSee(t *testing.T) {
	f := newStatusFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	masterOnly := map[string]any{"mode": "players", "playerIds": []string{}}
	for _, id := range []uuid.UUID{f.attackerID, f.npcID} {
		n := mc.count(game.MsgTypeStatusEffectsUpdated)
		sendApplyStatus(t, master, map[string]any{
			"characterIds": []string{id.String()}, "kind": "paralyzed",
			"params": map[string]any{"points": 3, "domains": []string{"attack"}}, "visibility": masterOnly,
		})
		if !awaitCount(mc, game.MsgTypeStatusEffectsUpdated, n+1, 2*time.Second) {
			t.Fatalf("apply never reached the master; errors: %v", errorCodes(t, mc))
		}
	}

	f.enqueueAttack(t, player)
	e := awaitBlockedError(t, pc, "blocked_by_status")
	if strings.Contains(e.Message, "paralyzed") || !strings.Contains(e.Message, "um efeito") {
		t.Errorf("owner's refusal = %q, want the generic cause and no kind", e.Message)
	}

	f.enqueueAttackAt(t, master, f.npcID, f.victimID)
	e = awaitBlockedError(t, mc, "blocked_by_status")
	if !strings.Contains(e.Message, "paralyzed") {
		t.Errorf("master's refusal = %q, want the kind named", e.Message)
	}
}
