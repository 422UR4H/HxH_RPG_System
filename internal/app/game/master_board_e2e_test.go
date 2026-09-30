package game_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// During a match the master moves, places and removes pieces through enqueue_master_action
// (spec §4.3, "Master action de peça", B14 + the `move` of B9 + B11 live), and every accepted
// master action — pieces, walls, and the generic ones that hang on the open turn — is recorded
// in master_actions the instant it is applied, with what each player saw of it live (spec §4.8).
//
// Driven over real sockets against a real Room and session; the only fakes are the stores.

// ─── fakes and helpers ──────────────────────────────────────────────────────

// fakeMasterActionStore stands in for pgmasteraction.Repository: it keeps every Record it was
// handed, in order.
type fakeMasterActionStore struct {
	mu   sync.Mutex
	recs []masteraction.Record
}

func (s *fakeMasterActionStore) Insert(_ context.Context, r masteraction.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recs = append(s.recs, r)
	return nil
}

func (s *fakeMasterActionStore) snapshot() []masteraction.Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]masteraction.Record(nil), s.recs...)
}

// await waits for at least n records. Reading the store under its own mutex is also what makes
// reading the session afterwards safe for the race detector: the room writes the session, then
// inserts here, on the same goroutine.
func (s *fakeMasterActionStore) await(t *testing.T, n int, d time.Duration) []masteraction.Record {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if recs := s.snapshot(); len(recs) >= n {
			return recs
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("master_actions has %d record(s), want at least %d", len(s.snapshot()), n)
	return nil
}

func sendMasterMove(t *testing.T, conn *websocket.Conn, charID uuid.UUID, pos [3]int) {
	t.Helper()
	sendWS(t, conn, string(game.MsgTypeEnqueueMasterAction), map[string]any{
		"targetIds": []string{charID.String()},
		"move":      map[string]any{"position": pos},
	})
}

func sendMasterRemove(t *testing.T, conn *websocket.Conn, charID uuid.UUID) {
	t.Helper()
	sendWS(t, conn, string(game.MsgTypeEnqueueMasterAction), map[string]any{
		"targetIds": []string{charID.String()},
		"remove":    map[string]any{},
	})
}

// chatBarrier sends a chat from the master and waits for it on every collector given. A chat
// goes out through r.broadcast AFTER the master's previous message was fully handled (the
// master's read pump runs one message at a time), and everything a piece action relays is put
// straight into each client's queue before that arm returns — so anything a player was going to
// be sent about the previous action has already arrived when the chat does.
func chatBarrier(t *testing.T, master *websocket.Conn, cs ...*collector) {
	t.Helper()
	before := make([]int, len(cs))
	for i, c := range cs {
		before[i] = c.count(game.MsgTypeChatMessage)
	}
	sendWS(t, master, string(game.MsgTypeChat), map[string]any{"message": "barrier"})
	for i, c := range cs {
		if !awaitCount(c, game.MsgTypeChatMessage, before[i]+1, 2*time.Second) {
			t.Fatal("the chat barrier never arrived")
		}
	}
}

// connectTable dials the master, the player and — the fixture must have withBystander — the
// bystander, and starts a collector on each once every one of them holds a board.
func (f *combatFixture) connectTable(t *testing.T) (master, player, blind *websocket.Conn, mc, pc, bc *collector) {
	t.Helper()
	master, player = f.connect(t)
	blind = connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	readMessage(t, blind) // room_state
	mc, pc, bc = collectFrom(master), collectFrom(player), collectFrom(blind)
	for name, c := range map[string]*collector{"master": mc, "player": pc, "bystander": bc} {
		if !c.await(game.MsgTypeMapFullState, 2*time.Second) {
			t.Fatalf("the %s never got the board — the fixture never started", name)
		}
	}
	return master, player, blind, mc, pc, bc
}

func pieceContentOf(t *testing.T, rec masteraction.Record) masteraction.PieceContent {
	t.Helper()
	var pc masteraction.PieceContent
	if err := json.Unmarshal(rec.Content, &pc); err != nil {
		t.Fatalf("unmarshal piece content %s: %v", rec.Content, err)
	}
	return pc
}

func lastPieceMoved(t *testing.T, c *collector) game.PieceMovedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypePieceMoved {
			continue
		}
		var p game.PieceMovedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal piece_moved: %v", err)
		}
		return p
	}
	t.Fatal("no piece_moved in the collected messages")
	return game.PieceMovedPayload{}
}

