package matchsession_test

import (
	"errors"
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
	s.SetRoundMode(enum.Free)
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
