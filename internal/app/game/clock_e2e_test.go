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
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// ─── fakes do relógio ───────────────────────────────────────────────────────

// fakeNow é o tempo real controlado pelo teste; a sala o repassa à sessão.
type fakeNow struct {
	mu sync.Mutex
	t  time.Time
}

func (f *fakeNow) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

func (f *fakeNow) advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

// fakeClockRepo guarda os checkpoints e os eventos, aplicando a fusão como o pgclock.
type fakeClockRepo struct {
	mu     sync.Mutex
	saves  []clock.Snapshot
	events []matchevent.Event
	fail   bool
}

func (r *fakeClockRepo) setFail(v bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fail = v
}

func (r *fakeClockRepo) Load(context.Context, uuid.UUID) (*clock.Snapshot, error) { return nil, nil }

func (r *fakeClockRepo) Save(_ context.Context, _ uuid.UUID, snap clock.Snapshot, ev *appmatch.ClockEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail {
		return errors.New("save failed")
	}
	r.saves = append(r.saves, snap)
	if ev == nil {
		return nil
	}
	if ev.MergeInto != uuid.Nil {
		for i := range r.events {
			if r.events[i].UUID == ev.MergeInto {
				r.events[i].Payload = ev.Event.Payload
			}
		}
		return nil
	}
	r.events = append(r.events, ev.Event)
	return nil
}

func (r *fakeClockRepo) eventsOf(kind matchevent.Kind) []matchevent.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []matchevent.Event
	for _, e := range r.events {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

func (r *fakeClockRepo) runningSaves() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, s := range r.saves {
		if s.Running {
			n++
		}
	}
	return n
}

type fakeMatchGetter struct{ storyStart time.Time }

func (g fakeMatchGetter) GetMatch(context.Context, uuid.UUID) (*match.Match, error) {
	return &match.Match{StoryStartAt: g.storyStart}, nil
}

// ─── fixture ────────────────────────────────────────────────────────────────

type clockFixture struct {
	*combatFixture
	now   *fakeNow
	repo  *fakeClockRepo
	clock *clock.Clock
	// summaries é o banco do get_combat_summary (combat_close_e2e_test.go).
	summaries *fakeSummaryRepo
	// ends é o banco do fim da partida (end_match_e2e_test.go).
	ends *fakeMatchEndRepo

	masterConn, playerConn *websocket.Conn
}

// newClockFixture monta a mesa de combate com o relógio ligado. unset deixa o relógio sem
// horário (o diálogo obrigatório do mestre); senão ele nasce em 2000-01-01T00:00:00, pausado.
func newClockFixture(t *testing.T, unset bool, checkpointEvery time.Duration) *clockFixture {
	t.Helper()
	cf := &clockFixture{
		now:       &fakeNow{t: time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)},
		repo:      &fakeClockRepo{},
		summaries: &fakeSummaryRepo{},
		ends:      &fakeMatchEndRepo{},
	}
	cf.combatFixture = newCombatFixture(t, withRoomDeps(func(d *game.RoomDeps) {
		d.ClockUC = appmatch.NewClockCommandsUC(fakeMatchGetter{storyStart: time.Date(1999, 4, 1, 0, 0, 0, 0, time.UTC)})
		d.ClockRepo = cf.repo
		d.Now = cf.now.Now
		d.ClockCheckpointEvery = checkpointEvery
		d.CombatSummaryRepo = cf.summaries
		d.MatchEndRepo = cf.ends
	}))
	if unset {
		cf.clock = clock.NewUnset()
	} else {
		cf.clock = clock.NewAt(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
	}
	cf.session.SetClock(cf.clock)
	return cf
}

func lastClockChanged(t *testing.T, c *collector) game.ClockChangedPayload {
	t.Helper()
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type == game.MsgTypeClockChanged {
			var p game.ClockChangedPayload
			if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
				t.Fatalf("unmarshal clock_changed: %v", err)
			}
			return p
		}
	}
	t.Fatal("no clock_changed received")
	return game.ClockChangedPayload{}
}

func awaitClockChanged(t *testing.T, c *collector, n int) game.ClockChangedPayload {
	t.Helper()
	if !awaitCount(c, game.MsgTypeClockChanged, n, 2*time.Second) {
		t.Fatalf("clock_changed #%d never arrived", n)
	}
	return lastClockChanged(t, c)
}

