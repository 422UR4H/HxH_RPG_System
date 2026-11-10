package clock_test

import (
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
)

var (
	realStart = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	story     = time.Date(2000, 1, 1, 8, 0, 0, 0, time.UTC)
)

func sec(n int) time.Duration { return time.Duration(n) * time.Second }

func TestNowWhileRunning(t *testing.T) {
	tests := []struct {
		name string
		at   time.Duration
		want time.Time
	}{
		{"90s", sec(90), story.Add(sec(90))},
		{"90.7s truncates", 90*time.Second + 700*time.Millisecond, story.Add(sec(90))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := clock.NewAt(story)
			if err := c.Play(realStart); err != nil {
				t.Fatal(err)
			}
			if got := c.Now(realStart.Add(tt.at)); !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnsetClock(t *testing.T) {
	c := clock.NewUnset()
	if c.IsSet() || !c.Now(realStart).IsZero() {
		t.Error("unset clock must not be set and Now must be zero")
	}
	if err := c.Play(realStart); !errors.Is(err, clock.ErrNotSet) {
		t.Errorf("Play: got %v", err)
	}
	if _, _, err := c.Adjust(sec(1), false, realStart); !errors.Is(err, clock.ErrNotSet) {
		t.Errorf("Adjust: got %v", err)
	}
	if err := c.Start(story); err != nil {
		t.Fatal(err)
	}
	if err := c.Start(story); !errors.Is(err, clock.ErrAlreadySet) {
		t.Errorf("second Start: got %v", err)
	}
}

func TestPlayAlreadyRunning(t *testing.T) {
	c := clock.NewAt(story)
	_ = c.Play(realStart)
	if err := c.Play(realStart); !errors.Is(err, clock.ErrAlreadyRunning) {
		t.Errorf("got %v", err)
	}
}

func TestAdjustRespectsLimit(t *testing.T) {
	c := clock.NewAt(story)
	c.RaiseLimit(clock.LimitTurn, realStart)

	if _, _, err := c.Adjust(-sec(1), false, realStart); !errors.Is(err, clock.ErrBelowLimit) {
		t.Fatalf("got %v", err)
	}
	if !c.Now(realStart).Equal(story) {
		t.Error("clock changed after failed Adjust")
	}
	from, to, err := c.Adjust(sec(3600), false, realStart)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)
	if !from.Equal(story) || !to.Equal(want) {
		t.Errorf("from %v to %v", from, to)
	}
	if _, _, err := c.Adjust(0, false, realStart); !errors.Is(err, clock.ErrZeroDelta) {
		t.Errorf("zero delta: got %v", err)
	}
}

func TestAdjustWhileRunningKeepsRunning(t *testing.T) {
	c := clock.NewAt(story)
	_ = c.Play(realStart)
	if _, _, err := c.Adjust(sec(60), false, realStart.Add(sec(30))); err != nil {
		t.Fatal(err)
	}
	if got := c.Now(realStart.Add(sec(30))); !got.Equal(story.Add(sec(90))) {
		t.Errorf("got %v", got)
	}
	if !c.Running() {
		t.Error("must keep running")
	}
}

func TestLongSkipNeedsConfirm(t *testing.T) {
	t.Run("adjust", func(t *testing.T) {
		c := clock.NewAt(story)
		if _, _, err := c.Adjust(sec(172800), false, realStart); err != nil {
			t.Fatal(err)
		}
		c = clock.NewAt(story)
		if _, _, err := c.Adjust(sec(172801), false, realStart); !errors.Is(err, clock.ErrLongSkipUnconfirmed) {
			t.Fatalf("got %v", err)
		}
		if _, _, err := c.Adjust(sec(172801), true, realStart); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("set", func(t *testing.T) {
		c := clock.NewAt(story)
		if _, err := c.Set(story.Add(sec(172800)), false, realStart); err != nil {
			t.Fatal(err)
		}
		c = clock.NewAt(story)
		if _, err := c.Set(story.Add(sec(172801)), false, realStart); !errors.Is(err, clock.ErrLongSkipUnconfirmed) {
			t.Fatalf("got %v", err)
		}
		if _, err := c.Set(story.Add(sec(172801)), true, realStart); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSetSameTimeIsZeroDelta(t *testing.T) {
	c := clock.NewAt(story)
	if _, err := c.Set(story, false, realStart); !errors.Is(err, clock.ErrZeroDelta) {
		t.Errorf("got %v", err)
	}
}

func TestRewindRealTimeDownToLimit(t *testing.T) {
	c := clock.NewAt(story)
	_ = c.Play(realStart)
	c.Pause(clock.PauseManual, realStart.Add(sec(600)))

	if _, err := c.Set(story.Add(sec(60)), false, realStart.Add(sec(600))); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Set(story.Add(-sec(1)), false, realStart.Add(sec(600))); !errors.Is(err, clock.ErrBelowLimit) {
		t.Errorf("got %v", err)
	}
}

func TestPauseReturnsElapsed(t *testing.T) {
	c := clock.NewAt(story)
	_ = c.Play(realStart)
	p := c.Pause(clock.PauseManual, realStart.Add(sec(192)))
	if p == nil {
		t.Fatal("expected pause")
	}
	if !p.From.Equal(story) || !p.To.Equal(story.Add(sec(192))) || p.Elapsed != sec(192) || p.Reason != clock.PauseManual {
		t.Errorf("got %+v", p)
	}
	if c.Pause(clock.PauseManual, realStart.Add(sec(200))) != nil {
		t.Error("second Pause must return nil")
	}
}

func TestAddRoundSecondRaisesLimit(t *testing.T) {
	c := clock.NewAt(story)
	c.AddRoundSecond(realStart)
	if got := c.Now(realStart); !got.Equal(story.Add(sec(1))) {
		t.Errorf("now %v", got)
	}
	at, reason := c.Limit()
	if !at.Equal(story.Add(sec(1))) || reason != clock.LimitRound {
		t.Errorf("limit %v %v", at, reason)
	}
}

func TestMinimum(t *testing.T) {
	t.Run("limit wins", func(t *testing.T) {
		c := clock.Reconstruct(clock.Snapshot{
			StoryAt: ptr(story.Add(sec(2))), LimitAt: ptr(story.Add(sec(5))), LimitReason: clock.LimitTurn,
		})
		if got := c.Minimum(sec(1), realStart); !got.Equal(story.Add(sec(5))) {
			t.Errorf("got %v", got)
		}
	})
	t.Run("now plus extra wins", func(t *testing.T) {
		c := clock.Reconstruct(clock.Snapshot{
			StoryAt: ptr(story.Add(sec(10))), LimitAt: ptr(story), LimitReason: clock.LimitStart,
		})
		if got := c.Minimum(sec(1), realStart); !got.Equal(story.Add(sec(11))) {
			t.Errorf("got %v", got)
		}
	})
}

func TestSnapshotCheckpointRoundTrip(t *testing.T) {
	c := clock.NewAt(story)
	_ = c.Play(realStart)
	at := realStart.Add(sec(40))
	snap := c.Snapshot(at)
	if snap.StoryAt == nil || !snap.StoryAt.Equal(story.Add(sec(40))) || !snap.Running || snap.RunningSince == nil || !snap.RunningSince.Equal(at) {
		t.Fatalf("snapshot %+v", snap)
	}
	r := clock.Reconstruct(snap)
	later := at.Add(sec(10))
	if !r.Now(later).Equal(c.Now(later)) || !r.Running() {
		t.Error("reconstructed clock diverges")
	}
	if clock.NewUnset().Snapshot(at).StoryAt != nil {
		t.Error("unset snapshot must have nil StoryAt")
	}
}

func TestStoryWireRoundTrip(t *testing.T) {
	const in = "2000-01-01T23:59:59"
	got, err := clock.ParseStory(in)
	if err != nil {
		t.Fatal(err)
	}
	if out := clock.FormatStory(got); out != in {
		t.Errorf("got %q", out)
	}
	for _, bad := range []string{"2000-01-01T23:59:59Z", "2000-01-01T23:59:59-03:00", "2000-01-01T23:59:59+00:00", "2000-01-01T23:59:59.5", "garbage", "0000-01-01T00:00:00", "10000-01-01T00:00:00"} {
		if _, err := clock.ParseStory(bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestStoryWireYearBounds(t *testing.T) {
	for _, ok := range []string{"0001-01-01T00:00:00", "9999-12-31T23:59:59"} {
		if _, err := clock.ParseStory(ok); err != nil {
			t.Errorf("%q must be accepted: %v", ok, err)
		}
	}
}

func TestRoundEquivalents(t *testing.T) {
	if got := clock.RoundEquivalents(story, story.Add(sec(90))); got != 90 {
		t.Errorf("got %d", got)
	}
}

func ptr(t time.Time) *time.Time { return &t }

func TestPauseAfterAdjustWhileRunningReportsWholeRun(t *testing.T) {
	c := clock.NewAt(story)
	_ = c.Play(realStart)
	if _, _, err := c.Adjust(sec(3600), false, realStart.Add(sec(100))); err != nil {
		t.Fatal(err)
	}
	p := c.Pause(clock.PauseManual, realStart.Add(sec(192)))
	if !p.From.Equal(story) || p.Elapsed != sec(192) || !p.To.Equal(story.Add(sec(3600+192))) {
		t.Errorf("got %+v", p)
	}
}

func TestPauseAfterRoundSecondWhileRunning(t *testing.T) {
	c := clock.NewAt(story)
	_ = c.Play(realStart)
	c.AddRoundSecond(realStart.Add(sec(10)))
	p := c.Pause(clock.PauseManual, realStart.Add(sec(20)))
	if !p.From.Equal(story) || p.Elapsed != sec(20) || !p.To.Equal(story.Add(sec(21))) {
		t.Errorf("got %+v", p)
	}
}

func TestSnapshotDoesNotDisturbPause(t *testing.T) {
	c := clock.NewAt(story)
	_ = c.Play(realStart)
	_ = c.Snapshot(realStart.Add(sec(50)))
	p := c.Pause(clock.PauseManual, realStart.Add(sec(192)))
	if !p.From.Equal(story) || p.Elapsed != sec(192) || !p.To.Equal(story.Add(sec(192))) {
		t.Errorf("got %+v", p)
	}
}
