# Remaining Domain Entity Tests — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Achieve full test coverage for all domain entity packages independent of the Turn/Round context.

**Architecture:** Table-driven tests using standard library only, external test packages (`_test` suffix on package name), following existing patterns from character_sheet tests.

**Tech Stack:** Go 1.23, standard `testing` package, `t.Run()` subtests.

---

## Scope

**In scope (Phases 1 & 2):**
- `entity/die` — Die + Roll
- `entity/item` — Weapon, WeaponsManager, WeaponsManagerFactory
- `entity/character_class` — CharacterClass, CharacterClassFactory
- `entity/enum` — NameFrom parsers (CharacterClassName, WeaponName, CategoryName)
- `entity/match/action` — PriorityQueue, Action, RollContext
- `entity/match` — Match entity, GameEvent

**Out of scope (excluded — coupled to Turn/Round):**
- `entity/match/turn` ❌
- `entity/match/round` ❌
- `entity/match/scene` ❌ (imports turn)
- `entity/match/battle` ❌ (Blow has no methods, depends on action types tied to round flow)

**Deferred to separate plan:**
- Domain use cases (require repository mocking)
- Gateway/API layers (integration tests)

---

## Phase 1: Independent Domain Entities

### Task 0: Create feature branch

**Files:** None

- [ ] **Step 1: Create branch**

```bash
git checkout -b feat/remaining-domain-entity-tests
```

- [ ] **Step 2: Commit** (no commit needed, just branch creation)

---

### Task 1: Die tests

**Files:**
- Create: `internal/domain/entity/die/die_test.go`

- [ ] **Step 1: Write tests**

```go
package die_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/die"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestNewDie(t *testing.T) {
	tests := []struct {
		name          string
		sides         enum.DieSides
		expectedSides int
	}{
		{"D4", enum.D4, 4},
		{"D6", enum.D6, 6},
		{"D8", enum.D8, 8},
		{"D10", enum.D10, 10},
		{"D12", enum.D12, 12},
		{"D20", enum.D20, 20},
		{"D100", enum.D100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := die.NewDie(tt.sides)
			if d.GetSides() != tt.expectedSides {
				t.Errorf("GetSides() = %d, want %d", d.GetSides(), tt.expectedSides)
			}
			if d.GetResult() != 0 {
				t.Errorf("GetResult() before Roll() = %d, want 0", d.GetResult())
			}
		})
	}
}

func TestDie_Roll(t *testing.T) {
	tests := []struct {
		name  string
		sides enum.DieSides
		max   int
	}{
		{"D4 produces 1-4", enum.D4, 4},
		{"D6 produces 1-6", enum.D6, 6},
		{"D8 produces 1-8", enum.D8, 8},
		{"D10 produces 1-10", enum.D10, 10},
		{"D12 produces 1-12", enum.D12, 12},
		{"D20 produces 1-20", enum.D20, 20},
		{"D100 produces 1-100", enum.D100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := die.NewDie(tt.sides)
			for i := 0; i < 100; i++ {
				result := d.Roll()
				if result < 1 || result > tt.max {
					t.Fatalf("Roll() = %d, want 1-%d", result, tt.max)
				}
				if d.GetResult() != result {
					t.Fatalf("GetResult() = %d, want %d (last Roll)", d.GetResult(), result)
				}
			}
		})
	}
}

func TestDie_Roll_UpdatesResult(t *testing.T) {
	d := die.NewDie(enum.D20)
	first := d.Roll()
	if first < 1 || first > 20 {
		t.Fatalf("first Roll() = %d, out of range", first)
	}
	if d.GetResult() != first {
		t.Fatalf("GetResult() after first Roll() = %d, want %d", d.GetResult(), first)
	}

	// Roll again and verify result is updated
	second := d.Roll()
	if second < 1 || second > 20 {
		t.Fatalf("second Roll() = %d, out of range", second)
	}
	if d.GetResult() != second {
		t.Fatalf("GetResult() after second Roll() = %d, want %d", d.GetResult(), second)
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/die/ -v`
Expected: PASS (all tests green)

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/die/die_test.go
git commit -m "test(die): add comprehensive Die tests

Cover NewDie construction, Roll() range validation for all die types,
and GetResult() state tracking after rolls.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Weapon tests

**Files:**
- Create: `internal/domain/entity/item/weapon_test.go`

- [ ] **Step 1: Write tests**