// awaitClockError espera um error com esse código.
func awaitClockError(t *testing.T, c *collector, code string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, m := range c.snapshotMessages() {
			if m.Type != game.MsgTypeError {
				continue
			}
			var p game.ErrorPayload
			_ = json.Unmarshal(m.Payload, &p)
			if p.Code == code {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	var got []string
	for _, m := range c.snapshotMessages() {
		got = append(got, string(m.Type)+" "+string(m.Payload))
	}
	t.Fatalf("error %q never arrived; got %v", code, got)
}

func strOf(p *string) string {
	if p == nil {
		return "<nil>"
	}
	return *p
}

func payloadMap(t *testing.T, e matchevent.Event) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(e.Payload, &m); err != nil {
		t.Fatalf("unmarshal event payload: %v", err)
	}
	return m
}

func (cf *clockFixture) connectCollectors(t *testing.T) (mc, pc *collector) {
	t.Helper()
	master, player := cf.connect(t)
	t.Cleanup(func() { master.Close(); player.Close() }) //nolint:errcheck
	mc, pc = newCollector(master), newCollector(player)
	cf.masterConn, cf.playerConn = master, player
	return mc, pc
}

// ─── testes ─────────────────────────────────────────────────────────────────

func TestClockStartSetsTimeOnce(t *testing.T) {
	cf := newClockFixture(t, true, 0)
	mc, pc := cf.connectCollectors(t)

	sendWS(t, cf.masterConn, "set_clock_start", map[string]any{"time": "08:30"})
	got := awaitClockChanged(t, pc, 1)
	if got.Change.Kind != "start" || strOf(got.Clock.StoryAt) != "1999-04-01T08:30:00" || got.Clock.Running {
		t.Errorf("clock_changed = %+v / %+v, want start at 1999-04-01T08:30:00, paused", got.Change, got.Clock)
	}
	if evs := cf.repo.eventsOf(matchevent.KindClockStarted); len(evs) != 1 || payloadMap(t, evs[0])["at"] != "1999-04-01T08:30:00" {
		t.Errorf("clockStarted events = %+v", evs)
	}

	sendWS(t, cf.masterConn, "set_clock_start", map[string]any{"time": "09:00:00"})
	awaitClockError(t, mc, "clock_already_set")
}

func TestAdjustClockBroadcastsToTable(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	_, pc := cf.connectCollectors(t)

	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 60})
	got := awaitClockChanged(t, pc, 1)
	if got.Change.Kind != "adjust" || strOf(got.Change.From) != "2000-01-01T00:00:00" ||
		strOf(got.Change.To) != "2000-01-01T00:01:00" || got.Change.DeltaSeconds != 60 {
		t.Errorf("change = %+v, want adjust 00:00:00 → 00:01:00 (+60)", got.Change)
	}
	if strOf(got.Clock.StoryAt) != "2000-01-01T00:01:00" {
		t.Errorf("clock.storyAt = %s", strOf(got.Clock.StoryAt))
	}
}

func TestAdjustBelowLimitRefused(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, _ := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": -60})
	awaitClockError(t, mc, "clock_below_limit")
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 0})
	awaitClockError(t, mc, "invalid_payload")
}

func TestLongSkipRequiresConfirm(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, pc := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 172801})
	awaitClockError(t, mc, "clock_long_skip_unconfirmed")

	sendWS(t, cf.masterConn, "set_clock", map[string]any{"storyAt": "2000-01-03T00:00:01", "confirmLongSkip": true})
	got := awaitClockChanged(t, pc, 1)
	if got.Change.Kind != "set" || strOf(got.Clock.StoryAt) != "2000-01-03T00:00:01" {
		t.Errorf("clock_changed = %+v / %+v", got.Change, got.Clock)
	}
}

func TestPlayRefusedInRace(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, _ := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "change_round_mode", map[string]any{"mode": "Race"})
	if !mc.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("no round_mode_changed")
	}
	sendWS(t, cf.masterConn, "play_clock", map[string]any{})
	awaitClockError(t, mc, "clock_play_in_race")
}

