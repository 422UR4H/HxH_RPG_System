package game_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Ações de um personagem Escondido (Task 15): quem não o percebe recebe o turn_opened, o
// reaction_opened e o openTurn reduzidos (sem ator, sem action, com actorHidden); um alvo
// Escondido sai de targetId e do resultado; a resolução assentada vai a todos (Q3); e o turno
// grava, por personagem, para quem cada um estava Escondido no instante.

// hiddenTable é a mesa sem tabuleiro (todos veem todos) com o NPC do mestre, o jogador (dono
// do atacante e da vítima) e o terceiro.
type hiddenTable struct {
	f                     *combatFixture
	master, player, blind *websocket.Conn
	mc, pc, bc            *collector
}

func newHiddenTable(t *testing.T) *hiddenTable {
	t.Helper()
	f := newCombatFixture(t, withBystander, withNPCTarget, withStatusEffectStore(&fakeStatusEffectStore{}))
	master, player, blind, mc, pc, bc := f.connectNoBoard(t)
	t.Cleanup(func() {
		_ = master.Close()
		_ = player.Close()
		_ = blind.Close()
	})
	return &hiddenTable{f: f, master: master, player: player, blind: blind, mc: mc, pc: pc, bc: bc}
}

// hide aplica Escondido em char (visibility nil = o padrão do tipo: só o mestre num NPC) e
// devolve o id do efeito.
func (h *hiddenTable) hide(t *testing.T, char uuid.UUID, visibility map[string]any) uuid.UUID {
	t.Helper()
	payload := map[string]any{"characterIds": []string{char.String()}, "kind": "hidden"}
	if visibility != nil {
		payload["visibility"] = visibility
	}
	got := applyAndWait(t, h.master, h.mc, payload)
	return got.StatusEffects[0].ID
}

// openNPCAttack: o NPC (do mestre) ataca a vítima do jogador e o turno abre para os três.
func (h *hiddenTable) openNPCAttack(t *testing.T) uuid.UUID {
	t.Helper()
	h.f.enqueueAttackAt(t, h.master, h.f.npcID, h.f.victimID)
	return h.openNext(t)
}

// openPCAttackOnNPC: o atacante do jogador ataca o NPC.
func (h *hiddenTable) openPCAttackOnNPC(t *testing.T) uuid.UUID {
	t.Helper()
	h.f.enqueueAttackAt(t, h.player, h.f.attackerID, h.f.npcID)
	return h.openNext(t)
}

func (h *hiddenTable) openNext(t *testing.T) uuid.UUID {
	t.Helper()
	if !h.mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the attack was never queued")
	}
	sendWS(t, h.master, string(game.MsgTypeOpenNextAction), struct{}{})
	for who, c := range map[string]*collector{"master": h.mc, "player": h.pc, "bystander": h.bc} {
		if !c.await(game.MsgTypeTurnOpened, 2*time.Second) {
			t.Fatalf("the %s never got turn_opened", who)
		}
	}
	return lastTurnOpened(t, h.mc).TurnID
}

// rawLast é o payload cru da última mensagem do tipo, chave a chave.
func rawLast(t *testing.T, who string, c *collector, typ game.MessageType) map[string]json.RawMessage {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != typ {
			continue
		}
		var out map[string]json.RawMessage
		if err := json.Unmarshal(msgs[i].Payload, &out); err != nil {
			t.Fatalf("%s: unmarshal %s: %v", who, typ, err)
		}
		return out
	}
	t.Fatalf("%s: no %s", who, typ)
	return nil
}

func assertReduced(t *testing.T, who string, p map[string]json.RawMessage, gone ...string) {
	t.Helper()
	for _, k := range gone {
		if _, ok := p[k]; ok {
			t.Fatalf("%s: key %q is on the wire: %s", who, k, p[k])
		}
	}
	if string(p["actorHidden"]) != "true" {
		t.Fatalf("%s: actorHidden = %s, want true", who, p["actorHidden"])
	}
	if _, ok := p["turnId"]; !ok {
		t.Fatalf("%s: no turnId", who)
	}
}

func assertWhole(t *testing.T, who string, p map[string]json.RawMessage, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := p[k]; !ok {
			t.Fatalf("%s: key %q is missing: %v", who, k, p)
		}
	}
	if _, ok := p["actorHidden"]; ok {
		t.Fatalf("%s: actorHidden is on the wire of a whole message", who)
	}
}

// targetIDsIn lê o targetId de um action cru.
func targetIDsIn(t *testing.T, raw json.RawMessage) []uuid.UUID {
	t.Helper()
	var a struct {
		TargetID []uuid.UUID `json:"targetId"`
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("unmarshal action: %v", err)
	}
	return a.TargetID
}

