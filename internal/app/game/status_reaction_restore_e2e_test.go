package game_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

// A reação que um status desfeito no turno trocou volta com o mesmo restoreCancelledActions
// (D3, revisão final): a mesma reação (id, dados, lugar), revalidada; as âncoras que a tinham
// seguido para a sentada voltam com ela; reaction_restored ao mestre e ao dono (quando o efeito
// era visível a ele), nada a mais à mesa.

// reactionTable é o hiddenTable com o NPC atacando a vítima do jogador e a esquiva dela anexada.
type reactionTable struct {
	*hiddenTable
	turnID, actionID, reactionID uuid.UUID
}

func newReactionTable(t *testing.T, beforeTurn func(h *hiddenTable)) *reactionTable {
	t.Helper()
	h := newHiddenTable(t)
	if beforeTurn != nil {
		beforeTurn(h)
	}
	rt := &reactionTable{hiddenTable: h}
	rt.turnID = h.openNPCAttack(t)
	rt.actionID = lastTurnOpened(t, h.mc).ActionID
	rt.reactionID = rt.attachVictim(t, "dodge")
	return rt
}

// attachVictim manda, pelo jogador, a reação da vítima e devolve o id dela.
func (rt *reactionTable) attachVictim(t *testing.T, kind string) uuid.UUID {
	t.Helper()
	before := rt.pc.count(game.MsgTypeReactionAttached)
	p := game.ActionPayload{ActorID: rt.f.victimID, ReactToID: rt.actionID, ReactionKind: kind}
	if kind == "dodge" {
		p.Dodge = &game.DodgePayload{}
	}
	sendWS(t, rt.player, string(game.MsgTypeAttachReaction), p)
	if !awaitCount(rt.pc, game.MsgTypeReactionAttached, before+1, 2*time.Second) {
		t.Fatalf("the victim's %s was never attached; errors: %v", kind, errorCodes(t, rt.pc))
	}
	return lastReactionAttached(t, rt.pc).ReactionID
}

// applyOnVictim aplica kind à vítima no turno aberto e devolve o id do efeito.
func (rt *reactionTable) applyOnVictim(t *testing.T, kind string, extra map[string]any) uuid.UUID {
	t.Helper()
	p := map[string]any{"characterIds": []string{rt.f.victimID.String()}, "kind": kind, "params": map[string]any{"points": 3}}
	for k, v := range extra {
		p[k] = v
	}
	before := rt.mc.count(game.MsgTypeStatusEffectsUpdated)
	sendApplyStatus(t, rt.master, p)
	awaitStatusCount(t, "master", rt.mc, before+1)
	// A mudança pode reenviar outro personagem (a âncora que segue a reação): espera o comando
	// inteiro e lê a lista da vítima.
	chatBarrier(t, rt.master, rt.mc)
	for _, e := range lastStatusUpdateFor(t, rt.mc, rt.f.victimID).StatusEffects {
		if string(e.Kind) == kind {
			return e.ID
		}
	}
	t.Fatalf("no %s in the master's update", kind)
	return uuid.Nil
}

func (rt *reactionTable) removeWithRestore(t *testing.T, effectID uuid.UUID) {
	t.Helper()
	before := rt.mc.count(game.MsgTypeStatusEffectsUpdated)
	sendWS(t, rt.master, string(game.MsgTypeRemoveStatusEffect),
		map[string]any{"statusEffectId": effectID, "restoreCancelledActions": true})
	awaitStatusCount(t, "master", rt.mc, before+1)
	chatBarrier(t, rt.master, rt.mc, rt.pc, rt.bc)
}

// pending é a reação da vítima na última resolução do mestre.
func (rt *reactionTable) pending(t *testing.T) (id uuid.UUID, seated bool) {
	t.Helper()
	for _, pr := range lastResolutionUpdated(t, rt.mc).PendingReactions {
		if pr.ActorID == rt.f.victimID {
			return pr.ReactionID, pr.Seated
		}
	}
	t.Fatal("the victim has no pending reaction")
	return uuid.Nil, false
}

func reactionRestoredOf(t *testing.T, c *collector) []game.ReactionRestoredPayload {
	t.Helper()
	var out []game.ReactionRestoredPayload
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeReactionRestored {
			continue
		}
		var p game.ReactionRestoredPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal reaction_restored: %v", err)
		}
		out = append(out, p)
	}
	return out
}

// masterSwappedOf é o swappedReactionIds do efeito no último status_effects_updated do mestre.
func masterSwappedOf(t *testing.T, mc *collector, char, effectID uuid.UUID) []uuid.UUID {
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
				return e.SwappedReactionIDs
			}
		}
		return nil
	}
	t.Fatalf("master: no status_effects_updated for %s", char)
	return nil
}