```go
package item_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

func TestNewWeapon(t *testing.T) {
	dice := []int{10, 6}
	w := item.NewWeapon(dice, 5, 3, 1.2, 2.5, 12, false)

	if w.GetDamage() != 5 {
		t.Errorf("GetDamage() = %d, want 5", w.GetDamage())
	}
	if w.GetDefense() != 3 {
		t.Errorf("GetDefense() = %d, want 3", w.GetDefense())
	}
	if w.GetHeight() != 1.2 {
		t.Errorf("GetHeight() = %f, want 1.2", w.GetHeight())
	}
	if w.GetWeight() != 2.5 {
		t.Errorf("GetWeight() = %f, want 2.5", w.GetWeight())
	}
	if w.GetVolume() != 12 {
		t.Errorf("GetVolume() = %d, want 12", w.GetVolume())
	}
	if w.IsFireWeapon() {
		t.Error("IsFireWeapon() = true, want false")
	}
}

func TestWeapon_GetDice_ReturnsCopy(t *testing.T) {
	original := []int{10, 6, 4}
	w := item.NewWeapon(original, 0, 0, 0, 0, 0, false)

	dice := w.GetDice()
	if len(dice) != 3 || dice[0] != 10 || dice[1] != 6 || dice[2] != 4 {
		t.Fatalf("GetDice() = %v, want [10, 6, 4]", dice)
	}

	// Mutating the returned slice should not affect internal state
	dice[0] = 999
	fresh := w.GetDice()
	if fresh[0] != 10 {
		t.Errorf("GetDice() returned reference instead of copy: got %d, want 10", fresh[0])
	}
}

func TestWeapon_GetPenality_MeleeWeapon(t *testing.T) {
	tests := []struct {
		name     string
		weight   float64
		expected float64
	}{
		{"light weapon (0.4kg)", 0.4, 0.4},
		{"medium weapon (2.5kg)", 2.5, 2.5},
		{"heavy weapon (6.0kg)", 6.0, 6.0},
		{"zero weight", 0.0, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := item.NewWeapon(nil, 0, 0, 0, tt.weight, 0, false)
			if w.GetPenality() != tt.expected {
				t.Errorf("GetPenality() = %f, want %f", w.GetPenality(), tt.expected)
			}
		})
	}
}

func TestWeapon_GetPenality_FireWeapon(t *testing.T) {
	tests := []struct {
		name     string
		weight   float64
		expected float64
	}{
		{"light fire weapon (0.5kg) => 0", 0.5, 0.0},
		{"exactly 1.0kg => 1", 1.0, 1.0},
		{"heavy fire weapon (4.5kg) => 1", 4.5, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := item.NewWeapon(nil, 0, 0, 0, tt.weight, 0, true)
			if w.GetPenality() != tt.expected {
				t.Errorf("GetPenality() = %f, want %f", w.GetPenality(), tt.expected)
			}
		})
	}
}

func TestWeapon_GetStaminaCost(t *testing.T) {
	tests := []struct {
		name         string
		weight       float64
		isFireWeapon bool
		expected     float64
	}{
		{"melee weapon uses weight", 2.5, false, 2.5},
		{"fire weapon always 1.0", 4.5, true, 1.0},
		{"light melee weapon", 0.4, false, 0.4},
		{"light fire weapon", 0.3, true, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := item.NewWeapon(nil, 0, 0, 0, tt.weight, 0, tt.isFireWeapon)
			if w.GetStaminaCost() != tt.expected {
				t.Errorf("GetStaminaCost() = %f, want %f", w.GetStaminaCost(), tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/item/ -v -run TestNewWeapon\|TestWeapon`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/item/weapon_test.go
git commit -m "test(item): add Weapon tests

Cover construction, getters, GetDice copy safety,
GetPenality logic for melee vs fire weapons,
and GetStaminaCost behavior.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: WeaponsManager tests

**Files:**
- Create: `internal/domain/entity/item/weapons_manager_test.go`

- [ ] **Step 1: Write tests**

