package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Efeitos de status ao vivo (Task 8): os três comandos do mestre e o status_effects_updated
// projetado por destinatário — a lista de quem vê (statuseffect.Listed) somada à visão
// (névoa): o jogador só recebe os efeitos de um personagem que enxerga; o dono e o mestre
// sempre enxergam. Mesa: o PC (victimID) é do jogador do fixture, o bystander tem o seu, e o
// NPC é do mestre.

// withStatusEffects dá à sala o caso de uso real dos efeitos de status.
var withStatusEffects = withRoomDeps(func(d *game.RoomDeps) {
	d.StatusEffectsUC = appmatch.NewStatusEffectsUC()
})

const (
	statusVictimPieceID = "piece-status-victim"
	statusNPCPieceID    = "piece-status-npc"
)

// statusNPCInSight põe o NPC à vista de todos (nenhuma parede no meio); statusNPCBehindWall o
// põe na metade esquerda do moveBoardWall, longe do bystander (20,4) e das peças do jogador.
var (
	statusNPCInSight    = [2]int{16, 4}
	statusNPCBehindWall = [2]int{4, 4}
)

// seedStatusBoard monta o tabuleiro: atacante (14,4) e vítima (18,4) do jogador, bystander
// (20,4) e o NPC em npcAt; walls separam quem a cena pedir.
func (f *combatFixture) seedStatusBoard(t *testing.T, npcAt [2]int, walls ...mapentity.WallSegment) {
	t.Helper()
	sq := func(c, r int) mapentity.PieceCoord {
		return mapentity.PieceCoord{Slot: mapentity.SquareCoord{Kind: "square", Col: c, Row: r}}
	}
	pieces := []mapentity.Piece{
		{ID: attackerPieceID, CharacterID: f.attackerID.String(), Coord: sq(14, 4), Visible: true},
		{ID: statusVictimPieceID, CharacterID: f.victimID.String(), Coord: sq(18, 4), Visible: true},
	}
	if f.npcID != uuid.Nil {
		pieces = append(pieces, mapentity.Piece{
			ID: statusNPCPieceID, CharacterID: f.npcID.String(), Coord: sq(npcAt[0], npcAt[1]), Visible: true,
		})
	}
	f.seedHiddenMoveBoard(t, pieces, walls...)
}

func newStatusFixture(t *testing.T, opts ...combatOpt) *combatFixture {
	t.Helper()
	return newCombatFixture(t, append([]combatOpt{withBystander, withNPCTarget, withStatusEffects}, opts...)...)
}

func sendApplyStatus(t *testing.T, conn *websocket.Conn, payload map[string]any) {
	t.Helper()
	sendWS(t, conn, string(game.MsgTypeApplyStatusEffect), payload)
}

// statusUpdates devolve, em ordem, todo status_effects_updated que o coletor recebeu.
func statusUpdates(t *testing.T, c *collector) []game.StatusEffectsUpdatedPayload {
	t.Helper()
	var out []game.StatusEffectsUpdatedPayload
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeStatusEffectsUpdated {
			continue
		}
		var p game.StatusEffectsUpdatedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal status_effects_updated: %v", err)
		}
		out = append(out, p)
	}
	return out
}

func lastStatusUpdate(t *testing.T, c *collector) game.StatusEffectsUpdatedPayload {
	t.Helper()
	all := statusUpdates(t, c)
	if len(all) == 0 {
		t.Fatal("no status_effects_updated arrived")
	}
	return all[len(all)-1]
}

// awaitStatusCount espera n status_effects_updated no coletor.
func awaitStatusCount(t *testing.T, who string, c *collector, n int) {
	t.Helper()
	if !awaitCount(c, game.MsgTypeStatusEffectsUpdated, n, 2*time.Second) {
		t.Fatalf("%s: got %d status_effects_updated, want %d", who, c.count(game.MsgTypeStatusEffectsUpdated), n)
	}
}