func TestHiddenActor_TurnOpenedIsReduced(t *testing.T) {
	h := newHiddenTable(t)
	h.hide(t, h.f.npcID, nil)
	h.openNPCAttack(t)

	assertWhole(t, "master", rawLast(t, "master", h.mc, game.MsgTypeTurnOpened), "actorId", "actionId", "action")
	for who, c := range map[string]*collector{"victim owner": h.pc, "bystander": h.bc} {
		assertReduced(t, who, rawLast(t, who, c, game.MsgTypeTurnOpened), "actorId", "actionId", "action")
	}
}

func TestHiddenActor_ListedPlayerSeesItWhole(t *testing.T) {
	h := newHiddenTable(t)
	h.hide(t, h.f.npcID, map[string]any{"mode": "players", "playerIds": []string{h.f.bystanderUUID.String()}})
	h.openNPCAttack(t)

	assertWhole(t, "bystander", rawLast(t, "bystander", h.bc, game.MsgTypeTurnOpened), "actorId", "actionId", "action")
	assertReduced(t, "victim owner", rawLast(t, "victim owner", h.pc, game.MsgTypeTurnOpened), "actorId", "actionId", "action")
}

func TestHiddenActor_HiddenTargetLeavesTargetIDs(t *testing.T) {
	h := newHiddenTable(t)
	h.hide(t, h.f.npcID, map[string]any{"mode": "players", "playerIds": []string{h.f.playerUUID.String()}})
	h.openPCAttackOnNPC(t)

	// O ator é visível: o turno chega inteiro a todos; só o alvo Escondido sai do terceiro.
	for who, c := range map[string]*collector{"master": h.mc, "player": h.pc} {
		p := rawLast(t, who, c, game.MsgTypeTurnOpened)
		assertWhole(t, who, p, "actorId", "actionId", "action")
		if got := targetIDsIn(t, p["action"]); !slices.Equal(got, []uuid.UUID{h.f.npcID}) {
			t.Fatalf("%s: targetId = %v, want the NPC", who, got)
		}
	}
	p := rawLast(t, "bystander", h.bc, game.MsgTypeTurnOpened)
	assertWhole(t, "bystander", p, "actorId", "actionId", "action")
	if got := targetIDsIn(t, p["action"]); len(got) != 0 {
		t.Fatalf("bystander: targetId = %v, want the hidden NPC cut", got)
	}
}

// attachNPCDodge: o mestre responde pelo NPC com uma esquiva e devolve o id da reação.
func (h *hiddenTable) attachNPCDodge(t *testing.T, actionID uuid.UUID) uuid.UUID {
	t.Helper()
	before := h.mc.count(game.MsgTypeReactionAttached)
	sendWS(t, h.master, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: h.f.npcID, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	if !awaitCount(h.mc, game.MsgTypeReactionAttached, before+1, 2*time.Second) {
		t.Fatalf("no reaction_attached; got %v", messageTypes(h.mc.snapshotMessages()))
	}
	return lastReactionAttached(t, h.mc).ReactionID
}

func (h *hiddenTable) openReaction(t *testing.T, reactionID uuid.UUID) {
	t.Helper()
	sendWS(t, h.master, string(game.MsgTypeOpenReaction), game.OpenReactionPayload{ReactionID: reactionID})
	for who, c := range map[string]*collector{"master": h.mc, "player": h.pc, "bystander": h.bc} {
		if !c.await(game.MsgTypeReactionOpened, 2*time.Second) {
			t.Fatalf("the %s never got reaction_opened", who)
		}
	}
}

func TestHiddenActor_ReactionOpenedIsReduced(t *testing.T) {
	h := newHiddenTable(t)
	h.hide(t, h.f.npcID, map[string]any{"mode": "players", "playerIds": []string{h.f.playerUUID.String()}})
	h.openPCAttackOnNPC(t)
	reactionID := h.attachNPCDodge(t, lastTurnOpened(t, h.mc).ActionID)
	h.openReaction(t, reactionID)

	assertWhole(t, "master", rawLast(t, "master", h.mc, game.MsgTypeReactionOpened), "reactionId", "reaction")
	assertWhole(t, "player", rawLast(t, "player", h.pc, game.MsgTypeReactionOpened), "reactionId", "reaction")
	assertReduced(t, "bystander", rawLast(t, "bystander", h.bc, game.MsgTypeReactionOpened), "reactionId", "reaction")
}

func settledTargets(t *testing.T, who string, c *collector) []uuid.UUID {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeResolutionUpdate {
			continue
		}
		var p game.ResolutionUpdatedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatal(err)
		}
		if !p.IsSettled {
			continue
		}
		out := make([]uuid.UUID, 0, len(p.Targets))
		for _, tr := range p.Targets {
			out = append(out, tr.TargetID)
		}
		return out
	}
	t.Fatalf("%s: no settled resolution_updated; got %v", who, messageTypes(msgs))
	return nil
}

