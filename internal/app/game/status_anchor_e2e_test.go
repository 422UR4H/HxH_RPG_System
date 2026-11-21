package game_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Âncoras do turno aberto (Task 15b): a âncora segue a reação que substitui a sentada (ou vai
// para a action quando a sentada sai), o mestre a troca com relink_status_effect, vê a âncora
// viva em turnActionId (só ele), e remover no mesmo turno um efeito aplicado nele desfaz a
// aplicação (nada é gravado).

// seatedOf é a reação sentada de char na última resolução que o mestre recebeu.
func seatedOf(t *testing.T, h *hiddenTable, char uuid.UUID) uuid.UUID {
	t.Helper()
	if !h.mc.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
		t.Fatal("the master never got the open turn's resolution")
	}
	for _, pr := range lastResolutionUpdated(t, h.mc).PendingReactions {
		if pr.ActorID == char && pr.Seated {
			return pr.ReactionID
		}
	}
	t.Fatalf("no seated reaction for %s", char)
	return uuid.Nil
}

// masterAnchorOf é o turnActionId do efeito no último status_effects_updated do mestre para
// char; nil = ausente.
func masterAnchorOf(t *testing.T, h *hiddenTable, char, effectID uuid.UUID) *uuid.UUID {
	t.Helper()
	msgs := h.mc.snapshotMessages()
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
				return e.TurnActionID
			}
		}
		return nil
	}
	t.Fatalf("master: no status_effects_updated for %s", char)
	return nil
}

func assertAnchor(t *testing.T, who string, got *uuid.UUID, want uuid.UUID) {
	t.Helper()
	if want == uuid.Nil {
		if got != nil {
			t.Fatalf("%s: anchor = %s, want none", who, *got)
		}
		return
	}
	if got == nil || *got != want {
		t.Fatalf("%s: anchor = %v, want %s", who, got, want)
	}
}

// closeAnchors fecha o turno e devolve a âncora de cada gravação do fechamento, por efeito.
func closeAnchors(t *testing.T, h *hiddenTable, turnID uuid.UUID) map[uuid.UUID]*uuid.UUID {
	t.Helper()
	closeTurnAndWait(t, h.master, h.mc)
	h.f.awaitPersistedTurn(t, turnID)
	out := map[uuid.UUID]*uuid.UUID{}
	for _, w := range lastCloseStatusOps(t, h.f) {
		if w.Event != nil {
			out[w.Effect.ID] = w.Event.ActionID
		}
	}
	return out
}

func (h *hiddenTable) applyAnchored(t *testing.T, char uuid.UUID, kind string, anchor uuid.UUID) uuid.UUID {
	t.Helper()
	got := applyAndWait(t, h.master, h.mc, map[string]any{
		"characterIds": []string{char.String()}, "kind": kind, "actionId": anchor})
	for _, e := range got.StatusEffects {
		if string(e.Kind) == kind {
			return e.ID
		}
	}
	t.Fatalf("no %s in the update", kind)
	return uuid.Nil
}

func TestStatusAnchor_FollowsTheReplacement(t *testing.T) {
	h := newHiddenTable(t)
	turnID := h.openPCAttackOnNPC(t)
	seated := seatedOf(t, h, h.f.npcID)
	eff := h.applyAnchored(t, h.f.bystanderID, "prone", seated)
	assertAnchor(t, "master after apply", masterAnchorOf(t, h, h.f.bystanderID, eff), seated)

	before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
	reaction := h.attachNPCDodge(t, lastTurnOpened(t, h.mc).ActionID)
	awaitStatusCount(t, "master", h.mc, before+1)
	assertAnchor(t, "master after the replacement", masterAnchorOf(t, h, h.f.bystanderID, eff), reaction)

	assertAnchor(t, "close", closeAnchors(t, h, turnID)[eff], reaction)
}

