package matchsession_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

// storyStart é o horário inicial que NewMatchSession usa (relógio pausado).
var storyStart = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

func raceSingle(t *testing.T) *matchsession.MatchSession {
	t.Helper()
	u := uuid.New()
	s, chars := sessionWithParticipants(u)
	s.GetActiveRound().SetMode(enum.Race)
	if err := s.EnqueueAction(u, makeActionWithSpeed(chars[0], 5)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	return s
}

func TestOpenTurnStampsStoryTimeAndRaisesLimit(t *testing.T) {
	s := raceSingle(t)
	real := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	s.SetNowFunc(func() time.Time { return real })
	if err := s.Clock().Play(real); err != nil {
		t.Fatal(err)
	}
	real = real.Add(90 * time.Second)

	tn := mustOpenNext(t, s).Opened
	want := storyStart.Add(90 * time.Second)
	if got := tn.GetStoryAt(); got == nil || !got.Equal(want) {
		t.Fatalf("turn story_at = %v, want %v", got, want)
	}
	if at, reason := s.Clock().Limit(); !at.Equal(want) || reason != clock.LimitTurn {
		t.Errorf("limit = %v/%v, want %v/turn", at, reason, want)
	}
}

func TestRaceRoundCountsOneSecond(t *testing.T) {
	s := raceSingle(t)
	closeRoundWhenNoActionCanPay(t, s)
	if got := s.Clock().Now(time.Now()); !got.Equal(storyStart.Add(time.Second)) {
		t.Errorf("Now = %v, want start+1s", got)
	}
}

func TestRoundDrainedFlag(t *testing.T) {
	t.Run("queue empty at close is drained", func(t *testing.T) {
		s := raceSingle(t)
		old := s.GetActiveRound()
		closeRoundWhenNoActionCanPay(t, s)
		if !old.IsDrained() || !old.IsCounted() {
			t.Errorf("drained=%v counted=%v, want both true", old.IsDrained(), old.IsCounted())
		}
		if s.GetActiveRound().IsCounted() {
			t.Error("the new round must not be counted")
		}
	})
}

func TestRoundWithActionsLeftIsNotDrained(t *testing.T) {
	u1, u2 := uuid.New(), uuid.New()
	s, c := sessionWithParticipants(u1, u2)
	s.GetActiveRound().SetMode(enum.Race)
	// com estas velocidades uma das ações não alcança o preço e fica na fila
	for i, v := range []int{50, 12, 12, 12} {
		who, id := u1, c[0]
		if i%2 == 1 {
			who, id = u2, c[1]
		}
		if err := s.EnqueueAction(who, makeActionWithSpeed(id, v)); err != nil {
			t.Fatal(err)
		}
	}
	old := s.GetActiveRound()
	closeRoundWhenNoActionCanPay(t, s)
	if len(s.PendingActions()) == 0 {
		t.Fatal("fixture: expected an action left in the queue")
	}
	if old.IsDrained() {
		t.Error("round closed with actions queued must not be drained")
	}
}

func TestFreeRoundDoesNotCount(t *testing.T) {
	u := uuid.New()
	s, chars := sessionWithParticipants(u)
	if err := s.EnqueueAction(u, makeActionWithSpeed(chars[0], 5)); err != nil {
		t.Fatal(err)
	}
	mustOpenNext(t, s)
	if _, err := s.CloseOpenTurn(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CloseRound(); err != nil {
		t.Fatal(err)
	}
	if got := s.Clock().Now(time.Now()); !got.Equal(storyStart) {
		t.Errorf("Now = %v, want unchanged %v", got, storyStart)
	}
}

func TestModeSwitchMidRoundStillCounts(t *testing.T) {
	s := raceSingle(t)
	mustOpenNext(t, s)
	if _, err := s.SetRoundMode(enum.Free); err != nil {
		t.Fatalf("SetRoundMode: %v", err)
	}
	if _, err := s.CloseOpenTurn(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CloseRound(); err != nil {
		t.Fatal(err)
	}
	if got := s.Clock().Now(time.Now()); !got.Equal(storyStart.Add(time.Second)) {
		t.Errorf("Now = %v, want start+1s", got)
	}
}

func TestOpenRefusedWhenClockUnset(t *testing.T) {
	s := raceSingle(t)
	s.SetClock(clock.NewUnset())
	if _, err := s.OpenNextAction(); !errors.Is(err, clock.ErrNotSet) {
		t.Errorf("OpenNextAction err = %v, want ErrNotSet", err)
	}
	if len(s.PendingActions()) != 1 {
		t.Error("the queue must be untouched")
	}
	id := s.PendingActions()[0].GetID()
	if _, err := s.PullAction(id); !errors.Is(err, clock.ErrNotSet) {
		t.Errorf("PullAction err = %v, want ErrNotSet", err)
	}
}

// Com o relógio rodando, fechar o turno sobe o limite até o horário do fechamento.
func TestTurnCloseRaisesLimitWhileRunning(t *testing.T) {
	u := uuid.New()
	s, chars := sessionWithParticipants(u)
	if err := s.EnqueueAction(u, makeActionWithSpeed(chars[0], 5)); err != nil {
		t.Fatal(err)
	}
	real := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	s.SetNowFunc(func() time.Time { return real })
	if err := s.Clock().Play(real); err != nil {
		t.Fatal(err)
	}
	mustOpenNext(t, s)
	real = real.Add(30 * time.Second)
	if _, err := s.CloseOpenTurn(); err != nil {
		t.Fatal(err)
	}
	want := storyStart.Add(30 * time.Second)
	if at, reason := s.Clock().Limit(); !at.Equal(want) || reason != clock.LimitTurn {
		t.Errorf("limit = %v/%v, want %v/turn", at, reason, want)
	}
}

// --- Tarefa 4: troca de cena, de regime e fim da partida movem o relógio ---

var realT0 = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

// runningSession devolve uma sessão com o relógio rodando desde realT0 e a fonte de tempo
// controlada por `*real`.
func runningSession(t *testing.T, real *time.Time) *matchsession.MatchSession {
	t.Helper()
	*real = realT0
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	s.SetNowFunc(func() time.Time { return *real })
	if err := s.Clock().Play(*real); err != nil {
		t.Fatal(err)
	}
	return s
}

type clockState struct {
	sceneID, roundID uuid.UUID
	now, limit       time.Time
	running          bool
	queue            int
}

func stateOf(s *matchsession.MatchSession, real time.Time) clockState {
	lim, _ := s.Clock().Limit()
	return clockState{
		sceneID: s.GetActiveScene().GetID(), roundID: s.GetActiveRound().GetID(),
		now: s.Clock().Now(real), limit: lim, running: s.Clock().Running(),
		queue: len(s.PendingActions()),
	}
}

func TestChangeSceneRejectsBelowMinimumWithoutMutating(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	s.GetActiveRound().MarkCounted()
	real = real.Add(10 * time.Second)
	before := stateOf(s, real)

	at := storyStart.Add(5 * time.Second) // abaixo de agora + 1 s
	_, _, _, err := s.ChangeScene(enum.Roleplay, "x", matchsession.SceneClose{StoryAt: &at})
	if !errors.Is(err, clock.ErrBelowLimit) {
		t.Fatalf("err = %v, want ErrBelowLimit", err)
	}
	if after := stateOf(s, real); after != before {
		t.Errorf("state changed on refusal: before %+v after %+v", before, after)
	}
}

func TestChangeSceneAddsActiveRoundSecond(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	s.GetActiveRound().MarkCounted()
	if _, _, _, err := s.ChangeScene(enum.Roleplay, "x", matchsession.SceneClose{}); err != nil {
		t.Fatal(err)
	}
	if got := s.Clock().Now(real); !got.Equal(storyStart.Add(time.Second)) {
		t.Errorf("Now = %v, want start+1s", got)
	}
	if s.Clock().Running() {
		t.Error("the clock must be paused by the scene change")
	}
	if _, reason := s.Clock().Limit(); reason != clock.LimitScene {
		t.Errorf("limit reason = %v, want scene", reason)
	}
}

func TestChangeScenePausesRealTime(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	real = real.Add(30 * time.Second)
	_, _, p, err := s.ChangeScene(enum.Roleplay, "x", matchsession.SceneClose{})
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || p.Reason != clock.PauseScene || p.Elapsed != 30*time.Second {
		t.Fatalf("pause = %+v, want scene/30s", p)
	}
	real = real.Add(time.Hour)
	if got := s.Clock().Now(real); !got.Equal(storyStart.Add(30 * time.Second)) {
		t.Errorf("Now = %v, want frozen at start+30s", got)
	}
}

func TestChangeSceneMovesClockToStoryAtAndStampsScenes(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	at := storyStart.Add(2 * time.Hour)
	desc := "  fim  "
	old, _, _, err := s.ChangeScene(enum.Roleplay, "x", matchsession.SceneClose{StoryAt: &at, BriefFinalDescription: &desc})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Clock().Now(real); !got.Equal(at) {
		t.Errorf("Now = %v, want %v", got, at)
	}
	if old.BriefFinalDescription == nil || *old.BriefFinalDescription != "fim" {
		t.Errorf("final description = %v, want trimmed", old.BriefFinalDescription)
	}
	if f := old.GetStoryFinishedAt(); f == nil || !f.Equal(at) {
		t.Errorf("old storyFinishedAt = %v", f)
	}
	if st := s.GetActiveScene().GetStoryStartedAt(); st == nil || !st.Equal(at) {
		t.Errorf("new storyStartedAt = %v", st)
	}
}

func TestChangeSceneBriefFinal(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	long := strings.Repeat("é", 256)
	before := stateOf(s, real)
	if _, _, _, err := s.ChangeScene(enum.Roleplay, "x", matchsession.SceneClose{BriefFinalDescription: &long}); !errors.Is(err, matchsession.ErrBriefFinalTooLong) {
		t.Fatalf("err = %v, want ErrBriefFinalTooLong", err)
	}
	if stateOf(s, real) != before {
		t.Error("state changed on refusal")
	}
	blank := "   "
	old, _, _, err := s.ChangeScene(enum.Roleplay, "x", matchsession.SceneClose{BriefFinalDescription: &blank})
	if err != nil {
		t.Fatal(err)
	}
	if old.BriefFinalDescription != nil {
		t.Error("blank description must be nil")
	}
}

func TestChangeSceneLongSkipNeedsConfirm(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	s.GetActiveRound().MarkCounted() // a baseline não inclui o segundo automático
	at := storyStart.Add(48*time.Hour + time.Second)
	before := stateOf(s, real)
	if _, _, _, err := s.ChangeScene(enum.Roleplay, "x", matchsession.SceneClose{StoryAt: &at}); !errors.Is(err, clock.ErrLongSkipUnconfirmed) {
		t.Fatalf("err = %v, want ErrLongSkipUnconfirmed", err)
	}
	if stateOf(s, real) != before {
		t.Error("state changed on refusal")
	}
	if _, _, _, err := s.ChangeScene(enum.Roleplay, "x", matchsession.SceneClose{StoryAt: &at, ConfirmLongSkip: true}); err != nil {
		t.Fatalf("confirmed skip: %v", err)
	}
	if got := s.Clock().Now(real); !got.Equal(at) {
		t.Errorf("Now = %v, want %v", got, at)
	}
}

func TestChangeSceneRefusedWhenClockUnset(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	s.SetClock(clock.NewUnset())
	id := s.GetActiveScene().GetID()
	if _, _, _, err := s.ChangeScene(enum.Roleplay, "x", matchsession.SceneClose{}); !errors.Is(err, clock.ErrNotSet) {
		t.Fatalf("err = %v, want ErrNotSet", err)
	}
	if s.GetActiveScene().GetID() != id {
		t.Error("scene changed on refusal")
	}
}

func TestSetRoundModePausesAndRaisesLimit(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	real = real.Add(20 * time.Second)
	p, err := s.SetRoundMode(enum.Race)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil || p.Reason != clock.PauseRoundMode {
		t.Fatalf("pause = %+v, want roundMode", p)
	}
	if at, reason := s.Clock().Limit(); reason != clock.LimitRoundMode || !at.Equal(storyStart.Add(20*time.Second)) {
		t.Errorf("limit = %v/%v", at, reason)
	}
	if s.GetActiveRound().GetMode() != enum.Race {
		t.Error("mode not applied")
	}
}

func TestSetRoundModeSameModeNoPause(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	real = real.Add(20 * time.Second)
	p, err := s.SetRoundMode(enum.Free)
	if err != nil || p != nil {
		t.Fatalf("pause=%v err=%v, want nil/nil", p, err)
	}
	if !s.Clock().Running() {
		t.Error("same mode must not pause")
	}
	if _, reason := s.Clock().Limit(); reason != clock.LimitStart {
		t.Errorf("limit reason = %v, want untouched", reason)
	}
}

func TestSetRoundModeRefusedWhenClockUnset(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	s.SetClock(clock.NewUnset())
	if _, err := s.SetRoundMode(enum.Race); !errors.Is(err, clock.ErrNotSet) {
		t.Fatalf("err = %v, want ErrNotSet", err)
	}
	if s.GetActiveRound().GetMode() != enum.Free {
		t.Error("mode changed on refusal")
	}
}

func TestEndMatchRefusedInBattle(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	if _, _, _, err := s.ChangeScene(enum.Battle, "b", matchsession.SceneClose{}); err != nil {
		t.Fatal(err)
	}
	before := stateOf(s, real)
	at := storyStart.Add(time.Hour)
	if _, err := s.EndMatchAt(at, false, real); !errors.Is(err, matchsession.ErrMatchInBattle) {
		t.Fatalf("err = %v, want ErrMatchInBattle", err)
	}
	if _, err := s.PlanEndMatch(at, false, real); !errors.Is(err, matchsession.ErrMatchInBattle) {
		t.Fatalf("check err = %v", err)
	}
	if stateOf(s, real) != before {
		t.Error("state changed on refusal")
	}
}

func TestEndMatchRefusedWithOpenTurn(t *testing.T) {
	u := uuid.New()
	s, chars := sessionWithParticipants(u)
	if err := s.EnqueueAction(u, makeActionWithSpeed(chars[0], 5)); err != nil {
		t.Fatal(err)
	}
	mustOpenNext(t, s)
	if _, err := s.EndMatchAt(storyStart.Add(time.Hour), false, time.Now()); !errors.Is(err, matchsession.ErrRoundHasOpenTurn) {
		t.Fatalf("err = %v, want ErrRoundHasOpenTurn", err)
	}
}

func TestEndMatchClosesAndFreezes(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	s.GetActiveRound().MarkCounted()
	real = real.Add(10 * time.Second)

	min := s.SceneCloseMinimum()
	if want := storyStart.Add(11 * time.Second); !min.Equal(want) {
		t.Fatalf("SceneCloseMinimum = %v, want %v", min, want)
	}
	if _, err := s.PlanEndMatch(min.Add(-time.Second), false, real); !errors.Is(err, clock.ErrBelowLimit) {
		t.Fatalf("below minimum: %v", err)
	}
	if _, err := s.PlanEndMatch(min, false, real); err != nil {
		t.Fatalf("check at minimum: %v", err)
	}

	at := storyStart.Add(3 * time.Hour)
	end, err := s.EndMatchAt(at, false, real)
	if err != nil {
		t.Fatal(err)
	}
	if !end.StoryAt.Equal(at) || end.Scene.GetFinishedAt() == nil || end.Round.GetFinishedAt() == nil {
		t.Errorf("end = %+v", end)
	}
	if end.Pause == nil || end.Pause.Reason != clock.PauseEnd {
		t.Errorf("pause = %+v, want end", end.Pause)
	}
	if f := end.Scene.GetStoryFinishedAt(); f == nil || !f.Equal(at) {
		t.Errorf("scene storyFinishedAt = %v", f)
	}
	real = real.Add(time.Hour)
	if s.Clock().Running() || !s.Clock().Now(real).Equal(at) {
		t.Errorf("clock must be frozen at %v, got %v running=%v", at, s.Clock().Now(real), s.Clock().Running())
	}
	if len(s.PendingActions()) != 0 {
		t.Error("queue must be discarded")
	}
}

func TestEndMatchSceneCloseMinimum(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	if got := s.SceneCloseMinimum(); !got.Equal(storyStart) {
		t.Errorf("minimum = %v, want start", got)
	}
	s.GetActiveRound().MarkCounted()
	if got := s.SceneCloseMinimum(); !got.Equal(storyStart.Add(time.Second)) {
		t.Errorf("minimum = %v, want start+1s", got)
	}
}

func TestSecondEndMatchRefused(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	s.GetActiveRound().MarkCounted()
	at := storyStart.Add(time.Hour)
	if _, err := s.EndMatchAt(at, false, real); err != nil {
		t.Fatal(err)
	}
	if !s.IsEnded() {
		t.Error("IsEnded must be true")
	}
	if _, err := s.EndMatchAt(at.Add(time.Hour), false, real); !errors.Is(err, matchsession.ErrMatchEnded) {
		t.Fatalf("err = %v, want ErrMatchEnded", err)
	}
	if _, err := s.PlanEndMatch(at.Add(time.Hour), false, real); !errors.Is(err, matchsession.ErrMatchEnded) {
		t.Fatalf("check err = %v", err)
	}
	if got := s.Clock().Now(real); !got.Equal(at) {
		t.Errorf("Now = %v, want unchanged %v", got, at)
	}
}

func TestEndMatchAtMinimumAddsRoundSecondOnce(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	s.GetActiveRound().MarkCounted()
	real = real.Add(10 * time.Second)
	min := s.SceneCloseMinimum()
	if _, err := s.EndMatchAt(min, false, real); err != nil {
		t.Fatal(err)
	}
	if got := s.Clock().Now(real); !got.Equal(min) {
		t.Errorf("Now = %v, want %v", got, min)
	}
}

// O plano do fim é o que EndMatchAt fará, sem mutar nada: a sala grava o plano antes de aplicar.
func TestPlanEndMatchMatchesEndMatchAt(t *testing.T) {
	var real time.Time
	s := runningSession(t, &real)
	s.GetActiveRound().MarkCounted()
	real = real.Add(10 * time.Second)
	at := storyStart.Add(2 * time.Hour)

	if _, err := s.PlanEndMatch(storyStart, false, real); !errors.Is(err, clock.ErrBelowLimit) {
		t.Fatalf("plan below minimum: %v", err)
	}
	before := stateOf(s, real)
	plan, err := s.PlanEndMatch(at, false, real)
	if err != nil {
		t.Fatal(err)
	}
	if after := stateOf(s, real); after != before || s.IsEnded() {
		t.Fatalf("PlanEndMatch mutated the session: %+v → %+v", before, after)
	}
	if plan.Pause == nil || plan.Pause.Reason != clock.PauseEnd || plan.Pause.Elapsed != 10*time.Second {
		t.Errorf("plan pause = %+v, want end with 10 s elapsed", plan.Pause)
	}
	if !plan.StoryAt.Equal(at) || plan.Clock.Running || plan.Clock.StoryAt == nil || !plan.Clock.StoryAt.Equal(at) ||
		plan.Clock.LimitReason != clock.LimitScene || !plan.Drained || !plan.ClosedAt.Equal(real) {
		t.Errorf("plan = %+v (clock %+v)", plan, plan.Clock)
	}

	real = real.Add(5 * time.Second) // o fim é aplicado depois da escrita, com o MESMO instante
	end, err := s.EndMatchAt(at, false, plan.ClosedAt)
	if err != nil {
		t.Fatal(err)
	}
	if *end.Pause != *plan.Pause || !end.StoryAt.Equal(plan.StoryAt) {
		t.Errorf("end pause %+v / story %v, want the plan's %+v / %v", end.Pause, end.StoryAt, plan.Pause, plan.StoryAt)
	}
	if f := end.Round.GetFinishedAt(); f == nil || !f.Equal(plan.ClosedAt) {
		t.Errorf("round finishedAt = %v, want %v", f, plan.ClosedAt)
	}
	if got := s.Clock().Snapshot(real); *got.StoryAt != *plan.Clock.StoryAt || got.Running {
		t.Errorf("clock after = %+v, want the plan's", got)
	}
}

// A primeira cena nasce sem início na história; ela é carimbada quando o relógio fica definido,
// uma vez só.
func TestStampSceneStartIfUnset(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	s.SetClock(clock.NewUnset())
	if s.StampSceneStartIfUnset() || s.GetActiveScene().GetStoryStartedAt() != nil {
		t.Fatal("an unset clock must not stamp the scene")
	}
	at := time.Date(1999, 4, 1, 8, 30, 0, 0, time.UTC)
	if err := s.Clock().Start(at); err != nil {
		t.Fatal(err)
	}
	if !s.StampSceneStartIfUnset() {
		t.Fatal("a set clock must stamp an unstamped scene")
	}
	if got := s.GetActiveScene().GetStoryStartedAt(); got == nil || !got.Equal(at) {
		t.Fatalf("story_started_at = %v, want %v", got, at)
	}
	if _, _, err := s.Clock().Adjust(time.Hour, false, time.Now()); err != nil {
		t.Fatal(err)
	}
	if s.StampSceneStartIfUnset() {
		t.Error("an already stamped scene must keep its start")
	}
	if got := s.GetActiveScene().GetStoryStartedAt(); !got.Equal(at) {
		t.Errorf("story_started_at moved to %v", got)
	}
}
