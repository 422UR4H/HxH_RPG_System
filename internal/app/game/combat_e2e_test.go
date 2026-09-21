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
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
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
//  2. the player receives no resolution_updated while the turn is open — the calculation is
//     the master's alone until it settles (Phase 5 then projects it to the table);
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
	roundRepo  *mockRoundRepoHandler
	// bystanderUUID/bystanderID are uuid.Nil unless withBystander was passed. See it there.
	bystanderUUID uuid.UUID
	bystanderID   uuid.UUID
}

// combatOpt tweaks the fixture before the session is built. Without one, newCombatFixture
// builds exactly the two-character table every other test in this file already relies on.
type combatOpt func(*combatFixture)

// withBystander seats a THIRD player at the table, with a character of their own.
//
// A fog gate cannot be proved with two people: the master sees everything and the mover owns
// the piece, so both are entitled to the news no matter what the gate does. The bystander is
// the only recipient whose line of sight actually decides, which is why the leak test needs
// one.
func withBystander(f *combatFixture) {
	f.bystanderUUID = uuid.New()
	f.bystanderID = uuid.New()
}

// setRollSource replaces the session's dice for one test. The session pointer is the
// fixture's own, and nothing is in flight when a test calls this.
func (f *combatFixture) setRollSource(src service.RollSource) { f.session.SetRollSource(src) }

