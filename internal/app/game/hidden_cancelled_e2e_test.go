package game_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Uma action cancelada de um ator Escondido (revisão final do PR 2, achado 2) grava o registro
// dele no instante do cancelamento (cancelled_actions.hidden_from), pelos três caminhos: fora de
// turno (gravado na hora), dentro de um turno (retido para o fechamento) e o status que tira a
// action da fila. Desfazer o Escondido no mesmo turno tira o registro.

// queueNPCAttack: o mestre põe na fila um ataque do NPC e devolve o id da action.
func (h *hiddenTable) queueNPCAttack(t *testing.T, queuedBefore int) uuid.UUID {
	t.Helper()
	h.f.enqueueAttackAt(t, h.master, h.f.npcID, h.f.victimID)
	if !awaitCount(h.mc, game.MsgTypeActionQueued, queuedBefore+1, 2*time.Second) {
		t.Fatal("the NPC attack was never queued")
	}
	return lastActionQueuedID(t, h.mc)
}

func TestHiddenActor_CancelledActionRecordsHiddenFrom(t *testing.T) {
	t.Run("a cancel with no turn open records the hidden actor at once", func(t *testing.T) {
		h := newHiddenTable(t)
		hid := h.hide(t, h.f.npcID, nil)
		actionID := h.queueNPCAttack(t, 0)
		sendWS(t, h.master, "cancel_action", map[string]any{"actionId": actionID.String()})
		recs := h.f.cancelled.await(t, 1, 2*time.Second)
		if len(recs) != 1 {
			t.Fatalf("cancelled_actions = %d record(s), want 1", len(recs))
		}
		want := statuseffect.HiddenFrom{h.f.npcID: hiddenRec(hid)}
		if !reflect.DeepEqual(recs[0].HiddenFrom, want) {
			t.Fatalf("HiddenFrom = %v, want %v", recs[0].HiddenFrom, want)
		}
	})

	t.Run("a cancel inside an open turn is held with the actor's record", func(t *testing.T) {
		h := newHiddenTable(t)
		hid := h.hide(t, h.f.npcID, map[string]any{"mode": "players", "playerIds": []string{h.f.playerUUID.String()}})
		turnID := h.openPCAttackOnNPC(t)
		actionID := h.queueNPCAttack(t, 1)
		sendWS(t, h.master, "cancel_action", map[string]any{"actionId": actionID.String()})
		if !h.mc.await(game.MsgTypeActionCancelled, 2*time.Second) {
			t.Fatal("the cancellation was never acknowledged")
		}
		closeTurnAndWait(t, h.master, h.mc)
		h.f.awaitPersistedTurn(t, turnID)
		got := lastClose(t, h.f).CancelledActions
		want := statuseffect.HiddenFrom{h.f.npcID: hiddenRec(hid, h.f.playerUUID)}
		if len(got) != 1 || !reflect.DeepEqual(got[0].HiddenFrom, want) {
			t.Fatalf("cancelled actions = %+v, want one with %v", got, want)
		}
	})

	t.Run("a status that drops the hidden actor's action records it", func(t *testing.T) {
		h := newHiddenTable(t)
		hid := h.hide(t, h.f.npcID, nil)
		h.queueNPCAttack(t, 0)
		applyAndWait(t, h.master, h.mc, map[string]any{"characterIds": []string{h.f.npcID.String()}, "kind": "unconscious"})
		recs := h.f.cancelled.await(t, 1, 2*time.Second)
		want := statuseffect.HiddenFrom{h.f.npcID: hiddenRec(hid)}
		if len(recs) != 1 || !reflect.DeepEqual(recs[0].HiddenFrom, want) {
			t.Fatalf("cancelled actions = %+v, want one with %v", recs, want)
		}
	})

	t.Run("a visible actor records nothing", func(t *testing.T) {
		h := newHiddenTable(t)
		actionID := h.queueNPCAttack(t, 0)
		sendWS(t, h.master, "cancel_action", map[string]any{"actionId": actionID.String()})
		recs := h.f.cancelled.await(t, 1, 2*time.Second)
		if len(recs) != 1 || recs[0].HiddenFrom != nil {
			t.Fatalf("cancelled actions = %+v, want one with no record", recs)
		}
	})

	t.Run("undoing the Hidden in the same turn drops the record", func(t *testing.T) {
		h := newHiddenTable(t)
		turnID := h.openPCAttackOnNPC(t)
		hid := h.hide(t, h.f.npcID, nil)
		actionID := h.queueNPCAttack(t, 1)
		sendWS(t, h.master, "cancel_action", map[string]any{"actionId": actionID.String()})
		if !h.mc.await(game.MsgTypeActionCancelled, 2*time.Second) {
			t.Fatal("the cancellation was never acknowledged")
		}
		before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
		sendWS(t, h.master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": hid})
		awaitStatusCount(t, "master", h.mc, before+1)
		closeTurnAndWait(t, h.master, h.mc)
		h.f.awaitPersistedTurn(t, turnID)
		got := lastClose(t, h.f).CancelledActions
		if len(got) != 1 || got[0].HiddenFrom != nil {
			t.Fatalf("cancelled actions = %+v, want one with no trace of the undone Hidden", got)
		}
	})
}