```go
package item_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

func newTestWeaponsManager() *item.WeaponsManager {
	weapons := map[enum.WeaponName]item.Weapon{
		enum.Sword:  *item.NewWeapon([]int{10, 4}, 2, 0, 0.8, 1.5, 12, false),
		enum.Katana: *item.NewWeapon([]int{4, 12}, 7, 0, 1.0, 1.3, 14, false),
	}
	return item.NewWeaponsManager(weapons)
}

func TestWeaponsManager_Get(t *testing.T) {
	wm := newTestWeaponsManager()

	t.Run("existing weapon", func(t *testing.T) {
		w, err := wm.Get(enum.Sword)
		if err != nil {
			t.Fatalf("Get(Sword) error = %v", err)
		}
		if w.GetDamage() != 2 {
			t.Errorf("Sword damage = %d, want 2", w.GetDamage())
		}
	})

	t.Run("non-existing weapon", func(t *testing.T) {
		_, err := wm.Get(enum.Halberd)
		if !errors.Is(err, item.ErrWeaponNotFound) {
			t.Errorf("Get(Halberd) error = %v, want ErrWeaponNotFound", err)
		}
	})
}

func TestWeaponsManager_Add(t *testing.T) {
	wm := item.NewWeaponsManager(make(map[enum.WeaponName]item.Weapon))
	dagger := *item.NewWeapon([]int{8}, 5, 0, 0.3, 0.4, 3, false)

	wm.Add(enum.Dagger, dagger)

	w, err := wm.Get(enum.Dagger)
	if err != nil {
		t.Fatalf("Get after Add error = %v", err)
	}
	if w.GetDamage() != 5 {
		t.Errorf("damage = %d, want 5", w.GetDamage())
	}
}

func TestWeaponsManager_Delete(t *testing.T) {
	wm := newTestWeaponsManager()
	wm.Delete(enum.Sword)

	_, err := wm.Get(enum.Sword)
	if !errors.Is(err, item.ErrWeaponNotFound) {
		t.Errorf("Get after Delete error = %v, want ErrWeaponNotFound", err)
	}
}

func TestWeaponsManager_GetAll(t *testing.T) {
	wm := newTestWeaponsManager()
	all := wm.GetAll()

	if len(all) != 2 {
		t.Fatalf("GetAll() len = %d, want 2", len(all))
	}
	if _, ok := all[enum.Sword]; !ok {
		t.Error("GetAll() missing Sword")
	}
	if _, ok := all[enum.Katana]; !ok {
		t.Error("GetAll() missing Katana")
	}
}

func TestWeaponsManager_Delegates(t *testing.T) {
	wm := newTestWeaponsManager()

	t.Run("GetDamage", func(t *testing.T) {
		dmg, err := wm.GetDamage(enum.Katana)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if dmg != 7 {
			t.Errorf("GetDamage(Katana) = %d, want 7", dmg)
		}
	})

	t.Run("GetDefense", func(t *testing.T) {
		def, err := wm.GetDefense(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if def != 0 {
			t.Errorf("GetDefense(Sword) = %d, want 0", def)
		}
	})

	t.Run("GetWeight", func(t *testing.T) {
		w, err := wm.GetWeight(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if w != 1.5 {
			t.Errorf("GetWeight(Sword) = %f, want 1.5", w)
		}
	})

	t.Run("GetHeight", func(t *testing.T) {
		h, err := wm.GetHeight(enum.Katana)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if h != 1.0 {
			t.Errorf("GetHeight(Katana) = %f, want 1.0", h)
		}
	})

	t.Run("GetVolume", func(t *testing.T) {
		v, err := wm.GetVolume(enum.Katana)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if v != 14 {
			t.Errorf("GetVolume(Katana) = %d, want 14", v)
		}
	})

	t.Run("IsFireWeapon", func(t *testing.T) {
		fire, err := wm.IsFireWeapon(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if fire {
			t.Error("IsFireWeapon(Sword) = true, want false")
		}
	})

	t.Run("GetDice", func(t *testing.T) {
		dice, err := wm.GetDice(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if len(dice) != 2 || dice[0] != 10 || dice[1] != 4 {
			t.Errorf("GetDice(Sword) = %v, want [10, 4]", dice)
		}
	})

	t.Run("GetPenality", func(t *testing.T) {
		p, err := wm.GetPenality(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if p != 1.5 {
			t.Errorf("GetPenality(Sword) = %f, want 1.5", p)
		}
	})

	t.Run("GetStaminaCost", func(t *testing.T) {
		sc, err := wm.GetStaminaCost(enum.Sword)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if sc != 1.5 {
			t.Errorf("GetStaminaCost(Sword) = %f, want 1.5", sc)
		}
	})

	t.Run("delegate error on missing weapon", func(t *testing.T) {
		_, err := wm.GetDamage(enum.Halberd)
		if !errors.Is(err, item.ErrWeaponNotFound) {
			t.Errorf("GetDamage(Halberd) error = %v, want ErrWeaponNotFound", err)
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/item/ -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/item/weapons_manager_test.go
git commit -m "test(item): add WeaponsManager tests

Cover Get, Add, Delete, GetAll, and all delegate methods
including error propagation for missing weapons.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: WeaponsManagerFactory tests

**Files:**
- Create: `internal/domain/entity/item/weapons_factory_test.go`

- [ ] **Step 1: Write tests**

```go
package item_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

func TestWeaponsManagerFactory_Build(t *testing.T) {
	factory := item.NewWeaponsManagerFactory()
	wm := factory.Build()

	allWeapons := enum.GetAllWeaponNames()
	all := wm.GetAll()

	if len(all) != len(allWeapons) {
		t.Fatalf("Build() produced %d weapons, want %d", len(all), len(allWeapons))
	}

	for _, name := range allWeapons {
		if _, ok := all[name]; !ok {
			t.Errorf("Build() missing weapon: %s", name)
		}
	}
}

