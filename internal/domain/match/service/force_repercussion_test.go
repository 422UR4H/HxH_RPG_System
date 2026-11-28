package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	sheetPkg "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// This file proves the LINK the spec's Task 2 verification actually asked for: two
// CHARACTERS, same weapon, same dice, different Force, different final damage — not just the
// RawDamage formula in isolation (damage_test.go's TestRawDamageAddsTheForce covers that, and
// still does). Round 1 review found this gap: TestRawDamageAddsTheForce calls
// service.RawDamage with force as a literal int, which never touches actorDamageSkill, skillValueOf,
// in.Sheets, or the two real call sites in turn_resolver.go. Without this file, actorDamageSkill
// could be reverted to always returning 0 (a bad merge, a wrong map key, Force swapped for
// another enum) and every test in the suite would still pass.

// sheetWithForce is plainSheet's (character_collision_test.go) non-zero-Force sibling: real XP
// raised on the real Force skill, read back through the same GetValueForTestOfSkill path the
// resolver itself uses — not a hardcoded number this test does not own.
func sheetWithForce(t *testing.T) (*sheetPkg.CharacterSheet, int) {
	t.Helper()
	cs := plainSheet(t)
	if err := cs.IncreaseExpForSkill(experience.NewUpgradeCascade(500), enum.Force); err != nil {
		t.Fatalf("IncreaseExpForSkill(Force): %v", err)
	}
	force, err := cs.GetValueForTestOfSkill(enum.Force)
	if err != nil {
		t.Fatalf("GetValueForTestOfSkill(Force): %v", err)
	}
	if force == 0 {
		t.Fatal("test setup: Force is still 0 — the fixture is not doing its job")
	}
	return cs, force
}

// TestSeedChain_ActorForceRaisesDamageAgainstACharacter is the character-branch half of the
// repercussion: two attackers, same sword, same dice — one with Force 0, one with real Force —
// against the same kind of target. Only the attacker's Force differs, and RawDamage on the
// resolved CharacterResult must differ by exactly that Force, end to end through
// TurnResolver.Resolve → seedChain → actorDamageSkill → skillValueOf.
func TestSeedChain_ActorForceRaisesDamageAgainstACharacter(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	sword := enum.Sword
	tn := attackTurn(actorID, targetID, []int{10, 8}, []int{9, 3}, &sword)

	weak := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, tn))

	forced, wantForce := sheetWithForce(t)
	in := resolveInput(t, actorID, targetID, tn)
	in.Sheets[actorID] = forced
	strong := service.TurnResolver{}.Resolve(in)

	if len(weak.CharacterResults) != 1 || len(strong.CharacterResults) != 1 {
		t.Fatalf("expected 1 character result each, got %d (weak) and %d (strong)",
			len(weak.CharacterResults), len(strong.CharacterResults))
	}
	gotRaw := strong.CharacterResults[0].RawDamage - weak.CharacterResults[0].RawDamage
	if gotRaw != wantForce {
		t.Fatalf("RawDamage moved by %d, want %d (the attacker's Force) — weak=%d strong=%d",
			gotRaw, wantForce, weak.CharacterResults[0].RawDamage, strong.CharacterResults[0].RawDamage)
	}
	// The armed attack lands against a bare-handed (plain-sheet) defense, which is not
	// reduced (see TestResolve_CharacterBranch), so the Force shows up whole in the final
	// number too — this is the "dano final" the spec's verification names.
	gotEffective := strong.CharacterResults[0].EffectiveDamage - weak.CharacterResults[0].EffectiveDamage
	if gotEffective != wantForce {
		t.Fatalf("EffectiveDamage moved by %d, want %d (the attacker's Force) — weak=%d strong=%d",
			gotEffective, wantForce,
			weak.CharacterResults[0].EffectiveDamage, strong.CharacterResults[0].EffectiveDamage)
	}
}

// TestWallBranch_ActorForceRaisesDamageAgainstAWall is the wall-branch half: the wall attack
// is NOT behind actorSheetMissing, so it is the one place a broken actorDamageSkill wire could hide
// even after the character branch was covered. Same weapon, same dice, an indestructible
// margin (huge HP, zero resistance) so EffectiveDamage tracks RawDamage 1:1 with no clamping.
func TestWallBranch_ActorForceRaisesDamageAgainstAWall(t *testing.T) {
	actorID := uuid.New()
	wallID := uuid.New()
	sword := enum.Sword
	wall := mapentity.WallSegment{ID: wallID.String(), HP: 1000, MaxHP: 1000, Resistance: 0}
	reader := mockWallReader{wallID: wallID.String(), wall: wall}

	buildAttackOnWall := func() *turn.Turn {
		atk := &action.Attack{
			Weapon: &sword,
			Damage: action.RollCheck{Attempts: action.RollAttempts{Primary: []int{9, 3}}},
		}
		a := action.NewAction(actorID, []uuid.UUID{wallID}, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, atk, nil, nil, nil, nil)
		return turn.NewTurn(*a)
	}
	baseInput := func() service.ResolveInput {
		return service.ResolveInput{
			Targets: reader,
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		}
	}

	weakIn := baseInput()
	weakIn.Turn = buildAttackOnWall()
	weak := service.TurnResolver{}.Resolve(weakIn)

	forced, wantForce := sheetWithForce(t)
	strongIn := baseInput()
	strongIn.Turn = buildAttackOnWall()
	strongIn.Sheets = map[uuid.UUID]*sheetPkg.CharacterSheet{actorID: forced}
	strong := service.TurnResolver{}.Resolve(strongIn)

	if len(weak.WallResults) != 1 || len(strong.WallResults) != 1 {
		t.Fatalf("expected 1 WallResult each, got %d (weak) and %d (strong)",
			len(weak.WallResults), len(strong.WallResults))
	}
	got := strong.WallResults[0].EffectiveDamage - weak.WallResults[0].EffectiveDamage
	if got != wantForce {
		t.Fatalf("wall EffectiveDamage moved by %d, want %d (the attacker's Force) — weak=%d strong=%d",
			got, wantForce, weak.WallResults[0].EffectiveDamage, strong.WallResults[0].EffectiveDamage)
	}
}