func TestStatusAnchor_FollowsAnUnseatToTheAction(t *testing.T) {
	h := newHiddenTable(t)
	turnID := h.openPCAttackOnNPC(t)
	actionID := lastTurnOpened(t, h.mc).ActionID
	seated := seatedOf(t, h, h.f.npcID)
	eff := h.applyAnchored(t, h.f.bystanderID, "prone", seated)

	// O mestre troca o alvo: a sentada do NPC sai do turno, sem substituta.
	before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
	sendWS(t, h.master, string(game.MsgTypeEditAction), map[string]any{"targetIds": []string{h.f.victimID.String()}})
	awaitStatusCount(t, "master", h.mc, before+1)
	assertAnchor(t, "master after the unseat", masterAnchorOf(t, h, h.f.bystanderID, eff), actionID)

	assertAnchor(t, "close", closeAnchors(t, h, turnID)[eff], actionID)
}

func TestStatusAnchor_Relink(t *testing.T) {
	relink := func(h *hiddenTable, eff uuid.UUID, anchor *uuid.UUID) {
		payload := map[string]any{"statusEffectId": eff, "actionId": nil}
		if anchor != nil {
			payload["actionId"] = *anchor
		}
		sendWS(t, h.master, string(game.MsgTypeRelinkStatusEffect), payload)
	}

	t.Run("to a reaction, to the action and to none", func(t *testing.T) {
		h := newHiddenTable(t)
		turnID := h.openPCAttackOnNPC(t)
		actionID := lastTurnOpened(t, h.mc).ActionID
		seated := seatedOf(t, h, h.f.npcID)
		eff := h.applyAnchored(t, h.f.bystanderID, "prone", actionID)

		for _, step := range []struct {
			name   string
			anchor *uuid.UUID
			want   uuid.UUID
		}{
			{"to the seated reaction", &seated, seated},
			{"to the action", &actionID, actionID},
			{"to none", nil, uuid.Nil},
		} {
			before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
			relink(h, eff, step.anchor)
			awaitStatusCount(t, "master", h.mc, before+1)
			assertAnchor(t, step.name, masterAnchorOf(t, h, h.f.bystanderID, eff), step.want)
		}
		if got, ok := closeAnchors(t, h, turnID)[eff]; !ok || got != nil {
			t.Fatalf("close: anchor = %v (written %v), want the event unanchored", got, ok)
		}
	})

	t.Run("refusals", func(t *testing.T) {
		h := newHiddenTable(t)
		refuse := func(label, code string, eff uuid.UUID, anchor *uuid.UUID) {
			t.Helper()
			before := len(errorCodes(t, h.mc))
			relink(h, eff, anchor)
			chatBarrier(t, h.master, h.mc)
			codes := errorCodes(t, h.mc)
			if len(codes) != before+1 || codes[len(codes)-1] != code {
				t.Fatalf("%s: errors = %v, want a new %s", label, codes, code)
			}
		}
		old := applyProneAndWait(t, h.master, h.mc, h.f.victimID).StatusEffects[0].ID
		refuse("no open turn", "status_effect_invalid", old, nil)

		h.openPCAttackOnNPC(t)
		refuse("an effect with no change in this turn", "status_effect_not_found", old, nil)
		eff := h.applyAnchored(t, h.f.bystanderID, "prone", lastTurnOpened(t, h.mc).ActionID)
		foreign := uuid.New()
		refuse("a foreign action id", "status_effect_invalid", eff, &foreign)

		// O jogador não reancora.
		before := len(errorCodes(t, h.pc))
		sendWS(t, h.player, string(game.MsgTypeRelinkStatusEffect), map[string]any{"statusEffectId": eff})
		chatBarrier(t, h.master, h.pc)
		if codes := errorCodes(t, h.pc); len(codes) != before+1 || codes[len(codes)-1] != "forbidden" {
			t.Fatalf("player: errors = %v, want forbidden", codes)
		}
	})
}