func newCombatFixture(t *testing.T, opts ...combatOpt) *combatFixture {
	t.Helper()

	f := &combatFixture{
		matchUUID:  uuid.New(),
		masterUUID: uuid.New(),
		playerUUID: uuid.New(),
		attackerID: uuid.New(),
		victimID:   uuid.New(),
		writer:     &recordingStatusWriter{},
	}
	for _, opt := range opts {
		opt(f)
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
	participants := []*match.Participant{attacker, victim}
	if f.bystanderUUID != uuid.Nil {
		sheets[f.bystanderID] = newCombatSheet(t)
		participants = append(participants, &match.Participant{
			UUID: uuid.New(), MatchUUID: f.matchUUID,
			Sheet: csEntity.Summary{UUID: f.bystanderID, PlayerUUID: &f.bystanderUUID},
		})
	}
	session := matchsession.NewMatchSession(f.matchUUID, sheets, participants)
	session.SetRollSource(topFaceSource{})
	f.session = session

	hub := game.NewHub()
	go hub.Run()

	roundRepo := &mockRoundRepoHandler{}
	f.roundRepo = roundRepo
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
		appmatch.NewEditActionUC(),
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

// enqueueAttack sends a plain sword attack from the fixture's attacker against its victim,
// over the given connection. It is the shape most combat e2e tests send inline; extracted
// here so a test that only cares about what happens after the enqueue does not have to
// spell it out again.
func (f *combatFixture) enqueueAttack(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	sendWS(t, conn, "enqueue_action", map[string]any{
		"actorId":  f.attackerID.String(),
		"targetId": []string{f.victimID.String()},
		"speed":    map[string]any{"bar": 0, "rollCheck": map[string]any{"skillName": enum.Legerity.String()}},
		"attack": map[string]any{
			"weapon": "Sword",
			"hit":    map[string]any{"skillName": enum.Accuracy.String()},
			"damage": map[string]any{"skillName": enum.Push.String()},
		},
	})
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
		// close_turn (Phase 5) closes the currently open turn directly, without touching the
		// scheduler's queue. An earlier version of this subtest enqueued a second action and
		// relied on open_next_action's implicit close-then-reopen instead; reopening from an
		// empty queue is a separate, pre-existing race in RoundOrchestrator.NextAction
		// (unrelated to resolution visibility) that intermittently made Execute return an
		// error and swallow the already-closed turn's resolution before anyone — master
		// included — ever saw it. close_turn sidesteps that path entirely and is the more
		// direct way to assert what this subtest is actually about.
		sendWS(t, master, "close_turn", map[string]any{"confirm": true})

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
		// Phase 5: resolution_updated is projected to the table once a turn SETTLES — this
		// replaces the old "the player receives nothing, ever" assertion, which encoded the
		// pre-Phase-5 master-only rule. The player here owns both the attacker and the victim
		// (see newCombatFixture), so they are entitled to the unredacted settled resolution,
		// same as the master's.
		if !playerMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
			t.Fatal("the player never received the settled resolution once the turn closed")
		}
		if n := playerMsgs.count(game.MsgTypeResolutionUpdate); n != 1 {
			t.Errorf("the player received %d resolution_updated messages, want exactly 1 (the settled one for the closed turn)", n)
		}
		var settled *game.ResolutionUpdatedPayload
		for _, m := range playerMsgs.snapshotMessages() {
			if m.Type != game.MsgTypeResolutionUpdate {
				continue
			}
			var p game.ResolutionUpdatedPayload
			if err := json.Unmarshal(m.Payload, &p); err != nil {
				t.Fatalf("unmarshal resolution_updated: %v", err)
			}
			if p.TurnID == opened.TurnID {
				settled = &p
				break
			}
		}
		if settled == nil {
			t.Fatal("no resolution_updated for the closed turn arrived on the player's connection")
		}
		if !settled.IsSettled {
			t.Error("the resolution the player received after close must be settled")
		}
		if len(settled.Targets) != 1 || settled.Targets[0].ProjectedDamage != projected {
			t.Errorf("player's settled targets = %+v, want one entry with ProjectedDamage %d", settled.Targets, projected)
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

// TestE2E_ReopeningAnEmptyQueueStillSettlesTheClosedTurn is Task 4b's delivery-level guarantee:
// a closed turn must never be dropped because the next one could not open.
//
// The fixture is Free mode (the default — no economy, no RoundExhausted branch), with exactly
// one queued action. The first open_next_action opens it as turn 1. The second finds an empty
// queue: MatchSession.OpenNextAction closes turn 1 (applying its damage) BEFORE it can fail to
// find a next action, so by the time Execute returns an error, turn 1 is already closed. This
// is deterministic, not a race — see the "closing the turn applies and persists the damage"
// subtest above, whose workaround exists precisely because this path used to swallow the
// closed turn's resolution.
func TestE2E_ReopeningAnEmptyQueueStillSettlesTheClosedTurn(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	hpBefore := f.victimHP(t)

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

	// Opens the only queued action as turn 1.
	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the first action never opened")
	}
	var turn1ID uuid.UUID
	for _, m := range masterMsgs.snapshotMessages() {
		if m.Type != game.MsgTypeTurnOpened {
			continue
		}
		var p game.TurnOpenedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal turn_opened: %v", err)
		}
		turn1ID = p.TurnID
	}
	if turn1ID == uuid.Nil {
		t.Fatal("no turn_opened payload carried a turn id")
	}

	// The queue is now empty. Reopening must still close turn 1 and report the failure to
	// open a next one — this is Task 4b's bug, reproduced over the real WS handler.
	sendWS(t, master, "open_next_action", map[string]any{})

	if !masterMsgs.await(game.MsgTypeError, 2*time.Second) {
		t.Fatal("the master was never told the reopen failed")
	}

	t.Run("the closed turn's settled resolution still reaches the table", func(t *testing.T) {
		findSettled := func(c *collector) *game.ResolutionUpdatedPayload {
			for _, m := range c.snapshotMessages() {
				if m.Type != game.MsgTypeResolutionUpdate {
					continue
				}
				var p game.ResolutionUpdatedPayload
				if err := json.Unmarshal(m.Payload, &p); err != nil {
					t.Fatalf("unmarshal resolution_updated: %v", err)
				}
				if p.TurnID == turn1ID && p.IsSettled {
					return &p
				}
			}
			return nil
		}
		if !masterMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
			t.Fatal("the master never received any resolution_updated for the failed reopen")
		}
		if findSettled(masterMsgs) == nil {
			t.Error("the master never received the settled resolution_updated for the closed turn")
		}
		// The player owns both the attacker and the victim (see newCombatFixture), so they are
		// entitled to the unredacted settled resolution too, same as in the sibling subtest
		// above.
		if !playerMsgs.await(game.MsgTypeResolutionUpdate, 2*time.Second) {
			t.Fatal("the player never received any resolution_updated for the closed turn")
		}
		if findSettled(playerMsgs) == nil {
			t.Error("the player never received the settled resolution_updated for the closed turn")
		}
	})

	t.Run("the closed turn was still persisted through PersistTurnClose", func(t *testing.T) {
		found := false
		for _, id := range f.roundRepo.persistedTurnIDs() {
			if id == turn1ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("persisted turn ids = %v, want %v among them", f.roundRepo.persistedTurnIDs(), turn1ID)
		}
	})

	t.Run("the damage was still applied and persisted to the sheet", func(t *testing.T) {
		if !f.writer.awaitPersisted(3 * time.Second) {
			t.Fatal("the closed turn's damage never persisted")
		}
		if got := f.victimHP(t); got >= hpBefore {
			t.Errorf("HP = %d, want less than %d (%d) — the hit that closed turn 1 must still land",
				got, hpBefore, hpBefore)
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

// O ack do jogador tem que nomear a MESMA ação que o mestre viu entrar na fila. Um actionId
// qualquer, não-zero, passaria num teste que só checasse "não é zero" — e um ID que não casa
// com o do mestre é pior que nenhum, porque pull_action falharia sem explicação.
func TestEnqueueActionAckNamesTheSameActionTheMasterSaw(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck

	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	f.enqueueAttack(t, player)

	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the player was never acknowledged for their own enqueue")
	}
	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master never saw the action land in the queue")
	}

	var ack game.ActionEnqueuedPayload
	for _, m := range playerMsgs.snapshotMessages() {
		if m.Type == game.MsgTypeActionEnqueued {
			if err := json.Unmarshal(m.Payload, &ack); err != nil {
				t.Fatalf("unmarshal action_enqueued: %v", err)
			}
			break
		}
	}

	var queued game.ActionQueuedPayload
	for _, m := range masterMsgs.snapshotMessages() {
		if m.Type == game.MsgTypeActionQueued {
			if err := json.Unmarshal(m.Payload, &queued); err != nil {
				t.Fatalf("unmarshal action_queued: %v", err)
			}
			break
		}
	}

	if ack.ActionID == uuid.Nil {
		t.Fatal("action_enqueued came back with a zero actionId; the player cannot address their own action")
	}
	if ack.ActionID != queued.ActionID {
		t.Fatalf("ack names %s but the master saw %s enter the queue", ack.ActionID, queued.ActionID)
	}
}

// latestBarsSeq returns the highest Seq among the bars_updated messages a collector has
// gathered so far. This is the test's proxy for "what seq had the table already seen before
// the late connection" — the number match_full_state's Bars.Seq must reproduce EXACTLY, not
// a fresh one, or a reconnecting client's guard against out-of-order snapshots would reset.
func latestBarsSeq(t *testing.T, msgs []game.Message) uint64 {
	t.Helper()
	var seq uint64
	found := false
	for _, m := range msgs {
		if m.Type != game.MsgTypeBarsUpdated {
			continue
		}
		var p game.BarsUpdatedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal bars_updated: %v", err)
		}
		if !found || p.Seq > seq {
			seq = p.Seq
			found = true
		}
	}
	if !found {
		t.Fatal("no bars_updated observed before the late connections — the test setup is wrong")
	}
	return seq
}

// findMessage returns the first message of the given type, or fails the test — every caller
// here already confirmed one arrived via collector.await, so a miss means the payload could
// not be found where await said it would be, not that it never came.
func findMessage(t *testing.T, msgs []game.Message, want game.MessageType) game.Message {
	t.Helper()
	for _, m := range msgs {
		if m.Type == want {
			return m
		}
	}
	t.Fatalf("no %s message found in %d collected messages", want, len(msgs))
	return game.Message{}
}

// Quem entra no meio — ou reconecta, e o hook do front reconecta até cinco vezes sozinho —
// fica sem barras, sem regime, sem cena, sem turno aberto e sem reações pendentes, até alguma
// coisa mudar por acaso. match_full_state fecha esse buraco no register arm de Run().
func TestMatchFullStateOnConnectCarriesTheCombatState(t *testing.T) {
	f := newCombatFixture(t)

	// The master stays connected for the whole test — it is the one who opens the turn, and
	// keeping it up means the room never goes empty (an empty room closes itself in Run(),
	// which would also reset r.barsSeq on the next connect — a false positive for the seq
	// assertion below).
	master := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	defer master.Close()   //nolint:errcheck
	readMessage(t, master) // room_state
	masterMsgs := newCollector(master)

	// A first player connection sends the attack, then drops — the room stays alive because
	// the master is still there.
	setupPlayer := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	readMessage(t, setupPlayer) // room_state
	setupPlayerMsgs := newCollector(setupPlayer)

	f.enqueueAttack(t, setupPlayer)
	if !setupPlayerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}
	setupPlayer.Close() //nolint:errcheck

	// The master opens it — this is what leaves a turn open for the late connections below.
	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeResolutionUpdate, 3*time.Second) {
		t.Fatal("the master never received the projection for the opened turn")
	}
	if !masterMsgs.await(game.MsgTypeBarsUpdated, 2*time.Second) {
		t.Fatal("no bars_updated observed before the late connections — nothing to compare seq against")
	}
	// The CURRENT seq, as the table already knew it, captured BEFORE either late connection —
	// this is the ground truth the Bars.Seq assertions below are checked against.
	wantSeq := latestBarsSeq(t, masterMsgs.snapshotMessages())

	// The player reconnects mid-turn — a genuine late joiner on this connection, exactly the
	// "reconnects up to five times on its own" case the front's hook produces.
	latePlayer := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	defer latePlayer.Close() //nolint:errcheck
	latePlayerMsgs := newCollector(latePlayer)
	if !latePlayerMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the late player never received match_full_state")
	}

	// The master reconnects too — closing and redialing the same connection while the turn is
	// still open. latePlayer is already registered at this point, so the room does not empty
	// out when the original master connection drops.
	master.Close() //nolint:errcheck
	lateMaster := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	defer lateMaster.Close() //nolint:errcheck
	lateMasterMsgs := newCollector(lateMaster)
	if !lateMasterMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the reconnecting master never received match_full_state")
	}

	var p, m game.MatchFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, latePlayerMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &p,
	); err != nil {
		t.Fatalf("unmarshal the late player's match_full_state: %v", err)
	}
	if err := json.Unmarshal(
		findMessage(t, lateMasterMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &m,
	); err != nil {
		t.Fatalf("unmarshal the reconnecting master's match_full_state: %v", err)
	}

	if p.OpenTurn == nil || p.OpenTurn.TurnID == uuid.Nil {
		t.Fatal("the late player did not learn that a turn is open")
	}
	if p.RoundMode == "" {
		t.Fatal("the late player did not learn the round regime")
	}
	// Eixo do TEMPO: o turno está aberto, logo o cálculo é do mestre.
	if p.Resolution != nil {
		t.Fatal("an open turn's resolution reached a player; that calculation is the master's")
	}
	if m.Resolution == nil {
		t.Fatal("the master reconnected into an open turn and lost the calculation")
	}
	// O seq atravessa a reconexão: se viesse um seq novo, a guarda do cliente zeraria e o
	// próximo bars_updated atrasado seria aplicado por cima de um estado mais novo.
	if p.Bars.Seq != wantSeq {
		t.Fatalf("match_full_state stamped seq %d for the player, want the current %d", p.Bars.Seq, wantSeq)
	}
	if m.Bars.Seq != wantSeq {
		t.Fatalf("match_full_state stamped seq %d for the master, want the current %d", m.Bars.Seq, wantSeq)
	}
}

