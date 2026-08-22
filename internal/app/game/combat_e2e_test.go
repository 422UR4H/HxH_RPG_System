package game_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file is the end-to-end guarantee for the first real collision: an attack sent over
// a real WebSocket, against a real Room, resolved by the real use cases, producing damage
// on the target's sheet.
//
// It drives the same sequence a browser would:
//
//	player  → enqueue_action  (attack, targeting another character)
//	master  → open_next_action  (opens the turn; the master receives the projection)
//	master  → open_next_action  (closes it; the damage is applied and persisted)
//
// and asserts the four things the phase owes:
//
//  1. the master sees the projected damage BEFORE anything is applied;
//  2. the player receives no resolution_updated at all — the calculation is the master's;
//  3. the target's HP does not move while the turn is open;
//  4. it moves by exactly the projected amount once the turn closes, and the same number
//     is written through to the sheet gateway.
//
// The dice are scripted, so the numbers are exact instead of lucky.

// ─── mocks ──────────────────────────────────────────────────────────────────

// combatSessionUC hands the handler one prepared session, so the test owns the sheets.
type combatSessionUC struct{ session *matchsession.MatchSession }

func (m *combatSessionUC) Init(_ context.Context, _ uuid.UUID) (*matchsession.MatchSession, error) {
	return m.session, nil
}

// recordingStatusWriter stands in for the sheet gateway.
type recordingStatusWriter struct {
	mu         sync.Mutex
	sheetUUIDs []string
	healthCurr []int
}

func (w *recordingStatusWriter) UpdateStatusBars(
	_ context.Context, sheetUUID string, health, _, _ status.IStatusBar,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sheetUUIDs = append(w.sheetUUIDs, sheetUUID)
	w.healthCurr = append(w.healthCurr, health.GetCurrent())
	return nil
}

// topFaceSource lands every die on its top face, so the attack always clears the passive
// dodge and the numbers in the assertions are exact.
type topFaceSource struct{}

func (topFaceSource) RollDie(sides enum.DieSides) int { return sides.GetSides() }

// scriptedFaces hands out faces in order and NEVER repeats: once exhausted, it records an
// overrun instead of silently replaying the last face.
//
// A silent repeat is a trap. An earlier version of this file's racing-round test scripted only
// 8 faces assuming a single 2D10 check per action, but MatchSession.rollActionDice rolls every
// check the action carries at enqueue time — Speed, Hit, and the weapon's damage dice — so an
// attack consumes far more than 4 faces. The test still passed: the victim's asserted speed of
// 16 came entirely from the fallback repeating the script's last face (which happened to equal
// the intended one), not from the script itself. A repeat can never be told apart from a
// genuine match by the test that relies on it. Failing loudly instead means every asserted
// number is provably scripted.
//
// Deliberately a different type from matchsession_test's same-named helper (which does repeat
// the last face, and other tests in that package rely on it) — a different package, so it is
// not importable from here anyway.
type scriptedFaces struct {
	faces   []int
	i       int
	overran bool // set once a roll asks for a face beyond what was scripted
}

func (s *scriptedFaces) RollDie(_ enum.DieSides) int {
	if s.i >= len(s.faces) {
		s.overran = true
		return -1 // an impossible face: anything that accidentally depends on it fails loudly
	}
	f := s.faces[s.i]
	s.i++
	return f
}

// ─── fixture ────────────────────────────────────────────────────────────────

type combatFixture struct {
	server     *httptest.Server
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	playerUUID uuid.UUID
	attackerID uuid.UUID // sheet UUID of the attacking character
	victimID   uuid.UUID // sheet UUID of the target
	victim     *csSheet.CharacterSheet
	writer     *recordingStatusWriter
	session    *matchsession.MatchSession
}

// setRollSource replaces the session's dice for one test. The session pointer is the
// fixture's own, and nothing is in flight when a test calls this.
func (f *combatFixture) setRollSource(src service.RollSource) { f.session.SetRollSource(src) }