func TestStatusAnchor_TurnActionIDIsTheMasters(t *testing.T) {
	h := newHiddenTable(t)
	h.openPCAttackOnNPC(t)
	actionID := lastTurnOpened(t, h.mc).ActionID
	eff := h.applyAnchored(t, h.f.bystanderID, "prone", actionID)
	assertAnchor(t, "master", masterAnchorOf(t, h, h.f.bystanderID, eff), actionID)
	awaitStatusCount(t, "bystander", h.bc, 1)
	chatBarrier(t, h.master, h.pc, h.bc)
	for who, c := range map[string]*collector{"player": h.pc, "bystander": h.bc} {
		for _, m := range c.snapshotMessages() {
			if strings.Contains(string(m.Payload), "turnActionId") {
				t.Fatalf("%s got turnActionId: %s %s", who, m.Type, m.Payload)
			}
		}
	}

	// Na reconexão do mestre também; o jogador, nunca.
	full := h.f.reconnectFullState(t, h.master, h.f.masterUUID)
	var found bool
	for _, e := range full[h.f.bystanderID] {
		if e.ID == eff {
			found = true
			assertAnchor(t, "master reconnect", e.TurnActionID, actionID)
		}
	}
	if !found {
		t.Fatal("master reconnect: the effect is missing")
	}
	for _, e := range h.f.reconnectFullState(t, h.blind, h.f.bystanderUUID)[h.f.bystanderID] {
		if e.TurnActionID != nil {
			t.Fatal("bystander reconnect got turnActionId")
		}
	}
}

func TestStatusAnchor_TurnActionIDLeavesWithTheClose(t *testing.T) {
	h := newHiddenTable(t)
	turnID := h.openPCAttackOnNPC(t)
	eff := h.applyAnchored(t, h.f.bystanderID, "prone", lastTurnOpened(t, h.mc).ActionID)
	before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
	closeTurnAndWait(t, h.master, h.mc)
	h.f.awaitPersistedTurn(t, turnID)
	awaitStatusCount(t, "master", h.mc, before+1)
	assertAnchor(t, "master after the close", masterAnchorOf(t, h, h.f.bystanderID, eff), uuid.Nil)
}