// This is the regression buildMatchFullState's round.HasOpenTurn() guard exists to prevent.
//
// round.CurrentTurn() returns the round's LAST turn regardless of whether it already closed —
// a round sits with its last turn closed for the whole window between close_turn and the next
// open_next_action. A late connection during that window must NOT be told a turn is open, and
// the master must not be handed a Resolution for it either: that closed turn's settled
// resolution was already broadcast to the table via resolution_updated when it closed, and
// match_full_state does not owe it again.
//
// A naive `if t := round.CurrentTurn(); t != nil` (rather than HasOpenTurn()) would pass every
// other assertion in this file — the sibling test above always reconnects while a turn is
// still open — so this scenario needs its own connection, made specifically in the closed
// window, to be caught at all.
func TestMatchFullStateOmitsOpenTurnBetweenCloseAndNextOpen(t *testing.T) {
	f := newCombatFixture(t)

	// The master stays connected for the whole test, same reasoning as the sibling test: an
	// empty room closes itself in Run(), which would reset r.barsSeq on the next connect.
	master := connectWS(t, f.server.URL, f.masterUUID, f.matchUUID)
	defer master.Close()   //nolint:errcheck
	readMessage(t, master) // room_state
	masterMsgs := newCollector(master)

	setupPlayer := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	readMessage(t, setupPlayer) // room_state
	setupPlayerMsgs := newCollector(setupPlayer)

	f.enqueueAttack(t, setupPlayer)
	if !setupPlayerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the action was never acknowledged as enqueued")
	}
	setupPlayer.Close() //nolint:errcheck

	sendWS(t, master, "open_next_action", map[string]any{})
	if !masterMsgs.await(game.MsgTypeResolutionUpdate, 3*time.Second) {
		t.Fatal("the master never received the projection for the opened turn")
	}

	// Close it — and deliberately do NOT open a next one. Nothing else is queued, so the round
	// itself stays open; only its last turn is closed. This is the exact window the fix
	// protects.
	sendWS(t, master, "close_turn", map[string]any{"confirm": true})
	if !f.writer.awaitPersisted(3 * time.Second) {
		t.Fatal("the turn never closed and persisted")
	}

	// A late connection now, with the round sitting on "last turn closed, nothing open yet".
	late := connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
	defer late.Close() //nolint:errcheck
	lateMsgs := newCollector(late)
	if !lateMsgs.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the late connection never received match_full_state")
	}

	var got game.MatchFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, lateMsgs.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &got,
	); err != nil {
		t.Fatalf("unmarshal match_full_state: %v", err)
	}

	if got.OpenTurn != nil {
		t.Fatalf("OpenTurn = %+v, want nil — the round's last turn already closed", got.OpenTurn)
	}
	if got.Resolution != nil {
		t.Fatal("Resolution present for a round with no open turn — the closed turn's " +
			"resolution was already broadcast to the table when it settled")
	}
}

