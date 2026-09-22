package game_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// add_npc puts a master-controlled character into the match over the socket, driven against
// a real Room and a real session, exactly as TestE2E_EditAction drives edit_action. The use
// case is the only mock: it stands in for the DB write + sheet load, and records whether it
// was reached at all — "the room refused before touching the database" is half of what the
// refusal tests assert.

// recordingAddLiveNPC stands in for AddLiveNPCUC: it hands back a real sheet (or a scripted
// error) and records every input it was called with.
type recordingAddLiveNPC struct {
	mu    sync.Mutex
	calls []appmatch.AddMatchNPCInput
	sheet *csSheet.CharacterSheet
	err   error
}

func (m *recordingAddLiveNPC) Execute(
	_ context.Context, in *appmatch.AddMatchNPCInput,
) (*csSheet.CharacterSheet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, *in)
	if m.err != nil {
		return nil, m.err
	}
	return m.sheet, nil
}

func (m *recordingAddLiveNPC) snapshot() []appmatch.AddMatchNPCInput {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]appmatch.AddMatchNPCInput(nil), m.calls...)
}

func sendAddNPC(t *testing.T, conn *websocket.Conn, sheetUUID uuid.UUID) {
	t.Helper()
	sendWS(t, conn, string(game.MsgTypeAddNPC), game.AddNPCPayload{CharacterSheetUUID: sheetUUID})
}

// errorCodes lists the codes of every error a collector has gathered, in arrival order.
func errorCodes(t *testing.T, c *collector) []string {
	t.Helper()
	var out []string
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeError {
			continue
		}
		var p game.ErrorPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal error payload: %v", err)
		}
		out = append(out, p.Code)
	}
	return out
}

// npcAddedIDs lists the characterId of every npc_added a collector has gathered, in order.
func npcAddedIDs(t *testing.T, c *collector) []uuid.UUID {
	t.Helper()
	var out []uuid.UUID
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeNPCAdded {
			continue
		}
		var p game.NPCAddedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal npc_added: %v", err)
		}
		out = append(out, p.CharacterID)
	}
	return out
}

func barsCarry(p game.BarsUpdatedPayload, charID uuid.UUID) bool {
	for _, c := range p.Characters {
		if c.CharacterID == charID {
			return true
		}
	}
	return false
}

// O caminho feliz: o mestre põe um NPC numa sala viva. A mesa inteira ouve npc_added e recebe
// barras NOVAS (seq maior) que já contam o NPC — e, a prova de que a sessão viva o conhece de
// verdade, o mestre consegue enfileirar uma ação com ele como ator logo em seguida.
func TestE2E_AddNPCPutsTheNPCIntoTheLiveSession(t *testing.T) {
	uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
	f := newCombatFixture(t, withAddLiveNPC(uc))
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	// A bars_updated the table has already seen, so "newer" below has something to beat.
	f.enqueueAttack(t, player)
	if !playerMsgs.await(game.MsgTypeBarsUpdated, 2*time.Second) {
		t.Fatal("no bars_updated after the enqueue — the fixture never started")
	}
	if !masterMsgs.await(game.MsgTypeBarsUpdated, 2*time.Second) {
		t.Fatal("the master never received the baseline bars_updated")
	}
	baseline := lastBarsUpdated(t, masterMsgs).Seq
	masterBars := masterMsgs.count(game.MsgTypeBarsUpdated)
	playerBars := playerMsgs.count(game.MsgTypeBarsUpdated)

	npcID := uuid.New()
	sendAddNPC(t, master, npcID)

	for name, c := range map[string]*collector{"master": masterMsgs, "player": playerMsgs} {
		if !c.await(game.MsgTypeNPCAdded, 2*time.Second) {
			t.Fatalf("the %s never received npc_added; they received: %v",
				name, messageTypes(c.snapshotMessages()))
		}
		if ids := npcAddedIDs(t, c); len(ids) != 1 || ids[0] != npcID {
			t.Fatalf("the %s got npc_added for %v, want exactly [%s]", name, ids, npcID)
		}
	}

	for name, tc := range map[string]struct {
		c      *collector
		before int
	}{"master": {masterMsgs, masterBars}, "player": {playerMsgs, playerBars}} {
		if !awaitCount(tc.c, game.MsgTypeBarsUpdated, tc.before+1, 2*time.Second) {
			t.Fatalf("the %s never received a bars_updated after add_npc", name)
		}
		bars := lastBarsUpdated(t, tc.c)
		if bars.Seq <= baseline {
			t.Fatalf("the %s's bars_updated after add_npc has seq %d, want > %d — a stale seq "+
				"would lose to the snapshot it is meant to replace", name, bars.Seq, baseline)
		}
		if !barsCarry(bars, npcID) {
			t.Fatalf("the %s's bars_updated after add_npc does not list the NPC %s", name, npcID)
		}
	}

	calls := uc.snapshot()
	if len(calls) != 1 {
		t.Fatalf("the use case was called %d time(s), want 1", len(calls))
	}
	want := appmatch.AddMatchNPCInput{RequesterUUID: f.masterUUID, MatchUUID: f.matchUUID, SheetUUID: npcID}
	if calls[0] != want {
		t.Fatalf("the use case got %+v, want %+v", calls[0], want)
	}

	// The live session knows the NPC: the master can act through it.
	f.enqueueAttackFrom(t, master, npcID)
	if !masterMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatalf("the master could not enqueue an action for the NPC; errors: %v",
			errorCodes(t, masterMsgs))
	}
	if codes := errorCodes(t, masterMsgs); len(codes) != 0 {
		t.Fatalf("the master received errors along the way: %v", codes)
	}
}

