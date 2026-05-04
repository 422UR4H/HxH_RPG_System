# Status Normalization Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Normalize persisted status bar values (`curr`) that exceed the calculated `max` after rule changes, correcting them proportionally at hydration time and persisting the correction asynchronously.

**Architecture:** `normalizeStatus` is a pure private function called inside `Wrap()` for each of the 3 status bars; `Wrap()` now returns `(bool, error)` to signal whether any correction occurred; `hydrateCharacterSheet()` fires a goroutine to persist corrected `min/curr/max` via a new repository method, so the GET response is not blocked.

**Tech Stack:** Go 1.23, `math.Round`, pgx v5, standard `testing` package.

---

## File Map

| Action | Path | Change |
|--------|------|--------|
| Modify | `internal/domain/character_sheet/get_character_sheet.go` | Add `normalizeStatus`, update `Wrap()` signature, update call site, add `persistNormalizedStatus` |
| Create | `internal/domain/character_sheet/normalize_test.go` | White-box unit tests for `normalizeStatus` (package `charactersheet`) |
| Modify | `internal/domain/character_sheet/i_repository.go` | Add `UpdateStatusBars` to `IRepository` |
| Modify | `internal/domain/testutil/mock_character_sheet_repo.go` | Add `UpdateStatusBarsFn` field and method |
| Create | `internal/gateway/pg/sheet/update_status_bars.go` | `Repository.UpdateStatusBars` implementation |
| Modify | `internal/gateway/pg/sheet/sheet_integration_test.go` | Add `TestUpdateStatusBars` integration test |
| Modify | `internal/domain/character_sheet/get_character_sheet_test.go` | Add test for status correction triggering async persist |
| Modify | `internal/domain/entity/character_sheet/status/status_bar.go` | Remove debug `fmt.Println` statements |
| Modify | `internal/domain/entity/character_sheet/status/status_manager.go` | Remove debug `fmt.Println` statements |
| Modify | `internal/domain/character_sheet/get_character_sheet.go` | Remove remaining debug `fmt.Println` from `Wrap()` |

---

## Task 1: `normalizeStatus` — test then implement

**Files:**
- Create: `internal/domain/character_sheet/normalize_test.go`
- Modify: `internal/domain/character_sheet/get_character_sheet.go`

- [ ] **Step 1: Write the failing test**

Create `internal/domain/character_sheet/normalize_test.go`:

```go
package charactersheet

import "testing"

func TestNormalizeStatus(t *testing.T) {
	tests := []struct {
		name          string
		curr          int
		oldMax        int
		newMax        int
		minVal        int
		wantCurr      int
		wantCorrected bool
	}{
		{"no correction - curr within new max", 70, 100, 90, 0, 70, false},
		{"no correction - curr equals new max", 90, 100, 90, 0, 90, false},
		{"no correction - newMax is zero", 80, 100, 0, 0, 80, false},
		{"no correction - curr is zero", 0, 100, 90, 0, 0, false},
		{"proportional correction", 80, 100, 90, 0, 72, true},
		{"fully healed correction", 100, 100, 90, 0, 90, true},
		{"oldMax zero fallback returns newMax", 50, 0, 90, 0, 90, true},
		{"result clamped to newMax", 100, 100, 50, 0, 50, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCurr, gotCorrected := normalizeStatus(tt.curr, tt.oldMax, tt.newMax, tt.minVal)
			if gotCurr != tt.wantCurr {
				t.Errorf("curr = %d, want %d", gotCurr, tt.wantCurr)
			}
			if gotCorrected != tt.wantCorrected {
				t.Errorf("corrected = %v, want %v", gotCorrected, tt.wantCorrected)
			}
		})
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test ./internal/domain/character_sheet/... -run TestNormalizeStatus -v
```

Expected: compilation error — `normalizeStatus` undefined.

- [ ] **Step 3: Implement `normalizeStatus`**

Add to `internal/domain/character_sheet/get_character_sheet.go`, before the `Wrap` function. Add `"math"` to the import block.