func lastMapFullState(t *testing.T, c *collector) game.MapFullStatePayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeMapFullState {
			continue
		}
		var p game.MapFullStatePayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal map_full_state: %v", err)
		}
		return p
	}
	t.Fatal("no map_full_state in the collected messages")
	return game.MapFullStatePayload{}
}

func ptrPos(p [3]int) *[3]int { return &p }

func samePos(a, b *[3]int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func assertViews(t *testing.T, got, want map[uuid.UUID]masteraction.View) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("views = %v, want %v", got, want)
	}
	for id, v := range want {
		if got[id] != v {
			t.Fatalf("views = %v, want %v", got, want)
		}
	}
}

// openAttackTurn enqueues the fixture's attack and opens it, returning the open turn's ID.
func (f *combatFixture) openAttackTurn(t *testing.T, master, player *websocket.Conn, mc *collector) uuid.UUID {
	t.Helper()
	f.enqueueAttack(t, player)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the attack was never queued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !mc.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn opened")
	}
	return lastTurnOpened(t, mc).TurnID
}

// ─── mover ──────────────────────────────────────────────────────────────────

// O mestre arrasta a peça de um jogador, sem turno aberto: aplica na hora, o dono e o mestre
// recebem piece_moved (o mestre também — a tela dele espera a confirmação), quem não vê nem a
// origem nem o destino não recebe nada, o eco master_action_enqueued vai SÓ para o mestre, o
// tabuleiro é salvo, e a master action é gravada fora de turno com o que cada jogador viu.
func TestMasterBoard_DragWithNoOpenTurn(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	f.seedBoard(t)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	saves := f.boards.saveCount()
	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})

	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatalf("the master never got master_action_enqueued; they received: %v", messageTypes(mc.snapshotMessages()))
	}
	if !mc.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the master was not sent piece_moved for their own drag — the spec wants the confirmation")
	}
	if !pc.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the owner was never relayed the drag of their own piece")
	}
	moved := lastPieceMoved(t, mc)
	if moved.PieceID != attackerPieceID || moved.Slot.Col == nil || *moved.Slot.Col != 6 || *moved.Slot.Row != 4 {
		t.Fatalf("piece_moved = %+v, want %s at (6,4)", moved, attackerPieceID)
	}
	if moved.Z != attackerElevation {
		t.Fatalf("piece_moved z = %v, want the elevation the piece already had (%v)", moved.Z, attackerElevation)
	}
	chatBarrier(t, master, pc, bc)

	for _, typ := range []game.MessageType{game.MsgTypePieceMoved, game.MsgTypePieceRemoved, game.MsgTypeMasterActionEnqueued} {
		if n := bc.count(typ); n != 0 {
			t.Fatalf("the bystander, who sees neither end, got %d %s", n, typ)
		}
	}
	if n := pc.count(game.MsgTypeMasterActionEnqueued); n != 0 {
		t.Fatalf("a player got master_action_enqueued (%d) — it carries the position and is master-only for pieces", n)
	}
	if got := f.boards.saveCount(); got <= saves {
		t.Fatalf("saveCount = %d, want more than %d — the drag was not persisted", got, saves)
	}

	recs := f.masterActions.await(t, 1, 2*time.Second)
	if len(recs) != 1 {
		t.Fatalf("records = %d, want 1", len(recs))
	}
	rec := recs[0]
	if rec.Kind != masteraction.KindMovePiece {
		t.Fatalf("kind = %q, want movePiece", rec.Kind)
	}
	if rec.TurnUUID != nil {
		t.Fatalf("turnUuid = %v, want nil — there is no open turn", *rec.TurnUUID)
	}
	if rec.MasterUUID != f.masterUUID || rec.MatchUUID != f.matchUUID {
		t.Fatalf("master/match = %s/%s, want %s/%s", rec.MasterUUID, rec.MatchUUID, f.masterUUID, f.matchUUID)
	}
	if rec.SceneUUID != f.session.GetActiveScene().GetID() || rec.RoundUUID != f.session.GetActiveRound().GetID() {
		t.Fatal("the record does not point at the ACTIVE scene/round")
	}
	pcContent := pieceContentOf(t, rec)
	if pcContent.CharacterID != f.attackerID.String() || pcContent.PieceID != attackerPieceID ||
		!samePos(pcContent.From, ptrPos([3]int{4, 4, 0})) || !samePos(pcContent.To, ptrPos([3]int{6, 4, 0})) {
		t.Fatalf("content = %s, want characterId/pieceId of the attacker, from [4 4 0] to [6 4 0]", rec.Content)
	}
	assertViews(t, rec.Views, map[uuid.UUID]masteraction.View{f.playerUUID: masteraction.ViewFull})

	ensured := f.roundRepo.ensuredRoundIDs()
	if len(ensured) == 0 || ensured[0] != rec.RoundUUID {
		t.Fatalf("EnsureSceneAndRound calls = %v, want the record's round %s — master_actions has an FK to it", ensured, rec.RoundUUID)
	}
}

