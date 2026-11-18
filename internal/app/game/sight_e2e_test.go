package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Visão (Task 10, decisões do dono 3–6): o Escondido tira a peça, as barras e os status de
// quem não está na lista dele; as barras e os status seguem a névoa; npc_added só ao mestre.
// Mesa: a do seedStatusBoard — o jogador (dono do atacante e da vítima), o bystander (dono da
// sua peça em (20,4)) e o NPC do mestre.

// pieceRemovedIDs lista, em ordem, o pieceId de todo piece_removed que o coletor recebeu.
func pieceRemovedIDs(t *testing.T, c *collector) []string {
	t.Helper()
	var out []string
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypePieceRemoved {
			continue
		}
		var p game.PieceRemovedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal piece_removed: %v", err)
		}
		out = append(out, p.PieceID)
	}
	return out
}

// piecesMovedOf lista, em ordem, todo piece_moved do coletor para a peça pieceID.
func piecesMovedOf(t *testing.T, c *collector, pieceID string) []game.PieceMovedPayload {
	t.Helper()
	var out []game.PieceMovedPayload
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypePieceMoved {
			continue
		}
		var p game.PieceMovedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal piece_moved: %v", err)
		}
		if p.PieceID == pieceID {
			out = append(out, p)
		}
	}
	return out
}

func countOf(ids []string, id string) int {
	n := 0
	for _, x := range ids {
		if x == id {
			n++
		}
	}
	return n
}

// awaitPieceRemoved espera que o coletor tenha n piece_removed de pieceID.
func awaitPieceRemoved(t *testing.T, who string, c *collector, pieceID string, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if countOf(pieceRemovedIDs(t, c), pieceID) >= n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s: got %d piece_removed for %s, want %d", who, countOf(pieceRemovedIDs(t, c), pieceID), pieceID, n)
}

// awaitPieceMoved espera que o coletor tenha n piece_moved de pieceID.
func awaitPieceMoved(t *testing.T, who string, c *collector, pieceID string, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(piecesMovedOf(t, c, pieceID)) >= n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s: got %d piece_moved for %s, want %d", who, len(piecesMovedOf(t, c, pieceID)), pieceID, n)
}

// firstIndexAfter é o índice da primeira mensagem do tipo want a partir de from; -1 se não há.
func firstIndexAfter(c *collector, want game.MessageType, from int) int {
	msgs := c.snapshotMessages()
	for i := from; i < len(msgs); i++ {
		if msgs[i].Type == want {
			return i
		}
	}
	return -1
}

// reconnectSight fecha a conexão de who, liga de novo e devolve a conexão nova, o coletor dela,
// o map_full_state e o match_full_state que ela recebeu.
func (f *combatFixture) reconnectSight(t *testing.T, old *websocket.Conn, who uuid.UUID) (
	*websocket.Conn, *collector, game.MapFullStatePayload, game.MatchFullStatePayload,
) {
	t.Helper()
	_ = old.Close()
	conn := connectWS(t, f.server.URL, who, f.matchUUID)
	t.Cleanup(func() { _ = conn.Close() })
	c := collectFrom(conn)
	if !c.await(game.MsgTypeMapFullState, 2*time.Second) || !c.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatalf("reconnect: got %v, want map_full_state and match_full_state", messageTypes(c.snapshotMessages()))
	}
	var match game.MatchFullStatePayload
	if err := json.Unmarshal(findMessage(t, c.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &match); err != nil {
		t.Fatalf("unmarshal match_full_state: %v", err)
	}
	return conn, c, lastMapFullState(t, c), match
}

func hasPiece(pieces []game.PieceMovedPayload, pieceID string) bool {
	for _, p := range pieces {
		if p.PieceID == pieceID {
			return true
		}
	}
	return false
}