func TestModeChangeToRacePausesClock(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	_, pc := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "play_clock", map[string]any{})
	if got := awaitClockChanged(t, pc, 1); got.Change.Kind != "play" || !got.Clock.Running {
		t.Fatalf("clock_changed = %+v / %+v, want play, running", got.Change, got.Clock)
	}
	cf.now.advance(30 * time.Second)

	sendWS(t, cf.masterConn, "change_round_mode", map[string]any{"mode": "Race"})
	got := awaitClockChanged(t, pc, 2)
	if got.Change.Kind != "pause" || got.Change.Reason != "roundMode" || got.Clock.Running ||
		strOf(got.Clock.StoryAt) != "2000-01-01T00:00:30" {
		t.Errorf("clock_changed = %+v / %+v, want pause roundMode at 00:00:30", got.Change, got.Clock)
	}
	if got.Clock.LimitReason != "roundMode" || strOf(got.Clock.LimitAt) != "2000-01-01T00:00:30" {
		t.Errorf("limit = %s (%s), want 00:00:30 (roundMode)", strOf(got.Clock.LimitAt), got.Clock.LimitReason)
	}
	evs := cf.repo.eventsOf(matchevent.KindClockRealTime)
	if len(evs) != 1 {
		t.Fatalf("clockRealTime events = %d, want 1", len(evs))
	}
	if m := payloadMap(t, evs[0]); m["elapsedSeconds"] != float64(30) || m["reason"] != "roundMode" ||
		m["from"] != "2000-01-01T00:00:00" || m["to"] != "2000-01-01T00:00:30" {
		t.Errorf("clockRealTime payload = %v", m)
	}
}

func TestMasterLeavingPausesClock(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	_, pc := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "play_clock", map[string]any{})
	awaitClockChanged(t, pc, 1)
	cf.now.advance(5 * time.Second)

	cf.masterConn.Close() //nolint:errcheck
	got := awaitClockChanged(t, pc, 2)
	if got.Change.Kind != "pause" || got.Change.Reason != "masterLeft" || got.Clock.Running {
		t.Errorf("clock_changed = %+v / %+v, want pause masterLeft", got.Change, got.Clock)
	}
	if evs := cf.repo.eventsOf(matchevent.KindClockRealTime); len(evs) != 1 || payloadMap(t, evs[0])["reason"] != "masterLeft" {
		t.Errorf("clockRealTime events = %+v", evs)
	}
}

func TestReplacedMasterTabDoesNotPause(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	_, pc := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "play_clock", map[string]any{})
	awaitClockChanged(t, pc, 1)

	second := connectWS(t, cf.server.URL, cf.masterUUID, cf.matchUUID)
	defer second.Close() //nolint:errcheck
	sc := newCollector(second)
	if !sc.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("the second master tab never got match_full_state")
	}
	time.Sleep(150 * time.Millisecond) // deixa o unregister da aba substituída chegar
	if n := pc.count(game.MsgTypeClockChanged); n != 1 {
		t.Fatalf("player got %d clock_changed, want only the play (the replacement paused)", n)
	}
	// Ainda rodando: a pausa manual da aba nova acontece de verdade.
	sendWS(t, second, "pause_clock", map[string]any{})
	if got := awaitClockChanged(t, pc, 2); got.Change.Reason != "manual" {
		t.Errorf("pause reason = %q, want manual", got.Change.Reason)
	}
}

func TestOpenNextActionRefusedWhenClockUnset(t *testing.T) {
	cf := newClockFixture(t, true, 0)
	mc, _ := cf.connectCollectors(t)
	cf.enqueueAttack(t, cf.playerConn)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master was never told an action was queued")
	}
	sendWS(t, cf.masterConn, "open_next_action", map[string]any{})
	awaitClockError(t, mc, "clock_not_set")
}

func TestMatchFullStateCarriesClock(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		cf := newClockFixture(t, false, 0)
		mc, _ := cf.connectCollectors(t)
		if !mc.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatal("no match_full_state")
		}
		var full game.MatchFullStatePayload
		_ = json.Unmarshal(findMessage(t, mc.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &full)
		if full.Clock == nil || strOf(full.Clock.StoryAt) != "2000-01-01T00:00:00" || full.Clock.Running ||
			full.Clock.LimitReason != "start" {
			t.Errorf("match_full_state.clock = %+v", full.Clock)
		}
	})
	t.Run("unset", func(t *testing.T) {
		cf := newClockFixture(t, true, 0)
		mc, _ := cf.connectCollectors(t)
		if !mc.await(game.MsgTypeMatchFullState, 2*time.Second) {
			t.Fatal("no match_full_state")
		}
		raw := findMessage(t, mc.snapshotMessages(), game.MsgTypeMatchFullState).Payload
		var probe struct {
			Clock map[string]json.RawMessage `json:"clock"`
		}
		_ = json.Unmarshal(raw, &probe)
		if probe.Clock == nil || string(probe.Clock["storyAt"]) != "null" {
			t.Errorf("match_full_state.clock = %s, want storyAt null", raw)
		}
	})
}