// Quem via a origem e não vê o destino recebeu só piece_removed ao vivo — e o registro diz
// `left` para ele, para o histórico mostrar a saída sem o destino.
func TestMasterBoard_DragSeenLeaving(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	f.seedBoard(t)
	master, player, blind, mc, _, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	// Into the bystander's half first: they see it arrive.
	sendMasterMove(t, master, f.attackerID, [3]int{22, 4, 0})
	if !bc.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the bystander did not see the piece arrive on their side — the premise is broken")
	}
	// And back behind the wall: they see it leave, not where it went.
	sendMasterMove(t, master, f.attackerID, [3]int{4, 4, 0})
	if !bc.await(game.MsgTypePieceRemoved, 2*time.Second) {
		t.Fatalf("the bystander was not told the piece left; they received: %v", messageTypes(bc.snapshotMessages()))
	}
	if !awaitCount(mc, game.MsgTypeMasterActionEnqueued, 2, 2*time.Second) {
		t.Fatal("the second drag was never acked")
	}

	recs := f.masterActions.await(t, 2, 2*time.Second)
	assertViews(t, recs[0].Views, map[uuid.UUID]masteraction.View{
		f.playerUUID: masteraction.ViewFull, f.bystanderUUID: masteraction.ViewFull,
	})
	assertViews(t, recs[1].Views, map[uuid.UUID]masteraction.View{
		f.playerUUID: masteraction.ViewFull, f.bystanderUUID: masteraction.ViewLeft,
	})
}

// Com turno aberto a master action também é pendurada no turno, e o registro leva o turnId.
func TestMasterBoard_DragDuringAnOpenTurn(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	turnID := f.openAttackTurn(t, master, player, mc)

	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
	recs := f.masterActions.await(t, 1, 2*time.Second)
	if recs[0].TurnUUID == nil || *recs[0].TurnUUID != turnID {
		t.Fatalf("turnUuid = %v, want the open turn %s", recs[0].TurnUUID, turnID)
	}
	if n := len(f.session.GetActiveRound().CurrentTurn().GetMasterActions()); n != 1 {
		t.Fatalf("the open turn carries %d master action(s), want 1", n)
	}
}

// ─── pôr ────────────────────────────────────────────────────────────────────

// Pôr o NPC do mestre que ainda não está na partida: inscrição de B11 (o mesmo enrollLiveNPC
// do add_npc — npc_added para a mesa e bars_updated), peça nova no slot, registro placePiece.
func TestMasterBoard_PlaceAnNPCThatIsNotAParticipant(t *testing.T) {
	uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
	f := newCombatFixture(t, withAddLiveNPC(uc))
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)
	if !pc.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board")
	}

	npcID := uuid.New()
	sendMasterMove(t, master, npcID, [3]int{6, 6, 0})

	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatalf("the place was never acked; the master received: %v (errors %v)",
			messageTypes(mc.snapshotMessages()), errorCodes(t, mc))
	}
	for name, c := range map[string]*collector{"master": mc, "player": pc} {
		if !c.await(game.MsgTypeNPCAdded, 2*time.Second) {
			t.Fatalf("the %s never received npc_added", name)
		}
		if !c.await(game.MsgTypePieceMoved, 2*time.Second) {
			t.Fatalf("the %s never saw the new piece", name)
		}
	}
	if !pc.await(game.MsgTypeBarsUpdated, 2*time.Second) || !barsCarry(lastBarsUpdated(t, pc), npcID) {
		t.Fatal("no bars_updated listing the enrolled NPC")
	}
	if calls := uc.snapshot(); len(calls) != 1 || calls[0].SheetUUID != npcID {
		t.Fatalf("add-NPC use case calls = %+v, want exactly one for %s", calls, npcID)
	}

	placed := lastPieceMoved(t, mc)
	if placed.CharacterID != npcID.String() || placed.PieceID == "" || placed.Slot.Kind != "square" ||
		placed.Slot.Col == nil || *placed.Slot.Col != 6 || *placed.Slot.Row != 6 ||
		placed.Visible == nil || !*placed.Visible {
		t.Fatalf("new piece = %+v, want a visible square piece of %s at (6,6)", placed, npcID)
	}

	recs := f.masterActions.await(t, 1, 2*time.Second)
	if recs[0].Kind != masteraction.KindPlacePiece {
		t.Fatalf("kind = %q, want placePiece", recs[0].Kind)
	}
	content := pieceContentOf(t, recs[0])
	if content.CharacterID != npcID.String() || content.PieceID != placed.PieceID ||
		content.From != nil || !samePos(content.To, ptrPos([3]int{6, 6, 0})) {
		t.Fatalf("content = %s, want the new piece, no from, to [6 6 0]", recs[0].Content)
	}
	assertViews(t, recs[0].Views, map[uuid.UUID]masteraction.View{f.playerUUID: masteraction.ViewFull})
}