func newCombatFixture(t *testing.T) *combatFixture {
	t.Helper()

	f := &combatFixture{
		matchUUID:  uuid.New(),
		masterUUID: uuid.New(),
		playerUUID: uuid.New(),
		attackerID: uuid.New(),
		victimID:   uuid.New(),
		writer:     &recordingStatusWriter{},
	}
	// Both characters belong to the same player, which is enough here: authorization is
	// per player and the test only needs one client to send the attack.
	victimPlayer := f.playerUUID
	attacker := &match.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.attackerID, PlayerUUID: &f.playerUUID},
	}
	victim := &match.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: f.victimID, PlayerUUID: &victimPlayer},
	}

	f.victim = newCombatSheet(t)
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		f.attackerID: newCombatSheet(t),
		f.victimID:   f.victim,
	}
	session := matchsession.NewMatchSession(
		f.matchUUID, sheets, []*match.Participant{attacker, victim},
	)
	session.SetRollSource(topFaceSource{})
	f.session = session

	hub := game.NewHub()
	go hub.Run()

	roundRepo := &mockRoundRepoHandler{}
	handler := game.NewHandler(
		hub,
		&fogMatchRepo{masterUUID: f.masterUUID, started: true},
		&mockEnrollmentChecker{enrolled: true},
		&mockStartMatchUC{},
		&mockKickPlayerUC{},
		&combatSessionUC{session: session},
		// The real use cases: this is what makes the test end-to-end rather than a mock
		// round-trip. closeRound is real too — TestE2E_AnExhaustedRoundClosesItself needs the
		// round to actually close when the bar economy runs out, not just report it.
		appmatch.NewOpenNextActionUC(f.writer, appmatch.NewCloseRoundUC(roundRepo)),
		appmatch.NewPullActionUC(f.writer, appmatch.NewCloseRoundUC(roundRepo)),
		appmatch.NewEnqueueActionUC(),
		appmatch.NewAttachReactionUC(),
		appmatch.NewOpenReactionUC(),
		appmatch.NewCloseTurnUC(f.writer),
		&mockChangeSceneUCHandler{},
		roundRepo,
		&mockEnqueueMasterActionUCHandler{},
		// The real UC: the exhaustion economy in TestE2E_AnExhaustedRoundClosesItself only
		// exists in Race mode, and the mock never actually flips the session's round mode.
		appmatch.NewChangeRoundModeUC(),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)
	f.server = httptest.NewServer(mux)
	t.Cleanup(func() {
		f.server.Close()
		hub.Stop()
	})
	return f
}

// connect dials the master first and then the player. The order matters: the room refuses
// a player with lobby_not_open until the master has opened it, which is also the path that
// rehydrates the session for an already-started match.
func (f *combatFixture) connect(t *testing.T) (master, player *websocket.Conn) {
	t.Helper()
	master = connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	readMessage(t, master) // room_state — confirms the client is registered in the room
	player = connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	readMessage(t, player) // room_state
	return master, player
}

func newCombatSheet(t *testing.T) *csSheet.CharacterSheet {
	t.Helper()
	playerUUID := uuid.New()
	cs, err := csSheet.NewCharacterSheetFactory().Build(
		&playerUUID, nil, nil,
		csSheet.CharacterProfile{NickName: "Combatant", FullName: "Combat Test Subject"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

// victimHP reads the live sheet. The room mutates that same sheet from its own goroutine,
// so callers must only read it when no message is in flight — before anything is sent, and
// after the writer has confirmed the close persisted.
func (f *combatFixture) victimHP(t *testing.T) int {
	t.Helper()
	bar, ok := f.victim.GetAllStatusBar()[enum.Health]
	if !ok {
		t.Fatal("the victim's sheet has no health bar")
	}
	return bar.GetCurrent()
}

// awaitPersisted waits for the gateway to have been written to, which is the room
// goroutine's last act on the closing path — after it, the sheet is safe to read.
func (w *recordingStatusWriter) awaitPersisted(d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		w.mu.Lock()
		n := len(w.sheetUUIDs)
		w.mu.Unlock()
		if n > 0 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func (w *recordingStatusWriter) snapshot() ([]string, []int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]string(nil), w.sheetUUIDs...), append([]int(nil), w.healthCurr...)
}

func sendWS(t *testing.T, conn *websocket.Conn, msgType string, payload any) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s payload: %v", msgType, err)
	}
	raw, err := json.Marshal(map[string]any{
		"type": msgType, "payload": json.RawMessage(data),
	})
	if err != nil {
		t.Fatalf("marshal %s message: %v", msgType, err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("send %s: %v", msgType, err)
	}
}

// collector drains a connection in the background into a slice.
//
// A blocking read that times out is fatal for a gorilla connection — the socket cannot be
// read again afterwards. Asserting "this client never received X" by waiting for a timeout
// would therefore break the very connection the rest of the test still needs, so the
// player's traffic is collected as it arrives instead.
type collector struct {
	mu   sync.Mutex
	msgs []game.Message
}

func newCollector(conn *websocket.Conn) *collector {
	c := &collector{}
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg game.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			c.mu.Lock()
			c.msgs = append(c.msgs, msg)
			c.mu.Unlock()
		}
	}()
	return c
}