func TestStatusAnchor_InTurnUndo(t *testing.T) {
	t.Run("an apply undone in its turn writes nothing; an older effect's removal is recorded", func(t *testing.T) {
		h := newHiddenTable(t)
		old := applyProneAndWait(t, h.master, h.mc, h.f.victimID).StatusEffects[0].ID
		turnID := h.openPCAttackOnNPC(t)
		fresh := applyProneAndWait(t, h.master, h.mc, h.f.bystanderID).StatusEffects[0].ID
		for _, id := range []uuid.UUID{fresh, old} {
			before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
			sendWS(t, h.master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": id})
			awaitStatusCount(t, "master", h.mc, before+1)
		}
		// Ao vivo, as duas são remoções.
		chatBarrier(t, h.master, h.bc)
		if got := lastStatusUpdateFor(t, h.bc, h.f.bystanderID); len(got.StatusEffects) != 0 {
			t.Fatalf("bystander still sees %+v", got.StatusEffects)
		}
		closeTurnAndWait(t, h.master, h.mc)
		h.f.awaitPersistedTurn(t, turnID)
		ops := lastCloseStatusOps(t, h.f)
		if len(ops) != 1 || ops[0].Effect.ID != old || !ops[0].Delete || ops[0].Event == nil ||
			ops[0].Event.Change != statuseffect.ChangeRemoved {
			t.Fatalf("close ops = %v, want only the older effect's removal", opsOn(ops))
		}
	})

	t.Run("undoing unconscious keeps its prone", func(t *testing.T) {
		h := newHiddenTable(t)
		turnID := h.openPCAttackOnNPC(t)
		got := applyAndWait(t, h.master, h.mc, map[string]any{"characterIds": []string{h.f.bystanderID.String()}, "kind": "unconscious"})
		var unconscious, prone uuid.UUID
		for _, e := range got.StatusEffects {
			switch e.Kind {
			case "unconscious":
				unconscious = e.ID
			case "prone":
				prone = e.ID
			}
		}
		before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
		sendWS(t, h.master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": unconscious})
		awaitStatusCount(t, "master", h.mc, before+1)
		closeTurnAndWait(t, h.master, h.mc)
		h.f.awaitPersistedTurn(t, turnID)
		ops := lastCloseStatusOps(t, h.f)
		if len(ops) != 1 || ops[0].Effect.ID != prone || ops[0].Delete {
			t.Fatalf("close ops = %v, want only prone's apply", opsOn(ops))
		}
	})
}

// lastStatusUpdateFor é o último status_effects_updated de char num coletor.
func lastStatusUpdateFor(t *testing.T, c *collector, char uuid.UUID) game.StatusEffectsUpdatedPayload {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		msgs := c.snapshotMessages()
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Type != game.MsgTypeStatusEffectsUpdated {
				continue
			}
			if p := decodeStatusUpdate(t, msgs[i]); p.CharacterID == char {
				return p
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no status_effects_updated for %s", char)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func decodeStatusUpdate(t *testing.T, m game.Message) game.StatusEffectsUpdatedPayload {
	t.Helper()
	var p game.StatusEffectsUpdatedPayload
	if err := json.Unmarshal(m.Payload, &p); err != nil {
		t.Fatalf("unmarshal status_effects_updated: %v", err)
	}
	return p
}

// Um status que refaz a sentada (Atordoado: sem poder reagir, a sentada vira nothing) leva as
// âncoras presas à sentada velha para a nova.
func TestStatusAnchor_FollowsAReseatByStatus(t *testing.T) {
	h := newHiddenTable(t)
	turnID := h.openPCAttackOnNPC(t)
	seated := seatedOf(t, h, h.f.npcID)
	eff := h.applyAnchored(t, h.f.bystanderID, "prone", seated)

	before := h.mc.count(game.MsgTypeResolutionUpdate)
	sendApplyStatus(t, h.master, map[string]any{
		"characterIds": []string{h.f.npcID.String()}, "kind": "stunned", "params": map[string]any{"points": 3}})
	if !awaitCount(h.mc, game.MsgTypeResolutionUpdate, before+1, 2*time.Second) {
		t.Fatal("the reseat never reached the master")
	}
	reseated := seatedOf(t, h, h.f.npcID)
	if reseated == seated {
		t.Fatal("the stun did not reseat the NPC")
	}
	chatBarrier(t, h.master, h.mc)
	assertAnchor(t, "master after the reseat", masterAnchorOf(t, h, h.f.bystanderID, eff), reseated)
	assertAnchor(t, "close", closeAnchors(t, h, turnID)[eff], reseated)
}

// Desfazer um Escondido aplicado no turno é "como se nunca tivesse existido": o id dele some de
// todo registro do turno que será gravado — o hidden_from da action e das reações e o
// HiddenFrom dos outros eventos do turno.
func TestStatusAnchor_UndoingAHiddenPurgesItsID(t *testing.T) {
	h := newHiddenTable(t)
	turnID := h.openPCAttackOnNPC(t)
	seated := seatedOf(t, h, h.f.npcID)
	hid := h.hide(t, h.f.npcID, nil)
	h.openReaction(t, seated)
	applyProneAndWait(t, h.master, h.mc, h.f.npcID)
	before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
	sendWS(t, h.master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": hid})
	awaitStatusCount(t, "master", h.mc, before+1)
	closeTurnAndWait(t, h.master, h.mc)
	h.f.awaitPersistedTurn(t, turnID)

	cd := lastClose(t, h.f)
	if cd.ActionHiddenFrom != nil || len(cd.ReactionHiddenFrom) != 0 {
		t.Fatalf("hidden from = %v / %v, want no trace of the undone Hidden %s", cd.ActionHiddenFrom, cd.ReactionHiddenFrom, hid)
	}
	if len(cd.StatusEffects) != 1 || cd.StatusEffects[0].Event == nil {
		t.Fatalf("close ops = %v, want only prone's apply", opsOn(cd.StatusEffects))
	}
	for pid, eff := range cd.StatusEffects[0].Event.HiddenFrom {
		if eff == hid {
			t.Fatalf("prone's event HiddenFrom names the undone Hidden for %s", pid)
		}
	}
}