// Personagem de jogador que não participa: not_participant, nada criado, nada gravado.
func TestMasterBoard_PlaceAPlayersCharacterThatIsNotAParticipant(t *testing.T) {
	uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
	f := newCombatFixture(t, withAddLiveNPC(uc))
	f.seedBoard(t)
	outsider := uuid.New()
	f.sheets.setPlayer(outsider, uuid.New())
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	if !mc.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board")
	}
	saves := f.boards.saveCount()

	sendMasterMove(t, master, outsider, [3]int{6, 6, 0})
	if !mc.await(game.MsgTypeError, 2*time.Second) {
		t.Fatal("placing a non-participating player's character was not refused")
	}
	if codes := errorCodes(t, mc); len(codes) != 1 || codes[0] != "not_participant" {
		t.Fatalf("error codes = %v, want [not_participant]", codes)
	}
	if n := mc.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("a piece was created anyway (%d piece_moved)", n)
	}
	if n := len(uc.snapshot()); n != 0 {
		t.Fatalf("the add-NPC use case was called %d time(s) for a player's sheet", n)
	}
	if got := f.boards.saveCount(); got != saves {
		t.Fatalf("saveCount = %d, want %d — a refused place must not persist", got, saves)
	}
	if n := len(f.masterActions.snapshot()); n != 0 {
		t.Fatalf("a refused master action was recorded (%d)", n)
	}
}

// ─── tirar ──────────────────────────────────────────────────────────────────

// Tirar a peça: piece_removed com fog, registro removePiece — e o personagem continua
// participante (não desinscreve, não mata).
func TestMasterBoard_Remove(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	f.seedBoard(t)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendMasterRemove(t, master, f.attackerID)
	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatalf("the remove was never acked; errors: %v", errorCodes(t, mc))
	}
	for name, c := range map[string]*collector{"master": mc, "player": pc} {
		if !c.await(game.MsgTypePieceRemoved, 2*time.Second) {
			t.Fatalf("the %s never received piece_removed", name)
		}
	}
	chatBarrier(t, master, bc)
	if n := bc.count(game.MsgTypePieceRemoved); n != 0 {
		t.Fatalf("the bystander, who never saw the piece, was told it was removed (%d)", n)
	}

	recs := f.masterActions.await(t, 1, 2*time.Second)
	if recs[0].Kind != masteraction.KindRemovePiece {
		t.Fatalf("kind = %q, want removePiece", recs[0].Kind)
	}
	content := pieceContentOf(t, recs[0])
	if content.PieceID != attackerPieceID || !samePos(content.From, ptrPos([3]int{4, 4, 0})) || content.To != nil {
		t.Fatalf("content = %s, want the attacker's piece, from [4 4 0], no to", recs[0].Content)
	}
	assertViews(t, recs[0].Views, map[uuid.UUID]masteraction.View{f.playerUUID: masteraction.ViewFull})

	board, _ := f.boards.Load(context.Background(), f.matchUUID)
	for _, p := range board.Pieces {
		if p.ID == attackerPieceID {
			t.Fatal("the saved board still has the removed piece")
		}
	}

	// Still a participant: the next bars_updated lists them.
	bars := pc.count(game.MsgTypeBarsUpdated)
	f.enqueueAttack(t, player)
	if !awaitCount(pc, game.MsgTypeBarsUpdated, bars+1, 2*time.Second) {
		t.Fatal("no bars_updated after the enqueue")
	}
	if !barsCarry(lastBarsUpdated(t, pc), f.attackerID) {
		t.Fatal("the character left the bars when their piece was removed — removing is not un-enrolling")
	}
}