// await waits for a message of the given type to have arrived.
func (c *collector) await(want game.MessageType, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if c.count(want) > 0 {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func (c *collector) count(want game.MessageType) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, m := range c.msgs {
		if m.Type == want {
			n++
		}
	}
	return n
}

// snapshotMessages returns a copy of what the collector has gathered so far, so a caller can
// pull a specific message's payload out after an await has already confirmed it arrived.
func (c *collector) snapshotMessages() []game.Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]game.Message(nil), c.msgs...)
}

// awaitResolution reads until a resolution_updated arrives or the deadline passes.
func awaitResolution(t *testing.T, conn *websocket.Conn, d time.Duration) *game.ResolutionUpdatedPayload {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, data, err := conn.ReadMessage()
		if err != nil {
			return nil
		}
		var msg game.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type != game.MsgTypeResolutionUpdate {
			continue
		}
		var p game.ResolutionUpdatedPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			t.Fatalf("unmarshal resolution_updated: %v", err)
		}
		return &p
	}
	return nil
}

// ─── the test ───────────────────────────────────────────────────────────────

func TestE2E_AttackAgainstACharacterProducesDamage(t *testing.T) {
	f := newCombatFixture(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	playerMsgs := newCollector(player)

	hpBefore := f.victimHP(t)
	if hpBefore <= 0 {
		t.Fatalf("the victim starts at %d HP — the fixture is wrong", hpBefore)
	}

	// The player sends an attack, exactly as a bottom sheet would.
	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId":  f.attackerID.String(),
		"targetId": []string{f.victimID.String()},
		"speed":    map[string]any{"bar": 0, "rollCheck": map[string]any{"skillName": enum.Legerity.String()}},
		"attack": map[string]any{
			"weapon": "Sword",
			"hit":    map[string]any{"skillName": enum.Accuracy.String()},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})

	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}

	// The master opens it. That is when the projection reaches them.
	sendWS(t, master, "open_next_action", map[string]any{})

	opened := awaitResolution(t, master, 3*time.Second)
	if opened == nil {
		t.Fatal("the master never received a resolution_updated for the opened turn")
	}

	var projected int

	t.Run("the master sees real dice and a projection", func(t *testing.T) {
		if len(opened.Action.DiceRolled) != 2 {
			t.Errorf("DiceRolled = %v, want the two individual 2D10 dice", opened.Action.DiceRolled)
		}
		if !opened.Action.IsCritical {
			t.Error("every die landed on its top face, so this must read as a critical")
		}
		if opened.Action.Total != 20 {
			t.Errorf("Total = %d, want 20 (10 + 10, skills at 0)", opened.Action.Total)
		}
		if opened.Action.Margin == nil {
			t.Fatal("the margin must be derived once the dodge gives the attack a CD")
		}
		// The passive reflex dodge is Reflex(0) + 11.
		if *opened.Action.Margin != 20-11 {
			t.Errorf("Margin = %d, want %d", *opened.Action.Margin, 20-11)
		}
		if len(opened.Targets) != 1 {
			t.Fatalf("Targets = %+v, want one entry", opened.Targets)
		}
		tgt := opened.Targets[0]
		if tgt.TargetID != f.victimID {
			t.Errorf("TargetID = %v, want %v", tgt.TargetID, f.victimID)
		}
		if tgt.Avoided {
			t.Error("a total of 20 must beat the passive dodge of 11")
		}
		if !tgt.Defended {
			t.Error("the passive defense should succeed at a CD one ladder step lower")
		}
		// A Sword is D10 + D4 with a flat 2: 10 + 4 + 2 = 16. An armed attack against a
		// bare-handed defense is not reduced while damage types do not exist.
		if tgt.RawDamage != 16 {
			t.Errorf("RawDamage = %d, want 16 (10 + 4 dice + 2 flat)", tgt.RawDamage)
		}
		if tgt.ProjectedDamage != 16 {
			t.Errorf("ProjectedDamage = %d, want 16", tgt.ProjectedDamage)
		}
		projected = tgt.ProjectedDamage
	})

	t.Run("the projection does not touch the sheet", func(t *testing.T) {
		// The gateway is the safe probe here: reading the live sheet from this goroutine
		// would race the room's, and a dry run that had written would have gone through
		// the gateway.
		if persisted, _ := f.writer.snapshot(); len(persisted) != 0 {
			t.Errorf("nothing should have been persisted yet, got %v", persisted)
		}
	})

	t.Run("the player is told the turn opened but not what it computed", func(t *testing.T) {
		// The mechanics of an action are public when it opens; the calculation is the
		// master's until the turn closes.
		if !playerMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
			t.Error("the player should have been told the turn opened")
		}
		if n := playerMsgs.count(game.MsgTypeResolutionUpdate); n != 0 {
			t.Errorf("the player received %d resolution_updated messages, want 0", n)
		}
	})

	t.Run("closing the turn applies and persists the damage", func(t *testing.T) {
		// A second action gives the master something to open, which closes the first turn.
		sendWS(t, player, "enqueue_action", map[string]any{
			"actorId": f.attackerID.String(),
			"speed":   map[string]any{"bar": 0},
		})
		if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
			t.Fatal("the second action was never acknowledged as enqueued")
		}
		sendWS(t, master, "open_next_action", map[string]any{})

		if !f.writer.awaitPersisted(3 * time.Second) {
			t.Fatal("the closing turn never persisted the damage")
		}
		// Safe to read the sheet now: persisting is the room goroutine's last act on this
		// path.
		if got := f.victimHP(t); got != hpBefore-projected {
			t.Errorf("HP = %d, want %d (%d - %d)", got, hpBefore-projected, hpBefore, projected)
		}
		persisted, healths := f.writer.snapshot()
		if len(persisted) != 1 {
			t.Fatalf("expected exactly one sheet persisted, got %v", persisted)
		}
		if persisted[0] != f.victimID.String() {
			t.Errorf("persisted %s, want the victim %s", persisted[0], f.victimID)
		}
		if healths[0] != hpBefore-projected {
			t.Errorf("persisted HP = %d, want %d", healths[0], hpBefore-projected)
		}
		if n := playerMsgs.count(game.MsgTypeResolutionUpdate); n != 0 {
			t.Errorf("the player received %d resolution_updated messages even after the close, want 0", n)
		}
	})
}

