package game_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// fakeMatchEndRepo é o banco do fim da partida: guarda o que foi gravado e o devolve na leitura,
// como matches.game_end_at + match_clocks.story_at.
type fakeMatchEndRepo struct {
	mu      sync.Mutex
	written []appmatch.MatchEndData
	state   *appmatch.MatchEndState
	fail    bool
}

func (f *fakeMatchEndRepo) setFail(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fail = v
}

func (f *fakeMatchEndRepo) finish(st appmatch.MatchEndState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = &st
}

func (f *fakeMatchEndRepo) PersistMatchEnd(_ context.Context, d appmatch.MatchEndData) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return errors.New("write failed")
	}
	if f.state != nil {
		return errors.New("already ended")
	}
	f.written = append(f.written, d)
	at := d.StoryAt
	f.state = &appmatch.MatchEndState{StoryAt: &at, GameEndAt: d.GameEndAt}
	return nil
}

func (f *fakeMatchEndRepo) ReadMatchEnd(context.Context, uuid.UUID) (*appmatch.MatchEndState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state == nil {
		return nil, nil
	}
	st := *f.state
	return &st, nil
}

func (f *fakeMatchEndRepo) writes() []appmatch.MatchEndData {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]appmatch.MatchEndData(nil), f.written...)
}

func awaitMatchEnded(t *testing.T, c *collector) game.MatchEndedPayload {
	t.Helper()
	if !c.await(game.MsgTypeMatchEnded, 2*time.Second) {
		t.Fatal("match_ended never arrived")
	}
	var p game.MatchEndedPayload
	_ = json.Unmarshal(findMessage(t, c.snapshotMessages(), game.MsgTypeMatchEnded).Payload, &p)
	return p
}

// awaitCloseCode espera a leitura do collector terminar e devolve o código do fechamento.
func awaitCloseCode(t *testing.T, c *collector) (int, string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		err := c.readErr
		c.mu.Unlock()
		if err != nil {
			var ce *websocket.CloseError
			if errors.As(err, &ce) {
				return ce.Code, ce.Text
			}
			t.Fatalf("connection ended with %v, want a close frame", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the connection was never closed")
	return 0, ""
}

func TestEndMatchBroadcastsAndCloses4002(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, pc := cf.connectCollectors(t)

	sendWS(t, cf.masterConn, "play_clock", map[string]any{})
	awaitClockChanged(t, pc, 1)
	cf.now.advance(30 * time.Second)

	sendWS(t, cf.masterConn, "end_match", map[string]any{
		"storyAt": "2000-01-01T05:00:00", "briefFinalDescription": "  A caravana chegou  ",
	})

	for name, c := range map[string]*collector{"player": pc, "master": mc} {
		got := awaitMatchEnded(t, c)
		if got.MatchID != cf.matchUUID || strOf(got.StoryAt) != "2000-01-01T05:00:00" || got.GameEndAt != "2026-11-08T12:00:30Z" {
			t.Errorf("%s match_ended = %+v, want match %s, story 05:00:00, gameEndAt 2026-11-08T12:00:30Z", name, got, cf.matchUUID)
		}
		if code, text := awaitCloseCode(t, c); code != 4002 || text != "match ended" {
			t.Errorf("%s close = %d %q, want 4002 \"match ended\"", name, code, text)
		}
	}

	ws := cf.ends.writes()
	if len(ws) != 1 {
		t.Fatalf("PersistMatchEnd calls = %d, want 1", len(ws))
	}
	d := ws[0]
	story := time.Date(2000, 1, 1, 5, 0, 0, 0, time.UTC)
	if d.MatchUUID != cf.matchUUID || !d.StoryAt.Equal(story) || !d.GameEndAt.Equal(cf.now.Now()) {
		t.Errorf("written = %+v, want story 05:00:00 at the injected real time", d)
	}
	if d.BriefFinalDescription == nil || *d.BriefFinalDescription != "A caravana chegou" {
		t.Errorf("briefFinalDescription = %v, want the trimmed text", d.BriefFinalDescription)
	}
	if d.Clock.Running || d.Clock.StoryAt == nil || !d.Clock.StoryAt.Equal(story) {
		t.Errorf("clock written = %+v, want paused at 05:00:00", d.Clock)
	}
	sc, rd := d.End.Scene, d.End.Round
	if sc == nil || rd == nil || sc.GetFinishedAt() == nil || rd.GetFinishedAt() == nil ||
		sc.GetStoryFinishedAt() == nil || !sc.GetStoryFinishedAt().Equal(story) {
		t.Fatalf("end pair = %+v / %+v, want both closed, story finished 05:00:00", sc, rd)
	}
	ev := d.RealTimeEvent
	if ev == nil || ev.Kind != matchevent.KindClockRealTime || ev.SceneUUID != sc.GetID() || ev.RoundUUID != rd.GetID() {
		t.Fatalf("real time event = %+v, want clockRealTime in the closed pair", ev)
	}
	if p := payloadMap(t, *ev); p["elapsedSeconds"] != float64(30) || p["reason"] != "end" {
		t.Errorf("real time event payload = %v, want 30 s, reason end", p)
	}
	if !cf.session.IsEnded() {
		t.Error("session not ended after a successful write")
	}
}

func TestEndMatchRefusedInBattleScene(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, pc := cf.connectCollectors(t)

	sendWS(t, cf.playerConn, "end_match", map[string]any{"storyAt": "2000-01-01T05:00:00"})
	awaitClockError(t, pc, "forbidden")
	sendWS(t, cf.masterConn, "end_match", map[string]any{"storyAt": "2000-01-01T05:00:00", "briefFinalDescription": strings.Repeat("x", 256)})
	awaitClockError(t, mc, "invalid_payload")
	sendWS(t, cf.masterConn, "end_match", map[string]any{"storyAt": "1999-12-31T23:00:00"})
	awaitClockError(t, mc, "clock_below_limit")
	sendWS(t, cf.masterConn, "end_match", map[string]any{"storyAt": "2000-01-05T00:00:00"})
	awaitClockError(t, mc, "clock_long_skip_unconfirmed")

	sendWS(t, cf.masterConn, "change_scene", map[string]any{"category": "battle"})
	awaitSceneChanged(t, pc, 1)
	sendWS(t, cf.masterConn, "end_match", map[string]any{"storyAt": "2000-01-01T05:00:00"})
	awaitClockError(t, mc, "match_in_battle")

	time.Sleep(100 * time.Millisecond)
	if n := pc.count(game.MsgTypeMatchEnded); n != 0 {
		t.Errorf("match_ended = %d after refusals, want 0", n)
	}
	if ws := cf.ends.writes(); len(ws) != 0 {
		t.Errorf("PersistMatchEnd calls = %d after refusals, want 0", len(ws))
	}
	if cf.session.IsEnded() {
		t.Error("a refused end_match ended the session")
	}
}

func TestEndMatchRefusedWithOpenTurn(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, _ := cf.connectCollectors(t)
	cf.enqueueAttack(t, cf.playerConn)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master was never told an action was queued")
	}
	sendWS(t, cf.masterConn, "open_next_action", map[string]any{})
	if !mc.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn_opened")
	}

	sendWS(t, cf.masterConn, "end_match", map[string]any{"storyAt": "2000-01-01T05:00:00"})
	awaitClockError(t, mc, "game_error")
	var msg string
	for _, m := range mc.snapshotMessages() {
		if m.Type == game.MsgTypeError {
			var p game.ErrorPayload
			_ = json.Unmarshal(m.Payload, &p)
			if p.Code == "game_error" {
				msg = p.Message
			}
		}
	}
	if msg != "cannot close round: current turn is still open" {
		t.Errorf("game_error = %q, want the change_scene open-turn text", msg)
	}
}