// Recusas do caminho de peça: fora de partida, alvo ambíguo, tirar quem não tem peça.
func TestMasterBoard_Refusals(t *testing.T) {
	t.Run("in the lobby the master uses piece_moved", func(t *testing.T) {
		f := newCombatFixture(t, inLobby)
		f.seedBoard(t)
		master, player := f.connect(t)
		defer master.Close() //nolint:errcheck
		defer player.Close() //nolint:errcheck
		mc := collectFrom(master)
		sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})
		if !mc.await(game.MsgTypeError, 2*time.Second) {
			t.Fatal("no error")
		}
		if codes := errorCodes(t, mc); codes[0] != "match_not_started" {
			t.Fatalf("error codes = %v, want match_not_started", codes)
		}
	})

	cases := []struct {
		name string
		send func(f *combatFixture, master *websocket.Conn)
	}{
		{"two targets", func(f *combatFixture, master *websocket.Conn) {
			sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
				"targetIds": []string{f.attackerID.String(), f.victimID.String()},
				"move":      map[string]any{"position": [3]int{6, 4, 0}},
			})
		}},
		{"no target", func(f *combatFixture, master *websocket.Conn) {
			sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
				"targetIds": []string{}, "remove": map[string]any{},
			})
		}},
		{"remove a character with no piece", func(f *combatFixture, master *websocket.Conn) {
			sendMasterRemove(t, master, f.victimID)
		}},
		{"move and remove at once", func(f *combatFixture, master *websocket.Conn) {
			sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
				"targetIds": []string{f.attackerID.String()},
				"move":      map[string]any{"position": [3]int{6, 4, 0}},
				"remove":    map[string]any{},
			})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCombatFixture(t)
			f.seedBoard(t)
			master, player := f.connect(t)
			defer master.Close() //nolint:errcheck
			defer player.Close() //nolint:errcheck
			mc := collectFrom(master)
			tc.send(f, master)
			if !mc.await(game.MsgTypeError, 2*time.Second) {
				t.Fatal("no error")
			}
			if codes := errorCodes(t, mc); codes[0] != "invalid_action" {
				t.Fatalf("error codes = %v, want invalid_action", codes)
			}
			if n := len(f.masterActions.snapshot()); n != 0 {
				t.Fatalf("a refused master action was recorded (%d)", n)
			}
		})
	}
}

// ─── paredes e o caminho genérico ───────────────────────────────────────────

var (
	doorID       = uuid.MustParse("0d000000-0000-4000-8000-000000000001")
	secretDoorID = uuid.MustParse("0d000000-0000-4000-8000-000000000002")
)

// seedBoardWithDoors is seedBoard plus a plain door and an unrevealed secret door, both with a
// UUID id — enqueue_master_action's targetIds are UUIDs, and moveBoardWall's is not. Both are
// short, far from every piece, and do not block sight, so they change nobody's line of sight.
func (f *combatFixture) seedBoardWithDoors(t *testing.T) {
	t.Helper()
	f.seedBoard(t)
	b, _ := f.boards.Load(context.Background(), f.matchUUID)
	sub := mapentity.DoorSubtypeBasic
	for _, w := range []mapentity.WallSegment{
		{ID: doorID.String(), P1: [2]float64{100, 1500}, P2: [2]float64{160, 1500}, WallType: mapentity.WallTypeDoor},
		{ID: secretDoorID.String(), P1: [2]float64{1000, 1500}, P2: [2]float64{1060, 1500}, WallType: mapentity.WallTypeSecretDoor},
	} {
		w.Material, w.DoorSubtype, w.Sense, w.HP, w.MaxHP = mapentity.WallMaterialStone, &sub, mapentity.SenseNone, 50, 50
		b.Walls = append(b.Walls, w)
	}
	f.boards.seed(f.matchUUID, b)
}

// Interagir com parede grava wallInteract, e as views são quem recebeu wall_state_changed ao
// vivo: a mesa inteira para uma porta; ninguém (só o mestre) para uma porta secreta não revelada.
func TestMasterBoard_WallInteractIsRecorded(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	f.seedBoardWithDoors(t)
	master, player, blind, mc, pc, bc := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
		"targetIds": []string{doorID.String()}, "interact": map[string]any{"kind": "open"},
	})
	for name, c := range map[string]*collector{"master": mc, "player": pc, "bystander": bc} {
		if !c.await(game.MsgTypeWallStateChanged, 2*time.Second) {
			t.Fatalf("the %s never saw the door open", name)
		}
	}
	recs := f.masterActions.await(t, 1, 2*time.Second)
	if recs[0].Kind != masteraction.KindWallInteract {
		t.Fatalf("kind = %q, want wallInteract", recs[0].Kind)
	}
	var content struct {
		WallIDs  []string `json:"wallIds"`
		Interact string   `json:"interact"`
	}
	if err := json.Unmarshal(recs[0].Content, &content); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if len(content.WallIDs) != 1 || content.WallIDs[0] != doorID.String() || content.Interact != "open" {
		t.Fatalf("content = %s, want the door and open", recs[0].Content)
	}
	assertViews(t, recs[0].Views, map[uuid.UUID]masteraction.View{
		f.playerUUID: masteraction.ViewFull, f.bystanderUUID: masteraction.ViewFull,
	})

	sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
		"targetIds": []string{secretDoorID.String()}, "interact": map[string]any{"kind": "open"},
	})
	recs = f.masterActions.await(t, 2, 2*time.Second)
	if recs[1].Kind != masteraction.KindWallInteract {
		t.Fatalf("kind = %q, want wallInteract", recs[1].Kind)
	}
	assertViews(t, recs[1].Views, map[uuid.UUID]masteraction.View{})
}

