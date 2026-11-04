package game

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

// Estes testes dirigem o handler real da sala (handleClientMessage) com os clientes do mestre
// e de um jogador acumulando no canal send — o mesmo harness do fog_regression_test.go, no
// tabuleiro do liveMatchRoom:
//
//	p1 (peça do jogador) na célula (1,1), centro (96,96). w1: parede vertical em x=192, y 0..320.
//	porta: porta horizontal em y=192, x 0..192 — fechada, prende p1 na caixa x<192, y<192.
//	p2 (peça de um NPC) começa na célula (5,1), atrás de w1, fora da visão de p1.
//
// Todo movimento e toda mudança de parede abaixo passam pelo enqueue_master_action, o único
// jeito de o tabuleiro mudar no meio da partida.

const pieceViewDoorID = "8d3c1f0e-0b7a-4c39-9d7e-0d2c6f1a9b10"

type pieceViewFixture struct {
	room           *Room
	master, player *Client
	sheetUUID      uuid.UUID
	npcChar        uuid.UUID
}

func newPieceViewFixture(t *testing.T) *pieceViewFixture {
	t.Helper()
	room, masterUUID, playerUUID, sheetUUID := liveMatchRoom(t)
	npcChar := uuid.New()

	door := mapentity.WallSegment{
		ID: pieceViewDoorID, P1: [2]float64{0, 192}, P2: [2]float64{192, 192},
		WallType: mapentity.WallTypeDoor, Material: mapentity.WallMaterialWood,
		Sense: mapentity.SenseSight, HP: 50, MaxHP: 50,
	}
	room.walls[door.ID] = door
	room.session.UpdateWall(door)
	room.pieces["p2"] = PieceMovedPayload{
		PieceID: "p2", CharacterID: npcChar.String(), Slot: squareSlot(5, 1),
	}

	f := &pieceViewFixture{
		room:      room,
		master:    NewClient(masterUUID, nil, "gm"),
		player:    NewClient(playerUUID, nil, "p1"),
		sheetUUID: sheetUUID,
		npcChar:   npcChar,
	}
	room.clients[masterUUID] = f.master
	room.clients[playerUUID] = f.player
	return f
}

func pieceViewRaw(t *testing.T, typ MessageType, payload any) []byte {
	t.Helper()
	p, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	raw, err := json.Marshal(Message{Type: typ, Payload: p})
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	return raw
}

func (f *pieceViewFixture) send(t *testing.T, c *Client, raw []byte) {
	t.Helper()
	withinTimeout(t, 3*time.Second, "handleClientMessage", func() {
		f.room.handleClientMessage(c, raw)
	})
}

func (f *pieceViewFixture) watch(t *testing.T, c *Client, pieceID string) {
	t.Helper()
	f.send(t, c, pieceViewRaw(t, MsgTypeWatchPieceView, WatchPieceViewPayload{PieceID: pieceID}))
}

func (f *pieceViewFixture) masterMove(t *testing.T, charID uuid.UUID, col, row int) {
	t.Helper()
	f.send(t, f.master, pieceViewRaw(t, MsgTypeEnqueueMasterAction, MasterActionPayload{
		TargetIDs: []uuid.UUID{charID},
		Move:      &MovePayload{Position: [3]int{col, row, 0}},
	}))
}

// drainAll devolve toda mensagem acumulada para c, na ordem de chegada.
func drainAll(t *testing.T, c *Client) []Message {
	t.Helper()
	var out []Message
	for {
		select {
		case data := <-c.send:
			var m Message
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatalf("unmarshal message: %v", err)
			}
			out = append(out, m)
		default:
			return out
		}
	}
}

func pieceViewsIn(t *testing.T, msgs []Message) []PieceViewPayload {
	t.Helper()
	var out []PieceViewPayload
	for _, m := range msgs {
		if m.Type != MsgTypePieceView {
			continue
		}
		var p PieceViewPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal piece_view: %v", err)
		}
		out = append(out, p)
	}
	return out
}