// assertStatusCount confere a contagem exata depois de uma barreira.
func assertStatusCount(t *testing.T, who string, c *collector, n int) {
	t.Helper()
	if got := c.count(game.MsgTypeStatusEffectsUpdated); got != n {
		t.Fatalf("%s: got %d status_effects_updated, want %d", who, got, n)
	}
}

func kindsOf(list []game.StatusEffectPayload) []string {
	out := make([]string, 0, len(list))
	for _, e := range list {
		out = append(out, e.Kind)
	}
	return out
}

func assertKinds(t *testing.T, who string, list []game.StatusEffectPayload, want ...string) {
	t.Helper()
	got := kindsOf(list)
	if len(got) != len(want) {
		t.Fatalf("%s: kinds = %v, want %v", who, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: kinds = %v, want %v", who, got, want)
		}
	}
}

func TestStatusEffects_ApplyPublic(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "prone"})
	awaitStatusCount(t, "master", mc, 1)
	awaitStatusCount(t, "owner", pc, 1)
	awaitStatusCount(t, "bystander", bc, 1)
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "master", mc, 1)
	assertStatusCount(t, "owner", pc, 1)
	assertStatusCount(t, "bystander", bc, 1)

	m := lastStatusUpdate(t, mc)
	if m.CharacterID != f.victimID {
		t.Fatalf("master: characterId = %s, want the victim", m.CharacterID)
	}
	assertKinds(t, "master", m.StatusEffects, "prone")
	e := m.StatusEffects[0]
	if e.Visibility == nil || e.Visibility.Mode != "default" {
		t.Fatalf("master: visibility = %+v, want mode default", e.Visibility)
	}
	if e.Public == nil || !*e.Public {
		t.Fatalf("master: public = %v, want true", e.Public)
	}
	if e.ID == uuid.Nil || e.AppliedAt.IsZero() {
		t.Fatalf("master: id/appliedAt missing: %+v", e)
	}
	for who, c := range map[string]*collector{"owner": pc, "bystander": bc} {
		p := lastStatusUpdate(t, c)
		assertKinds(t, who, p.StatusEffects, "prone")
		if p.StatusEffects[0].Visibility != nil || p.StatusEffects[0].Public != nil {
			t.Fatalf("%s: a player got the viewer list: %+v", who, p.StatusEffects[0])
		}
	}
}

func TestStatusEffects_ApplyHiddenOnPC(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "poisoned"})
	awaitStatusCount(t, "master", mc, 1)
	awaitStatusCount(t, "owner", pc, 1)
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "bystander", bc, 0)

	m := lastStatusUpdate(t, mc).StatusEffects
	assertKinds(t, "master", m, "poisoned")
	if m[0].Params.HPPerRound == nil || *m[0].Params.HPPerRound != 1 {
		t.Fatalf("master: hpPerRound = %v, want 1", m[0].Params.HPPerRound)
	}
	if m[0].Public == nil || *m[0].Public {
		t.Fatalf("master: public = %v, want false", m[0].Public)
	}
	assertKinds(t, "owner", lastStatusUpdate(t, pc).StatusEffects, "poisoned")
}

func TestStatusEffects_ApplyHiddenOnNPC(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.npcID.String()}, "kind": "poisoned"})
	awaitStatusCount(t, "master", mc, 1)
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "player", pc, 0)
	assertStatusCount(t, "bystander", bc, 0)
}