```go
func normalizeStatus(curr, oldMax, newMax, minVal int) (int, bool) {
	if newMax == 0 || curr <= newMax {
		return curr, false
	}
	if oldMax <= 0 {
		// TODO(logger): status normalized (fallback, old_max=0): curr %d → new_max %d
		fmt.Printf("TODO(logger): status normalized (fallback): curr %d → new_max %d\n", curr, newMax)
		return newMax, true
	}
	corrected := int(math.Round(float64(newMax) * float64(curr) / float64(oldMax)))
	corrected = max(minVal, min(newMax, corrected))
	fmt.Printf("TODO(logger): status normalized: curr %d → %d (old_max: %d, new_max: %d)\n", curr, corrected, oldMax, newMax)
	return corrected, true
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/domain/character_sheet/... -run TestNormalizeStatus -v
```

Expected: all 8 cases PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/character_sheet/normalize_test.go \
        internal/domain/character_sheet/get_character_sheet.go
git commit -m "$(cat <<'EOF'
feat(domain): add normalizeStatus for proportional status correction on load

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Update `Wrap()` signature and normalization calls

**Files:**
- Modify: `internal/domain/character_sheet/get_character_sheet.go`

- [ ] **Step 1: Change `Wrap()` signature and all return statements**

In `get_character_sheet.go`, update the `Wrap` function:

1. Change signature from `func Wrap(...) error` to `func Wrap(...) (wasCorrected bool, err error)`.
2. Replace every `return fmt.Errorf(...)` in the function body (before the status section) with `return false, fmt.Errorf(...)`.
3. Replace the three `SetCurrStatus` blocks with the normalization loop below.
4. Replace the final `return nil` with `return wasCorrected, nil`.

The status section replaces lines 258–267 in the current file. Replace:

```go
if err := charSheet.SetCurrStatus(enum.Health, modelSheet.Health.Curr); err != nil {
    return fmt.Errorf("%w (health): %v", domainSheet.ErrFailedToSetStatus, err)
}
if err := charSheet.SetCurrStatus(enum.Stamina, modelSheet.Stamina.Curr); err != nil {
    return fmt.Errorf("%w (stamina): %v", domainSheet.ErrFailedToSetStatus, err)
}
if err := charSheet.SetCurrStatus(enum.Aura, modelSheet.Aura.Curr); err != nil {
    fmt.Println("xiiii")
    return fmt.Errorf("%w (aura): %v", domainSheet.ErrFailedToSetStatus, err)
}
fmt.Println("status definidos")
```

With:

```go
type statusEntry struct {
    name   enum.StatusName
    curr   int
    oldMax int
}
for _, e := range []statusEntry{
    {enum.Health, modelSheet.Health.Curr, modelSheet.Health.Max},
    {enum.Stamina, modelSheet.Stamina.Curr, modelSheet.Stamina.Max},
    {enum.Aura, modelSheet.Aura.Curr, modelSheet.Aura.Max},
} {
    newMax, _ := charSheet.GetMaxOfStatus(e.name)
    minVal, _ := charSheet.GetMinOfStatus(e.name)
    corrected, correctionApplied := normalizeStatus(e.curr, e.oldMax, newMax, minVal)
    if correctionApplied {
        wasCorrected = true
    }
    if err := charSheet.SetCurrStatus(e.name, corrected); err != nil {
        return false, fmt.Errorf("%w (%s): %v", domainSheet.ErrFailedToSetStatus, e.name, err)
    }
}
```

- [ ] **Step 2: Update the call site in `hydrateCharacterSheet`**

Replace (line 124):

```go
err = Wrap(characterSheet, modelSheet)
if err != nil {
    return nil, err
}
```

With (using `_` for now — goroutine dispatch added in Task 5):

```go
_, err = Wrap(characterSheet, modelSheet)
if err != nil {
    return nil, err
}
```

- [ ] **Step 3: Run all existing tests to confirm no regressions**

```bash
go test ./internal/domain/character_sheet/... -v
```

Expected: all existing tests PASS. `TestNormalizeStatus` also passes.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/character_sheet/get_character_sheet.go
git commit -m "$(cat <<'EOF'
feat(domain): update Wrap() to normalize status curr on load

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add `UpdateStatusBars` to interface and mock

