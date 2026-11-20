package game_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Devolver as ações que um status desfeito no turno cancelou (D3): o mestre remove, no mesmo
// turno aberto, um status aplicado nele com restoreCancelledActions, e as ações da fila que esse
// status tirou voltam (mesmos ids e dados), revalidadas; nada delas é gravado como cancelado.
// Fora da janela do desfazer, a flag não muda nada.

// restoreTable é a mesa do D3: o turno do atacante aberto e duas ações do alvo (do jogador) na
// fila.
type restoreTable struct {
	f              *combatFixture
	master, player *websocket.Conn
	mc, pc         *collector
	turnID         uuid.UUID
	queued         []uuid.UUID
}

func newRestoreTable(t *testing.T) *restoreTable {
	t.Helper()
	f := newStatusFixture(t)
	master, player := f.connect(t)
	t.Cleanup(func() {
		_ = master.Close()
		_ = player.Close()
	})
	h := &restoreTable{f: f, master: master, player: player, mc: collectFrom(master), pc: collectFrom(player)}
	h.turnID = f.openAttackTurn(t, master, player, h.mc)
	for i, target := range []uuid.UUID{f.attackerID, f.npcID} {
		f.enqueueAttackAt(t, player, f.victimID, target)
		if !awaitCount(h.mc, game.MsgTypeActionQueued, i+2, 2*time.Second) {
			t.Fatalf("the victim's action %d was never queued; errors: %v", i+1, errorCodes(t, h.pc))
		}
		h.queued = append(h.queued, lastActionQueuedID(t, h.mc))
	}
	return h
}

// paralyze aplica Paralisado (ataque) ao alvo, espera os dois action_cancelled e devolve o id
// do efeito.
func (h *restoreTable) paralyze(t *testing.T) uuid.UUID {
	t.Helper()
	got := applyAndWait(t, h.master, h.mc, map[string]any{
		"characterIds": []string{h.f.victimID.String()}, "kind": "paralyzed",
		"params": map[string]any{"points": 3, "domains": []string{"attack"}},
	})
	for _, id := range h.queued {
		awaitCancelled(t, "master", h.mc, id)
		awaitCancelled(t, "owner", h.pc, id)
	}
	for _, e := range got.StatusEffects {
		if e.Kind == "paralyzed" {
			return e.ID
		}
	}
	t.Fatal("no paralyzed in the master's update")
	return uuid.Nil
}

// remove manda o remove_status_effect e espera o status_effects_updated do mestre.
func (h *restoreTable) remove(t *testing.T, effectID uuid.UUID, restore bool) {
	t.Helper()
	before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
	p := map[string]any{"statusEffectId": effectID}
	if restore {
		p["restoreCancelledActions"] = true
	}
	sendWS(t, h.master, string(game.MsgTypeRemoveStatusEffect), p)
	awaitStatusCount(t, "master", h.mc, before+1)
}

// masterCancelledOf é o cancelledActionIds do efeito no último status_effects_updated do mestre
// para char.
func masterCancelledOf(t *testing.T, mc *collector, char, effectID uuid.UUID) []uuid.UUID {
	t.Helper()
	msgs := mc.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeStatusEffectsUpdated {
			continue
		}
		p := decodeStatusUpdate(t, msgs[i])
		if p.CharacterID != char {
			continue
		}
		for _, e := range p.StatusEffects {
			if e.ID == effectID {
				return e.CancelledActionIDs
			}
		}
		return nil
	}
	t.Fatalf("master: no status_effects_updated for %s", char)
	return nil
}

func queuedIDs(t *testing.T, c *collector) []uuid.UUID {
	t.Helper()
	var ids []uuid.UUID
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeActionQueued {
			continue
		}
		var p game.ActionQueuedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal action_queued: %v", err)
		}
		ids = append(ids, p.ActionID)
	}
	return ids
}

func assertNoKeyIn(t *testing.T, who string, c *collector, key string) {
	t.Helper()
	for _, m := range c.snapshotMessages() {
		if strings.Contains(string(m.Payload), key) {
			t.Fatalf("%s got %s: %s %s", who, key, m.Type, m.Payload)
		}
	}
}