// A payload without actorId cannot be attributed to a character, so it is refused at the
// boundary instead of resolving against a sheet nobody owns.
func TestE2E_AttackWithoutActorIDIsRefused(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	sendWS(t, player, "enqueue_action", map[string]any{
		"targetId": []string{f.victimID.String()},
		"attack": map[string]any{
			"hit":    map[string]any{"skillName": enum.Accuracy.String()},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})

	if code := awaitErrorCode(t, player, 2*time.Second); code != "invalid_action" {
		t.Errorf("error code = %q, want invalid_action", code)
	}
}

// An unknown skill name is a client bug and must come back as a WS error, not become a
// silent zero deep inside the resolver.
func TestE2E_AttackWithUnknownSkillIsRefused(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId":  f.attackerID.String(),
		"targetId": []string{f.victimID.String()},
		"attack": map[string]any{
			"hit":    map[string]any{"skillName": "Kamehameha"},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})

	if code := awaitErrorCode(t, player, 2*time.Second); code != "invalid_action" {
		t.Errorf("error code = %q, want invalid_action", code)
	}
}

// A player cannot act through a character they do not own.
func TestE2E_ActingThroughAnotherPlayersCharacterIsRefused(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	sendWS(t, player, "enqueue_action", map[string]any{
		"actorId":  uuid.New().String(), // a character nobody in this match owns
		"targetId": []string{f.victimID.String()},
	})

	if code := awaitErrorCode(t, player, 2*time.Second); code != "game_error" {
		t.Errorf("error code = %q, want game_error", code)
	}
}