**Files:**
- Modify: `internal/domain/character_sheet/i_repository.go`
- Modify: `internal/domain/testutil/mock_character_sheet_repo.go`

- [ ] **Step 1: Add method to `IRepository`**

In `internal/domain/character_sheet/i_repository.go`, add to the interface:

```go
UpdateStatusBars(
    ctx context.Context,
    sheetUUID string,
    health, stamina, aura model.StatusBar,
) error
```

- [ ] **Step 2: Add field and method to mock**

In `internal/domain/testutil/mock_character_sheet_repo.go`:

Add field to `MockCharacterSheetRepo`:
```go
UpdateStatusBarsFn func(ctx context.Context, uuid string, health, stamina, aura model.StatusBar) error
```

Add method:
```go
func (m *MockCharacterSheetRepo) UpdateStatusBars(ctx context.Context, uuid string, health, stamina, aura model.StatusBar) error {
    if m.UpdateStatusBarsFn != nil {
        return m.UpdateStatusBarsFn(ctx, uuid, health, stamina, aura)
    }
    return nil
}
```

- [ ] **Step 3: Verify compilation**

```bash
go build ./internal/...
```

Expected: compiles without errors. Gateway will fail to compile until Task 4 — run only the domain packages for now:

```bash
go build ./internal/domain/...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/character_sheet/i_repository.go \
        internal/domain/testutil/mock_character_sheet_repo.go
git commit -m "$(cat <<'EOF'
feat(domain): add UpdateStatusBars to IRepository interface and mock

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Gateway implementation + integration test

**Files:**
- Create: `internal/gateway/pg/sheet/update_status_bars.go`
- Modify: `internal/gateway/pg/sheet/sheet_integration_test.go`

- [ ] **Step 1: Write the failing integration test**

Append to `internal/gateway/pg/sheet/sheet_integration_test.go`:

```go
func TestUpdateStatusBars(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := sheet.NewRepository(pool)
	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		playerUUID := uuid.New()
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Gon")

		health := model.StatusBar{Min: 0, Curr: 17, Max: 20}
		stamina := model.StatusBar{Min: 0, Curr: 0, Max: 0}
		aura := model.StatusBar{Min: 0, Curr: 0, Max: 0}

		err := repo.UpdateStatusBars(ctx, sheetUUID, health, stamina, aura)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		updated, err := repo.GetCharacterSheetByUUID(ctx, sheetUUID)
		if err != nil {
			t.Fatalf("failed to fetch sheet: %v", err)
		}
		if updated.Health.Curr != 17 {
			t.Errorf("health curr = %d, want 17", updated.Health.Curr)
		}
		if updated.Health.Max != 20 {
			t.Errorf("health max = %d, want 20", updated.Health.Max)
		}
		if updated.Health.Min != 0 {
			t.Errorf("health min = %d, want 0", updated.Health.Min)
		}
	})

	t.Run("sheet not found is a no-op", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		health := model.StatusBar{Min: 0, Curr: 10, Max: 20}
		stamina := model.StatusBar{Min: 0, Curr: 5, Max: 10}
		aura := model.StatusBar{Min: 0, Curr: 0, Max: 0}

		err := repo.UpdateStatusBars(ctx, uuid.New().String(), health, stamina, aura)
		if err != nil {
			t.Errorf("expected no error for missing sheet, got: %v", err)
		}
	})
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test -tags=integration ./internal/gateway/pg/sheet/... -run TestUpdateStatusBars -v
```

Expected: compilation error — `UpdateStatusBars` undefined on `*Repository`.

- [ ] **Step 3: Implement `UpdateStatusBars`**

Create `internal/gateway/pg/sheet/update_status_bars.go`:

```go
package sheet

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
)