func TestStatusEffects_VisibilityChange(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "poisoned"})
	awaitStatusCount(t, "master", mc, 1)
	awaitStatusCount(t, "owner", pc, 1)
	chatBarrier(t, master, mc, pc, bc)
	id := lastStatusUpdate(t, mc).StatusEffects[0].ID

	// Escondido → público: o bystander passa a vê-lo; o dono já o via, nada muda para ele.
	sendWS(t, master, string(game.MsgTypeEditStatusEffect), map[string]any{
		"statusEffectId": id, "visibility": map[string]any{"mode": "public"},
	})
	awaitStatusCount(t, "master", mc, 2)
	awaitStatusCount(t, "bystander", bc, 1)
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "owner", pc, 1)
	assertKinds(t, "bystander", lastStatusUpdate(t, bc).StatusEffects, "poisoned")
	if v := lastStatusUpdate(t, mc).StatusEffects[0].Visibility; v == nil || v.Mode != "public" {
		t.Fatalf("master: visibility = %+v, want public", v)
	}

	// Público → só o mestre: dono e bystander recebem a lista vazia.
	sendWS(t, master, string(game.MsgTypeEditStatusEffect), map[string]any{
		"statusEffectId": id, "visibility": map[string]any{"mode": "players", "playerIds": []string{}},
	})
	awaitStatusCount(t, "master", mc, 3)
	awaitStatusCount(t, "owner", pc, 2)
	awaitStatusCount(t, "bystander", bc, 2)
	for who, c := range map[string]*collector{"owner": pc, "bystander": bc} {
		p := lastStatusUpdate(t, c)
		if p.StatusEffects == nil || len(p.StatusEffects) != 0 {
			t.Fatalf("%s: statusEffects = %v, want []", who, p.StatusEffects)
		}
	}
	if v := lastStatusUpdate(t, mc).StatusEffects[0].Visibility; v == nil || v.Mode != "players" || len(v.PlayerIDs) != 0 {
		t.Fatalf("master: visibility = %+v, want players []", v)
	}

	// Só o apelido: ninguém além do mestre o vê.
	sendWS(t, master, string(game.MsgTypeEditStatusEffect), map[string]any{"statusEffectId": id, "alias": "Peçonha"})
	awaitStatusCount(t, "master", mc, 4)
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "owner", pc, 2)
	assertStatusCount(t, "bystander", bc, 2)
	if a := lastStatusUpdate(t, mc).StatusEffects[0].Alias; a != "Peçonha" {
		t.Fatalf("master: alias = %q", a)
	}
}

func TestStatusEffects_PublicNeedsSight(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCBehindWall, moveBoardWall)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.npcID.String()}, "kind": "prone"})
	awaitStatusCount(t, "master", mc, 1)
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "master", mc, 1)
	assertStatusCount(t, "bystander", bc, 0)
	assertStatusCount(t, "player", pc, 0)
}

// connectNoBoard liga a mesa sem tabuleiro: espera o match_full_state de cada um, em vez do
// map_full_state que connectTable espera.
func (f *combatFixture) connectNoBoard(t *testing.T) (master, player, blind *websocket.Conn, mc, pc, bc *collector) {
	t.Helper()
	master, player = f.connect(t)
	blind = connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	readMessage(t, blind) // room_state
	mc, pc, bc = collectFrom(master), collectFrom(player), collectFrom(blind)
	for name, c := range map[string]*collector{"master": mc, "player": pc, "bystander": bc} {
		if !c.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatalf("the %s never got match_full_state", name)
		}
	}
	return master, player, blind, mc, pc, bc
}

func TestStatusEffects_NoBoardEveryoneSees(t *testing.T) {
	f := newStatusFixture(t)
	master, player, blind, mc, pc, bc := f.connectNoBoard(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.npcID.String()}, "kind": "prone"})
	awaitStatusCount(t, "master", mc, 1)
	awaitStatusCount(t, "bystander", bc, 1)
	awaitStatusCount(t, "player", pc, 1)
	assertKinds(t, "bystander", lastStatusUpdate(t, bc).StatusEffects, "prone")
}

func TestStatusEffects_Remove(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "prone"})
	awaitStatusCount(t, "master", mc, 1)
	awaitStatusCount(t, "owner", pc, 1)
	awaitStatusCount(t, "bystander", bc, 1)
	id := lastStatusUpdate(t, mc).StatusEffects[0].ID

	sendWS(t, master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": id})
	for who, c := range map[string]*collector{"master": mc, "owner": pc, "bystander": bc} {
		awaitStatusCount(t, who, c, 2)
		p := lastStatusUpdate(t, c)
		if p.CharacterID != f.victimID || p.StatusEffects == nil || len(p.StatusEffects) != 0 {
			t.Fatalf("%s: got %+v, want the victim with []", who, p)
		}
	}
}