// Revelar vai a todos ao vivo, então é full para todo jogador da sessão.
func TestMasterBoard_RevealIsRecordedFullForEveryone(t *testing.T) {
	f := newCombatFixture(t, withBystander)
	f.seedBoardWithDoors(t)
	master, player, blind, _, _, _ := f.connectTable(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	defer blind.Close()  //nolint:errcheck

	sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
		"targetIds": []string{secretDoorID.String()}, "interact": map[string]any{"kind": "reveal"},
	})
	recs := f.masterActions.await(t, 1, 2*time.Second)
	if recs[0].Kind != masteraction.KindRevealWall {
		t.Fatalf("kind = %q, want revealWall", recs[0].Kind)
	}
	assertViews(t, recs[0].Views, map[uuid.UUID]masteraction.View{
		f.playerUUID: masteraction.ViewFull, f.bystanderUUID: masteraction.ViewFull,
	})
}

// O caminho genérico (só alvos e perícias, pendurado no turno aberto) grava turnNote, sem views:
// não chega à mesa ao vivo como ação, fica só para o mestre. O eco continua indo à mesa inteira.
func TestMasterBoard_GenericMasterActionIsATurnNote(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)
	turnID := f.openAttackTurn(t, master, player, mc)

	sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
		"targetIds": []string{f.attackerID.String()},
		"skills":    []map[string]any{{"skillName": "Accuracy"}},
	})
	if !pc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatal("the generic master action's echo no longer reaches the table")
	}
	recs := f.masterActions.await(t, 1, 2*time.Second)
	if recs[0].Kind != masteraction.KindTurnNote {
		t.Fatalf("kind = %q, want turnNote", recs[0].Kind)
	}
	if recs[0].TurnUUID == nil || *recs[0].TurnUUID != turnID {
		t.Fatalf("turnUuid = %v, want %s", recs[0].TurnUUID, turnID)
	}
	assertViews(t, recs[0].Views, map[uuid.UUID]masteraction.View{})
	var content game.MasterActionPayload
	if err := json.Unmarshal(recs[0].Content, &content); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if len(content.TargetIDs) != 1 || content.TargetIDs[0] != f.attackerID || len(content.Skills) != 1 {
		t.Fatalf("content = %s, want the payload the master sent", recs[0].Content)
	}
}

// edit_action também pendura uma MasterAction no turno, mas NÃO é master action: não grava.
func TestMasterBoard_EditActionIsNotRecorded(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)
	f.openAttackTurn(t, master, player, mc)

	sendWS(t, master, string(game.MsgTypeEditAction), map[string]any{
		"conditions": []map[string]any{{"field": "hit", "modifier": 5}},
	})
	if !mc.await(game.MsgTypeActionEdited, 2*time.Second) {
		t.Fatal("the edit never went through")
	}
	if n := len(f.masterActions.snapshot()); n != 0 {
		t.Fatalf("edit_action was recorded as a master action (%d)", n)
	}
}

// Jogador não manda master action: forbidden, nada gravado.
func TestMasterBoard_APlayerCannotSendAMasterAction(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	pc := collectFrom(player)

	sendMasterMove(t, player, f.attackerID, [3]int{6, 4, 0})
	if !pc.await(game.MsgTypeError, 2*time.Second) {
		t.Fatal("a player's master action was not refused")
	}
	if codes := errorCodes(t, pc); codes[0] != "forbidden" {
		t.Fatalf("error codes = %v, want forbidden", codes)
	}
	if n := len(f.masterActions.snapshot()); n != 0 {
		t.Fatalf("a refused master action was recorded (%d)", n)
	}
}