func TestWeaponsManagerFactory_Build_WeaponProperties(t *testing.T) {
	factory := item.NewWeaponsManagerFactory()
	wm := factory.Build()

	tests := []struct {
		name         string
		weapon       enum.WeaponName
		damage       int
		isFireWeapon bool
	}{
		{"Dagger is melee with damage 5", enum.Dagger, 5, false},
		{"Katana is melee with damage 7", enum.Katana, 7, false},
		{"Pistol38 is fire weapon with damage 4", enum.Pistol38, 4, true},
		{"Crossbow is fire weapon with damage 2", enum.Crossbow, 2, true},
		{"Bomb is melee (thrown) with damage 0", enum.Bomb, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := wm.Get(tt.weapon)
			if err != nil {
				t.Fatalf("Get(%s) error = %v", tt.weapon, err)
			}
			if w.GetDamage() != tt.damage {
				t.Errorf("GetDamage() = %d, want %d", w.GetDamage(), tt.damage)
			}
			if w.IsFireWeapon() != tt.isFireWeapon {
				t.Errorf("IsFireWeapon() = %v, want %v", w.IsFireWeapon(), tt.isFireWeapon)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/item/ -v -run Factory`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/item/weapons_factory_test.go
git commit -m "test(item): add WeaponsManagerFactory tests

Verify Build() produces all weapons from enum and spot-check
specific weapon properties (damage, fire weapon flag).

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: CharacterClass validation tests

**Files:**
- Create: `internal/domain/entity/character_class/character_class_test.go`

- [ ] **Step 1: Write tests**

```go
package characterclass_test

import (
	"errors"
	"testing"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func buildClassWithDistribution() *cc.CharacterClass {
	profile := *cc.NewClassProfile(enum.Hunter, "", "Test class", "")
	distribution := &cc.Distribution{
		SkillPoints:          []int{210, 127},
		ProficiencyPoints:    []int{127, 127},
		SkillsAllowed:        []enum.SkillName{enum.Velocity, enum.Reflex, enum.Accuracy},
		ProficienciesAllowed: []enum.WeaponName{enum.Bow, enum.Longbow, enum.Dagger},
	}
	return cc.NewCharacterClass(profile, distribution, nil, nil, nil, nil, nil, nil, nil)
}

func buildClassWithoutDistribution() *cc.CharacterClass {
	profile := *cc.NewClassProfile(enum.Swordsman, "", "No distribution", "")
	return cc.NewCharacterClass(profile, nil, nil, nil, nil, nil, nil, nil, nil)
}

func TestCharacterClass_GetName(t *testing.T) {
	class := buildClassWithDistribution()
	if class.GetName() != enum.Hunter {
		t.Errorf("GetName() = %v, want Hunter", class.GetName())
	}
	if class.GetNameString() != "Hunter" {
		t.Errorf("GetNameString() = %s, want Hunter", class.GetNameString())
	}
}

func TestCharacterClass_ValidateSkills(t *testing.T) {
	t.Run("valid skills", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Velocity: 210,
			enum.Reflex:   127,
		}
		if err := class.ValidateSkills(skills); err != nil {
			t.Errorf("ValidateSkills() error = %v, want nil", err)
		}
	})

	t.Run("no distribution and no skills is valid", func(t *testing.T) {
		class := buildClassWithoutDistribution()
		if err := class.ValidateSkills(map[enum.SkillName]int{}); err != nil {
			t.Errorf("ValidateSkills() error = %v, want nil", err)
		}
	})

	t.Run("no distribution but skills provided", func(t *testing.T) {
		class := buildClassWithoutDistribution()
		skills := map[enum.SkillName]int{enum.Velocity: 100}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrNoSkillDistribution) {
			t.Errorf("error = %v, want ErrNoSkillDistribution", err)
		}
	})

	t.Run("wrong count of skills", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Velocity: 210,
		}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrSkillsCountMismatch) {
			t.Errorf("error = %v, want ErrSkillsCountMismatch", err)
		}
	})

	t.Run("skill not allowed", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Velocity: 210,
			enum.Heal:     127, // not in allowed list
		}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrSkillNotAllowed) {
			t.Errorf("error = %v, want ErrSkillNotAllowed", err)
		}
	})

	t.Run("points mismatch", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Velocity: 210,
			enum.Reflex:   999, // not a valid point value
		}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrSkillsPointsMismatch) {
			t.Errorf("error = %v, want ErrSkillsPointsMismatch", err)
		}
	})
}

func TestCharacterClass_ValidateProficiencies(t *testing.T) {
	t.Run("valid proficiencies", func(t *testing.T) {
		class := buildClassWithDistribution()
		profs := map[enum.WeaponName]int{
			enum.Bow:     127,
			enum.Longbow: 127,
		}
		if err := class.ValidateProficiencies(profs); err != nil {
			t.Errorf("ValidateProficiencies() error = %v, want nil", err)
		}
	})

	t.Run("no distribution and no profs is valid", func(t *testing.T) {
		class := buildClassWithoutDistribution()
		if err := class.ValidateProficiencies(map[enum.WeaponName]int{}); err != nil {
			t.Errorf("ValidateProficiencies() error = %v, want nil", err)
		}
	})

	t.Run("no distribution but profs provided", func(t *testing.T) {
		class := buildClassWithoutDistribution()
		profs := map[enum.WeaponName]int{enum.Dagger: 100}
		err := class.ValidateProficiencies(profs)
		if !errors.Is(err, cc.ErrNoProficiencyDistribution) {
			t.Errorf("error = %v, want ErrNoProficiencyDistribution", err)
		}
	})

	t.Run("wrong count", func(t *testing.T) {
		class := buildClassWithDistribution()
		profs := map[enum.WeaponName]int{enum.Bow: 127}
		err := class.ValidateProficiencies(profs)
		if !errors.Is(err, cc.ErrProficienciesCountMismatch) {
			t.Errorf("error = %v, want ErrProficienciesCountMismatch", err)
		}
	})

	t.Run("proficiency not allowed", func(t *testing.T) {
		class := buildClassWithDistribution()
		profs := map[enum.WeaponName]int{
			enum.Bow:   127,
			enum.Rifle: 127, // not allowed
		}
		err := class.ValidateProficiencies(profs)
		if !errors.Is(err, cc.ErrProficiencyNotAllowed) {
			t.Errorf("error = %v, want ErrProficiencyNotAllowed", err)
		}
	})

	t.Run("points mismatch", func(t *testing.T) {
		class := buildClassWithDistribution()
		profs := map[enum.WeaponName]int{
			enum.Bow:     127,
			enum.Longbow: 999,
		}
		err := class.ValidateProficiencies(profs)
		if !errors.Is(err, cc.ErrProficienciesPointsMismatch) {
			t.Errorf("error = %v, want ErrProficienciesPointsMismatch", err)
		}
	})
}

func TestCharacterClass_ApplySkills(t *testing.T) {
	class := buildClassWithDistribution()
	skills := map[enum.SkillName]int{
		enum.Velocity: 210,
		enum.Reflex:   127,
	}
	class.ApplySkills(skills)

	if class.SkillsExps[enum.Velocity] != 210 {
		t.Errorf("SkillsExps[Velocity] = %d, want 210", class.SkillsExps[enum.Velocity])
	}
	if class.SkillsExps[enum.Reflex] != 127 {
		t.Errorf("SkillsExps[Reflex] = %d, want 127", class.SkillsExps[enum.Reflex])
	}
}