func TestRemoveStatusEffect_InTurnUndo_RestoresTheSwappedReaction(t *testing.T) {
	rt := newReactionTable(t, nil)
	// Uma âncora presa à esquiva: segue a reação para a sentada e volta com ela.
	anchored := rt.applyAnchored(t, rt.f.bystanderID, "prone", rt.reactionID)

	stun := rt.applyOnVictim(t, "stunned", nil)
	chatBarrier(t, rt.master, rt.mc)
	seat, seated := rt.pending(t)
	if !seated || seat == rt.reactionID {
		t.Fatalf("under the stun the victim's pending reaction = %s (seated %v), want a seat", seat, seated)
	}
	assertAnchor(t, "master under the stun", masterAnchorOf(t, rt.hiddenTable, rt.f.bystanderID, anchored), seat)
	if got := masterSwappedOf(t, rt.mc, rt.f.victimID, stun); len(got) != 1 || got[0] != rt.reactionID {
		t.Fatalf("master swappedReactionIds = %v, want [%s]", got, rt.reactionID)
	}
	for _, e := range lastStatusUpdateFor(t, rt.pc, rt.f.victimID).StatusEffects {
		if len(e.SwappedReactionIDs) != 0 {
			t.Fatalf("the player got swappedReactionIds: %v", e.SwappedReactionIDs)
		}
	}

	rt.removeWithRestore(t, stun)

	if codes := errorCodes(t, rt.mc); len(codes) != 0 {
		t.Fatalf("master errors = %v, want none", codes)
	}
	m := reactionRestoredOf(t, rt.mc)
	if len(m) != 1 || m[0].ReactionID != rt.reactionID || m[0].ActorID != rt.f.victimID || m[0].TurnID != rt.turnID ||
		m[0].ReactionKind != "dodge" || m[0].ReplacedReactionID == nil {
		t.Fatalf("master reaction_restored = %+v, want the victim's dodge %s", m, rt.reactionID)
	}
	o := reactionRestoredOf(t, rt.pc)
	if len(o) != 1 || o[0].ReactionID != rt.reactionID || o[0].ReplacedReactionID != nil {
		t.Fatalf("owner reaction_restored = %+v, want the dodge without the seat id", o)
	}
	if n := rt.bc.count(game.MsgTypeReactionRestored); n != 0 {
		t.Fatalf("the bystander got %d reaction_restored", n)
	}
	if id, seated := rt.pending(t); id != rt.reactionID || seated {
		t.Fatalf("pending reaction = %s (seated %v), want the dodge %s back, not seated", id, seated, rt.reactionID)
	}
	assertAnchor(t, "master after the restore", masterAnchorOf(t, rt.hiddenTable, rt.f.bystanderID, anchored), rt.reactionID)
	assertAnchor(t, "close", closeAnchors(t, rt.hiddenTable, rt.turnID)[anchored], rt.reactionID)
}

func TestRemoveStatusEffect_InTurnUndo_ReactionResentIsRefused(t *testing.T) {
	rt := newReactionTable(t, nil)
	stun := rt.applyOnVictim(t, "stunned", nil)
	// Atordoada, a vítima só pode responder nothing — e responde: a palavra é dela.
	resent := rt.attachVictim(t, "nothing")

	rt.removeWithRestore(t, stun)

	refusals := restoreRefusals(t, rt.mc)
	if len(refusals) != 1 || !strings.HasPrefix(refusals[0].Message, "Não foi possível devolver a reação: ") ||
		!strings.Contains(refusals[0].Message, "outra reação") {
		t.Fatalf("master restore_refused = %+v, want one for the reaction, saying another was sent", refusals)
	}
	if n := rt.mc.count(game.MsgTypeReactionRestored) + rt.pc.count(game.MsgTypeReactionRestored); n != 0 {
		t.Fatalf("got %d reaction_restored, want none", n)
	}
	if codes := errorCodes(t, rt.pc); len(codes) != 0 {
		t.Fatalf("player errors = %v, want none", codes)
	}
	if id, seated := rt.pending(t); id != resent || seated {
		t.Fatalf("pending reaction = %s (seated %v), want the re-sent nothing %s", id, seated, resent)
	}
}