// TestE2E_AnExhaustedRoundClosesItself proves the second done-criterion: the round ends on
// its own when nothing pending can still pay, and the whole table is told.
func TestE2E_AnExhaustedRoundClosesItself(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	masterMsgs := newCollector(master)
	// Started at connect time, before anything is sent — otherwise the round_closed
	// broadcast can arrive before this test ever asks for it and be missed entirely.
	playerMsgs := newCollector(player)

	// Race, so the economy is on at all.
	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{
		Mode: string(enum.Race),
	})
	if !masterMsgs.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("the regime switch was never announced")
	}

	// One action, from the attacker. topFaceSource gives 10+10 = 20 on every check, so the
	// price is 20 and the single action pays for itself exactly.
	sendWS(t, player, string(game.MsgTypeEnqueueAction), game.ActionPayload{
		ActorID:  f.attackerID,
		TargetID: []uuid.UUID{f.victimID},
		Attack:   &game.AttackPayload{Hit: game.RollCheckPayload{SkillName: enum.Accuracy.String()}},
	})
	// Wait for the enqueue to land before opening — enqueue_action (player) and
	// open_next_action (master) travel on different connections with no ordering guarantee
	// between them, so opening too early would find an empty queue.
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}

	// First open consumes it; the second finds nothing that can pay.
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the first action never opened")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})

	if !masterMsgs.await(game.MsgTypeRoundClosed, 2*time.Second) {
		t.Fatal("the round ran out and nobody was told")
	}

	t.Run("the payload names the regime that just ended", func(t *testing.T) {
		payload := awaitRoundClosed(t, masterMsgs)
		if payload.RoundMode != string(enum.Race) {
			t.Errorf("roundMode = %q, want %q", payload.RoundMode, enum.Race)
		}
	})

	t.Run("the players hear it too — the round is table state, not master state", func(t *testing.T) {
		if !playerMsgs.await(game.MsgTypeRoundClosed, 2*time.Second) {
			t.Error("the player should have heard round_closed too — it is table state, not master state")
		}
	})
}