func TestStatusEffects_MultiTargetAllOrNothing(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "prone"})
	awaitStatusCount(t, "master", mc, 1)

	// O atacante não tem; a vítima já tem: nada muda para nenhum dos dois.
	sendApplyStatus(t, master, map[string]any{
		"characterIds": []string{f.attackerID.String(), f.victimID.String()}, "kind": "prone",
	})
	if ep := awaitErrorWithCode(t, mc, "status_effect_exists", 2*time.Second); ep.Message == "" {
		t.Fatal("status_effect_exists with no message")
	}
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "master", mc, 1)

	// Com os dois livres, os dois recebem.
	sendApplyStatus(t, master, map[string]any{
		"characterIds": []string{f.attackerID.String(), f.npcID.String()}, "kind": "airborne",
	})
	awaitStatusCount(t, "master", mc, 3)
	got := map[uuid.UUID]bool{}
	for _, u := range statusUpdates(t, mc)[1:] {
		got[u.CharacterID] = true
	}
	if !got[f.attackerID] || !got[f.npcID] {
		t.Fatalf("master updates = %v, want attacker and npc", got)
	}
}

func TestStatusEffects_Refusals(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, player, map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "prone"})
	awaitErrorWithCode(t, pc, "forbidden", 2*time.Second)

	tests := []struct {
		name    string
		msgType game.MessageType
		payload map[string]any
		code    string
	}{
		{"unknown kind", game.MsgTypeApplyStatusEffect,
			map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "flying"}, "status_effect_invalid"},
		{"param foreign to the kind", game.MsgTypeApplyStatusEffect,
			map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "prone", "params": map[string]any{"points": 2}},
			"status_effect_invalid"},
		{"unknown character", game.MsgTypeApplyStatusEffect,
			map[string]any{"characterIds": []string{uuid.New().String()}, "kind": "prone"}, "unknown_character"},
		{"empty character ids", game.MsgTypeApplyStatusEffect,
			map[string]any{"characterIds": []string{}, "kind": "prone"}, "invalid_payload"},
		{"repeated character ids", game.MsgTypeApplyStatusEffect,
			map[string]any{"characterIds": []string{f.victimID.String(), f.victimID.String()}, "kind": "prone"}, "invalid_payload"},
		{"unknown sense", game.MsgTypeApplyStatusEffect,
			map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "sense", "sense": "sixth",
				"params": map[string]any{"level": "impaired"}}, "invalid_payload"},
		{"unknown visibility mode", game.MsgTypeApplyStatusEffect,
			map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "prone",
				"visibility": map[string]any{"mode": "some"}}, "invalid_payload"},
		{"audience player outside the match", game.MsgTypeApplyStatusEffect,
			map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "prone",
				"visibility": map[string]any{"mode": "players", "playerIds": []string{uuid.New().String()}}}, "status_effect_invalid"},
		{"edit unknown id", game.MsgTypeEditStatusEffect,
			map[string]any{"statusEffectId": uuid.New().String(), "alias": "x"}, "status_effect_not_found"},
		{"remove unknown id", game.MsgTypeRemoveStatusEffect,
			map[string]any{"statusEffectId": uuid.New().String()}, "status_effect_not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := mc.count(game.MsgTypeError)
			sendWS(t, master, string(tt.msgType), tt.payload)
			if !awaitCount(mc, game.MsgTypeError, before+1, 2*time.Second) {
				t.Fatal("no error arrived")
			}
			if got := lastError(t, mc); got.Code != tt.code {
				t.Fatalf("code = %q (%s), want %q", got.Code, got.Message, tt.code)
			}
		})
	}
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "master", mc, 0)

	t.Run("lobby", func(t *testing.T) {
		lf := newStatusFixture(t, inLobby)
		lm := connectWS(t, lf.server.URL, lf.masterUUID, lf.matchUUID)
		defer lm.Close()   //nolint:errcheck
		readMessage(t, lm) // room_state
		lc := collectFrom(lm)
		sendApplyStatus(t, lm, map[string]any{"characterIds": []string{lf.victimID.String()}, "kind": "prone"})
		awaitErrorWithCode(t, lc, "match_not_started", 2*time.Second)
	})
}

