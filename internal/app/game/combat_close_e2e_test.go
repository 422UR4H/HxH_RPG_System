package game_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
)

// fakeSummaryRepo é o banco do resumo do combate: devolve os rounds fechados que o teste pôs.
type fakeSummaryRepo struct {
	mu     sync.Mutex
	rounds []appmatch.SummaryRound
	asked  []uuid.UUID
}

func (f *fakeSummaryRepo) set(rounds ...appmatch.SummaryRound) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rounds = rounds
}

func (f *fakeSummaryRepo) ListSceneRoundsForSummary(_ context.Context, sceneUUID uuid.UUID) ([]appmatch.SummaryRound, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, sceneUUID)
	return append([]appmatch.SummaryRound(nil), f.rounds...), nil
}

func awaitSceneChanged(t *testing.T, c *collector, n int) game.SceneChangedPayload {
	t.Helper()
	if !awaitCount(c, game.MsgTypeSceneChanged, n, 2*time.Second) {
		t.Fatalf("scene_changed #%d never arrived", n)
	}
	msgs := c.snapshotMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Type == game.MsgTypeSceneChanged {
			var p game.SceneChangedPayload
			_ = json.Unmarshal(msgs[i].Payload, &p)
			return p
		}
	}
	return game.SceneChangedPayload{}
}

func TestCombatSummaryOnlyInBattle(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, pc := cf.connectCollectors(t)

	sendWS(t, cf.playerConn, "get_combat_summary", map[string]any{})
	awaitClockError(t, pc, "forbidden")
	sendWS(t, cf.masterConn, "get_combat_summary", map[string]any{})
	awaitClockError(t, mc, "not_battle_scene")

	sendWS(t, cf.masterConn, "change_scene", map[string]any{"category": "battle", "briefInitialDescription": "Arena"})
	battle := awaitSceneChanged(t, pc, 1)

	// Um round fechado no banco: Race, drenado, com um turno — o hiato abre nele e fecha no
	// primeiro turno do round ativo.
	before := appmatch.SummaryTurn{TurnUUID: uuid.New(), ActorUUID: cf.victimID, ActionKind: "move"}
	cf.summaries.set(appmatch.SummaryRound{
		UUID: uuid.New(), Mode: enum.Race, Counted: true, Drained: true, First: &before, Last: &before,
	})

	sendWS(t, cf.masterConn, "change_round_mode", map[string]any{"mode": "Race"})
	if !mc.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("no round_mode_changed")
	}
	cf.enqueueAttack(t, cf.playerConn)
	if !mc.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master was never told an action was queued")
	}
	sendWS(t, cf.masterConn, "open_next_action", map[string]any{})
	if !mc.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("no turn_opened")
	}

	sendWS(t, cf.masterConn, "get_combat_summary", map[string]any{})
	if !mc.await(game.MsgTypeCombatSummary, 2*time.Second) {
		t.Fatal("the master never got combat_summary")
	}
	var got game.CombatSummaryPayload
	_ = json.Unmarshal(findMessage(t, mc.snapshotMessages(), game.MsgTypeCombatSummary).Payload, &got)
	if got.SceneID != battle.SceneID || got.CountedRounds != 1 || got.CountedSeconds != 1 || !got.ActiveRoundCounts {
		t.Errorf("summary = %+v, want scene %s, 1 counted round / 1 s, active round counting", got, battle.SceneID)
	}
	if got.MinimumStoryAt != "2000-01-01T00:00:01" {
		t.Errorf("minimumStoryAt = %q, want 2000-01-01T00:00:01 (clock + the active round's second)", got.MinimumStoryAt)
	}
	if len(got.Hiatuses) != 1 {
		t.Fatalf("hiatuses = %+v, want one", got.Hiatuses)
	}
	h := got.Hiatuses[0]
	if h.Before.RoundNumber != 1 || h.Before.TurnID != before.TurnUUID || h.Before.ActorID != cf.victimID || h.Before.ActionKind != "move" {
		t.Errorf("before = %+v", h.Before)
	}
	if h.After.RoundNumber != 2 || h.After.ActorID != cf.attackerID || h.After.ActionKind != "attack" || h.After.TurnID == uuid.Nil {
		t.Errorf("after = %+v, want round 2, the attacker's attack", h.After)
	}

	time.Sleep(100 * time.Millisecond)
	if n := pc.count(game.MsgTypeCombatSummary); n != 0 {
		t.Errorf("the player got %d combat_summary, want none — it answers only the sender", n)
	}
}