func errorCodesIn(t *testing.T, msgs []Message) []string {
	t.Helper()
	var out []string
	for _, m := range msgs {
		if m.Type != MsgTypeError {
			continue
		}
		var p ErrorPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		out = append(out, p.Code)
	}
	return out
}

func TestWatchPieceView_MasterGetsPieceView(t *testing.T) {
	f := newPieceViewFixture(t)

	f.watch(t, f.master, "p1")

	msgs := drainAll(t, f.master)
	views := pieceViewsIn(t, msgs)
	if len(views) != 1 {
		t.Fatalf("expected exactly one piece_view, got %d (messages: %v)", len(views), msgs)
	}
	v := views[0]
	if v.PieceID != "p1" {
		t.Errorf("pieceId = %q, want p1", v.PieceID)
	}
	if len(v.VisiblePolygons) != 1 || len(v.VisiblePolygons[0]) == 0 {
		t.Errorf("expected one non-empty polygon, got %v", v.VisiblePolygons)
	}
	if !slices.Contains(v.WallIDs, "w1") {
		t.Errorf("wallIds = %v, want it to contain the in-sight wall w1", v.WallIDs)
	}
	if !slices.Contains(v.PieceIDs, "p1") {
		t.Errorf("pieceIds = %v, want it to contain the watched piece p1", v.PieceIDs)
	}
	if slices.Contains(v.PieceIDs, "p2") {
		t.Errorf("pieceIds = %v, p2 is behind w1 and must not be listed", v.PieceIDs)
	}
	// O jogador não fica sabendo nada do ponto de vista do mestre.
	if got := pieceViewsIn(t, drainAll(t, f.player)); len(got) != 0 {
		t.Errorf("the player received %d piece_view — it is master-only", len(got))
	}
}