// ─── o movimento aplicado ao tabuleiro ──────────────────────────────────────
//
// Até aqui uma ação de mover acontecia e a peça não saía do lugar: o motor resolvia a
// velocidade e checava a parede, mas nada aplicava o resultado ao tabuleiro — e o fog nunca
// recalculava, porque do ponto de vista do servidor nada tinha se movido.

// moveBoardGrid is the board the two move tests share: 30x30 square cells of 64px, so cell
// (c,r) centres on ((c+0.5)*64, (r+0.5)*64).
var moveBoardGrid = mapentity.GridShape{
	Kind: mapentity.GridKindSquare, Cols: 30, Rows: 30, CellSize: 64, SkewRatio: 1,
}

// moveBoardWall runs the full height of the board at x=640, splitting it in two halves that
// cannot see each other. It is what makes the bystander genuinely blind to the move instead
// of merely forgotten by the room.
var moveBoardWall = mapentity.WallSegment{
	ID: "divider", P1: [2]float64{640, 0}, P2: [2]float64{640, 1920},
	WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
	Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
}

const (
	attackerPieceID  = "piece-attacker"
	bystanderPieceID = "piece-bystander"
	// attackerElevation is the mover's virtual height in metres, non-zero so that a
	// horizontal move that flattened it would be caught instead of looking like a no-op.
	attackerElevation = 2.5
)