// Um jogador não põe NPC na mesa — e a recusa vem antes do banco: o use case nem é chamado.
func TestE2E_AddNPCFromAPlayerIsForbidden(t *testing.T) {
	uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
	f := newCombatFixture(t, withAddLiveNPC(uc))
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	sendAddNPC(t, player, uuid.New())
	if code := awaitErrorCode(t, player, 2*time.Second); code != "forbidden" {
		t.Fatalf("error code = %q, want forbidden", code)
	}
	// The refusal is sent from the same goroutine, before the use case would be reached, so
	// by the time it has arrived the call either happened or never will.
	if n := len(uc.snapshot()); n != 0 {
		t.Fatalf("the use case was called %d time(s) for a player's add_npc, want 0", n)
	}
}

// O mesmo NPC duas vezes: o banco tolera (o UC é chamado de novo — Decisão 2), a SESSÃO recusa,
// e a mesa não ouve um segundo npc_added.
func TestE2E_AddNPCTwiceIsRefusedByTheSession(t *testing.T) {
	uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
	f := newCombatFixture(t, withAddLiveNPC(uc))
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	npcID := uuid.New()
	sendAddNPC(t, master, npcID)
	if !masterMsgs.await(game.MsgTypeNPCAdded, 2*time.Second) {
		t.Fatal("the first add_npc never produced npc_added")
	}

	sendAddNPC(t, master, npcID)
	if !masterMsgs.await(game.MsgTypeError, 2*time.Second) {
		t.Fatal("the second add_npc of the same NPC was not refused")
	}
	if codes := errorCodes(t, masterMsgs); len(codes) != 1 || codes[0] != "npc_already_in_match" {
		t.Fatalf("error codes = %v, want [npc_already_in_match]", codes)
	}
	if n := len(uc.snapshot()); n != 2 {
		t.Fatalf("the use case was called %d time(s), want 2 — the database tolerates the "+
			"duplicate, it is the session that refuses it", n)
	}

	// The barrier. npc_added travels through r.broadcast from a goroutine, so there is no
	// same-goroutine message to wait on behind it; a SECOND, different NPC is the closest
	// thing — the table must end up with exactly [first, second], in that order, and a
	// stray duplicate announcement would show up between them.
	second := uuid.New()
	sendAddNPC(t, master, second)
	for name, c := range map[string]*collector{"master": masterMsgs, "player": playerMsgs} {
		if !awaitCount(c, game.MsgTypeNPCAdded, 2, 2*time.Second) {
			t.Fatalf("the %s never heard about the second NPC", name)
		}
		ids := npcAddedIDs(t, c)
		if len(ids) != 2 || ids[0] != npcID || ids[1] != second {
			t.Fatalf("the %s got npc_added for %v, want [%s %s] — the refused duplicate was "+
				"announced anyway", name, ids, npcID, second)
		}
	}
}