func (h *restoreTable) closeAndCancelled(t *testing.T) []uuid.UUID {
	t.Helper()
	closeTurnAndWait(t, h.master, h.mc)
	h.f.awaitPersistedTurn(t, h.turnID)
	closes := h.f.roundRepo.closeData()
	if len(closes) != 1 {
		t.Fatalf("PersistTurnClose ran %d time(s), want 1", len(closes))
	}
	var ids []uuid.UUID
	for _, rec := range closes[0].CancelledActions {
		ids = append(ids, rec.UUID)
	}
	return ids
}

func TestRemoveStatusEffect_InTurnUndo_RestoresCancelledActions(t *testing.T) {
	h := newRestoreTable(t)
	eff := h.paralyze(t)

	// Na janela do desfazer, o mestre vê o que o efeito tirou da fila; o jogador nunca.
	if got := masterCancelledOf(t, h.mc, h.f.victimID, eff); !slices.Equal(got, h.queued) {
		t.Fatalf("master cancelledActionIds = %v, want %v", got, h.queued)
	}
	queuedBefore := len(queuedIDs(t, h.mc))
	enqueuedBefore := len(actionIDsInOrder(t, h.pc))
	h.remove(t, eff, true)
	if !awaitCount(h.mc, game.MsgTypeActionQueued, queuedBefore+2, 2*time.Second) {
		t.Fatalf("the master never got the restored actions; errors: %v", errorCodes(t, h.mc))
	}
	if got := queuedIDs(t, h.mc)[queuedBefore:]; !slices.Equal(got, h.queued) {
		t.Fatalf("master action_queued = %v, want %v (same ids, in order)", got, h.queued)
	}
	if !awaitCount(h.pc, game.MsgTypeActionEnqueued, enqueuedBefore+2, 2*time.Second) {
		t.Fatal("the owner never got the restored actions")
	}
	if got := actionIDsInOrder(t, h.pc)[enqueuedBefore:]; !slices.Equal(got, h.queued) {
		t.Fatalf("owner action_enqueued = %v, want %v", got, h.queued)
	}
	chatBarrier(t, h.master, h.mc, h.pc)
	assertNoKeyIn(t, "player", h.pc, "cancelledActionIds")
	if codes := errorCodes(t, h.mc); len(codes) != 0 {
		t.Fatalf("master errors = %v, want none", codes)
	}

	if got := h.closeAndCancelled(t); len(got) != 0 {
		t.Fatalf("cancelled_actions at the close = %v, want none (the restore erased them)", got)
	}
}

func TestRemoveStatusEffect_InTurnUndo_WithoutRestore_KeepsCancelled(t *testing.T) {
	h := newRestoreTable(t)
	eff := h.paralyze(t)
	queuedBefore := h.mc.count(game.MsgTypeActionQueued)
	h.remove(t, eff, false)
	chatBarrier(t, h.master, h.mc, h.pc)
	if n := h.mc.count(game.MsgTypeActionQueued); n != queuedBefore {
		t.Fatalf("the master got %d new action_queued without restore", n-queuedBefore)
	}
	if got := h.closeAndCancelled(t); !slices.Equal(got, h.queued) {
		t.Fatalf("cancelled_actions at the close = %v, want %v", got, h.queued)
	}
}

func TestRemoveStatusEffect_InTurnUndo_RefusalGoesToTheMasterOnly(t *testing.T) {
	h := newRestoreTable(t)
	eff := h.paralyze(t)
	applyStatus(t, h.master, h.mc, h.f.victimID, "stunned", map[string]any{"points": 3})
	queuedBefore := h.mc.count(game.MsgTypeActionQueued)
	playerErrors := len(errorCodes(t, h.pc))

	h.remove(t, eff, true)
	chatBarrier(t, h.master, h.mc, h.pc)
	if n := h.mc.count(game.MsgTypeActionQueued); n != queuedBefore {
		t.Fatalf("a still-stunned character got %d action(s) back", n-queuedBefore)
	}
	var refusals []game.ErrorPayload
	for _, m := range h.mc.snapshotMessages() {
		if m.Type != game.MsgTypeError {
			continue
		}
		var p game.ErrorPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if p.Code == "restore_refused" {
			refusals = append(refusals, p)
		}
	}
	if len(refusals) != 2 {
		t.Fatalf("restore_refused = %+v, want one per action", refusals)
	}
	for _, r := range refusals {
		if !strings.HasPrefix(r.Message, "Não foi possível devolver a ação: ") || !strings.Contains(r.Message, "stunned") {
			t.Errorf("message = %q, want the PT prefix and what still blocks", r.Message)
		}
	}
	if got := len(errorCodes(t, h.pc)); got != playerErrors {
		t.Fatalf("the player got %d error(s) from the master's restore", got-playerErrors)
	}
	if got := h.closeAndCancelled(t); !slices.Equal(got, h.queued) {
		t.Fatalf("cancelled_actions at the close = %v, want %v (refused stays cancelled)", got, h.queued)
	}
}