// syncBoard seeds the in-memory board from the master, exactly as the real client does once
// its REST map has loaded. The room answers it by re-pushing map_full_state to everyone.
//
// The attacker starts at (4,4) → (288,288), west of the wall. The bystander, when the fixture
// has one, sits at (20,4) → (1312,288), east of it: nothing west of x=640 is in their line of
// sight, which is the whole point of them.
func (f *combatFixture) syncBoard(t *testing.T, master *websocket.Conn) {
	t.Helper()
	col, row := 4, 4
	pieces := []game.PieceMovedPayload{{
		PieceID:     attackerPieceID,
		CharacterID: f.attackerID.String(),
		Slot:        game.SlotPayload{Kind: "square", Col: &col, Row: &row},
		// The mover starts ELEVATED on purpose. Z is a virtual height in metres and
		// Move.Position[2] is a grid index; a horizontal step must not flatten the piece.
		Z: attackerElevation,
	}}
	if f.bystanderUUID != uuid.Nil {
		bcol, brow := 20, 4
		pieces = append(pieces, game.PieceMovedPayload{
			PieceID:     bystanderPieceID,
			CharacterID: f.bystanderID.String(),
			Slot:        game.SlotPayload{Kind: "square", Col: &bcol, Row: &brow},
		})
	}
	grid := toGridShapePayload(moveBoardGrid)
	sendWS(t, master, "map_state_sync", game.MapStateSyncPayload{
		Pieces: &pieces,
		Walls:  []game.WallSegmentPayload{toWallSegmentPayload(moveBoardWall)},
		Grid:   &grid,
	})
}