func TestHiddenActor_SettledResolution(t *testing.T) {
	t.Run("a hidden actor's settled resolution reaches every reader", func(t *testing.T) {
		h := newHiddenTable(t)
		h.hide(t, h.f.npcID, nil)
		h.openNPCAttack(t)
		closeTurnAndWait(t, h.master, h.mc)
		for who, c := range map[string]*collector{"master": h.mc, "victim owner": h.pc, "bystander": h.bc} {
			if !c.await(game.MsgTypeTurnClosed, 2*time.Second) {
				t.Fatalf("%s: no turn_closed", who)
			}
			chatBarrier(t, h.master, c)
			if got := settledTargets(t, who, c); !slices.Equal(got, []uuid.UUID{h.f.victimID}) {
				t.Fatalf("%s: targets = %v, want the victim", who, got)
			}
		}
	})

	t.Run("a hidden target's result leaves the reader who does not perceive it", func(t *testing.T) {
		h := newHiddenTable(t)
		h.hide(t, h.f.npcID, map[string]any{"mode": "players", "playerIds": []string{h.f.playerUUID.String()}})
		h.openPCAttackOnNPC(t)
		closeTurnAndWait(t, h.master, h.mc)
		chatBarrier(t, h.master, h.mc, h.pc, h.bc)
		for who, c := range map[string]*collector{"master": h.mc, "player": h.pc} {
			if got := settledTargets(t, who, c); !slices.Equal(got, []uuid.UUID{h.f.npcID}) {
				t.Fatalf("%s: targets = %v, want the NPC", who, got)
			}
		}
		if got := settledTargets(t, "bystander", h.bc); len(got) != 0 {
			t.Fatalf("bystander: targets = %v, want the hidden NPC cut", got)
		}
	})
}

// reconnectOpenTurn fecha a conexão de who, liga de novo e devolve o openTurn cru.
func (f *combatFixture) reconnectOpenTurn(t *testing.T, old *websocket.Conn, who uuid.UUID) map[string]json.RawMessage {
	t.Helper()
	_ = old.Close()
	conn := connectWS(t, f.server.URL, who, f.matchUUID)
	t.Cleanup(func() { _ = conn.Close() })
	c := collectFrom(conn)
	if !c.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("no match_full_state on reconnect")
	}
	p := rawLast(t, "reconnect", c, game.MsgTypeMatchFullState)
	var ot map[string]json.RawMessage
	if err := json.Unmarshal(p["openTurn"], &ot); err != nil || ot == nil {
		t.Fatalf("openTurn = %s (%v)", p["openTurn"], err)
	}
	return ot
}

func TestHiddenActor_ReconnectOpenTurnIsReduced(t *testing.T) {
	t.Run("a hidden actor's open turn is reduced", func(t *testing.T) {
		h := newHiddenTable(t)
		h.hide(t, h.f.npcID, nil)
		h.openNPCAttack(t)
		assertReduced(t, "bystander", h.f.reconnectOpenTurn(t, h.blind, h.f.bystanderUUID), "actorId", "actionId", "action")
		assertWhole(t, "master", h.f.reconnectOpenTurn(t, h.master, h.f.masterUUID), "actorId", "actionId", "action")
	})

	t.Run("a hidden reactor's opened reaction and a hidden target are left out", func(t *testing.T) {
		h := newHiddenTable(t)
		h.hide(t, h.f.npcID, map[string]any{"mode": "players", "playerIds": []string{h.f.playerUUID.String()}})
		h.openPCAttackOnNPC(t)
		h.openReaction(t, h.attachNPCDodge(t, lastTurnOpened(t, h.mc).ActionID))

		ot := h.f.reconnectOpenTurn(t, h.blind, h.f.bystanderUUID)
		assertWhole(t, "bystander", ot, "actorId", "actionId", "action")
		if got := targetIDsIn(t, ot["action"]); len(got) != 0 {
			t.Fatalf("bystander: targetId = %v, want the hidden NPC cut", got)
		}
		if _, ok := ot["reactions"]; ok {
			t.Fatalf("bystander: reactions = %s, want the hidden reactor's left out", ot["reactions"])
		}
		ot = h.f.reconnectOpenTurn(t, h.player, h.f.playerUUID)
		var reactions []json.RawMessage
		if err := json.Unmarshal(ot["reactions"], &reactions); err != nil || len(reactions) != 1 {
			t.Fatalf("player: reactions = %s, want the NPC's", ot["reactions"])
		}
	})
}