func TestSight_HiddenRemovesPiece(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCInSight)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	// Escondido no NPC, percebido só pelo jogador: o bystander perde a peça, o jogador não.
	sendApplyStatus(t, master, map[string]any{
		"characterIds": []string{f.npcID.String()}, "kind": "hidden",
		"visibility": map[string]any{"mode": "players", "playerIds": []string{f.playerUUID.String()}},
	})
	awaitStatusCount(t, "master", mc, 1)
	hiddenID := lastStatusUpdate(t, mc).StatusEffects[0].ID
	awaitPieceRemoved(t, "bystander", bc, statusNPCPieceID, 1)
	chatBarrier(t, master, mc, pc, bc)
	if n := countOf(pieceRemovedIDs(t, pc), statusNPCPieceID); n != 0 {
		t.Fatalf("owner: got %d piece_removed for the NPC, want 0", n)
	}
	if barsCarry(lastBarsUpdated(t, bc), f.npcID) {
		t.Fatal("bystander: bars_updated still lists the hidden NPC")
	}

	// O NPC anda: o bystander não sabe de nada; o jogador vê.
	sendMasterMove(t, master, f.npcID, [3]int{16, 5, 0})
	awaitPieceMoved(t, "owner", pc, statusNPCPieceID, 1)
	chatBarrier(t, master, mc, pc, bc)
	if n := len(piecesMovedOf(t, bc, statusNPCPieceID)); n != 0 {
		t.Fatalf("bystander: got %d piece_moved for the hidden NPC, want 0", n)
	}
	if n := countOf(pieceRemovedIDs(t, bc), statusNPCPieceID); n != 1 {
		t.Fatalf("bystander: got %d piece_removed for the hidden NPC, want 1", n)
	}

	// Reconectado, o bystander não recebe a peça nem as barras do NPC.
	blind, bc, mapState, matchState := f.reconnectSight(t, blind, f.bystanderUUID)
	defer blind.Close() //nolint:errcheck
	if hasPiece(mapState.Pieces, statusNPCPieceID) {
		t.Fatal("bystander reconnect: map_full_state has the hidden NPC piece")
	}
	if barsCarry(matchState.Bars, f.npcID) {
		t.Fatal("bystander reconnect: match_full_state.bars has the hidden NPC")
	}
	if !barsCarry(matchState.Bars, f.bystanderID) {
		t.Fatal("bystander reconnect: match_full_state.bars lost the bystander's own PC")
	}

	// O mestre tira o Escondido: a peça volta ao bystander, no slot de agora, e as barras dela.
	sendWS(t, master, string(game.MsgTypeRemoveStatusEffect), map[string]any{"statusEffectId": hiddenID})
	awaitPieceMoved(t, "bystander", bc, statusNPCPieceID, 1)
	back := piecesMovedOf(t, bc, statusNPCPieceID)[0]
	if back.Slot.Col == nil || back.Slot.Row == nil || *back.Slot.Col != 16 || *back.Slot.Row != 5 {
		t.Fatalf("bystander: piece_moved slot = %+v, want (16,5)", back.Slot)
	}
	if !awaitCount(bc, game.MsgTypeBarsUpdated, 1, 2*time.Second) {
		t.Fatal("bystander: no bars_updated after the NPC came back")
	}
	if !barsCarry(lastBarsUpdated(t, bc), f.npcID) {
		t.Fatal("bystander: bars_updated after the reveal does not list the NPC")
	}
}