// TestE2E_ARacingRoundRunsOnTheBars drives a whole Race round over real WebSockets: two
// characters enqueue, the master opens, the order comes out of the bars rather than out of
// insertion, and the balances that cross into the next round are the ones the rules say.
//
// The dice are scripted, so the numbers are exact instead of lucky.
func TestE2E_ARacingRoundRunsOnTheBars(t *testing.T) {
	f := newCombatFixture(t)
	// Legerity is 0 on a factory sheet, so actionSpeed IS the dice total. Both enqueued actions
	// below carry Attack{Hit: Accuracy} with no Weapon (unarmed → Fist, 3 damage dice), and
	// rollActionDice rolls EVERY check an action carries, once, at enqueue time — not just
	// Speed. Each action therefore consumes, in this order: Speed (2D10 primary + 2D10
	// secondary — only the primary pair sums into the result), Hit (same 4-face shape), then
	// Damage (Fist's 3 dice). 11 faces per action, 22 total, attacker enqueued before victim:
	//
	//	attacker Speed   faces[ 0: 4] = 3,3,3,3 → primary 3+3 = 6   (asserted: the frozen price)
	//	attacker Hit     faces[ 4: 8] = 5,5,5,5 → unasserted
	//	attacker Damage  faces[ 8:11] = 2,2,2   → unasserted
	//	victim   Speed   faces[11:15] = 8,8,8,8 → primary 8+8 = 16  (asserted: leads the bar)
	//	victim   Hit     faces[15:19] = 5,5,5,5 → unasserted
	//	victim   Damage  faces[19:22] = 2,2,2   → unasserted
	//
	// scriptedFaces fails loudly on any roll past face 22 instead of repeating one, so both
	// asserted numbers are provably driven by the script, not a coincidence of a fallback.
	src := &scriptedFaces{faces: []int{
		3, 3, 3, 3, // attacker speed  (asserted: 6)
		5, 5, 5, 5, // attacker hit    (unasserted)
		2, 2, 2, // attacker damage    (unasserted)
		8, 8, 8, 8, // victim speed    (asserted: 16)
		5, 5, 5, 5, // victim hit      (unasserted)
		2, 2, 2, // victim damage      (unasserted)
	}}
	f.setRollSource(src)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{
		Mode: string(enum.Race),
	})
	if !masterMsgs.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("the regime switch was never announced")
	}

	// bars_updated is re-broadcast after EVERY mutation (regime switch, each enqueue, each
	// open) and both connections see every broadcast, so by the time the regime-switch ack
	// landed there is already one bars_updated in flight. Plain `collector.await` only checks
	// "has at least one arrived, ever" — it would trivially re-pass on that first, stale one
	// instead of waiting for the fresh one each step below actually depends on. awaitCount
	// pins a specific, growing count instead, which is what makes each wait mean "the event
	// this step caused", not "some event of this type, at some point".
	if !awaitCount(playerMsgs, game.MsgTypeBarsUpdated, 1, 2*time.Second) {
		t.Fatal("bars_updated never followed the regime switch")
	}

	// enqueue_action (player) and open_next_action (master) travel on independent connections
	// with no ordering guarantee between them — awaiting the enqueue's own ack count (not a
	// bare await, for the same staleness reason as above) is the real happens-before edge.
	enqueue := func(actor uuid.UUID, wantAcks, wantBars int) {
		t.Helper()
		sendWS(t, player, string(game.MsgTypeEnqueueAction), game.ActionPayload{
			ActorID: actor,
			Attack:  &game.AttackPayload{Hit: game.RollCheckPayload{SkillName: enum.Accuracy.String()}},
		})
		if !awaitCount(playerMsgs, game.MsgTypeActionEnqueued, wantAcks, 2*time.Second) {
			t.Fatalf("actor %v: enqueue ack #%d never arrived", actor, wantAcks)
		}
		if !awaitCount(playerMsgs, game.MsgTypeBarsUpdated, wantBars, 2*time.Second) {
			t.Fatalf("actor %v: bars_updated #%d never followed the enqueue", actor, wantBars)
		}
	}
	enqueue(f.attackerID, 1, 2) // 6
	enqueue(f.victimID, 2, 3)   // 16

	// All of this round's dice already fell — rollActionDice runs entirely at enqueue time,
	// never on open — so this is the earliest point an overrun could have happened, and the
	// only thing that can rescue an under-scripted face list from becoming a silent 16-by-luck
	// again.
	if src.overran {
		t.Fatal("the scripted dice ran out — a roll consumed more faces than this test accounted for")
	}

	t.Run("bars_updated announces the order before anything opens", func(t *testing.T) {
		bars := lastBarsUpdated(t, playerMsgs)
		if len(bars.Order) != 2 {
			t.Fatalf("order = %d slots, want 2", len(bars.Order))
		}
		if bars.Order[0].ActorID != f.victimID {
			t.Error("the character who rolled 16 leads the general bar, not the one inserted first")
		}
	})

	t.Run("the master opens, and the faster character goes first", func(t *testing.T) {
		sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
		if !awaitCount(masterMsgs, game.MsgTypeTurnOpened, 1, 2*time.Second) {
			t.Fatal("nothing opened")
		}
		// Pins the bars_updated this open produced before "the price froze" reads it —
		// that subtest sends nothing itself, so it depends entirely on this wait.
		if !awaitCount(playerMsgs, game.MsgTypeBarsUpdated, 4, 2*time.Second) {
			t.Fatal("bars_updated never followed the first open")
		}
		opened := lastTurnOpened(t, masterMsgs)
		if opened.ActorID != f.victimID {
			t.Errorf("actor = %v, want the faster character %v", opened.ActorID, f.victimID)
		}
	})

	t.Run("the price froze at the slowest pending speed", func(t *testing.T) {
		bars := lastBarsUpdated(t, playerMsgs)
		if got := bars.Prices[string(action.BarAction)]; got != 6 {
			t.Errorf("price = %d, want the slowest pending speed, 6", got)
		}
	})

	t.Run("the second open takes the slower character", func(t *testing.T) {
		sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
		// count 2, not a bare await: turn_opened already fired once for the first open, so a
		// bare await would trivially re-pass on that stale message before the second one lands.
		if !awaitCount(masterMsgs, game.MsgTypeTurnOpened, 2, 2*time.Second) {
			t.Fatal("the second action never opened")
		}
		opened := lastTurnOpened(t, masterMsgs)
		if opened.ActorID != f.attackerID {
			t.Errorf("actor = %v, want %v", opened.ActorID, f.attackerID)
		}
	})

	t.Run("the third open finds nothing that can pay, and the round closes itself", func(t *testing.T) {
		sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
		if !masterMsgs.await(game.MsgTypeRoundClosed, 2*time.Second) {
			t.Fatal("the round ran out and nobody was told")
		}
	})

	t.Run("the carry crossed over, ceiling applied", func(t *testing.T) {
		// The closing bars_updated (broadcastBars runs unconditionally on every
		// open_next_action, before it decides whether a turn or the round closed) and
		// round_closed are two independent broadcasts; nothing pins their relative arrival
		// order at this connection. Waiting for the 6th bars_updated directly — count pinning
		// again — is what guarantees the carry-over numbers below are the post-close ones.
		if !awaitCount(playerMsgs, game.MsgTypeBarsUpdated, 6, 2*time.Second) {
			t.Fatal("bars_updated never followed the round closing")
		}
		bars := lastBarsUpdated(t, playerMsgs)
		byChar := map[uuid.UUID]game.CharacterBarsPayload{}
		for _, c := range bars.Characters {
			byChar[c.CharacterID] = c
		}
		// The fast one kept 16 − 6 = 10, clipped to the ceiling of 6.
		if got := byChar[f.victimID].ActionBalance; got != 6 {
			t.Errorf("fast balance = %v, want 6 — a leftover of 10 is clipped to the round price", got)
		}
		// The slowest of the round starts the next one from zero.
		if got := byChar[f.attackerID].ActionBalance; got != 0 {
			t.Errorf("slow balance = %v, want 0", got)
		}
		if len(byChar[f.attackerID].ActionSpeeds) != 0 {
			t.Error("the round's speed history is cleared; only the balance crosses over")
		}
	})
}