// ─── a linha de visão do dono (reestabelecido de T4) ────────────────────────
//
// Os três testes abaixo existiam sobre o piece_moved do mestre no meio da partida, que B14
// proibiu (T4). O arrasto do mestre agora é a master action `move`, que passa pelo MESMO
// relayPieceMove — então as mesmas garantias são provadas por ela.

// polygonsFromPayload rebuilds the domain polygons from the wire shape, so a test can ask the
// fog gate's own question — "is this world point inside what the player can see?" — instead of
// asserting a geometry it merely assumes. Origin is left zero: IsVisible never reads it.
func polygonsFromPayload(polys [][]game.Point2DPayload) []service.VisibilityPolygon {
	out := make([]service.VisibilityPolygon, 0, len(polys))
	for _, poly := range polys {
		vs := make([]service.Point2D, 0, len(poly))
		for _, p := range poly {
			vs = append(vs, service.Point2D{X: p.X, Y: p.Y})
		}
		out = append(out, service.VisibilityPolygon{Vertices: vs})
	}
	return out
}

// O mestre arrasta a peça de um JOGADOR: o map_full_state é decidido pelo dono do personagem,
// não pelo remetente — o jogador recebe a linha de visão refeita, com a peça no lugar novo.
func TestE2E_TheMasterDraggingAPlayersPieceRefreshesThatPlayersSight(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)
	if !pc.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}

	sendMasterMove(t, master, f.attackerID, [3]int{6, 4, 0})

	if !awaitAtLeast(pc, game.MsgTypeMapFullState, 2, 2*time.Second) {
		t.Fatal("the master moved the player's piece and the player's line of sight was never " +
			"recomputed — the owner is being resolved from the sender again")
	}
	board := lastMapFullState(t, pc)
	var seen *game.PieceMovedPayload
	for i := range board.Pieces {
		if board.Pieces[i].PieceID == attackerPieceID {
			seen = &board.Pieces[i]
		}
	}
	if seen == nil {
		t.Fatal("the refreshed board does not carry the player's own piece")
	}
	if seen.Slot.Col == nil || *seen.Slot.Col != 6 || seen.Slot.Row == nil || *seen.Slot.Row != 4 {
		t.Fatalf("the refreshed board still shows the piece at %+v, want (6,4)", seen.Slot)
	}
	if !pc.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the player was never relayed the move the master made")
	}
	// Unlike the old client drag, the master is not skipped: a server-applied move has no
	// sender, and the master's screen waits for this confirmation (spec §4.3).
	if !mc.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the master was not sent the confirmation of their own drag")
	}
}

// O dono de uma peça que o mestre leva para FORA do campo de visão anterior não pode receber
// piece_removed dela: o recompute do dono roda ANTES do dispatch, então ele é julgado pelo
// polígono novo, não pelo do slot que acabou de deixar.
func TestE2E_TheOwnerIsNotToldTheirOwnPieceVanishedWhenItLeavesItsOldSight(t *testing.T) {
	f := newCombatFixture(t)
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	pc := collectFrom(player)
	if !pc.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}

	// The premise, asserted rather than assumed: from (4,4) the player sees their own slot and
	// NOT (20,4), behind the wall at x=640.
	polys := polygonsFromPayload(lastMapFullState(t, pc).VisiblePolygons)
	origin := service.Point2D{X: 4.5 * 64, Y: 4.5 * 64}
	destination := service.Point2D{X: 20.5 * 64, Y: 4.5 * 64}
	if !service.IsVisible(origin, polys) {
		t.Fatal("the player cannot see the slot their own piece is standing on — this test would prove nothing")
	}
	if service.IsVisible(destination, polys) {
		t.Fatal("the destination is already visible from the old slot: the move never leaves the old field of view")
	}

	sendMasterMove(t, master, f.attackerID, [3]int{20, 4, 0})

	// The ordering barrier: relayPieceMove dispatches the relay before it sends the owner this
	// second map_full_state, on the same goroutine.
	if !awaitAtLeast(pc, game.MsgTypeMapFullState, 2, 2*time.Second) {
		t.Fatal("the owner's line of sight was never recomputed: no second map_full_state")
	}
	if n := pc.count(game.MsgTypePieceRemoved); n != 0 {
		t.Fatalf("the owner was told their own piece vanished %d time(s); they received: %v",
			n, messageTypes(pc.snapshotMessages()))
	}
	if n := pc.count(game.MsgTypePieceMoved); n == 0 {
		t.Fatalf("the owner was never relayed the move of their own piece; they received: %v",
			messageTypes(pc.snapshotMessages()))
	}
	recs := f.masterActions.await(t, 1, 2*time.Second)
	assertViews(t, recs[0].Views, map[uuid.UUID]masteraction.View{f.playerUUID: masteraction.ViewFull})
}