func (r *Repository) UpdateStatusBars(
	ctx context.Context,
	sheetUUID string,
	health, stamina, aura model.StatusBar,
) error {
	const query = `
		UPDATE character_sheets
		SET
			health_min_pts  = $1,
			health_curr_pts = $2,
			health_max_pts  = $3,
			stamina_min_pts = $4,
			stamina_curr_pts = $5,
			stamina_max_pts = $6,
			aura_min_pts    = $7,
			aura_curr_pts   = $8,
			aura_max_pts    = $9
		WHERE uuid = $10
	`
	_, err := r.q.Exec(ctx, query,
		health.Min, health.Curr, health.Max,
		stamina.Min, stamina.Curr, stamina.Max,
		aura.Min, aura.Curr, aura.Max,
		sheetUUID,
	)
	return err
}
```

- [ ] **Step 4: Run integration tests**

```bash
go test -tags=integration ./internal/gateway/pg/sheet/... -run TestUpdateStatusBars -v
```

Expected: both sub-tests PASS.

- [ ] **Step 5: Run all gateway sheet tests to confirm no regressions**

```bash
go test -tags=integration ./internal/gateway/pg/sheet/... -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/gateway/pg/sheet/update_status_bars.go \
        internal/gateway/pg/sheet/sheet_integration_test.go
git commit -m "$(cat <<'EOF'
feat(gateway): implement UpdateStatusBars for status normalization persistence

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: `persistNormalizedStatus` + goroutine dispatch + test

**Files:**
- Modify: `internal/domain/character_sheet/get_character_sheet.go`
- Modify: `internal/domain/character_sheet/get_character_sheet_test.go`

- [ ] **Step 1: Add test case for status correction**

Append inside `TestGetCharacterSheet` in `get_character_sheet_test.go`:

```go
t.Run("triggers async persist when status curr exceeds new max", func(t *testing.T) {
    sheetMap := newTestSheetMap()
    factory := newTestFactory()
    masterUUID := uuid.New()

    modelSheet := newValidModelSheet(nil, &masterUUID, nil)
    // HP_BASE_VALUE = 20 for a base sheet with zero XP.
    // Setting curr=25, max=30 forces normalizeStatus to compute round(20*25/30)=17.
    modelSheet.Health.Curr = 25
    modelSheet.Health.Max = 30

    done := make(chan struct{})
    mockRepo := &testutil.MockCharacterSheetRepo{
        GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
            return modelSheet, nil
        },
        UpdateStatusBarsFn: func(ctx context.Context, uuid string, health, stamina, aura model.StatusBar) error {
            close(done)
            return nil
        },
    }
    mockCampaignRepo := &testutil.MockCampaignRepo{}

    uc := charactersheet.NewGetCharacterSheetUC(sheetMap, factory, mockRepo, mockCampaignRepo)
    result, err := uc.GetCharacterSheet(ctx, modelSheet.UUID, masterUUID)
    if err != nil {
        t.Fatalf("expected no error, got: %v", err)
    }
    if result == nil {
        t.Fatal("expected character sheet, got nil")
    }

    select {
    case <-done:
    case <-time.After(100 * time.Millisecond):
        t.Error("expected UpdateStatusBars to be called within 100ms")
    }
})
```

Add `"time"` to the import block in `get_character_sheet_test.go` if not already present.

- [ ] **Step 2: Run to confirm it fails**

```bash
go test ./internal/domain/character_sheet/... -run TestGetCharacterSheet/triggers_async -v
```

Expected: FAIL — `UpdateStatusBars` is never called because the goroutine dispatch does not exist yet.

- [ ] **Step 3: Add `persistNormalizedStatus` method**

In `get_character_sheet.go`, add this method to `GetCharacterSheetUC` (after `hydrateCharacterSheet`):

```go
func (uc *GetCharacterSheetUC) persistNormalizedStatus(
	ctx context.Context,
	sheetUUID uuid.UUID,
	charSheet *domainSheet.CharacterSheet,
) {
	allBars := charSheet.GetAllStatusBar()
	toBar := func(name enum.StatusName) model.StatusBar {
		bar := allBars[name]
		return model.StatusBar{Min: bar.GetMin(), Curr: bar.GetCurrent(), Max: bar.GetMax()}
	}
	if err := uc.repo.UpdateStatusBars(ctx, sheetUUID.String(),
		toBar(enum.Health), toBar(enum.Stamina), toBar(enum.Aura),
	); err != nil {
		fmt.Printf("TODO(logger): failed to persist normalized status for sheet %s: %v\n", sheetUUID, err)
	}
}
```