func TestChangeSceneClosesCombatWithStoryAt(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	_, pc := cf.connectCollectors(t)

	sendWS(t, cf.masterConn, "change_scene", map[string]any{"category": "battle"})
	battle := awaitSceneChanged(t, pc, 1)
	// Sem pausa (o relógio estava parado): só o clock_changed scene, sem mover.
	if got := awaitClockChanged(t, pc, 1); got.Change.Kind != "scene" ||
		strOf(got.Change.From) != "2000-01-01T00:00:00" || strOf(got.Change.To) != "2000-01-01T00:00:00" {
		t.Fatalf("clock_changed = %+v, want scene 00:00:00 → 00:00:00", got.Change)
	}

	sendWS(t, cf.masterConn, "play_clock", map[string]any{})
	awaitClockChanged(t, pc, 2)
	cf.now.advance(30 * time.Second)

	sendWS(t, cf.masterConn, "change_scene", map[string]any{
		"category": "roleplay", "briefInitialDescription": "Depois",
		"briefFinalDescription": "  Os bandidos fugiram  ", "storyAt": "2000-01-01T00:10:00",
	})
	next := awaitSceneChanged(t, pc, 2)
	scene2 := awaitClockChanged(t, pc, 4)

	// Ordem: scene_changed, pause (tempo real corria), scene.
	var kinds []string
	var pause game.ClockChangePayload
	seenScene := 0
	for _, m := range pc.snapshotMessages() {
		if m.Type == game.MsgTypeSceneChanged {
			seenScene++
		}
		if seenScene == 2 && m.Type == game.MsgTypeClockChanged {
			var p game.ClockChangedPayload
			_ = json.Unmarshal(m.Payload, &p)
			kinds = append(kinds, p.Change.Kind+":"+p.Change.Reason)
			if p.Change.Kind == "pause" {
				pause = p.Change
			}
		}
	}
	if strings.Join(kinds, ",") != "pause:scene,scene:" {
		t.Fatalf("clock_changed after the 2nd scene_changed = %v, want [pause:scene scene:]", kinds)
	}
	// R8: a pausa diz o tempo real que correu.
	if pause.ElapsedSeconds != 30 {
		t.Errorf("pause elapsedSeconds = %d, want 30", pause.ElapsedSeconds)
	}
	// O from da troca é o to da pausa (sem sobrepor um segundo).
	if strOf(scene2.Change.From) != strOf(pause.To) {
		t.Errorf("scene change from = %s, want the pause's to %s", strOf(scene2.Change.From), strOf(pause.To))
	}
	// O clockRealTime é do par que estava rodando (a cena de combate), não do novo.
	evs := cf.repo.eventsOf(matchevent.KindClockRealTime)
	if len(evs) != 1 || evs[0].SceneUUID != battle.SceneID || evs[0].SceneUUID == next.SceneID {
		t.Errorf("clockRealTime events = %+v, want one in the battle scene %s", evs, battle.SceneID)
	}
	if scene2.Change.Kind != "scene" || strOf(scene2.Change.From) != "2000-01-01T00:00:30" ||
		strOf(scene2.Change.To) != "2000-01-01T00:10:00" || scene2.Change.DeltaSeconds != 570 {
		t.Errorf("scene change = %+v, want 00:00:30 → 00:10:00 (+570)", scene2.Change)
	}
	if strOf(scene2.Clock.StoryAt) != "2000-01-01T00:10:00" || scene2.Clock.Running || scene2.Clock.LimitReason != "scene" {
		t.Errorf("clock = %+v, want 00:10:00, paused, limit scene", scene2.Clock)
	}

	// O par antigo fecha com o resumo final, o fim na história e o relógio, numa escrita só.
	calls := cf.roundRepo.sceneCloseCalls()
	if len(calls) != 2 {
		t.Fatalf("CloseSceneAndRound calls = %d, want 2", len(calls))
	}
	last := calls[1]
	if f := last.end.Scene.BriefFinalDescription; f == nil || *f != "Os bandidos fugiram" {
		t.Errorf("closed scene final description = %v, want the trimmed text", f)
	}
	if at := last.end.Scene.GetStoryFinishedAt(); at == nil || !at.Equal(time.Date(2000, 1, 1, 0, 10, 0, 0, time.UTC)) {
		t.Errorf("closed scene storyFinishedAt = %v, want 00:10:00", at)
	}
	if last.clock == nil || last.clock.StoryAt == nil || !last.clock.StoryAt.Equal(time.Date(2000, 1, 1, 0, 10, 0, 0, time.UTC)) {
		t.Errorf("clock written with the close = %+v, want story_at 00:10:00", last.clock)
	}
	// A cena nova nasce com o início na história no mesmo instante.
	var born *scene.Scene
	for _, p := range cf.roundRepo.handedPointers() {
		if sc, ok := p.(*scene.Scene); ok && sc.GetID() == next.SceneID {
			born = sc
		}
	}
	if born == nil || born.GetStoryStartedAt() == nil || !born.GetStoryStartedAt().Equal(time.Date(2000, 1, 1, 0, 10, 0, 0, time.UTC)) {
		t.Errorf("new scene written = %+v, want storyStartedAt 00:10:00", born)
	}
}