func TestRemoveStatusEffect_RestoreFlagOutsideUndoWindow_IsIgnored(t *testing.T) {
	f := newStatusFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	f.enqueueAttack(t, player)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatalf("never queued; errors: %v", errorCodes(t, pc))
	}
	actionID := lastActionQueuedID(t, mc)
	got := applyAndWait(t, master, mc, map[string]any{
		"characterIds": []string{f.attackerID.String()}, "kind": "paralyzed",
		"params": map[string]any{"points": 3, "domains": []string{"attack"}},
	})
	awaitCancelled(t, "master", mc, actionID)
	if recs := f.cancelled.await(t, 1, 2*time.Second); len(recs) != 1 || recs[0].UUID != actionID {
		t.Fatalf("cancelled_actions = %+v, want the attack written at once", recs)
	}
	eff := got.StatusEffects[0].ID
	if ids := masterCancelledOf(t, mc, f.attackerID, eff); len(ids) != 0 {
		t.Fatalf("cancelledActionIds = %v outside a turn, want none", ids)
	}

	before := mc.count(game.MsgTypeStatusEffectsUpdated)
	sendWS(t, master, string(game.MsgTypeRemoveStatusEffect), map[string]any{
		"statusEffectId": eff, "restoreCancelledActions": true})
	awaitStatusCount(t, "master", mc, before+1)
	chatBarrier(t, master, mc, pc)
	if n := mc.count(game.MsgTypeActionQueued); n != 1 {
		t.Fatalf("the master got %d action_queued, want only the original", n)
	}
	if codes := errorCodes(t, mc); len(codes) != 0 {
		t.Fatalf("master errors = %v, want none (the flag is ignored)", codes)
	}
	assertNoKeyIn(t, "master", mc, "cancelledActionIds")
}

func TestRemoveStatusEffect_AfterClose_NothingToRestore(t *testing.T) {
	h := newRestoreTable(t)
	eff := h.paralyze(t)
	before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
	if got := h.closeAndCancelled(t); !slices.Equal(got, h.queued) {
		t.Fatalf("cancelled_actions at the close = %v, want %v", got, h.queued)
	}
	// O cancelledActionIds que o mestre via some com o fechamento.
	awaitStatusCount(t, "master", h.mc, before+1)
	if ids := masterCancelledOf(t, h.mc, h.f.victimID, eff); len(ids) != 0 {
		t.Fatalf("cancelledActionIds after the close = %v, want none", ids)
	}

	queuedBefore := h.mc.count(game.MsgTypeActionQueued)
	h.remove(t, eff, true)
	chatBarrier(t, h.master, h.mc, h.pc)
	if n := h.mc.count(game.MsgTypeActionQueued); n != queuedBefore {
		t.Fatalf("the master got %d action(s) back after the close", n-queuedBefore)
	}
	if codes := errorCodes(t, h.mc); len(codes) != 0 {
		t.Fatalf("master errors = %v, want none", codes)
	}
}

func TestRemoveStatusEffect_CancelledActionIDsAreTheMasters(t *testing.T) {
	h := newRestoreTable(t)
	eff := h.paralyze(t)
	chatBarrier(t, h.master, h.mc, h.pc)
	assertNoKeyIn(t, "player", h.pc, "cancelledActionIds")

	// Na reconexão do mestre também; a do dono, nunca.
	for _, e := range h.f.reconnectFullState(t, h.player, h.f.playerUUID)[h.f.victimID] {
		if len(e.CancelledActionIDs) != 0 {
			t.Fatal("owner reconnect got cancelledActionIds")
		}
	}
	var found bool
	for _, e := range h.f.reconnectFullState(t, h.master, h.f.masterUUID)[h.f.victimID] {
		if e.ID == eff {
			found = true
			if !slices.Equal(e.CancelledActionIDs, h.queued) {
				t.Fatalf("master reconnect cancelledActionIds = %v, want %v", e.CancelledActionIDs, h.queued)
			}
		}
	}
	if !found {
		t.Fatal("master reconnect: the effect is missing")
	}
}