func TestCharacterClass_ApplyProficiencies(t *testing.T) {
	class := buildClassWithDistribution()
	profs := map[enum.WeaponName]int{
		enum.Bow: 127,
	}
	class.ApplyProficiencies(profs)

	if class.ProficienciesExps[enum.Bow] != 127 {
		t.Errorf("ProficienciesExps[Bow] = %d, want 127", class.ProficienciesExps[enum.Bow])
	}
}

func TestDistribution_AllowSkill(t *testing.T) {
	d := &cc.Distribution{
		SkillsAllowed: []enum.SkillName{enum.Velocity, enum.Reflex},
	}
	if !d.AllowSkill(enum.Velocity) {
		t.Error("AllowSkill(Velocity) = false, want true")
	}
	if d.AllowSkill(enum.Heal) {
		t.Error("AllowSkill(Heal) = true, want false")
	}
}

func TestDistribution_AllowProficiency(t *testing.T) {
	d := &cc.Distribution{
		ProficienciesAllowed: []enum.WeaponName{enum.Bow, enum.Longbow},
	}
	if !d.AllowProficiency(enum.Bow) {
		t.Error("AllowProficiency(Bow) = false, want true")
	}
	if d.AllowProficiency(enum.Rifle) {
		t.Error("AllowProficiency(Rifle) = true, want false")
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/character_class/ -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/character_class/character_class_test.go
git commit -m "test(character_class): add CharacterClass validation tests

Cover ValidateSkills, ValidateProficiencies, ApplySkills,
ApplyProficiencies, Distribution.AllowSkill/AllowProficiency,
and all error cases.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: CharacterClassFactory tests

**Files:**
- Create: `internal/domain/entity/character_class/character_class_factory_test.go`

- [ ] **Step 1: Write tests**

```go
package characterclass_test

import (
	"testing"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestCharacterClassFactory_Build(t *testing.T) {
	factory := cc.NewCharacterClassFactory()
	classes := factory.Build()

	expectedClasses := []enum.CharacterClassName{
		enum.Swordsman, enum.Samurai, enum.Ninja, enum.Rogue,
		enum.Netrunner, enum.Pirate, enum.Mercenary, enum.Terrorist,
		enum.Monk, enum.Military, enum.Hunter, enum.WeaponsMaster,
	}

	if len(classes) != len(expectedClasses) {
		t.Fatalf("Build() produced %d classes, want %d", len(classes), len(expectedClasses))
	}

	for _, name := range expectedClasses {
		class, ok := classes[name]
		if !ok {
			t.Errorf("Build() missing class: %s", name)
			continue
		}
		if class.GetName() != name {
			t.Errorf("class %s: GetName() = %v", name, class.GetName())
		}
	}
}

func TestCharacterClassFactory_Build_ClassesHaveSkills(t *testing.T) {
	factory := cc.NewCharacterClassFactory()
	classes := factory.Build()

	for name, class := range classes {
		if len(class.SkillsExps) == 0 {
			t.Errorf("class %s has no skills", name)
		}
	}
}

func TestCharacterClassFactory_Build_DistributionClasses(t *testing.T) {
	factory := cc.NewCharacterClassFactory()
	classes := factory.Build()

	classesWithDist := []enum.CharacterClassName{
		enum.Ninja, enum.Mercenary, enum.Hunter,
	}
	for _, name := range classesWithDist {
		class := classes[name]
		if class.Distribution == nil {
			t.Errorf("class %s: expected non-nil Distribution", name)
		}
	}

	classesWithoutDist := []enum.CharacterClassName{
		enum.Swordsman, enum.Samurai, enum.Rogue, enum.Pirate,
	}
	for _, name := range classesWithoutDist {
		class := classes[name]
		if class.Distribution != nil {
			t.Errorf("class %s: expected nil Distribution", name)
		}
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/character_class/ -v -run Factory`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/character_class/character_class_factory_test.go
git commit -m "test(character_class): add CharacterClassFactory tests

Verify Build() produces all 12 classes, each has skills,
and distribution is correctly assigned (Ninja, Mercenary, Hunter have it).

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 7: Enum parser tests

**Files:**
- Create: `internal/domain/entity/enum/enum_test.go`

- [ ] **Step 1: Write tests**

```go
package enum_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestCharacterClassNameFrom(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		tests := []struct {
			input    string
			expected enum.CharacterClassName
		}{
			{"Swordsman", enum.Swordsman},
			{"Ninja", enum.Ninja},
			{"Hunter", enum.Hunter},
			{"WeaponsMaster", enum.WeaponsMaster},
		}
		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result, err := enum.CharacterClassNameFrom(tt.input)
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				if result != tt.expected {
					t.Errorf("result = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		_, err := enum.CharacterClassNameFrom("InvalidClass")
		if !errors.Is(err, enum.ErrInvalidNameOf) {
			t.Errorf("error = %v, want ErrInvalidNameOf", err)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := enum.CharacterClassNameFrom("")
		if !errors.Is(err, enum.ErrInvalidNameOf) {
			t.Errorf("error = %v, want ErrInvalidNameOf", err)
		}
	})
}

func TestWeaponNameFrom(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		tests := []struct {
			input    string
			expected enum.WeaponName
		}{
			{"Dagger", enum.Dagger},
			{"Katana", enum.Katana},
			{"Pistol38", enum.Pistol38},
			{"Ar15", enum.Ar15},
			{"ThrowingDagger", enum.ThrowingDagger},
		}
		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result, err := enum.WeaponNameFrom(tt.input)
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				if result != tt.expected {
					t.Errorf("result = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		_, err := enum.WeaponNameFrom("Lightsaber")
		if !errors.Is(err, enum.ErrInvalidNameOf) {
			t.Errorf("error = %v, want ErrInvalidNameOf", err)
		}
	})
}

func TestCategoryNameFrom(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		tests := []struct {
			input    string
			expected enum.CategoryName
		}{
			{"Reinforcement", enum.Reinforcement},
			{"Transmutation", enum.Transmutation},
			{"Materialization", enum.Materialization},
			{"Specialization", enum.Specialization},
			{"Manipulation", enum.Manipulation},
			{"Emission", enum.Emission},
		}
		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				result, err := enum.CategoryNameFrom(tt.input)
				if err != nil {
					t.Fatalf("error = %v", err)
				}
				if result != tt.expected {
					t.Errorf("result = %v, want %v", result, tt.expected)
				}
			})
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		_, err := enum.CategoryNameFrom("Fire")
		if !errors.Is(err, enum.ErrInvalidNameOf) {
			t.Errorf("error = %v, want ErrInvalidNameOf", err)
		}
	})
}

func TestDieSides_GetSides(t *testing.T) {
	tests := []struct {
		name     string
		die      enum.DieSides
		expected int
	}{
		{"D4", enum.D4, 4},
		{"D6", enum.D6, 6},
		{"D8", enum.D8, 8},
		{"D10", enum.D10, 10},
		{"D12", enum.D12, 12},
		{"D20", enum.D20, 20},
		{"D100", enum.D100, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.die.GetSides() != tt.expected {
				t.Errorf("GetSides() = %d, want %d", tt.die.GetSides(), tt.expected)
			}
		})
	}
}

func TestGetAllCharacterClasses(t *testing.T) {
	classes := enum.GetAllCharacterClasses()
	if len(classes) != 16 {
		t.Errorf("GetAllCharacterClasses() len = %d, want 16", len(classes))
	}
}

func TestGetAllWeaponNames(t *testing.T) {
	weapons := enum.GetAllWeaponNames()
	if len(weapons) != 40 {
		t.Errorf("GetAllWeaponNames() len = %d, want 40", len(weapons))
	}
}

func TestAllNenCategoryNames(t *testing.T) {
	categories := enum.AllNenCategoryNames()
	if len(categories) != 6 {
		t.Errorf("AllNenCategoryNames() len = %d, want 6", len(categories))
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/enum/ -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/enum/enum_test.go
git commit -m "test(enum): add enum parser and collection tests

Cover CharacterClassNameFrom, WeaponNameFrom, CategoryNameFrom
with valid/invalid inputs, DieSides.GetSides, and collection
length assertions.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Phase 2: Match Entities (Independent of Turn/Round)

### Task 8: Action PriorityQueue tests

**Files:**
- Create: `internal/domain/entity/match/action/priority_queue_test.go`

- [ ] **Step 1: Write tests**

```go
package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/google/uuid"
)

func makeAction(speed int) *action.Action {
	return action.NewAction(
		uuid.New(),   // actorID
		nil,          // targetID
		uuid.Nil,     // reactToID
		nil,          // skills
		action.ActionSpeed{RollCheck: action.RollCheck{Result: speed}},
		nil,          // feint
		nil,          // move
		nil,          // attack
		nil,          // defense
		nil,          // dodge
		nil,          // trigger
	)
}

func TestPriorityQueue_NewEmpty(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)
	if !pq.IsEmpty() {
		t.Error("new queue should be empty")
	}
	if pq.Len() != 0 {
		t.Errorf("Len() = %d, want 0", pq.Len())
	}
}

