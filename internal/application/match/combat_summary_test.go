package match_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

func summaryTurn(kind string) *match.SummaryTurn {
	return &match.SummaryTurn{TurnUUID: uuid.New(), ActorUUID: uuid.New(), ActionKind: kind}
}

func summaryRound(mode enum.RoundMode, counted, drained bool, first, last *match.SummaryTurn) match.SummaryRound {
	return match.SummaryRound{UUID: uuid.New(), Mode: mode, Counted: counted, Drained: drained, First: first, Last: last}
}

func assertHiatusTurn(t *testing.T, label string, got match.HiatusTurn, round int, want *match.SummaryTurn) {
	t.Helper()
	if got.RoundNumber != round || got.TurnUUID != want.TurnUUID || got.ActorUUID != want.ActorUUID ||
		got.ActionKind != want.ActionKind {
		t.Errorf("%s = %+v, want round %d turn %+v", label, got, round, *want)
	}
}

func TestBuildCombatSummaryCountsAndHiatuses(t *testing.T) {
	t1, t2 := summaryTurn("attack"), summaryTurn("move")
	t3, t4 := summaryTurn("attack"), summaryTurn("dodge")
	t5 := summaryTurn("skill")
	rounds := []match.SummaryRound{
		summaryRound(enum.Race, true, false, t1, t2),
		summaryRound(enum.Race, true, true, t3, t4),
		summaryRound(enum.Race, true, false, t5, t5),
	}
	minimum := time.Date(2000, 1, 1, 0, 0, 3, 0, time.UTC)

	got := match.BuildCombatSummary(rounds, summaryRound(enum.Race, false, false, nil, nil), minimum)

	if got.CountedRounds != 3 || got.CountedSeconds != 3 {
		t.Errorf("counted = %d rounds / %d s, want 3 / 3", got.CountedRounds, got.CountedSeconds)
	}
	if got.ActiveRoundCounts {
		t.Error("ActiveRoundCounts = true for an active round with no turn")
	}
	if !got.MinimumStoryAt.Equal(minimum) {
		t.Errorf("MinimumStoryAt = %v, want %v", got.MinimumStoryAt, minimum)
	}
	if len(got.Hiatuses) != 1 {
		t.Fatalf("hiatuses = %+v, want one", got.Hiatuses)
	}
	assertHiatusTurn(t, "before", got.Hiatuses[0].Before, 2, t4)
	assertHiatusTurn(t, "after", got.Hiatuses[0].After, 3, t5)
}

func TestDrainedLastRoundIsNotHiatus(t *testing.T) {
	tu := summaryTurn("attack")
	rounds := []match.SummaryRound{summaryRound(enum.Race, true, true, tu, tu)}

	got := match.BuildCombatSummary(rounds, summaryRound(enum.Free, false, false, nil, nil), time.Time{})

	if len(got.Hiatuses) != 0 {
		t.Errorf("hiatuses = %+v, want none — a drained round with no successor that has a turn ended the combat", got.Hiatuses)
	}
	if got.CountedRounds != 1 || got.CountedSeconds != 1 {
		t.Errorf("counted = %d / %d, want 1 / 1", got.CountedRounds, got.CountedSeconds)
	}
}

func TestUncountedDrainedRoundIsNotHiatus(t *testing.T) {
	a, b := summaryTurn("move"), summaryTurn("move")
	rounds := []match.SummaryRound{
		summaryRound(enum.Free, false, true, a, a),
		summaryRound(enum.Free, false, false, b, b),
	}

	got := match.BuildCombatSummary(rounds, summaryRound(enum.Free, false, false, nil, nil), time.Time{})

	if len(got.Hiatuses) != 0 {
		t.Errorf("hiatuses = %+v, want none — only a contested (counted) round makes a hiatus", got.Hiatuses)
	}
	if got.CountedRounds != 0 || got.CountedSeconds != 0 {
		t.Errorf("counted = %d / %d, want 0 / 0", got.CountedRounds, got.CountedSeconds)
	}
}

// Um round disputado (contado) que o mestre passou para Free antes de drenar continua hiato:
// o que vale é ter sido disputado, não o regime em que fechou (Ruling R9).
func TestCountedRoundSwitchedToFreeIsHiatus(t *testing.T) {
	a, b := summaryTurn("attack"), summaryTurn("move")
	rounds := []match.SummaryRound{
		summaryRound(enum.Free, true, true, a, a),
		summaryRound(enum.Free, false, false, b, b),
	}

	got := match.BuildCombatSummary(rounds, summaryRound(enum.Free, false, false, nil, nil), time.Time{})

	if len(got.Hiatuses) != 1 {
		t.Fatalf("hiatuses = %+v, want one — a counted, drained round is a hiatus whatever its last mode", got.Hiatuses)
	}
	assertHiatusTurn(t, "before", got.Hiatuses[0].Before, 1, a)
	assertHiatusTurn(t, "after", got.Hiatuses[0].After, 2, b)
}

func TestActiveRoundJoinsSummary(t *testing.T) {
	t1, t2 := summaryTurn("attack"), summaryTurn("feint")
	rounds := []match.SummaryRound{
		summaryRound(enum.Race, true, true, t1, t1),
		// um round sem turno no meio não existe (round vazio não fecha), mas é pulado
		summaryRound(enum.Race, false, false, nil, nil),
	}
	active := summaryRound(enum.Race, true, false, t2, t2)
	minimum := time.Date(2000, 1, 1, 0, 0, 2, 0, time.UTC)

	got := match.BuildCombatSummary(rounds, active, minimum)

	if !got.ActiveRoundCounts {
		t.Error("ActiveRoundCounts = false for a counted active round")
	}
	if got.CountedRounds != 1 || got.CountedSeconds != 1 {
		t.Errorf("counted = %d / %d, want 1 / 1 — the active round's second is added at close", got.CountedRounds, got.CountedSeconds)
	}
	if len(got.Hiatuses) != 1 {
		t.Fatalf("hiatuses = %+v, want one", got.Hiatuses)
	}
	assertHiatusTurn(t, "before", got.Hiatuses[0].Before, 1, t1)
	assertHiatusTurn(t, "after", got.Hiatuses[0].After, 3, t2)
}