- [ ] **Step 4: Wire goroutine dispatch in `hydrateCharacterSheet`**

Replace the temporary `_, err = Wrap(...)` call site (from Task 2) with:

```go
wasCorrected, err := Wrap(characterSheet, modelSheet)
if err != nil {
    return nil, err
}
if wasCorrected {
    go uc.persistNormalizedStatus(context.Background(), modelSheet.UUID, characterSheet)
}
```

- [ ] **Step 5: Run all character sheet tests**

```bash
go test ./internal/domain/character_sheet/... -v
```

Expected: all tests PASS, including the new async persist test.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/character_sheet/get_character_sheet.go \
        internal/domain/character_sheet/get_character_sheet_test.go
git commit -m "$(cat <<'EOF'
feat(domain): persist normalized status bars asynchronously after detection

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Remove debug `fmt.Println` statements

**Files:**
- Modify: `internal/domain/entity/character_sheet/status/status_bar.go`
- Modify: `internal/domain/entity/character_sheet/status/status_manager.go`
- Modify: `internal/domain/character_sheet/get_character_sheet.go`

- [ ] **Step 1: Remove debug prints from `status_bar.go`**

In `internal/domain/entity/character_sheet/status/status_bar.go`, remove lines 40–44 (the five `fmt.Println` inside `SetCurrent`) and the `"fmt"` import:

```go
func (b *Bar) SetCurrent(value int) error {
	if value < b.min || value > b.max {
		return ErrInvalidValue
	}
	b.curr = value
	return nil
}
```

- [ ] **Step 2: Remove debug prints from `status_manager.go`**

In `internal/domain/entity/character_sheet/status/status_manager.go`, remove the `fmt.Println("status", name)` line (line 43) from `SetCurrent` and the `"fmt"` import if it becomes unused:

```go
func (sm *Manager) SetCurrent(name enum.StatusName, value int) error {
	status, err := sm.Get(name)
	if err != nil {
		return err
	}
	if err := status.SetCurrent(value); err != nil {
		return err
	}
	return nil
}
```

- [ ] **Step 3: Remove remaining debug prints from `Wrap()` in `get_character_sheet.go`**

Scan `Wrap()` for any remaining `fmt.Println` calls unrelated to normalization (e.g., `fmt.Println("ref retornada, mas não validada")`, `fmt.Println("peguei a referência...")`, `fmt.Println("terminei o loop...")`) and remove them. Keep only the `fmt.Printf` calls added by `normalizeStatus` and `persistNormalizedStatus`.

- [ ] **Step 4: Verify unused imports are removed**

```bash
go build ./internal/...
```

Expected: compiles without errors or unused-import warnings.

- [ ] **Step 5: Run all tests**

```bash
go test ./internal/domain/... -v
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/entity/character_sheet/status/status_bar.go \
        internal/domain/entity/character_sheet/status/status_manager.go \
        internal/domain/character_sheet/get_character_sheet.go
git commit -m "$(cat <<'EOF'
chore(cleanup): remove debug fmt.Println from status and hydration code

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review Checklist

- [x] **Spec coverage:** normalizeStatus ✓, proportional correction ✓, log with TODO ✓, async goroutine ✓, min/curr/max updated ✓, `persistNormalizedStatus` named without "Async" ✓, edge cases (oldMax=0, newMax=0, curr=0) ✓
- [x] **No placeholders:** all steps include exact code, commands, and expected output
- [x] **Type consistency:** `model.StatusBar` used throughout; `enum.StatusName` (Health/Stamina/Aura) consistent across all tasks; `GetMaxOfStatus`/`GetMinOfStatus`/`GetAllStatusBar` match character_sheet.go:253-393
- [x] **Column names:** `health_min_pts`, `health_curr_pts`, `health_max_pts` confirmed from `create_character_sheet.go`
- [x] **Mock field naming:** `UpdateStatusBarsFn` follows the existing pattern in `mock_character_sheet_repo.go`