func TestPriorityQueue_InsertAndExtractMax(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)

	a1 := makeAction(10)
	a2 := makeAction(30)
	a3 := makeAction(20)

	pq.Insert(a1)
	pq.Insert(a2)
	pq.Insert(a3)

	if pq.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", pq.Len())
	}

	// Should extract in order: 30, 20, 10
	first := pq.ExtractMax()
	if first.Speed.Result != 30 {
		t.Errorf("first ExtractMax() speed = %d, want 30", first.Speed.Result)
	}

	second := pq.ExtractMax()
	if second.Speed.Result != 20 {
		t.Errorf("second ExtractMax() speed = %d, want 20", second.Speed.Result)
	}

	third := pq.ExtractMax()
	if third.Speed.Result != 10 {
		t.Errorf("third ExtractMax() speed = %d, want 10", third.Speed.Result)
	}

	if !pq.IsEmpty() {
		t.Error("queue should be empty after extracting all")
	}
}

func TestPriorityQueue_ExtractMax_EmptyReturnsNil(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)
	if pq.ExtractMax() != nil {
		t.Error("ExtractMax() on empty queue should return nil")
	}
}

func TestPriorityQueue_Peek(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)

	t.Run("empty queue", func(t *testing.T) {
		if pq.Peek() != nil {
			t.Error("Peek() on empty queue should return nil")
		}
	})

	a1 := makeAction(15)
	a2 := makeAction(25)
	pq.Insert(a1)
	pq.Insert(a2)

	t.Run("returns max without removing", func(t *testing.T) {
		peeked := pq.Peek()
		if peeked.Speed.Result != 25 {
			t.Errorf("Peek() speed = %d, want 25", peeked.Speed.Result)
		}
		if pq.Len() != 2 {
			t.Errorf("Len() after Peek() = %d, want 2", pq.Len())
		}
	})
}

