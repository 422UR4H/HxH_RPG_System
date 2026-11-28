package service

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// forcedSheet builds a fresh sheet and raises its Force through real XP, so the test reads
// whatever the character sheet's own skill math actually produces instead of asserting a
// number this test does not own. A round1-review fix: TestRawDamageAddsTheForce proves the
// FORMULA (an int in, an int out) but never exercises actorDamageSkill, skillValueOf or in.Sheets —
// the real link the spec asked for. This file and force_repercussion_test.go close that gap.
func forcedSheet(t *testing.T) (*csSheet.CharacterSheet, int) {
	t.Helper()
	playerUUID := uuid.New()
	cs, err := csSheet.NewCharacterSheetFactory().Build(
		&playerUUID, nil, nil,
		csSheet.CharacterProfile{NickName: "Forcer", FullName: "Force Test Subject"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	if err := cs.IncreaseExpForSkill(experience.NewUpgradeCascade(500), enum.Force); err != nil {
		t.Fatalf("IncreaseExpForSkill(Force): %v", err)
	}
	force, err := cs.GetValueForTestOfSkill(enum.Force)
	if err != nil {
		t.Fatalf("GetValueForTestOfSkill(Force): %v", err)
	}
	if force == 0 {
		t.Fatal("test setup: Force is still 0 after raising its exp — the fixture is not doing its job")
	}
	return cs, force
}

// TestActorForce_ReadsTheAttackersSheet proves actorDamageSkill is not a dead wire: given a real
// sheet with a non-zero Force, in.Sheets, and the actor's own ID, it returns exactly what
// GetValueForTestOfSkill(Force) reads off that sheet — not a hardcoded, decoupled number.
func TestActorForce_ReadsTheAttackersSheet(t *testing.T) {
	actorID := uuid.New()
	cs, wantForce := forcedSheet(t)

	a := action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil)
	in := ResolveInput{Sheets: map[uuid.UUID]*csSheet.CharacterSheet{actorID: cs}}

	if got := (TurnResolver{}).actorDamageSkill(in, *a); got != wantForce {
		t.Fatalf("actorDamageSkill() = %d, want %d (the sheet's own Force)", got, wantForce)
	}
}

// TestActorForce_MissingSheetIsZero pins the nil-guard the wall branch depends on: it is NOT
// behind actorSheetMissing, so an action whose actor sheet never reached the resolver must
// not panic here — zero is the honest answer.
func TestActorForce_MissingSheetIsZero(t *testing.T) {
	actorID := uuid.New()
	a := action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil)

	t.Run("the actor's sheet was never handed to the resolver", func(t *testing.T) {
		in := ResolveInput{Sheets: map[uuid.UUID]*csSheet.CharacterSheet{}}
		if got := (TurnResolver{}).actorDamageSkill(in, *a); got != 0 {
			t.Fatalf("actorDamageSkill() = %d, want 0 for an absent sheet", got)
		}
	})

	t.Run("Sheets itself is nil — the wall branch's own shape", func(t *testing.T) {
		in := ResolveInput{}
		if got := (TurnResolver{}).actorDamageSkill(in, *a); got != 0 {
			t.Fatalf("actorDamageSkill() = %d, want 0 when Sheets is nil", got)
		}
	})
}

// TestDamageAlwaysMeasuredByForce: the raw damage of an attack is the weapon's dice + flat +
// the attacker's Force, and nothing on the attack can change which skill is added.
func TestDamageAlwaysMeasuredByForce(t *testing.T) {
	actorID := uuid.New()
	cs, force := forcedSheet(t)
	a := action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
	in := ResolveInput{Sheets: map[uuid.UUID]*csSheet.CharacterSheet{actorID: cs}}
	if got := (TurnResolver{}).actorDamageSkill(in, *a); got != force {
		t.Fatalf("actorDamageSkill() = %d, want the sheet's Force %d", got, force)
	}
}