// Arrastar um NPC não refaz linha de visão de ninguém: o dono de um NPC é o mestre, cuja
// visão não tem fog — nada de recompute, PlayerMemory do mestre ou map_full_state extra. E o
// caso de controle, na mesma mesa: arrastar a peça de um jogador ainda refaz a do dono.
func TestE2E_TheMasterDraggingAnNPCSkipsTheLineOfSightRecompute(t *testing.T) {
	uc := &recordingAddLiveNPC{sheet: newCombatSheet(t)}
	f := newCombatFixture(t, withAddLiveNPC(uc))
	f.seedBoard(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc, pc := collectFrom(master), collectFrom(player)

	// Puts the NPC on the board at (6,6), on the player's side of the wall — enrolling it on the
	// way (B11), so the drag below is of a piece the master owns.
	npcID := uuid.New()
	sendMasterMove(t, master, npcID, [3]int{6, 6, 0})
	if !mc.await(game.MsgTypeMasterActionEnqueued, 2*time.Second) {
		t.Fatalf("the NPC was never placed; errors: %v", errorCodes(t, mc))
	}
	if !pc.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the player never saw the NPC's piece appear on the board")
	}

	t.Run("the master's NPC drag reaches the table without refreshing the master", func(t *testing.T) {
		masterBoards := mc.count(game.MsgTypeMapFullState)
		playerMoves := pc.count(game.MsgTypePieceMoved)
		acks := mc.count(game.MsgTypeMasterActionEnqueued)

		sendMasterMove(t, master, npcID, [3]int{6, 4, 0})

		if !awaitCount(pc, game.MsgTypePieceMoved, playerMoves+1, 2*time.Second) {
			t.Fatalf("the player, who sees (6,4), was never relayed the NPC's move; they received: %v",
				messageTypes(pc.snapshotMessages()))
		}
		// The ack is the barrier: it is the arm's last send, after the relay and after where
		// the owner's map_full_state would have gone.
		if !awaitCount(mc, game.MsgTypeMasterActionEnqueued, acks+1, 2*time.Second) {
			t.Fatal("the drag was never acked")
		}
		if n := mc.count(game.MsgTypeMapFullState); n != masterBoards {
			t.Fatalf("the master got %d new map_full_state for dragging an NPC — the whole board "+
				"resent for a view that has no fog", n-masterBoards)
		}
		if _, ok := f.session.GetPlayerMemory(f.masterUUID); ok {
			t.Fatal("the drag created a PlayerMemory for the master, which nobody ever reads")
		}
	})

	t.Run("the master's drag of a player's piece still refreshes its owner", func(t *testing.T) {
		playerBoards := pc.count(game.MsgTypeMapFullState)
		sendMasterMove(t, master, f.attackerID, [3]int{5, 4, 0})
		if !awaitAtLeast(pc, game.MsgTypeMapFullState, playerBoards+1, 2*time.Second) {
			t.Fatal("the owner's line of sight was not recomputed after the master moved their " +
				"piece — the NPC shortcut swallowed the player's case too")
		}
	})
}

// B9 (spec §2, decision closed with the product owner): the master attacks through an NPC
// with enqueue_action — an "attack" on enqueue_master_action is refused outright, unconditionally,
// telling the master the correct path. The check reads the RAW JSON (the decoded
// MasterActionPayload no longer even carries an Attack field), so a client that still sends
// one is refused instead of having the key silently dropped.
func TestE2E_MasterActionWithAttackIsRefused(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	mc := collectFrom(master)

	sendWS(t, master, string(game.MsgTypeEnqueueMasterAction), map[string]any{
		"targetIds": []string{f.attackerID.String()},
		"attack": map[string]any{
			"weapon": "Sword",
			"hit":    map[string]any{"skillName": "Accuracy"},
			"damage": map[string]any{"skillName": "Push"},
		},
	})

	got := awaitErrorWithCode(t, mc, "invalid_action", 2*time.Second)
	wantMsg := "the master attacks through an NPC with enqueue_action"
	if got.Message != wantMsg {
		t.Fatalf("error message = %q, want %q", got.Message, wantMsg)
	}
	if n := len(f.masterActions.snapshot()); n != 0 {
		t.Fatalf("a refused master action was recorded %d time(s), want 0", n)
	}
}
