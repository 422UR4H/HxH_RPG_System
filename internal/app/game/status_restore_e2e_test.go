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

func restoredOf(t *testing.T, c *collector) []game.ActionRestoredPayload {
	t.Helper()
	var out []game.ActionRestoredPayload
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeActionRestored {
			continue
		}
		var p game.ActionRestoredPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal action_restored: %v", err)
		}
		out = append(out, p)
	}
	return out
}

func restoredIDs(ps []game.ActionRestoredPayload) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.ActionID)
	}
	return out
}

func restoreRefusals(t *testing.T, c *collector) []game.ErrorPayload {
	t.Helper()
	var out []game.ErrorPayload
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeError {
			continue
		}
		var p game.ErrorPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if p.Code == "restore_refused" {
			out = append(out, p)
		}
	}
	return out
}

// awaitRestored espera o action_restored das ações em c, na ordem dada.
func awaitRestored(t *testing.T, who string, c *collector, want []uuid.UUID) []game.ActionRestoredPayload {
	t.Helper()
	if !awaitCount(c, game.MsgTypeActionRestored, len(want), 2*time.Second) {
		t.Fatalf("%s got %d action_restored, want %d", who, c.count(game.MsgTypeActionRestored), len(want))
	}
	got := restoredOf(t, c)
	if !slices.Equal(restoredIDs(got), want) {
		t.Fatalf("%s action_restored = %v, want %v", who, restoredIDs(got), want)
	}
	return got
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
	// Uma ação nova do jogador chega depois do cancelamento: as devolvidas voltam antes dela.
	f := h.f
	f.enqueueAttackAt(t, h.player, f.attackerID, f.npcID)
	if !awaitCount(h.mc, game.MsgTypeActionQueued, 4, 2*time.Second) {
		t.Fatalf("the later action was never queued; errors: %v", errorCodes(t, h.pc))
	}
	later := lastActionQueuedID(t, h.mc)
	queuedBefore := len(queuedIDs(t, h.mc))
	enqueuedBefore := h.pc.count(game.MsgTypeActionEnqueued)
	h.remove(t, eff, true)
	master := awaitRestored(t, "master", h.mc, h.queued)
	owner := awaitRestored(t, "owner", h.pc, h.queued)
	if !awaitCount(h.mc, game.MsgTypeActionQueued, queuedBefore+2, 2*time.Second) {
		t.Fatalf("the master never got the queue entries; errors: %v", errorCodes(t, h.mc))
	}
	if got := queuedIDs(t, h.mc)[queuedBefore:]; !slices.Equal(got, h.queued) {
		t.Fatalf("master action_queued = %v, want %v (same ids, in order)", got, h.queued)
	}
	// O lugar: as duas, anunciadas na ordem da fila, voltam diante da ação que chegou depois.
	for who, ps := range map[string][]game.ActionRestoredPayload{"master": master, "owner": owner} {
		for i, p := range ps {
			if p.BeforeActionID == nil || *p.BeforeActionID != later {
				t.Errorf("%s restored #%d beforeActionId = %v, want %v", who, i, p.BeforeActionID, later)
			}
		}
	}
	// O corte: o mestre vê os números, o dono o que declarou (sem velocidade).
	if master[0].Action.Speed.RollCheck.Result == nil {
		t.Error("master action_restored has no speed result, want actionwire.Full")
	}
	if owner[0].Action.Speed.RollCheck.Result != nil {
		t.Error("owner action_restored carries the speed result, want the ownQueue cut")
	}
	chatBarrier(t, h.master, h.mc, h.pc)
	if pending, want := idsOfPending(h.f), append(append([]uuid.UUID(nil), h.queued...), later); !slices.Equal(pending, want) {
		t.Fatalf("queue = %v, want %v (the restored ones back in their place)", pending, want)
	}
	// Nenhum ack sem pedido: o action_enqueued é casado pela ordem dos envios do cliente.
	if n := h.pc.count(game.MsgTypeActionEnqueued); n != enqueuedBefore {
		t.Fatalf("the owner got %d unsolicited action_enqueued", n-enqueuedBefore)
	}
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
	refusals := restoreRefusals(t, h.mc)
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

// Dois status aplicados no turno recusam as mesmas ações: desfazer os dois as devolve, em
// qualquer ordem. A recusa do primeiro passa o registro ao segundo.
func TestRemoveStatusEffect_InTurnUndo_TwoBlockersInEitherOrder(t *testing.T) {
	stun := func(t *testing.T, h *restoreTable) uuid.UUID {
		t.Helper()
		got := applyAndWait(t, h.master, h.mc, map[string]any{
			"characterIds": []string{h.f.victimID.String()}, "kind": "stunned", "params": map[string]any{"points": 3}})
		for _, e := range got.StatusEffects {
			if e.Kind == "stunned" {
				return e.ID
			}
		}
		t.Fatal("no stunned in the master's update")
		return uuid.Nil
	}

	t.Run("the first applied is undone first", func(t *testing.T) {
		h := newRestoreTable(t)
		paralyzed := h.paralyze(t)
		stunned := stun(t, h)
		h.remove(t, paralyzed, true)
		chatBarrier(t, h.master, h.mc)
		if n := len(restoreRefusals(t, h.mc)); n != 2 {
			t.Fatalf("restore_refused = %d, want 2 (stunned still blocks)", n)
		}
		if got := masterCancelledOf(t, h.mc, h.f.victimID, stunned); !slices.Equal(got, h.queued) {
			t.Fatalf("stunned cancelledActionIds = %v, want %v (the record moved to it)", got, h.queued)
		}
		h.remove(t, stunned, true)
		awaitRestored(t, "master", h.mc, h.queued)
		awaitRestored(t, "owner", h.pc, h.queued)
		if got := h.closeAndCancelled(t); len(got) != 0 {
			t.Fatalf("cancelled_actions at the close = %v, want none", got)
		}
	})

	t.Run("the last applied is undone first", func(t *testing.T) {
		h := newRestoreTable(t)
		paralyzed := h.paralyze(t)
		stunned := stun(t, h)
		h.remove(t, stunned, true)
		chatBarrier(t, h.master, h.mc)
		if n := h.mc.count(game.MsgTypeActionRestored); n != 0 {
			t.Fatalf("undoing the stun gave back %d action(s), want none (it cancelled nothing)", n)
		}
		h.remove(t, paralyzed, true)
		awaitRestored(t, "master", h.mc, h.queued)
		awaitRestored(t, "owner", h.pc, h.queued)
		if got := h.closeAndCancelled(t); len(got) != 0 {
			t.Fatalf("cancelled_actions at the close = %v, want none", got)
		}
	})
}

// Uma cadeia de movimentos empilhados volta inteira, na ordem, ainda empilhada.
func TestRemoveStatusEffect_InTurnUndo_RestoresAStackedChain(t *testing.T) {
	f := newStatusFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)
	turnID := f.openAttackTurn(t, master, player, mc)
	h := &restoreTable{f: f, master: master, player: player, mc: mc, pc: pc, turnID: turnID}

	for i, to := range [][3]int{{4, 6, 0}, {5, 6, 0}} {
		sendWS(t, player, "enqueue_action", map[string]any{
			"actorId": f.victimID.String(),
			"move":    map[string]any{"category": "Dash", "from": [3]int{0, 0, 0}, "position": to},
		})
		if !awaitCount(mc, game.MsgTypeActionQueued, i+2, 2*time.Second) {
			t.Fatalf("dash %d was never queued; errors: %v", i+1, errorCodes(t, pc))
		}
	}
	moves := queuedMoves(t, mc)
	if len(moves) != 2 || moves[1].stackedOn == nil || *moves[1].stackedOn != moves[0].id {
		t.Fatalf("setup: moves = %+v, want the second stacked on the first", moves)
	}
	h.queued = []uuid.UUID{moves[0].id, moves[1].id}

	got := applyAndWait(t, master, mc, map[string]any{
		"characterIds": []string{f.victimID.String()}, "kind": "paralyzed",
		"params": map[string]any{"points": 3, "domains": []string{"movement"}},
	})
	for _, id := range h.queued {
		awaitCancelled(t, "owner", pc, id)
	}
	h.remove(t, got.StatusEffects[0].ID, true)
	restored := awaitRestored(t, "master", mc, h.queued)
	awaitRestored(t, "owner", pc, h.queued)
	if s := restored[1].Action.Move; s == nil || s.StackedOn == nil || *s.StackedOn != h.queued[0] {
		t.Fatalf("restored second move = %+v, want still stacked on the first", s)
	}
	chatBarrier(t, master, mc, pc)
	// O arm terminou (a barreira passou): ler a sessão é seguro.
	if pending := idsOfPending(f); !slices.Equal(pending, h.queued) {
		t.Fatalf("queue = %v, want the chain %v", pending, h.queued)
	}
	if got := h.closeAndCancelled(t); len(got) != 0 {
		t.Fatalf("cancelled_actions at the close = %v, want none", got)
	}
}