func lastClose(t *testing.T, f *combatFixture) appmatch.TurnCloseData {
	t.Helper()
	closes := f.roundRepo.closeData()
	if len(closes) == 0 {
		t.Fatal("no PersistTurnClose succeeded")
	}
	return closes[len(closes)-1]
}

func TestHiddenActor_RecordsHiddenFrom(t *testing.T) {
	t.Run("a hidden actor is recorded for every player but the master", func(t *testing.T) {
		h := newHiddenTable(t)
		hid := h.hide(t, h.f.npcID, nil)
		turnID := h.openNPCAttack(t)
		closeTurnAndWait(t, h.master, h.mc)
		h.f.awaitPersistedTurn(t, turnID)
		got := lastClose(t, h.f).ActionHiddenFrom
		want := map[uuid.UUID]uuid.UUID{h.f.playerUUID: hid, h.f.bystanderUUID: hid}
		if len(got) != 1 || len(got[h.f.npcID]) != 2 ||
			got[h.f.npcID][h.f.playerUUID] != hid || got[h.f.npcID][h.f.bystanderUUID] != hid {
			t.Fatalf("ActionHiddenFrom = %v, want {npc: %v}", got, want)
		}
	})

	t.Run("a hidden target and its reactions are recorded at the opening or at the close", func(t *testing.T) {
		h := newHiddenTable(t)
		hid := h.hide(t, h.f.npcID, map[string]any{"mode": "players", "playerIds": []string{h.f.playerUUID.String()}})
		turnID := h.openPCAttackOnNPC(t)
		var seatedID uuid.UUID
		for _, pr := range lastResolutionUpdated(t, h.mc).PendingReactions {
			if pr.ActorID == h.f.npcID && pr.Seated {
				seatedID = pr.ReactionID
			}
		}
		if seatedID == uuid.Nil {
			t.Fatal("no seated reaction for the NPC")
		}
		// O mestre abre a reação sentada: o reator é registrado na abertura dela. Depois a lista
		// do Escondido muda (o jogador também sai dela): o registro da abertura é o que fica.
		h.openReaction(t, seatedID)
		before := h.mc.count(game.MsgTypeStatusEffectsUpdated)
		sendWS(t, h.master, string(game.MsgTypeEditStatusEffect), map[string]any{
			"statusEffectId": hid, "visibility": map[string]any{"mode": "players", "playerIds": []string{}}})
		awaitStatusCount(t, "master", h.mc, before+1)
		closeTurnAndWait(t, h.master, h.mc)
		h.f.awaitPersistedTurn(t, turnID)
		cd := lastClose(t, h.f)
		if len(cd.ActionHiddenFrom) != 1 || len(cd.ActionHiddenFrom[h.f.npcID]) != 1 || cd.ActionHiddenFrom[h.f.npcID][h.f.bystanderUUID] != hid {
			t.Fatalf("ActionHiddenFrom = %v, want {npc: {bystander: %s}}", cd.ActionHiddenFrom, hid)
		}
		r := cd.ReactionHiddenFrom[seatedID]
		if len(cd.ReactionHiddenFrom) != 1 || len(r) != 1 || len(r[h.f.npcID]) != 1 || r[h.f.npcID][h.f.bystanderUUID] != hid {
			t.Fatalf("ReactionHiddenFrom = %v, want {%s: {npc: {bystander: %s}}}", cd.ReactionHiddenFrom, seatedID, hid)
		}
	})

	t.Run("a never-opened seated reaction is recorded at the close", func(t *testing.T) {
		h := newHiddenTable(t)
		turnID := h.openPCAttackOnNPC(t)
		var seatedID uuid.UUID
		for _, pr := range lastResolutionUpdated(t, h.mc).PendingReactions {
			if pr.ActorID == h.f.npcID && pr.Seated {
				seatedID = pr.ReactionID
			}
		}
		// Escondido aplicado DEPOIS da abertura: o alvo não estava Escondido no instante da
		// abertura, mas o reator está no fechamento, que é quando a reação nunca aberta é
		// registrada.
		hid := h.hide(t, h.f.npcID, nil)
		closeTurnAndWait(t, h.master, h.mc)
		h.f.awaitPersistedTurn(t, turnID)
		cd := lastClose(t, h.f)
		if cd.ActionHiddenFrom != nil {
			t.Fatalf("ActionHiddenFrom = %v, want nil — nobody was hidden at the opening", cd.ActionHiddenFrom)
		}
		r := cd.ReactionHiddenFrom[seatedID]
		if len(r) != 1 || len(r[h.f.npcID]) != 2 || r[h.f.npcID][h.f.playerUUID] != hid || r[h.f.npcID][h.f.bystanderUUID] != hid {
			t.Fatalf("ReactionHiddenFrom = %v, want {%s: {npc: {player, bystander: %s}}}", cd.ReactionHiddenFrom, seatedID, hid)
		}
	})

	// Um PC Escondido (padrão do tipo: o mestre e o dono): o dono nunca entra no registro.
	t.Run("a hidden PC never records its owner", func(t *testing.T) {
		h := newHiddenTable(t)
		hid := h.hide(t, h.f.victimID, nil)
		turnID := h.openNPCAttack(t)
		closeTurnAndWait(t, h.master, h.mc)
		h.f.awaitPersistedTurn(t, turnID)
		got := lastClose(t, h.f).ActionHiddenFrom
		if len(got) != 1 || len(got[h.f.victimID]) != 1 || got[h.f.victimID][h.f.bystanderUUID] != hid {
			t.Fatalf("ActionHiddenFrom = %v, want {victim: {bystander: %s}} — the owner %s never in it",
				got, hid, h.f.playerUUID)
		}
	})

	t.Run("nobody hidden records nothing", func(t *testing.T) {
		h := newHiddenTable(t)
		turnID := h.openPCAttackOnNPC(t)
		closeTurnAndWait(t, h.master, h.mc)
		h.f.awaitPersistedTurn(t, turnID)
		cd := lastClose(t, h.f)
		if cd.ActionHiddenFrom != nil || len(cd.ReactionHiddenFrom) != 0 {
			t.Fatalf("hidden from = %v / %v, want nil", cd.ActionHiddenFrom, cd.ReactionHiddenFrom)
		}
	})
}