// enqueueDash sends a Dash from the fixture's attacker over the given connection.
//
// Dash is one of the only two categories the mapper accepts (the other is Shift), and neither
// rolls against a DC — which is precisely the branch that displaces the piece on opening.
func (f *combatFixture) enqueueDash(t *testing.T, conn *websocket.Conn, from, to [3]int) {
	t.Helper()
	sendWS(t, conn, "enqueue_action", map[string]any{
		"actorId": f.attackerID.String(),
		"move": map[string]any{
			"category": string(enum.Dash),
			"from":     from,
			"position": to,
		},
	})
}

// collectFrom starts a collector after clearing the read deadline readMessage left behind.
//
// A collector whose connection still carries that 2s deadline stops reading two seconds after
// the handshake. In a test that proves a NEGATIVE — "this player was never told" — that would
// turn "nobody was told" into "we stopped listening", and the test would pass for the wrong
// reason.
func collectFrom(conn *websocket.Conn) *collector {
	_ = conn.SetReadDeadline(time.Time{})
	return newCollector(conn)
}

// messageTypes lists what a collector gathered, for a failure message. The raw payloads are
// json.RawMessage, which %v prints as thousands of byte values — unreadable in a test log.
func messageTypes(msgs []game.Message) []game.MessageType {
	out := make([]game.MessageType, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Type)
	}
	return out
}

// indexOfMessage returns the arrival position of the first message of that type, or -1.
// The collector appends in arrival order, so comparing indices compares arrival order.
func indexOfMessage(msgs []game.Message, want game.MessageType) int {
	for i, m := range msgs {
		if m.Type == want {
			return i
		}
	}
	return -1
}

// Na ABERTURA, não no fechamento: o dano espera o fechamento porque pode ser editado; a
// posição não pode, porque as reactions seguintes dependem de onde a peça está — e a mesa não
// pode ver o turno abrir com a peça ainda no slot velho.
func TestE2E_OpeningAMoveActionMovesThePieceBeforeTurnOpened(t *testing.T) {
	f := newCombatFixture(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	f.syncBoard(t, master)
	if !masterMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the master never got the board back after map_state_sync — the fixture never started")
	}

	f.enqueueDash(t, player, [3]int{4, 4, 0}, [3]int{6, 4, 0})
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the dash was never enqueued")
	}

	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})

	if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("no piece_moved: the resolved move was never applied to the board")
	}
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn_opened: the action never opened, so this test measured nothing")
	}

	msgs := masterMsgs.snapshotMessages()
	moved := findMessage(t, msgs, game.MsgTypePieceMoved)

	var mp game.PieceMovedPayload
	if err := json.Unmarshal(moved.Payload, &mp); err != nil {
		t.Fatalf("unmarshal piece_moved: %v", err)
	}
	if mp.PieceID != attackerPieceID {
		t.Fatalf("piece_moved carried pieceId %q, want %q — the engine moved the wrong piece",
			mp.PieceID, attackerPieceID)
	}
	if mp.Slot.Col == nil || mp.Slot.Row == nil {
		t.Fatalf("piece_moved carried no square slot: %+v", mp.Slot)
	}
	if *mp.Slot.Col != 6 || *mp.Slot.Row != 4 {
		t.Fatalf("piece landed on (%d,%d), want (6,4)", *mp.Slot.Col, *mp.Slot.Row)
	}
	if mp.Slot.Kind != "square" {
		t.Fatalf("slot kind = %q, want the piece's own kind %q", mp.Slot.Kind, "square")
	}
	// A horizontal step must not drop an elevated piece to the ground. The front draws the
	// position that arrives without recomputing it, so writing Move.Position[2] into Z here
	// would visibly flatten the token on screen.
	if mp.Z != attackerElevation {
		t.Fatalf("piece_moved carried z=%v, want the elevation it already had (%v): a "+
			"sideways move must not change the piece's height", mp.Z, attackerElevation)
	}
	// The server is the author of this one: no browser predicted it.
	if moved.SenderID != uuid.Nil {
		t.Fatalf("piece_moved senderId = %s, want the zero UUID of a server message", moved.SenderID)
	}

	opened := findMessage(t, msgs, game.MsgTypeTurnOpened)

	// The assertion that actually bites. Both envelopes are stamped when they are BUILT, and
	// both are built by the SAME goroutine — the master connection's ReadPump, which runs
	// handleClientMessage's open_next_action arm from end to end. Comparing the stamps
	// therefore compares the order the server decided, with no scheduling in between.
	// Arrival order alone cannot do that job here:
	// turn_opened travels through r.broadcast (a 256-slot buffered channel drained by
	// Room.Run) while piece_moved goes straight into each client's queue, so a server that
	// applied the move late would still, most of the time, have its piece_moved overtake the
	// broadcast on the way out. Verified by injecting exactly that regression.
	if opened.Timestamp.Before(moved.Timestamp) {
		t.Fatalf("turn_opened was built at %s, BEFORE piece_moved at %s: the move was applied "+
			"after the turn opened, so the table saw the turn open with the piece still in "+
			"the old slot", opened.Timestamp, moved.Timestamp)
	}

	// And the table's own experience: same client, same queue, piece_moved first.
	movedAt := indexOfMessage(msgs, game.MsgTypePieceMoved)
	openedAt := indexOfMessage(msgs, game.MsgTypeTurnOpened)
	if movedAt > openedAt {
		t.Fatalf("turn_opened (index %d) arrived before piece_moved (index %d); the table saw "+
			"the turn open with the piece still in the old slot", openedAt, movedAt)
	}
}

