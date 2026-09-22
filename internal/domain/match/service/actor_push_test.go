package service

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// pushedSheet builds a fresh sheet and raises its Push through real XP, so the test reads
// whatever the character sheet's own skill math actually produces instead of asserting a
// number this test does not own. A round1-review fix: TestRawDamageAddsThePush proves the
// FORMULA (an int in, an int out) but never exercises actorPush, skillValueOf or in.Sheets —
// the real link the spec asked for. This file and push_repercussion_test.go close that gap.
func pushedSheet(t *testing.T) (*csSheet.CharacterSheet, int) {
	t.Helper()
	playerUUID := uuid.New()
	cs, err := csSheet.NewCharacterSheetFactory().Build(
		&playerUUID, nil, nil,
		csSheet.CharacterProfile{NickName: "Pusher", FullName: "Push Test Subject"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	if err := cs.IncreaseExpForSkill(experience.NewUpgradeCascade(500), enum.Push); err != nil {
		t.Fatalf("IncreaseExpForSkill(Push): %v", err)
	}
	push, err := cs.GetValueForTestOfSkill(enum.Push)
	if err != nil {
		t.Fatalf("GetValueForTestOfSkill(Push): %v", err)
	}
	if push == 0 {
		t.Fatal("test setup: Push is still 0 after raising its exp — the fixture is not doing its job")
	}
	return cs, push
}

// TestActorPush_ReadsTheAttackersSheet proves actorPush is not a dead wire: given a real
// sheet with a non-zero Push, in.Sheets, and the actor's own ID, it returns exactly what
// GetValueForTestOfSkill(Push) reads off that sheet — not a hardcoded, decoupled number.
func TestActorPush_ReadsTheAttackersSheet(t *testing.T) {
	actorID := uuid.New()
	cs, wantPush := pushedSheet(t)

	a := action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil)
	in := ResolveInput{Sheets: map[uuid.UUID]*csSheet.CharacterSheet{actorID: cs}}

	if got := (TurnResolver{}).actorPush(in, *a); got != wantPush {
		t.Fatalf("actorPush() = %d, want %d (the sheet's own Push)", got, wantPush)
	}
}

// TestActorPush_MissingSheetIsZero pins the nil-guard the wall branch depends on: it is NOT
// behind actorSheetMissing, so an action whose actor sheet never reached the resolver must
// not panic here — zero is the honest answer.
func TestActorPush_MissingSheetIsZero(t *testing.T) {
	actorID := uuid.New()
	a := action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil)

	t.Run("the actor's sheet was never handed to the resolver", func(t *testing.T) {
		in := ResolveInput{Sheets: map[uuid.UUID]*csSheet.CharacterSheet{}}
		if got := (TurnResolver{}).actorPush(in, *a); got != 0 {
			t.Fatalf("actorPush() = %d, want 0 for an absent sheet", got)
		}
	})

	t.Run("Sheets itself is nil — the wall branch's own shape", func(t *testing.T) {
		in := ResolveInput{}
		if got := (TurnResolver{}).actorPush(in, *a); got != 0 {
			t.Fatalf("actorPush() = %d, want 0 when Sheets is nil", got)
		}
	})
}