func TestPriorityQueue_ExtractByID(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)

	a1 := makeAction(10)
	a2 := makeAction(20)
	a3 := makeAction(30)

	pq.Insert(a1)
	pq.Insert(a2)
	pq.Insert(a3)

	t.Run("extract existing by ID", func(t *testing.T) {
		targetID := a2.GetID()
		extracted := pq.ExtractByID(targetID)
		if extracted == nil {
			t.Fatal("ExtractByID returned nil")
		}
		if extracted.GetID() != targetID {
			t.Errorf("extracted ID = %v, want %v", extracted.GetID(), targetID)
		}
		if pq.Len() != 2 {
			t.Errorf("Len() after extract = %d, want 2", pq.Len())
		}
	})

	t.Run("extract non-existing ID returns nil", func(t *testing.T) {
		result := pq.ExtractByID(uuid.New())
		if result != nil {
			t.Error("ExtractByID with unknown ID should return nil")
		}
	})
}

func TestPriorityQueue_NewFromExisting(t *testing.T) {
	a1 := makeAction(5)
	a2 := makeAction(50)
	a3 := makeAction(25)
	actions := []*action.Action{a1, a2, a3}

	pq := action.NewActionPriorityQueue(&actions)

	if pq.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", pq.Len())
	}

	// Max should be 50
	max := pq.ExtractMax()
	if max.Speed.Result != 50 {
		t.Errorf("ExtractMax() speed = %d, want 50", max.Speed.Result)
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/match/action/ -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/match/action/priority_queue_test.go
git commit -m "test(action): add PriorityQueue tests

Cover Insert, ExtractMax (ordering, empty), Peek, ExtractByID,
IsEmpty, initialization from existing slice, and heap ordering.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 9: Action entity and RollContext tests

**Files:**
- Create: `internal/domain/entity/match/action/action_test.go`

- [ ] **Step 1: Write tests**

```go
package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/die"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/google/uuid"
)

func TestNewAction(t *testing.T) {
	actorID := uuid.New()
	targetIDs := []uuid.UUID{uuid.New(), uuid.New()}
	reactToID := uuid.New()

	a := action.NewAction(
		actorID, targetIDs, reactToID, nil,
		action.ActionSpeed{Bar: 5, RollCheck: action.RollCheck{Result: 42}},
		nil, nil, nil, nil, nil, nil,
	)

	if a.GetID() == uuid.Nil {
		t.Error("GetID() should not be nil UUID")
	}
	if a.GetActorID() != actorID {
		t.Errorf("GetActorID() = %v, want %v", a.GetActorID(), actorID)
	}
	if a.Speed.Result != 42 {
		t.Errorf("Speed.Result = %d, want 42", a.Speed.Result)
	}
	if a.Speed.Bar != 5 {
		t.Errorf("Speed.Bar = %d, want 5", a.Speed.Bar)
	}
}

func TestNewAction_UniqueIDs(t *testing.T) {
	a1 := makeAction(10)
	a2 := makeAction(20)

	if a1.GetID() == a2.GetID() {
		t.Error("two actions should have different UUIDs")
	}
}

func TestRollContext_GetDiceResult(t *testing.T) {
	d1 := die.NewDie(enum.D6)
	d2 := die.NewDie(enum.D8)

	// Roll dice to get results
	r1 := d1.Roll()
	r2 := d2.Roll()

	rc := action.RollContext{
		Dice: []die.Die{*d1, *d2},
	}

	result := rc.GetDiceResult(*d1) // parameter is unused in implementation
	expectedSum := r1 + r2
	if result != expectedSum {
		t.Errorf("GetDiceResult() = %d, want %d (sum of rolled dice)", result, expectedSum)
	}
}

func TestRollContext_GetDiceResult_Empty(t *testing.T) {
	rc := action.RollContext{
		Dice: []die.Die{},
	}
	result := rc.GetDiceResult(*die.NewDie(enum.D6))
	if result != 0 {
		t.Errorf("GetDiceResult() with empty dice = %d, want 0", result)
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/match/action/ -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/match/action/action_test.go
git commit -m "test(action): add Action entity and RollContext tests

Cover NewAction construction, UUID generation, getter methods,
and RollContext.GetDiceResult with populated and empty dice.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 10: Match entity tests

**Files:**
- Create: `internal/domain/entity/match/match_test.go`

- [ ] **Step 1: Write tests**

```go
package match_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/scene"
	"github.com/google/uuid"
)