// O use case recusa: cada erro do pacote match vira o código que o front sabe tratar, e a
// sessão fica intocada — o mestre não consegue agir pelo NPC recusado.
func TestE2E_AddNPCMapsTheUseCaseErrors(t *testing.T) {
	cases := []struct {
		err  error
		code string
	}{
		{appmatch.ErrNotMatchMaster, "forbidden"},
		{appmatch.ErrMatchNotFound, "not_found"},
		{appmatch.ErrCharacterSheetNotFound, "not_found"},
		{appmatch.ErrSheetNotNPC, "invalid_npc"},
		{appmatch.ErrSheetNotOwnedByMaster, "invalid_npc"},
		{appmatch.ErrMatchAlreadyFinished, "invalid_npc"},
		{errors.New("connection reset by peer"), "game_error"},
	}
	for _, tc := range cases {
		t.Run(tc.code+"/"+tc.err.Error(), func(t *testing.T) {
			uc := &recordingAddLiveNPC{err: tc.err}
			f := newCombatFixture(t, withAddLiveNPC(uc))
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			masterMsgs := collectFrom(master)

			npcID := uuid.New()
			sendAddNPC(t, master, npcID)
			if !masterMsgs.await(game.MsgTypeError, 2*time.Second) {
				t.Fatal("the refused add_npc produced no error")
			}
			if codes := errorCodes(t, masterMsgs); len(codes) != 1 || codes[0] != tc.code {
				t.Fatalf("error codes = %v, want [%s]", codes, tc.code)
			}

			// The session was not touched: acting through the refused NPC fails.
			f.enqueueAttackFrom(t, master, npcID)
			if !awaitCount(masterMsgs, game.MsgTypeError, 2, 2*time.Second) {
				t.Fatal("the master enqueued an action for an NPC the use case refused")
			}
			if n := masterMsgs.count(game.MsgTypeNPCAdded); n != 0 {
				t.Fatalf("a refused add_npc was announced %d time(s)", n)
			}
			if n := masterMsgs.count(game.MsgTypeActionEnqueued); n != 0 {
				t.Fatal("the master enqueued an action for an NPC the use case refused")
			}
		})
	}
}

// Payload sem ficha — ou que nem é JSON de objeto — não chega ao banco.
func TestE2E_AddNPCWithAnInvalidPayloadIsRefused(t *testing.T) {
	for name, payload := range map[string]any{
		"zero uuid":   game.AddNPCPayload{},
		"wrong shape": []int{1, 2, 3},
	} {
		t.Run(name, func(t *testing.T) {
			uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
			f := newCombatFixture(t, withAddLiveNPC(uc))
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck

			sendWS(t, master, string(game.MsgTypeAddNPC), payload)
			if code := awaitErrorCode(t, master, 2*time.Second); code != "invalid_payload" {
				t.Fatalf("error code = %q, want invalid_payload", code)
			}
			if n := len(uc.snapshot()); n != 0 {
				t.Fatalf("the use case was called %d time(s) for an invalid payload, want 0", n)
			}
		})
	}
}

// Sala sem sessão (lobby — Decisão 3): o banco é gravado (o Init trará o NPC quando a partida
// começar) e a mesa ouve npc_added — mas não há barras para publicar.
func TestE2E_AddNPCInTheLobbyRostersItWithoutBars(t *testing.T) {
	uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
	f := newCombatFixture(t, withAddLiveNPC(uc), inLobby)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	npcID := uuid.New()
	sendAddNPC(t, master, npcID)
	for name, c := range map[string]*collector{"master": masterMsgs, "player": playerMsgs} {
		if !c.await(game.MsgTypeNPCAdded, 2*time.Second) {
			t.Fatalf("the %s never received npc_added in the lobby; they received: %v",
				name, messageTypes(c.snapshotMessages()))
		}
	}
	if n := len(uc.snapshot()); n != 1 {
		t.Fatalf("the use case was called %d time(s), want 1 — the lobby still rosters the NPC", n)
	}

	// The barrier: a chat goes out through the same r.broadcast the bars would, and is sent
	// after the add_npc arm returned (the master's messages are handled one at a time), so a
	// bars_updated from that arm would have to be queued no later than this lands.
	sendWS(t, master, string(game.MsgTypeChat), map[string]any{"message": "barrier"})
	if !playerMsgs.await(game.MsgTypeChatMessage, 2*time.Second) {
		t.Fatal("the chat barrier never arrived")
	}
	for name, c := range map[string]*collector{"master": masterMsgs, "player": playerMsgs} {
		if n := c.count(game.MsgTypeBarsUpdated); n != 0 {
			t.Fatalf("the %s got %d bars_updated in a lobby, which has no bars", name, n)
		}
		if codes := errorCodes(t, c); len(codes) != 0 {
			t.Fatalf("the %s received errors: %v", name, codes)
		}
	}
}