// O que de fato prova a projeção: quem não enxerga nem a origem nem o destino não é avisado.
//
// Sem esse teste, o anterior prova só que UMA mensagem foi emitida — não que o gate de fog
// decide quem a recebe.
func TestE2E_OpeningAMoveActionDoesNotLeakToWhoCannotSeeIt(t *testing.T) {
	f := newCombatFixture(t, withBystander)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	blind := connectWS(t, f.server.URL, f.bystanderUUID, f.matchUUID)
	defer blind.Close()   //nolint:errcheck
	readMessage(t, blind) // room_state

	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)
	blindMsgs := collectFrom(blind)

	f.syncBoard(t, master)
	if !blindMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the bystander never got a board — they are not really at the table")
	}

	// The premise of the whole test: the bystander sees their own piece and NOT the mover's.
	// If the wall ever stops splitting the board, this fails here instead of quietly turning
	// the assertion below into a tautology.
	var board game.MapFullStatePayload
	if err := json.Unmarshal(
		findMessage(t, blindMsgs.snapshotMessages(), game.MsgTypeMapFullState).Payload, &board,
	); err != nil {
		t.Fatalf("unmarshal map_full_state: %v", err)
	}
	sees := map[string]bool{}
	for _, p := range board.Pieces {
		sees[p.PieceID] = true
	}
	if !sees[bystanderPieceID] {
		t.Fatal("the bystander cannot even see their own piece — their line of sight is empty, " +
			"so this test would prove nothing about the fog gate")
	}
	if sees[attackerPieceID] {
		t.Fatal("the bystander can see the mover's piece: the wall is not splitting the board " +
			"and there is nothing blind about this player")
	}

	f.enqueueDash(t, player, [3]int{4, 4, 0}, [3]int{6, 4, 0})
	if !playerMsgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the dash was never enqueued")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})

	// The move really happened — otherwise "the bystander heard nothing" is trivially true.
	if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("no piece_moved reached the master: nothing moved, so the gate was never exercised")
	}
	// turn_opened is the barrier, not a sleep: the room dispatches piece_moved before it, on
	// the same goroutine, so anything the bystander was entitled to is already in their queue
	// by the time turn_opened lands.
	if !blindMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the bystander received nothing at all — this connection is dead, not gated")
	}

	if n := blindMsgs.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("a player who sees neither end of the move was told about it %d time(s); "+
			"they received: %v", n, messageTypes(blindMsgs.snapshotMessages()))
	}
	// piece_removed is the other half of the pair: it is for whoever could see the ORIGIN and
	// no longer can. The bystander could see neither end, so they get that one no more than
	// the other.
	if n := blindMsgs.count(game.MsgTypePieceRemoved); n != 0 {
		t.Fatalf("the bystander was told the piece left a slot they never saw it in (%d time(s))", n)
	}
}

// ─── o caminho cliente→servidor do mesmo helper ─────────────────────────────
//
// Os dois testes acima rodam com origin == uuid.Nil e portanto NUNCA entram nas linhas que a
// extração mudou: o "pule o remetente" e o envelope NewClientMessage. Quem entra nelas é o
// piece_moved que um cliente manda — e `case MsgTypePieceMoved` não tem gate de mestre nem de
// fase, então esse caminho está vivo no meio do combate, não só no lobby.