func TestSight_StatusOnEnteringSight(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCBehindWall, moveBoardWall)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendApplyStatus(t, master, map[string]any{"characterIds": []string{f.npcID.String()}, "kind": "prone"})
	awaitStatusCount(t, "master", mc, 1)
	proneID := lastStatusUpdate(t, mc).StatusEffects[0].ID
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "bystander", bc, 0)

	// Entra na visão: piece_moved e só então a lista atual.
	start := len(bc.snapshotMessages())
	sendMasterMove(t, master, f.npcID, [3]int{16, 4, 0})
	awaitStatusCount(t, "bystander", bc, 1)
	moved := firstIndexAfter(bc, game.MsgTypePieceMoved, start)
	status := firstIndexAfter(bc, game.MsgTypeStatusEffectsUpdated, start)
	if moved < 0 || status < moved {
		t.Fatalf("bystander: piece_moved at %d, status_effects_updated at %d — the piece goes first", moved, status)
	}
	got := lastStatusUpdate(t, bc)
	if got.CharacterID != f.npcID {
		t.Fatalf("bystander: status for %s, want the NPC", got.CharacterID)
	}
	assertKinds(t, "bystander", got.StatusEffects, "prone")

	// Sai da visão: piece_removed, nenhum status.
	sendMasterMove(t, master, f.npcID, [3]int{4, 4, 0})
	awaitPieceRemoved(t, "bystander", bc, statusNPCPieceID, 1)
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "bystander", bc, 1)

	// Editado fora da visão: nada.
	sendWS(t, master, string(game.MsgTypeEditStatusEffect), map[string]any{"statusEffectId": proneID, "alias": "Caído"})
	awaitStatusCount(t, "master", mc, 2)
	chatBarrier(t, master, mc, pc, bc)
	assertStatusCount(t, "bystander", bc, 1)

	// De volta à visão: a lista nova.
	sendMasterMove(t, master, f.npcID, [3]int{16, 4, 0})
	awaitStatusCount(t, "bystander", bc, 2)
	got = lastStatusUpdate(t, bc)
	assertKinds(t, "bystander", got.StatusEffects, "prone")
	if got.StatusEffects[0].Alias != "Caído" {
		t.Fatalf("bystander: alias = %q, want the edited one", got.StatusEffects[0].Alias)
	}
}

func TestSight_BarsFollowSight(t *testing.T) {
	f := newStatusFixture(t)
	f.seedStatusBoard(t, statusNPCBehindWall, moveBoardWall)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	// Reconectado com o NPC fora da visão: o match_full_state do bystander não o lista.
	blind, bc, _, matchState := f.reconnectSight(t, blind, f.bystanderUUID)
	defer blind.Close() //nolint:errcheck
	if barsCarry(matchState.Bars, f.npcID) {
		t.Fatal("bystander: match_full_state.bars lists the NPC out of sight")
	}
	if !barsCarry(matchState.Bars, f.bystanderID) {
		t.Fatal("bystander: match_full_state.bars lost the bystander's own PC")
	}

	sendMasterMove(t, master, f.npcID, [3]int{16, 4, 0})
	if !awaitCount(bc, game.MsgTypeBarsUpdated, 1, 2*time.Second) {
		t.Fatal("bystander: no bars_updated when the NPC came into sight")
	}
	in := lastBarsUpdated(t, bc)
	if !barsCarry(in, f.npcID) || !barsCarry(in, f.bystanderID) {
		t.Fatalf("bystander in sight: bars = %+v, want the NPC and the own PC", in.Characters)
	}

	sendMasterMove(t, master, f.npcID, [3]int{4, 4, 0})
	if !awaitCount(bc, game.MsgTypeBarsUpdated, 2, 2*time.Second) {
		t.Fatal("bystander: no bars_updated when the NPC left sight")
	}
	out := lastBarsUpdated(t, bc)
	if barsCarry(out, f.npcID) || !barsCarry(out, f.bystanderID) {
		t.Fatalf("bystander out of sight: bars = %+v, want only the own PC side", out.Characters)
	}

	chatBarrier(t, master, mc, pc, bc)
	if !barsCarry(lastBarsUpdated(t, mc), f.npcID) {
		t.Fatal("master: bars_updated lost the NPC")
	}
}

func TestSight_NpcAddedMasterOnly(t *testing.T) {
	uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
	f := newCombatFixture(t, withAddLiveNPC(uc))
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	npcID := uuid.New()
	sendAddNPC(t, master, npcID)
	if !mc.await(game.MsgTypeNPCAdded, 2*time.Second) {
		t.Fatal("master: no npc_added")
	}
	chatBarrier(t, master, mc, pc)
	if n := pc.count(game.MsgTypeNPCAdded); n != 0 {
		t.Fatalf("player: got %d npc_added, want 0 — the NPC reaches players through the fog", n)
	}
}