func TestConsecutiveAdjustsMergeInHistory(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	_, pc := cf.connectCollectors(t)

	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 60})
	awaitClockChanged(t, pc, 1)
	cf.now.advance(3 * time.Second)
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 30})
	awaitClockChanged(t, pc, 2)

	evs := cf.repo.eventsOf(matchevent.KindClockAdjusted)
	if len(evs) != 1 {
		t.Fatalf("clockAdjusted events = %d, want 1 (merged)", len(evs))
	}
	if m := payloadMap(t, evs[0]); m["from"] != "2000-01-01T00:00:00" || m["to"] != "2000-01-01T00:01:30" ||
		m["deltaSeconds"] != float64(90) {
		t.Errorf("merged payload = %v, want 00:00:00 → 00:01:30 (+90)", m)
	}

	// Fora da janela: linha nova.
	cf.now.advance(11 * time.Second)
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 10})
	awaitClockChanged(t, pc, 3)
	if n := len(cf.repo.eventsOf(matchevent.KindClockAdjusted)); n != 2 {
		t.Errorf("clockAdjusted events = %d, want 2 after the window closed", n)
	}
}

// O tempo real que corre entre dois ajustes fundidos não entra no deltaSeconds: é a soma do
// que o mestre pediu.
func TestConsecutiveAdjustsMergeWhileRunning(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	_, pc := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "play_clock", map[string]any{})
	awaitClockChanged(t, pc, 1)

	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 60})
	awaitClockChanged(t, pc, 2)
	cf.now.advance(3 * time.Second)
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 30})
	awaitClockChanged(t, pc, 3)

	evs := cf.repo.eventsOf(matchevent.KindClockAdjusted)
	if len(evs) != 1 {
		t.Fatalf("clockAdjusted events = %d, want 1 (merged)", len(evs))
	}
	if m := payloadMap(t, evs[0]); m["from"] != "2000-01-01T00:00:00" || m["to"] != "2000-01-01T00:01:33" ||
		m["deltaSeconds"] != float64(90) {
		t.Errorf("merged payload = %v, want 00:00:00 → 00:01:33 with deltaSeconds 90", m)
	}
}

// Sem gravação do evento, a janela não abre: o próximo ajuste é uma linha nova.
func TestAdjustNotRecordedDoesNotOpenMergeWindow(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	_, pc := cf.connectCollectors(t)
	cf.repo.setFail(true)
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 60})
	awaitClockChanged(t, pc, 1)
	cf.repo.setFail(false)
	cf.now.advance(time.Second)
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 30})
	awaitClockChanged(t, pc, 2)

	evs := cf.repo.eventsOf(matchevent.KindClockAdjusted)
	if len(evs) != 1 {
		t.Fatalf("clockAdjusted events = %d, want 1", len(evs))
	}
	if m := payloadMap(t, evs[0]); m["from"] != "2000-01-01T00:01:00" || m["deltaSeconds"] != float64(30) {
		t.Errorf("payload = %v, want a fresh row from 00:01:00 (+30)", m)
	}
}

func TestClockChangedCarriesIncreasingSeq(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, pc := cf.connectCollectors(t)
	if !mc.await(game.MsgTypeMatchFullState, 2*time.Second) {
		t.Fatal("no match_full_state")
	}
	var full game.MatchFullStatePayload
	_ = json.Unmarshal(findMessage(t, mc.snapshotMessages(), game.MsgTypeMatchFullState).Payload, &full)
	if full.Clock == nil || full.Clock.Seq == 0 {
		t.Fatalf("match_full_state.clock.seq missing: %+v", full.Clock)
	}

	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 60})
	first := awaitClockChanged(t, pc, 1)
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 60})
	second := awaitClockChanged(t, pc, 2)
	if full.Clock.Seq >= first.Clock.Seq || first.Clock.Seq >= second.Clock.Seq {
		t.Errorf("seq = %d (full) → %d → %d, want strictly increasing", full.Clock.Seq, first.Clock.Seq, second.Clock.Seq)
	}
}

func TestNonMasterClockCommandForbidden(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	_, pc := cf.connectCollectors(t)
	sendWS(t, cf.playerConn, "adjust_clock", map[string]any{"deltaSeconds": 60})
	awaitClockError(t, pc, "forbidden")
}

func TestRunningClockCheckpoints(t *testing.T) {
	cf := newClockFixture(t, false, 20*time.Millisecond)
	_, pc := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "play_clock", map[string]any{})
	awaitClockChanged(t, pc, 1)
	before := cf.repo.runningSaves()
	time.Sleep(150 * time.Millisecond)
	if after := cf.repo.runningSaves(); after <= before {
		t.Errorf("running checkpoints = %d → %d, want more while running", before, after)
	}
	if n := pc.count(game.MsgTypeClockChanged); n != 1 {
		t.Errorf("player got %d clock_changed, want 1 (a checkpoint announces nothing)", n)
	}
}