// awaitAtLeast waits until at least n messages of that type have arrived.
//
// The move tests need counts, not presence: every client already holds one map_full_state from
// the board sync, so "the owner was refreshed" is the SECOND one, not the first.
func awaitAtLeast(c *collector, want game.MessageType, n int, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if c.count(want) >= n {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// sendPieceMoved drags a piece from a client, the way a browser does when someone moves a
// token with the mouse.
func sendPieceMoved(t *testing.T, conn *websocket.Conn, pieceID, characterID string, col, row int) {
	t.Helper()
	sendWS(t, conn, string(game.MsgTypePieceMoved), map[string]any{
		"pieceId":     pieceID,
		"characterId": characterID,
		"slot":        map[string]any{"kind": "square", "col": col, "row": row},
	})
}

// O jogador arrasta a própria peça: o mestre é avisado, o remetente não recebe eco (o browser
// dele já desenhou), o envelope leva o UUID dele, e a visão dele é recalculada.
func TestE2E_APlayerDraggingTheirOwnPieceIsNotEchoedBackToThemselves(t *testing.T) {
	f := newCombatFixture(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	f.syncBoard(t, master)
	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}

	sendPieceMoved(t, player, attackerPieceID, f.attackerID.String(), 6, 4)

	if !masterMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the master was never told the player moved a piece")
	}
	moved := findMessage(t, masterMsgs.snapshotMessages(), game.MsgTypePieceMoved)
	if moved.SenderID != f.playerUUID {
		t.Fatalf("piece_moved senderId = %s, want the mover's own UUID %s — a relayed client "+
			"move is not a server message", moved.SenderID, f.playerUUID)
	}

	// The owner's line of sight moved with the piece, so a second map_full_state must reach
	// them. It is also the ordering barrier for the assertion below: the room dispatches the
	// relay BEFORE it sends this, on the same goroutine, so any echo would already be queued.
	if !awaitAtLeast(playerMsgs, game.MsgTypeMapFullState, 2, 2*time.Second) {
		t.Fatal("the mover's line of sight was never recomputed: no second map_full_state")
	}
	if n := playerMsgs.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("the mover was echoed their own move back %d time(s); they received: %v",
			n, messageTypes(playerMsgs.snapshotMessages()))
	}
}

// O mestre arrasta a peça de um JOGADOR. Esta é a única prova possível do delta declarado na
// extração: o map_full_state é decidido pelo dono do personagem, não pelo remetente. Antes de
// d0b4191 o jogador não recebia nada e ficava com o fog velho.
func TestE2E_TheMasterDraggingAPlayersPieceRefreshesThatPlayersSight(t *testing.T) {
	f := newCombatFixture(t)

	master, player := f.connect(t)
	defer master.Close() //nolint:errcheck
	defer player.Close() //nolint:errcheck
	masterMsgs := collectFrom(master)
	playerMsgs := collectFrom(player)

	f.syncBoard(t, master)
	if !playerMsgs.await(game.MsgTypeMapFullState, 2*time.Second) {
		t.Fatal("the player never got the board — the fixture never started")
	}

	// The master drags a piece that belongs to the PLAYER.
	sendPieceMoved(t, master, attackerPieceID, f.attackerID.String(), 6, 4)

	if !awaitAtLeast(playerMsgs, game.MsgTypeMapFullState, 2, 2*time.Second) {
		t.Fatal("the master moved the player's piece and the player's line of sight was never " +
			"recomputed — the owner is being resolved from the sender again")
	}

	var board game.MapFullStatePayload
	msgs := playerMsgs.snapshotMessages()
	var last *game.Message
	for i := range msgs {
		if msgs[i].Type == game.MsgTypeMapFullState {
			last = &msgs[i]
		}
	}
	if err := json.Unmarshal(last.Payload, &board); err != nil {
		t.Fatalf("unmarshal map_full_state: %v", err)
	}
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

	// The player is not the sender, so they are relayed the move as well.
	if !playerMsgs.await(game.MsgTypePieceMoved, 2*time.Second) {
		t.Fatal("the player was never relayed the move the master made")
	}
	// The master IS the sender, so they are not echoed their own drag.
	if n := masterMsgs.count(game.MsgTypePieceMoved); n != 0 {
		t.Fatalf("the master was echoed their own drag back %d time(s); they received: %v",
			n, messageTypes(masterMsgs.snapshotMessages()))
	}
}