func TestWatchPieceView_EmptyListsEncodeAsArrays(t *testing.T) {
	f := newPieceViewFixture(t)
	// Prender p1 sem nenhuma parede à vista não dá neste tabuleiro; em vez disso, confere o
	// JSON cru de uma visão real para as listas, que nunca podem ser null.
	f.watch(t, f.master, "p1")
	msgs := drainAll(t, f.master)
	for _, m := range msgs {
		if m.Type != MsgTypePieceView {
			continue
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(m.Payload, &raw); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		for _, k := range []string{"wallIds", "pieceIds", "visiblePolygons"} {
			if string(raw[k]) == "null" || raw[k] == nil {
				t.Errorf("%s encoded as %s, want an array", k, raw[k])
			}
		}
		return
	}
	t.Fatal("no piece_view received")
}

func TestWatchPieceView_PlayerForbidden(t *testing.T) {
	f := newPieceViewFixture(t)

	f.watch(t, f.player, "p1")

	msgs := drainAll(t, f.player)
	if codes := errorCodesIn(t, msgs); !slices.Contains(codes, "forbidden") {
		t.Fatalf("player error codes = %v, want forbidden", codes)
	}
	if got := pieceViewsIn(t, msgs); len(got) != 0 {
		t.Fatalf("the player received a piece_view")
	}
	f.room.mu.RLock()
	watched := f.room.masterViewPieceID
	f.room.mu.RUnlock()
	if watched != "" {
		t.Fatalf("a player's watch_piece_view set the master's watch to %q", watched)
	}
}

func TestWatchPieceView_InvalidPayload(t *testing.T) {
	f := newPieceViewFixture(t)
	raw, _ := json.Marshal(map[string]any{"type": MsgTypeWatchPieceView, "payload": "not-an-object"})

	f.send(t, f.master, raw)

	if codes := errorCodesIn(t, drainAll(t, f.master)); !slices.Contains(codes, "invalid_payload") {
		t.Fatalf("master error codes = %v, want invalid_payload", codes)
	}
}

func TestWatchPieceView_NoLiveMatch(t *testing.T) {
	masterUUID := uuid.New()
	room := newFogRoom(uuid.New(), masterUUID)
	room.pieces["p1"] = PieceMovedPayload{PieceID: "p1", CharacterID: uuid.NewString(), Slot: squareSlot(1, 1)}
	master := NewClient(masterUUID, nil, "gm")
	room.clients[masterUUID] = master
	f := &pieceViewFixture{room: room, master: master}
	// Uma observação anterior (de antes de a sessão cair) precisa ser apagada pelo watch novo.
	room.masterViewPieceID = "p1"

	f.watch(t, master, "p1")

	// Silencioso: o front reenvia o watch a cada match_full_state; responder erro só acenderia
	// o banner do mestre.
	msgs := drainAll(t, master)
	if codes := errorCodesIn(t, msgs); len(codes) != 0 {
		t.Fatalf("master error codes = %v, want none (watch on a lobby is silent)", codes)
	}
	if got := pieceViewsIn(t, msgs); len(got) != 0 {
		t.Fatalf("a lobby answered with a piece_view")
	}
	room.mu.RLock()
	watched := room.masterViewPieceID
	room.mu.RUnlock()
	if watched != "" {
		t.Fatalf("the watch still names %q after a watch with no live match", watched)
	}
}

func TestWatchPieceView_UnknownPiece(t *testing.T) {
	f := newPieceViewFixture(t)
	f.watch(t, f.master, "p1")
	if got := pieceViewsIn(t, drainAll(t, f.master)); len(got) != 1 {
		t.Fatalf("precondition: watching p1 should answer one piece_view, got %d", len(got))
	}

	f.watch(t, f.master, "ghost")

	// Silencioso: depois de um restart o watch do onopen chega antes de o tabuleiro carregar;
	// o front reenvia a cada match_full_state, então um erro aqui só acenderia o banner.
	msgs := drainAll(t, f.master)
	if codes := errorCodesIn(t, msgs); len(codes) != 0 {
		t.Fatalf("master error codes = %v, want none (unknown piece is silent)", codes)
	}
	if got := pieceViewsIn(t, msgs); len(got) != 0 {
		t.Fatalf("an unknown piece answered with a piece_view")
	}

	f.masterMove(t, f.npcChar, 1, 2)
	if got := pieceViewsIn(t, drainAll(t, f.master)); len(got) != 0 {
		t.Fatalf("a move after an unknown piece produced %d piece_view — the p1 watch must be cleared", len(got))
	}
}

func TestWatchPieceView_MasterUnregisterClearsWatch(t *testing.T) {
	f := newPieceViewFixture(t)
	f.watch(t, f.master, "p1")
	drainAll(t, f.master)

	if !f.room.removeClient(f.player) {
		t.Fatal("precondition: the player should be removed")
	}
	f.room.mu.RLock()
	watched := f.room.masterViewPieceID
	f.room.mu.RUnlock()
	if watched != "p1" {
		t.Fatalf("a player's unregister changed the master's watch to %q", watched)
	}

	if !f.room.removeClient(f.master) {
		t.Fatal("the master should be removed")
	}
	f.room.mu.RLock()
	watched = f.room.masterViewPieceID
	f.room.mu.RUnlock()
	if watched != "" {
		t.Fatalf("the watch still names %q after the master disconnected", watched)
	}
}

func TestWatchPieceView_UpdatesWhenAnotherPieceMoves(t *testing.T) {
	f := newPieceViewFixture(t)
	f.watch(t, f.master, "p1")
	drainAll(t, f.master)

	// p2 (NPC) sai de trás de w1 para a caixa de p1: movimento de NPC toma a saída owner == Nil do relayPieceMove.
	f.masterMove(t, f.npcChar, 2, 2)

	msgs := drainAll(t, f.master)
	movedAt := slices.IndexFunc(msgs, func(m Message) bool {
		if m.Type != MsgTypePieceMoved {
			return false
		}
		var p PieceMovedPayload
		_ = json.Unmarshal(m.Payload, &p)
		return p.PieceID == "p2"
	})
	if movedAt < 0 {
		t.Fatalf("the master got no piece_moved for p2 (messages: %v)", msgs)
	}
	viewAt := slices.IndexFunc(msgs, func(m Message) bool { return m.Type == MsgTypePieceView })
	if viewAt < 0 {
		t.Fatalf("the master got no piece_view after p2 moved")
	}
	if viewAt < movedAt {
		t.Fatalf("piece_view (index %d) arrived before p2's piece_moved (index %d)", viewAt, movedAt)
	}
	views := pieceViewsIn(t, msgs[movedAt:])
	last := views[len(views)-1]
	if !slices.Contains(last.PieceIDs, "p2") {
		t.Fatalf("pieceIds = %v after p2 stepped into p1's sight, want p2 listed", last.PieceIDs)
	}
}

func TestWatchPieceView_UpdatesOnWallChange(t *testing.T) {
	f := newPieceViewFixture(t)
	// p2 fica logo abaixo da porta fechada: só entra na visão de p1 quando ela abre.
	f.masterMove(t, f.npcChar, 1, 4)
	f.watch(t, f.master, "p1")
	before := pieceViewsIn(t, drainAll(t, f.master))
	if len(before) != 1 || slices.Contains(before[0].PieceIDs, "p2") {
		t.Fatalf("precondition: one piece_view without p2, got %v", before)
	}

	f.send(t, f.master, pieceViewRaw(t, MsgTypeEnqueueMasterAction, MasterActionPayload{
		TargetIDs: []uuid.UUID{uuid.MustParse(pieceViewDoorID)},
		Interact:  &InteractPayload{Kind: "open"},
	}))

	views := pieceViewsIn(t, drainAll(t, f.master))
	if len(views) == 0 {
		t.Fatal("opening the door produced no piece_view")
	}
	if last := views[len(views)-1]; !slices.Contains(last.PieceIDs, "p2") {
		t.Fatalf("pieceIds = %v after the door opened, want p2 listed", last.PieceIDs)
	}
}

func TestWatchPieceView_StopsOnEmptyAndOnRemoval(t *testing.T) {
	t.Run("empty pieceId stops watching", func(t *testing.T) {
		f := newPieceViewFixture(t)
		f.watch(t, f.master, "p1")
		drainAll(t, f.master)

		f.watch(t, f.master, "")
		f.masterMove(t, f.npcChar, 2, 2)

		msgs := drainAll(t, f.master)
		if got := pieceViewsIn(t, msgs); len(got) != 0 {
			t.Fatalf("got %d piece_view after the watch stopped", len(got))
		}
		if codes := errorCodesIn(t, msgs); len(codes) != 0 {
			t.Fatalf("stopping the watch answered errors %v", codes)
		}
	})

	t.Run("removing the watched piece stops watching", func(t *testing.T) {
		f := newPieceViewFixture(t)
		f.watch(t, f.master, "p1")
		drainAll(t, f.master)

		f.send(t, f.master, pieceViewRaw(t, MsgTypeEnqueueMasterAction, MasterActionPayload{
			TargetIDs: []uuid.UUID{f.sheetUUID},
			Remove:    &RemovePiecePayload{},
		}))
		f.masterMove(t, f.npcChar, 2, 2)

		if got := pieceViewsIn(t, drainAll(t, f.master)); len(got) != 0 {
			t.Fatalf("got %d piece_view after the watched piece left the board", len(got))
		}
		f.room.mu.RLock()
		watched := f.room.masterViewPieceID
		f.room.mu.RUnlock()
		if watched != "" {
			t.Fatalf("the watch still names %q after its piece was removed", watched)
		}
	})
}