func TestChangeSceneBelowMinimumRefused(t *testing.T) {
	cf := newClockFixture(t, false, 0)
	mc, pc := cf.connectCollectors(t)
	sendWS(t, cf.masterConn, "adjust_clock", map[string]any{"deltaSeconds": 60})
	awaitClockChanged(t, pc, 1)

	sendWS(t, cf.masterConn, "change_scene", map[string]any{"category": "battle", "storyAt": "2000-01-01T00:00:30"})
	awaitClockError(t, mc, "clock_below_limit")

	sendWS(t, cf.masterConn, "change_scene", map[string]any{"category": "battle", "storyAt": "2000-01-01T00:01:00Z"})
	sendWS(t, cf.masterConn, "change_scene", map[string]any{
		"category": "battle", "briefFinalDescription": strings.Repeat("x", 256),
	})
	deadline := time.Now().Add(2 * time.Second)
	var msgs []string
	for time.Now().Before(deadline) {
		msgs = msgs[:0]
		for _, m := range mc.snapshotMessages() {
			if m.Type == game.MsgTypeError {
				var p game.ErrorPayload
				_ = json.Unmarshal(m.Payload, &p)
				if p.Code == "invalid_payload" {
					msgs = append(msgs, p.Message)
				}
			}
		}
		if len(msgs) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(msgs) != 2 || msgs[1] != "briefFinalDescription is too long" {
		t.Fatalf("invalid_payload errors = %q, want the bad storyAt and the too-long final description", msgs)
	}

	time.Sleep(100 * time.Millisecond)
	if n := pc.count(game.MsgTypeSceneChanged); n != 0 {
		t.Errorf("scene_changed = %d after refusals, want 0", n)
	}
	if n := pc.count(game.MsgTypeClockChanged); n != 1 {
		t.Errorf("clock_changed = %d, want only the adjust — a refusal moves nothing", n)
	}
	if calls := cf.roundRepo.sceneCloseCalls(); len(calls) != 0 {
		t.Errorf("CloseSceneAndRound calls = %d after refusals, want 0", len(calls))
	}
}