// reconnectFullState fecha a conexão de who, liga de novo e devolve o statusEffects do
// match_full_state que a nova conexão recebe.
func (f *combatFixture) reconnectFullState(t *testing.T, old *websocket.Conn, who uuid.UUID) map[uuid.UUID][]game.StatusEffectPayload {
	t.Helper()
	_ = old.Close()
	conn := connectWS(t, f.server.URL, who, f.matchUUID)
	t.Cleanup(func() { _ = conn.Close() })
	c := collectFrom(conn)
	if !c.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("no match_full_state on reconnect")
	}
	var p game.MatchFullStatePayload
	if err := json.Unmarshal(findMessage(t, c.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &p); err != nil {
		t.Fatalf("unmarshal match_full_state: %v", err)
	}
	return p.StatusEffects
}

func TestStatusEffects_Reconnect(t *testing.T) {
	for _, tc := range []struct {
		name    string
		npcAt   [2]int
		walls   []mapentity.WallSegment
		inSight bool
	}{
		{"npc in sight", statusNPCInSight, nil, true},
		{"npc out of sight", statusNPCBehindWall, []mapentity.WallSegment{moveBoardWall}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newStatusFixture(t)
			f.seedStatusBoard(t, tc.npcAt, tc.walls...)
			master, player, blind, mc, _, _ := f.connectTable(t)

			sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.npcID.String()}, "kind": "prone"})
			sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.npcID.String()}, "kind": "poisoned"})
			sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.victimID.String()}, "kind": "poisoned"})
			awaitStatusCount(t, "master", mc, 3)

			// bystander: só o público do NPC, e só se o vê.
			got := f.reconnectFullState(t, blind, f.bystanderUUID)
			if tc.inSight {
				if len(got) != 1 {
					t.Fatalf("bystander: statusEffects = %v, want only the npc", got)
				}
				assertKinds(t, "bystander npc", got[f.npcID], "prone")
				if got[f.npcID][0].Visibility != nil || got[f.npcID][0].Public != nil {
					t.Fatal("bystander: got the viewer list")
				}
			} else if len(got) != 0 {
				t.Fatalf("bystander: statusEffects = %v, want none", got)
			}

			// dono: o público do NPC (se o vê) e o escondido do próprio PC.
			got = f.reconnectFullState(t, player, f.playerUUID)
			assertKinds(t, "owner victim", got[f.victimID], "poisoned")
			if tc.inSight {
				if len(got) != 2 {
					t.Fatalf("owner: statusEffects = %v, want npc + victim", got)
				}
				assertKinds(t, "owner npc", got[f.npcID], "prone")
			} else if len(got) != 1 {
				t.Fatalf("owner: statusEffects = %v, want only the victim", got)
			}

			// mestre: tudo, com a lista de quem vê.
			got = f.reconnectFullState(t, master, f.masterUUID)
			if len(got) != 2 {
				t.Fatalf("master: statusEffects = %v, want npc + victim", got)
			}
			assertKinds(t, "master npc", got[f.npcID], "prone", "poisoned")
			assertKinds(t, "master victim", got[f.victimID], "poisoned")
			for _, list := range got {
				for _, e := range list {
					if e.Visibility == nil || e.Public == nil {
						t.Fatalf("master: %s with no visibility/public", e.Kind)
					}
				}
			}
		})
	}
}