func TestEndMatchClockNotSet(t *testing.T) {
	cf := newClockFixture(t, true, 0)
	mc, _ := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "end_match", map[string]any{"storyAt": "2000-01-01T05:00:00"})
	awaitClockError(t, mc, "clock_not_set")
}

func TestEndMatchWriteFailureKeepsRoom(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, pc := cf.connectCollectors(t)
	cf.ends.setFail(true)

	sendWS(t, cf.masterConn, "end_match", map[string]any{"storyAt": "2000-01-01T05:00:00"})
	awaitClockError(t, mc, "game_error")
	time.Sleep(100 * time.Millisecond)
	if n := pc.count(game.MsgTypeMatchEnded); n != 0 {
		t.Fatalf("match_ended = %d after a failed write, want 0", n)
	}
	if cf.session.IsEnded() {
		t.Fatal("a failed write ended the session")
	}

	// A sala continua de pé: o mestre tenta de novo e encerra.
	cf.ends.setFail(false)
	sendWS(t, cf.masterConn, "end_match", map[string]any{"storyAt": "2000-01-01T05:00:00"})
	awaitMatchEnded(t, pc)
	if code, _ := awaitCloseCode(t, pc); code != 4002 {
		t.Errorf("close = %d, want 4002", code)
	}
}

func TestReconnectToFinishedMatchGets4002(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	story := time.Date(2000, 1, 1, 5, 0, 0, 0, time.UTC)
	cf.ends.finish(appmatch.MatchEndState{StoryAt: &story, GameEndAt: time.Date(2026, 11, 8, 15, 4, 5, 0, time.UTC)})

	for name, user := range map[string]uuid.UUID{"master": cf.masterUUID, "player": cf.playerUUID} {
		conn := connectWS(t, cf.server.URL, user, cf.matchUUID)
		got := readMessage(t, conn)
		var p game.MatchEndedPayload
		_ = json.Unmarshal(got.Payload, &p)
		if got.Type != game.MsgTypeMatchEnded || p.MatchID != cf.matchUUID ||
			strOf(p.StoryAt) != "2000-01-01T05:00:00" || p.GameEndAt != "2026-11-08T15:04:05Z" {
			t.Errorf("%s first message = %s %+v, want match_ended from the database", name, got.Type, p)
		}
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err := conn.ReadMessage()
		var ce *websocket.CloseError
		if !errors.As(err, &ce) || ce.Code != 4002 || ce.Text != "match ended" {
			t.Errorf("%s close = %v, want 4002 \"match ended\"", name, err)
		}
		conn.Close() //nolint:errcheck
	}
}