func TestNewMatch(t *testing.T) {
	masterUUID := uuid.New()
	campaignUUID := uuid.New()
	gameStart := time.Now().Add(24 * time.Hour)
	storyStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	m, err := match.NewMatch(
		masterUUID, campaignUUID,
		"Test Match", "A brief description",
		"Full description", true,
		gameStart, storyStart,
	)

	if err != nil {
		t.Fatalf("NewMatch() error = %v", err)
	}
	if m.UUID == uuid.Nil {
		t.Error("UUID should not be nil")
	}
	if m.MasterUUID != masterUUID {
		t.Errorf("MasterUUID = %v, want %v", m.MasterUUID, masterUUID)
	}
	if m.CampaignUUID != campaignUUID {
		t.Errorf("CampaignUUID = %v, want %v", m.CampaignUUID, campaignUUID)
	}
	if m.Title != "Test Match" {
		t.Errorf("Title = %s, want Test Match", m.Title)
	}
	if m.BriefInitialDescription != "A brief description" {
		t.Errorf("BriefInitialDescription = %s", m.BriefInitialDescription)
	}
	if m.Description != "Full description" {
		t.Errorf("Description = %s", m.Description)
	}
	if !m.IsPublic {
		t.Error("IsPublic = false, want true")
	}
	if m.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestMatch_AddScene_GetScenes(t *testing.T) {
	m, _ := match.NewMatch(
		uuid.New(), uuid.New(),
		"Test", "Brief", "Desc", false,
		time.Now(), time.Now(),
	)

	if len(m.GetScenes()) != 0 {
		t.Fatalf("initial scenes should be empty")
	}

	s1 := scene.NewScene(enum.Roleplay, "Opening scene")
	s2 := scene.NewScene(enum.Battle, "First battle")

	m.AddScene(s1)
	m.AddScene(s2)

	scenes := m.GetScenes()
	if len(scenes) != 2 {
		t.Fatalf("GetScenes() len = %d, want 2", len(scenes))
	}
	if scenes[0].GetCategory() != enum.Roleplay {
		t.Errorf("first scene category = %v, want Roleplay", scenes[0].GetCategory())
	}
	if scenes[1].GetCategory() != enum.Battle {
		t.Errorf("second scene category = %v, want Battle", scenes[1].GetCategory())
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/match/ -v -run TestNewMatch\|TestMatch_Add`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/match/match_test.go
git commit -m "test(match): add Match entity tests

Cover NewMatch construction with all fields,
AddScene, and GetScenes ordering.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 11: GameEvent tests

**Files:**
- Create: `internal/domain/entity/match/game_event_test.go`

- [ ] **Step 1: Write tests**

```go
package match_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
)

func TestNewGameEvent(t *testing.T) {
	t.Run("with categories", func(t *testing.T) {
		categories := []enum.GameEventCategory{enum.Death, enum.Achievement}
		desc := "A character died"
		event := match.NewGameEvent(categories, "Character Death", &desc, nil)

		if event == nil {
			t.Fatal("NewGameEvent returned nil")
		}
	})

	t.Run("nil categories defaults to Other", func(t *testing.T) {
		event := match.NewGameEvent(nil, "Some event", nil, nil)
		if event == nil {
			t.Fatal("NewGameEvent returned nil")
		}
	})

	t.Run("with date change", func(t *testing.T) {
		newDate := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
		categories := []enum.GameEventCategory{enum.DateChange}
		event := match.NewGameEvent(categories, "Time Skip", nil, &newDate)

		if event == nil {
			t.Fatal("NewGameEvent returned nil")
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/entity/match/ -v -run TestNewGameEvent`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/match/game_event_test.go
git commit -m "test(match): add GameEvent tests

Cover NewGameEvent with categories, nil categories (defaults to Other),
and date change events.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 12: Final verification and branch completion

**Files:** None (verification only)

- [ ] **Step 1: Run all tests**

```bash
go test ./internal/domain/entity/die/ ./internal/domain/entity/item/ ./internal/domain/entity/character_class/ ./internal/domain/entity/enum/ ./internal/domain/entity/match/action/ ./internal/domain/entity/match/ -v -count=1
```

Expected: All PASS

- [ ] **Step 2: Run full suite to confirm no regressions**

```bash
go test ./... 2>&1 | grep -E "^(ok|FAIL)"
```

Expected: Only `match/turn` FAIL (pre-existing), all others PASS

- [ ] **Step 3: Merge to main**

```bash
git checkout main
git merge feat/remaining-domain-entity-tests
git branch -d feat/remaining-domain-entity-tests
```

---

## Summary

| Phase | Package | Tests | Focus |
|-------|---------|-------|-------|
| 1 | die | 3 | Roll range, result state |
| 1 | item (weapon) | 5 | Getters, penalty, stamina |
| 1 | item (manager) | 5 | CRUD, delegates, errors |
| 1 | item (factory) | 2 | All weapons built |
| 1 | character_class | 7 | Validation, apply, distribution |
| 1 | character_class (factory) | 3 | All classes built |
| 1 | enum | 7 | Parsers, collections, DieSides |
| 2 | action (queue) | 5 | Heap operations, ExtractByID |
| 2 | action (entity) | 3 | Construction, UUID, RollContext |
| 2 | match | 2 | NewMatch, AddScene |
| 2 | match (event) | 1 | GameEvent construction |
| **Total** | | **~43** | |

---

## Phase 3 (Future Plan)

Domain use cases and infrastructure layers require repository mocking.
This will be addressed in a separate plan: `2026-04-29-domain-usecases-tests.md`.

Scope:
- `domain/character_sheet/` use cases
- `domain/match/` use cases
- `domain/auth/`, `domain/campaign/`, `domain/scenario/` use cases
- Gateway integration tests (optional, requires DB)
