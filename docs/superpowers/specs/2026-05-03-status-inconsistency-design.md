# Status Inconsistency Design

**Date:** 2026-05-03
**Branch:** feat/match-enrollments-listing
**Scope:** `internal/domain/entity/character_sheet/status/`, `internal/domain/character_sheet/`

## Problem

Character sheets are built at runtime from base data persisted in the database. Status bars (Health, Stamina, Aura) have a `max` calculated from skills, attributes, and abilities — and a `curr` that is both calculated and persisted.

When game rules change, the formula for `max` can decrease below a previously valid `curr` in the database. This causes `Bar.SetCurrent()` to return `ErrInvalidValue` during hydration, crashing `GET /character-sheets/{uuid}`.

## Decisions

| Question | Decision |
|---|---|
| Behavior when `curr_db > new_max` | Proportional correction: `curr_new = round(new_max * curr_db / old_max_db)` |
| Persistence | Async goroutine — user does not wait, DB updated in background |
| What to persist | `min`, `curr`, and `max` for all 3 status bars |
| Logging | `fmt.Printf` with the four values; `// TODO: replace with structured logger` |

## Architecture

The fix touches two layers only. No new routes, no external libraries, no changes to entity invariants.

```
GET /character-sheets/{uuid}
  → hydrateCharacterSheet()
      → factory.Build()              # builds sheet with max from current rules
      → Wrap()                       # applies persisted data to built sheet
          → normalizeStatus()        # per status: detect and correct inconsistency
          → returns (wasCorrected bool, err error)
      → if wasCorrected:
          go persistNormalizedStatus(context.Background(), sheetUUID, sheet)
      → returns sheet immediately
```

## Components

### `normalizeStatus(curr, oldMax, newMax, minVal int) (int, bool)`

Private function in the `charactersheet` use case package. Pure — no side effects.

- If `curr <= newMax`: returns `(curr, false)` — no correction needed
- If `newMax == 0`: returns `(curr, false)` with anomaly log — avoids division by zero
- If `oldMax > 0`: `corrected = clamp(round(newMax * curr / oldMax), minVal, newMax)`
- If `oldMax == 0`: fallback — `corrected = newMax` (character treated as full)
- Logs the correction: `fmt.Printf("TODO(logger): status normalized: curr %d → %d (old_max: %d, new_max: %d)\n", ...)`
- Returns `(corrected, true)`

### `Wrap()` — signature change

```go
// before
func Wrap(charSheet *CharacterSheet, modelSheet *model.CharacterSheet) error

// after
func Wrap(charSheet *CharacterSheet, modelSheet *model.CharacterSheet) (wasCorrected bool, err error)
```

For each of the 3 status bars:
1. Reads `newMax` and `minVal` from the already-built bar in memory
2. Calls `normalizeStatus(curr_db, max_db, newMax, minVal)`
3. Calls `SetCurrStatus` with the (possibly corrected) value
4. If any status was corrected, sets `wasCorrected = true`

`Bar.SetCurrent()` remains strict — validates `[min, max]` for gameplay operations. Normalization happens before calling it.

### `persistNormalizedStatus(ctx context.Context, sheetUUID string, charSheet *CharacterSheet)`

Private method on `GetCharacterSheetUC`. Reads current `min/curr/max` from the built sheet for all 3 status bars and calls `repo.UpdateStatusBars()`. Logs errors but does not propagate — the GET has already returned.

```go
if wasCorrected {
    go uc.persistNormalizedStatus(context.Background(), sheetUUID, characterSheet)
}
```

The `go` keyword at the call site communicates the async nature. The function name does not include "Async" — idiomatic Go.

### `IRepository.UpdateStatusBars(ctx, sheetUUID, health, stamina, aura model.StatusBar) error`

New method on the character sheet repository interface and its PostgreSQL implementation. Updates the 6 fields (`min/curr/max × 3`) in one query.

## Edge Cases

| Case | Behavior |
|---|---|
| `old_max_db == 0` | Fallback: `corrected = new_max` (character treated as full) |
| `curr_db == 0` | No correction needed regardless of max values |
| `new_max == 0` | No correction; logs anomaly to avoid division by zero |
| `persistNormalizedStatus` fails | Logs error, does not propagate — GET already responded correctly |
| GET called again before persist completes | Normalization re-applied in memory, second goroutine attempts update — idempotent |
| Rule changes again before persist completes | Goroutine persists values from this correction; next GET detects and corrects again |

## What Does Not Change

- `Bar.SetCurrent()` — remains strict, validates `[min, max]`
- `IStatusBar`, `Manager`, `HealthPoints`, `StaminaPoints`, `AuraPoints` — no changes
- No new routes, no external libraries
- `Bar.min` is `0` for all status bars until the min TODO is implemented; `normalizeStatus` receives `minVal` from the bar so it will work correctly once min is defined

## Future Directions

- Replace `fmt.Printf` with a structured logger when logging infrastructure is added
- Consider storing `loss` (damage taken / stamina spent) instead of absolute `curr` — this would eliminate the inconsistency class entirely for future rule changes (schema migration required)
- The `old_max_db` used for the ratio reflects the max at the time of the last DB write (gameplay event or normalization). Multiple sequential rule changes between writes may accumulate small ratio drift — acceptable for now