// hasReactToID diz se a chave reactToId está numa reação crua.
func hasReactToID(t *testing.T, raw json.RawMessage) bool {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal reaction: %v", err)
	}
	_, ok := m["reactToId"]
	return ok
}

// A vítima (PC visível) responde a um NPC Escondido: a reação dela chega inteira a quem a
// percebe, mas sem reactToId (o id da action escondida), ao vivo e na reconexão — e o openTurn
// reduzido ainda a traz.
func TestHiddenActor_VisibleReactionToAHiddenActor(t *testing.T) {
	h := newHiddenTable(t)
	h.hide(t, h.f.npcID, nil)
	h.openNPCAttack(t)
	actionID := lastTurnOpened(t, h.mc).ActionID
	before := h.mc.count(game.MsgTypeReactionAttached)
	sendWS(t, h.player, string(game.MsgTypeAttachReaction), game.ActionPayload{
		ActorID: h.f.victimID, ReactToID: actionID, ReactionKind: "dodge", Dodge: &game.DodgePayload{},
	})
	if !awaitCount(h.mc, game.MsgTypeReactionAttached, before+1, 2*time.Second) {
		t.Fatalf("no reaction_attached; got %v", messageTypes(h.mc.snapshotMessages()))
	}
	h.openReaction(t, lastReactionAttached(t, h.mc).ReactionID)

	if p := rawLast(t, "master", h.mc, game.MsgTypeReactionOpened); !hasReactToID(t, p["reaction"]) {
		t.Fatal("master: reactToId is missing")
	}
	p := rawLast(t, "bystander", h.bc, game.MsgTypeReactionOpened)
	assertWhole(t, "bystander", p, "reactionId", "reaction")
	if hasReactToID(t, p["reaction"]) {
		t.Fatalf("bystander: reactToId names the hidden action: %s", p["reaction"])
	}

	ot := h.f.reconnectOpenTurn(t, h.blind, h.f.bystanderUUID)
	assertReduced(t, "bystander openTurn", ot, "actorId", "actionId", "action")
	var reactions []json.RawMessage
	if err := json.Unmarshal(ot["reactions"], &reactions); err != nil || len(reactions) != 1 {
		t.Fatalf("bystander: openTurn.reactions = %s, want the victim's", ot["reactions"])
	}
	if hasReactToID(t, reactions[0]) {
		t.Fatalf("bystander: openTurn.reactions[0] names the hidden action: %s", reactions[0])
	}
}