// ─── débito de LOS (Decisão 7) ──────────────────────────────────────────────

const npcPieceID = "piece-npc"

// syncBoardWithNPC seeds the same board syncBoard does — the attacker at (4,4), west of the
// wall — plus an NPC piece at (6,6), on the same side, so the player sees it move.
func (f *combatFixture) syncBoardWithNPC(t *testing.T, master *websocket.Conn, npcID uuid.UUID) {
	t.Helper()
	col, row := 4, 4
	ncol, nrow := 6, 6
	pieces := []game.PieceMovedPayload{
		{
			PieceID:     attackerPieceID,
			CharacterID: f.attackerID.String(),
			Slot:        game.SlotPayload{Kind: "square", Col: &col, Row: &row},
		},
		{
			PieceID:     npcPieceID,
			CharacterID: npcID.String(),
			Slot:        game.SlotPayload{Kind: "square", Col: &ncol, Row: &nrow},
		},
	}
	grid := toGridShapePayload(moveBoardGrid)
	sendWS(t, master, "map_state_sync", game.MapStateSyncPayload{
		Pieces: &pieces,
		Walls:  []game.WallSegmentPayload{toWallSegmentPayload(moveBoardWall)},
		Grid:   &grid,
	})
}

// O mestre arrasta a peça de um NPC. O NPC é "dele" no charToPlayer, mas o mestre vê o
// tabuleiro sem filtro: nada de recompute de LOS, nada de PlayerMemory para o mestre, nada de
// map_full_state a cada arrasto. A mesa continua recebendo o piece_moved de sempre. E o
// controle, na mesma mesa: arrastar a peça de um JOGADOR continua refrescando o dono.
func TestE2E_TheMasterDraggingAnNPCSkipsTheLineOfSightRecompute(t *testing.T) {
	uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
	f := newCombatFixture(t, withAddLiveNPC(uc))
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	npcID := uuid.New()
	sendAddNPC(t, master, npcID)
	if !masterMsgs.await(game.MsgTypeNPCAdded, 2*time.Second) {
		t.Fatal("the NPC never joined the session — this test would drag a piece nobody owns")
	}

	f.syncBoardWithNPC(t, master, npcID)
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board back after map_state_sync")
	}
	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}

	t.Run("the master's NPC drag reaches the table without refreshing the master", func(t *testing.T) {
		masterBoards := masterMsgs.count(game.MsgTypeMapFullState)

		sendPieceMoved(t, master, npcPieceID, npcID.String(), 6, 4)

		if !playerMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
			t.Fatalf("the player, who sees (6,4), was never relayed the NPC's move; they "+
				"received: %v", messageTypes(playerMsgs.snapshotMessages()))
		}

		// The ordering barrier: an unknown message type is answered with a direct error from
		// the master's own read loop, which handles one message at a time — so the piece_moved
		// arm, map_full_state included, finished before this reply was even built.
		sendWS(t, master, "barrier", map[string]any{})
		if !masterMsgs.await(game.MsgTypeError, 2*time.Second) {
			t.Fatal("the barrier never came back")
		}

		if n := masterMsgs.count(game.MsgTypeMapFullState); n != masterBoards {
			t.Fatalf("the master got %d new map_full_state for dragging an NPC — the whole "+
				"board resent for a view that has no fog", n-masterBoards)
		}
		if _, ok := f.session.GetPlayerMemory(f.masterUUID); ok {
			t.Fatal("the drag created a PlayerMemory for the master, which nobody ever reads")
		}
	})

	t.Run("the master's drag of a player's piece still refreshes its owner", func(t *testing.T) {
		playerBoards := playerMsgs.count(game.MsgTypeMapFullState)

		sendPieceMoved(t, master, attackerPieceID, f.attackerID.String(), 5, 4)

		if !awaitAtLeast(playerMsgs, game.MsgTypeMapFullState, playerBoards+1, 2*time.Second) {
			t.Fatal("the owner's line of sight was not recomputed after the master moved " +
				"their piece — the NPC shortcut swallowed the player's case too")
		}
	})
}