// awaitRoundClosed pulls the round_closed payload out of what the collector already has.
func awaitRoundClosed(t *testing.T, c *collector) game.RoundClosedPayload {
	t.Helper()
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeRoundClosed {
			continue
		}
		var p game.RoundClosedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal round_closed: %v", err)
		}
		return p
	}
	t.Fatal("no round_closed in the collected messages")
	return game.RoundClosedPayload{}
}

// awaitCount waits until a collector has gathered at least n messages of the given type.
//
// Plain `collector.await` only checks "has at least one arrived, ever" — once a message of
// that type has already been seen earlier in a test (bars_updated fires after every enqueue
// and every open; turn_opened after every open that doesn't close the round), that check is
// satisfied by the STALE one and returns immediately, racing whatever produced the fresh one.
// Waiting for a specific, growing count is what actually pins a fresh occurrence in place.
func awaitCount(c *collector, want game.MessageType, n int, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if c.count(want) >= n {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// lastBarsUpdated pulls the most recent bars_updated payload out of what the collector has
// gathered so far — the bars are re-broadcast after every enqueue and every open, so callers
// want the latest snapshot, not the first one.
func lastBarsUpdated(t *testing.T, c *collector) game.BarsUpdatedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeBarsUpdated {
			continue
		}
		var p game.BarsUpdatedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal bars_updated: %v", err)
		}
		return p
	}
	t.Fatal("no bars_updated in the collected messages")
	return game.BarsUpdatedPayload{}
}

// lastTurnOpened pulls the most recent turn_opened payload out of what the collector has
// gathered so far.
func lastTurnOpened(t *testing.T, c *collector) game.TurnOpenedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type != game.MsgTypeTurnOpened {
			continue
		}
		var p game.TurnOpenedPayload
		if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
			t.Fatalf("unmarshal turn_opened: %v", err)
		}
		return p
	}
	t.Fatal("no turn_opened in the collected messages")
	return game.TurnOpenedPayload{}
}

func awaitErrorCode(t *testing.T, conn *websocket.Conn, d time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, data, err := conn.ReadMessage()
		if err != nil {
			return ""
		}
		var msg game.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type != game.MsgTypeError {
			continue
		}
		var p game.ErrorPayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			t.Fatalf("unmarshal error payload: %v", err)
		}
		return p.Code
	}
	return ""
}