func TestRemoveStatusEffect_InTurnUndo_ReactionStillBlockedComesBackWithTheOther(t *testing.T) {
	rt := newReactionTable(t, nil)
	stun := rt.applyOnVictim(t, "stunned", nil)
	para := rt.applyOnVictim(t, "paralyzed", map[string]any{"params": map[string]any{"points": 3, "domains": []string{"dodge"}}})

	rt.removeWithRestore(t, stun)
	refusals := restoreRefusals(t, rt.mc)
	if len(refusals) != 1 || !strings.Contains(refusals[0].Message, "paralisado") {
		t.Fatalf("master restore_refused = %+v, want one naming Paralisado", refusals)
	}
	if got := masterSwappedOf(t, rt.mc, rt.f.victimID, para); len(got) != 1 || got[0] != rt.reactionID {
		t.Fatalf("paralyzed swappedReactionIds = %v, want the dodge passed to it", got)
	}

	rt.removeWithRestore(t, para)
	if m := reactionRestoredOf(t, rt.mc); len(m) != 1 || m[0].ReactionID != rt.reactionID {
		t.Fatalf("master reaction_restored = %+v, want the dodge back", m)
	}
	if id, seated := rt.pending(t); id != rt.reactionID || seated {
		t.Fatalf("pending reaction = %s (seated %v), want the dodge", id, seated)
	}
}

// Um efeito que o dono não via não é anunciado pela volta da reação; a mesa não ouve nada de um
// reator Escondido.
func TestRemoveStatusEffect_InTurnUndo_ReactionRestoreTellsNoOneWhatTheyCouldNotSee(t *testing.T) {
	rt := newReactionTable(t, func(h *hiddenTable) { h.hide(t, h.f.victimID, nil) })
	stun := rt.applyOnVictim(t, "stunned", map[string]any{
		"visibility": map[string]any{"mode": "players", "playerIds": []string{rt.f.bystanderUUID.String()}},
	})

	rt.removeWithRestore(t, stun)

	if m := reactionRestoredOf(t, rt.mc); len(m) != 1 || m[0].ReactionID != rt.reactionID {
		t.Fatalf("master reaction_restored = %+v, want the dodge back", m)
	}
	if n := rt.pc.count(game.MsgTypeReactionRestored); n != 0 {
		t.Fatalf("the owner got %d reaction_restored for an effect they could not see", n)
	}
	if n := rt.bc.count(game.MsgTypeReactionRestored); n != 0 {
		t.Fatalf("the bystander got %d reaction_restored", n)
	}
	assertNoKeyIn(t, "bystander", rt.bc, rt.reactionID.String())
}

// Sem o pedido, o desfazer é o de antes: a sentada fica no lugar e nada é anunciado.
func TestRemoveStatusEffect_InTurnUndo_WithoutRestore_ReactionStaysSwapped(t *testing.T) {
	rt := newReactionTable(t, nil)
	stun := rt.applyOnVictim(t, "stunned", nil)
	before := rt.mc.count(game.MsgTypeStatusEffectsUpdated)
	sendWS(t, rt.master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": stun})
	awaitStatusCount(t, "master", rt.mc, before+1)
	chatBarrier(t, rt.master, rt.mc, rt.pc)

	if n := rt.mc.count(game.MsgTypeReactionRestored) + rt.pc.count(game.MsgTypeReactionRestored); n != 0 {
		t.Fatalf("got %d reaction_restored without the flag", n)
	}
	if id, seated := rt.pending(t); id == rt.reactionID || !seated {
		t.Fatalf("pending reaction = %s (seated %v), want the default seat, not the dodge", id, seated)
	}
}

// A sentada da sessão que volta não é resposta do dono: nunca está em ownReactions, e o
// reaction_restored dela vai só ao mestre — mesmo com o efeito visível ao dono.
func TestRemoveStatusEffect_InTurnUndo_ARestoredSeatIsTheMastersOnly(t *testing.T) {
	h := newHiddenTable(t)
	rt := &reactionTable{hiddenTable: h}
	rt.turnID = h.openNPCAttack(t)
	rt.actionID = lastTurnOpened(t, h.mc).ActionID
	seat, seated := rt.pending(t)
	if !seated {
		t.Fatalf("the victim's pending reaction %s is not a seat", seat)
	}

	// Atordoado é público: o dono o vê. A padrão sentada vira nothing.
	stun := rt.applyOnVictim(t, "stunned", nil)
	if got := masterSwappedOf(t, rt.mc, rt.f.victimID, stun); len(got) != 1 || got[0] != seat {
		t.Fatalf("master swappedReactionIds = %v, want the seat %s", got, seat)
	}

	rt.removeWithRestore(t, stun)

	if m := reactionRestoredOf(t, rt.mc); len(m) != 1 || m[0].ReactionID != seat {
		t.Fatalf("master reaction_restored = %+v, want the seat %s back", m, seat)
	}
	if n := rt.pc.count(game.MsgTypeReactionRestored); n != 0 {
		t.Fatalf("the owner got %d reaction_restored for a seat they never sent", n)
	}
	if n := rt.bc.count(game.MsgTypeReactionRestored); n != 0 {
		t.Fatalf("the bystander got %d reaction_restored", n)
	}
	if id, seated := rt.pending(t); id != seat || !seated {
		t.Fatalf("pending reaction = %s (seated %v), want the original seat %s back as a seat", id, seated, seat)
	}
}