func idsOfPending(f *combatFixture) []uuid.UUID {
	var out []uuid.UUID
	for _, a := range f.session.PendingActions() {
		out = append(out, a.GetID())
	}
	return out
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

// Uma cadeia cuja base é recusada por outro status do turno: o empilhado segue a base para o
// registro desse status, e desfazer os dois devolve a cadeia inteira, em qualquer ordem.
func TestRemoveStatusEffect_InTurnUndo_AChainFollowsItsBlockedBaseInEitherOrder(t *testing.T) {
	setup := func(t *testing.T) (h *restoreTable, stunned, paralyzed uuid.UUID) {
		t.Helper()
		f := newStatusFixture(t)
		master, player := f.connect(t)
		t.Cleanup(func() {
			_ = master.Close()
			_ = player.Close()
		})
		mc, pc := collectFrom(master), collectFrom(player)
		turnID := f.openAttackTurn(t, master, player, mc)
		h = &restoreTable{f: f, master: master, player: player, mc: mc, pc: pc, turnID: turnID}
		f.enqueueMoveAttackFrom(t, player, f.victimID, f.attackerID, "Sword", [3]int{0, 0, 0}, [3]int{4, 6, 0})
		sendWS(t, player, "enqueue_action", map[string]any{
			"actorId": f.victimID.String(),
			"move":    map[string]any{"category": "Dash", "from": [3]int{0, 0, 0}, "position": [3]int{5, 6, 0}},
		})
		if !awaitCount(mc, game.MsgTypeActionQueued, 3, 2*time.Second) {
			t.Fatalf("the chain was never queued; errors: %v", errorCodes(t, pc))
		}
		moves := queuedMoves(t, mc)
		if len(moves) != 2 || moves[1].stackedOn == nil || *moves[1].stackedOn != moves[0].id {
			t.Fatalf("setup: moves = %+v, want the dash stacked on the attack that moves", moves)
		}
		h.queued = []uuid.UUID{moves[0].id, moves[1].id}
		idOf := func(kind string, payload map[string]any) uuid.UUID {
			for _, e := range applyAndWait(t, master, mc, payload).StatusEffects {
				if string(e.Kind) == kind {
					return e.ID
				}
			}
			t.Fatalf("no %s in the master's update", kind)
			return uuid.Nil
		}
		stunned = idOf("stunned", map[string]any{
			"characterIds": []string{f.victimID.String()}, "kind": "stunned", "params": map[string]any{"points": 3}})
		for _, id := range h.queued {
			awaitCancelled(t, "owner", pc, id)
		}
		paralyzed = idOf("paralyzed", map[string]any{
			"characterIds": []string{f.victimID.String()}, "kind": "paralyzed",
			"params": map[string]any{"points": 3, "domains": []string{"attack"}}})
		return h, stunned, paralyzed
	}

	for _, tc := range []struct {
		name  string
		first func(stunned, paralyzed uuid.UUID) uuid.UUID
	}{
		{"the one that cancelled is undone first", func(s, _ uuid.UUID) uuid.UUID { return s }},
		{"the other one is undone first", func(_, p uuid.UUID) uuid.UUID { return p }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, stunned, paralyzed := setup(t)
			first := tc.first(stunned, paralyzed)
			second := stunned
			if first == stunned {
				second = paralyzed
			}
			h.remove(t, first, true)
			chatBarrier(t, h.master, h.mc)
			if n := h.mc.count(game.MsgTypeActionRestored); n != 0 {
				t.Fatalf("the first undo gave back %d action(s), want none (the other status still blocks the base)", n)
			}
			h.remove(t, second, true)
			awaitRestored(t, "master", h.mc, h.queued)
			awaitRestored(t, "owner", h.pc, h.queued)
			chatBarrier(t, h.master, h.mc, h.pc)
			if pending := idsOfPending(h.f); !slices.Equal(pending, h.queued) {
				t.Fatalf("queue = %v, want the whole chain %v", pending, h.queued)
			}
			if got := h.closeAndCancelled(t); len(got) != 0 {
				t.Fatalf("cancelled_actions at the close = %v, want none", got)
			}
		})
	}
}
