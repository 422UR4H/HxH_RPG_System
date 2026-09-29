package match

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
	"time"

	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// update rewrites testdata/history_golden.json with this run's output instead of comparing
// against it. It exists so this file can be regenerated deliberately (as it was the first
// time, against today's get_match_history.go, BEFORE actionwire.From ever existed — see
// design spec §4.1 and task-6-brief.md Step 1) and never by accident: a plain `go test` run
// always compares.
var update = flag.Bool("update", false, "rewrite testdata/history_golden.json instead of comparing against it")

const goldenPath = "testdata/history_golden.json"

// fixtureGoldenHistoryTurn is one closed turn built to exercise every field
// toHistoryTurnResponse (and, after actionwire.From replaces its guts, every wire.Action
// field) can carry: skills, an actionSpeed roll, a feint, a trigger, a move with both a
// moveSpeed roll and a charge roll, an attack with a charge roll and a non-default spread,
// an interact, a non-zero systemBias, and three reactions — dodge, defense and repel — each
// with their own weapon/roll. Every UUID and timestamp is a fixed literal, never uuid.New()
// or time.Now(): the point of a golden file is that re-running this test without -update
// reproduces the exact same JSON every time.
func fixtureGoldenHistoryTurn() matchUC.HistoryTurn {
	turnUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	targetA := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	targetB := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	dodgeReactor := uuid.MustParse("00000000-0000-0000-0000-000000000005")
	defenseReactor := uuid.MustParse("00000000-0000-0000-0000-000000000006")
	repelReactor := uuid.MustParse("00000000-0000-0000-0000-000000000007")
	// Action/reaction ids are reconstructed to fixed literals too — NewAction otherwise mints
	// a fresh uuid.New() for each, which would make this "golden" file compare unequal to
	// itself on every run that is not -update. WithReconstructedID is exactly the escape
	// hatch NewAction documents for a caller rebuilding an Action whose identity must be
	// stable (see action.WithReconstructedID's own doc) — a gateway reading a persisted row
	// is the usual caller, and a fixed test fixture has the exact same requirement.
	mainActionID := uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	dodgeReactionID := uuid.MustParse("00000000-0000-0000-0000-0000000000bb")
	defenseReactionID := uuid.MustParse("00000000-0000-0000-0000-0000000000cc")
	repelReactionID := uuid.MustParse("00000000-0000-0000-0000-0000000000dd")
	createdAt := time.Date(2026, 9, 27, 10, 0, 0, 0, time.UTC)
	finishedAt := time.Date(2026, 9, 27, 10, 0, 30, 0, time.UTC)

	rc := func(skill string, value int, primary, secondary []int, result int) action.RollCheck {
		return action.RollCheck{
			SkillName: skill, SkillValue: value,
			Attempts: action.RollAttempts{Primary: primary, Secondary: secondary},
			Result:   result,
		}
	}
	weaponHalberd := enum.Halberd
	weaponDagger := enum.Dagger
	weaponThrowing := enum.ThrowingDagger
	difficulty := 12

	moveSpeed := rc(enum.Legerity.String(), 4, []int{2, 5}, nil, 13)
	moveCharge := rc(enum.Legerity.String(), 4, []int{1}, nil, 6)
	attackCharge := rc(enum.Legerity.String(), 3, []int{4}, nil, 9)
	feint := rc(enum.Feint.String(), 2, []int{6}, nil, 8)

	mainAction := action.NewAction(
		actorID, []uuid.UUID{targetA, targetB}, uuid.Nil,
		[]action.Skill{
			{SkillName: enum.Legerity.String(), Difficulty: &difficulty, RollCheck: rc(enum.Legerity.String(), 5, []int{3, 4}, []int{1, 2}, 11)},
			{SkillName: enum.Evasion.String(), RollCheck: rc(enum.Evasion.String(), 3, []int{2}, nil, 5)},
		},
		action.ActionSpeed{Bar: 3, RollCheck: rc(enum.Legerity.String(), 5, []int{5, 6}, []int{2, 3}, 14)},
		&feint,
		&action.Move{
			Category: enum.Dash, From: [3]int{1, 1, 0}, Position: [3]int{4, 4, 0},
			Speed: &moveSpeed, Charge: &moveCharge, FinalSpeed: 9,
		},
		&action.Attack{
			Weapon: &weaponHalberd, Hit: rc(enum.Legerity.String(), 6, []int{4, 5}, nil, 17),
			Damage: rc(enum.Legerity.String(), 4, []int{2, 3}, nil, 10),
			Charge: &attackCharge, Spread: action.SpreadSimultaneous, RelativeVelocity: 12.5,
		},
		nil, nil, &action.Trigger{}, &action.Interact{Kind: action.InteractReveal},
		action.WithReconstructedID(mainActionID),
	)
	mainAction.SystemBias = -1

	dodgeReaction := action.NewAction(
		dodgeReactor, nil, mainAction.GetID(), nil, action.ActionSpeed{},
		nil, nil, nil, nil,
		&action.Dodge{RollCheck: rc(enum.Legerity.String(), 4, []int{3, 3}, nil, 9)},
		nil, nil,
		action.WithReconstructedID(dodgeReactionID),
	)
	dodgeReaction.ReactionKind = action.ReactDodge

	defenseReaction := action.NewAction(
		defenseReactor, nil, mainAction.GetID(), nil, action.ActionSpeed{},
		nil, nil, nil,
		&action.Defense{Weapon: &weaponDagger, RollCheck: rc(enum.Legerity.String(), 4, []int{5}, nil, 10)},
		nil, nil, nil,
		action.WithReconstructedID(defenseReactionID),
	)

	repelReaction := action.NewAction(
		repelReactor, nil, mainAction.GetID(), nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
		action.WithReconstructedID(repelReactionID),
	)
	repelReaction.Repel = &action.Repel{Weapon: &weaponThrowing, RollCheck: rc(enum.Legerity.String(), 5, []int{6}, nil, 15)}
	repelReaction.ReactionKind = action.ReactRepel

	return matchUC.HistoryTurn{
		UUID: turnUUID, CreatedAt: createdAt, FinishedAt: finishedAt,
		Action: *mainAction,
		Reactions: []action.Action{
			*dodgeReaction, *defenseReaction, *repelReaction,
		},
	}
}

// TestGetMatchHistoryGolden pins the REST Action History's JSON. actionwire.From replaces
// toActionResponse's guts right after this test is committed (task-6-brief.md, design spec
// §4.1) — this file's whole job is proving that swap changes not one byte of what the REST
// history already emits. Run with -update to (re)write the golden file; a plain run compares.
func TestGetMatchHistoryGolden(t *testing.T) {
	got, err := json.MarshalIndent(toHistoryTurnResponse(fixtureGoldenHistoryTurn()), "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')

	if *update {
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("wrote %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run `go test -run TestGetMatchHistoryGolden -update ./internal/app/api/match/` to create it): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("history JSON changed:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
