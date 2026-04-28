# Phase 4: CharacterSheet Use Case Tests — Design Spec

**Date:** 2026-04-29  
**Scope:** Unit tests for all 4 CharacterSheet domain use cases  
**Branch:** `feat/character-sheet-uc-tests`

## Problem

The CharacterSheet domain package has 4 use cases without test coverage:
1. `CreateCharacterSheetUC` — complex entity construction with multi-step validation
2. `GetCharacterSheetUC` — permission checks + model-to-entity hydration
3. `ListCharacterSheetsUC` — simple delegation to repository
4. `UpdateNenHexagonValueUC` — method dispatch + persistence

These were deferred from Phase 3 due to their complexity (200+ lines for Create, 130+ for Get).

## Approach

### Test Strategy

Use **real domain objects** (factory, character classes) combined with **mock repositories**:

- `CharacterSheetFactory` is a pure function (no I/O) — use it directly
- `CharacterClass` instances are built from the existing factory (e.g., `BuildSwordsman()`)
- Repository interactions are mocked via existing `testutil.MockCharacterSheetRepo` and `testutil.MockCampaignRepo`
- `sync.Map` is populated in test setup with real class data

### Test Helper

A shared helper in `internal/domain/character_sheet/testutil_test.go` (external test package) providing:
- `newTestFactory()` — returns a `*sheet.CharacterSheetFactory`
- `newTestClassMap()` — returns a `*sync.Map` pre-loaded with Swordsman class
- `newValidInput()` — returns a `*CreateCharacterSheetInput` that passes all validation
- `newValidModelSheet()` — returns a `*model.CharacterSheet` for GetCharacterSheet hydration

## Test Cases

### CreateCharacterSheetUC (13 cases)

| # | Case | Expected |
|---|------|----------|
| 1 | Happy path (player, no campaign) | Returns `*CharacterSheet`, no error |
| 2 | Happy path (master, no campaign) | Returns `*CharacterSheet`, no error |
| 3 | Happy path (player, with campaign) | Returns `*CharacterSheet`, no error |
| 4 | Class not found | `ErrCharacterClassNotFound` (wrapped) |
| 5 | Invalid skills distribution | CharacterClass validation error |
| 6 | Invalid proficiencies distribution | CharacterClass validation error |
| 7 | Nickname matches class name | `ErrNicknameNotAllowed` (wrapped) |
| 8 | Player limit exceeded (>=20) | `ErrMaxCharacterSheetsLimit` |
| 9 | CountCharacters repo error | Propagated error |
| 10 | Campaign not found | `domainCampaign.ErrCampaignNotFound` |
| 11 | Campaign repo error | Propagated error |
| 12 | Not campaign owner | `domainCampaign.ErrNotCampaignOwner` |
| 13 | Create repo error | Propagated error |

### GetCharacterSheetUC (9 cases)

| # | Case | Expected |
|---|------|----------|
| 1 | Happy path — user is master | Returns hydrated `*CharacterSheet` |
| 2 | Happy path — user is player | Returns hydrated `*CharacterSheet` |
| 3 | Happy path — user is campaign master | Returns hydrated `*CharacterSheet` |
| 4 | Sheet not found | `ErrCharacterSheetNotFound` |
| 5 | Get repo error | Propagated error |
| 6 | Insufficient permissions (no campaign) | `auth.ErrInsufficientPermissions` |
| 7 | Insufficient permissions (not campaign master) | `auth.ErrInsufficientPermissions` |
| 8 | Campaign not found during permission check | `domainCampaign.ErrCampaignNotFound` |
| 9 | Campaign repo error during permission check | Propagated error |

### ListCharacterSheetsUC (3 cases)

| # | Case | Expected |
|---|------|----------|
| 1 | Happy path — returns list | `[]model.CharacterSheetSummary`, no error |
| 2 | Happy path — empty list | Empty slice, no error |
| 3 | Repo error | Propagated error |

### UpdateNenHexagonValueUC (5 cases)

| # | Case | Expected |
|---|------|----------|
| 1 | Happy path — increase | Returns `*NenHexagonUpdateResult` |
| 2 | Happy path — decrease | Returns `*NenHexagonUpdateResult` |
| 3 | Invalid method | `ErrInvalidUpdateHexValMethod` |
| 4 | Increase/Decrease entity error | Propagated error |
| 5 | Repo update error | `domain.DBError` wrapping repo error |

**Total: 30 test cases**

## File Structure

```
internal/domain/character_sheet/
├── create_character_sheet_test.go   (13 cases)
├── get_character_sheet_test.go      (9 cases)
├── list_character_sheets_test.go    (3 cases)
├── update_nen_hexagon_value_test.go (5 cases)
└── testutil_test.go                 (shared helpers)
```

## Key Design Decisions

1. **External test package** (`charactersheet_test`) — tests import the package under test as a consumer would
2. **Real factory** — avoids mocking complex entity construction; ensures integration correctness
3. **Swordsman class** — simplest class (no Distribution), so `SkillsExps` and `ProficienciesExps` must be empty to pass validation
4. **Ninja class for distribution tests** — has `Distribution` with specific allowed proficiencies/points, enabling validation error tests
5. **Gateway error imports** — tests import `pgCampaign` and `pgSheet` error packages to trigger the error translation paths in UCs
6. **Table-driven tests** with `t.Run()` sub-tests — consistent with Phase 3 pattern

## Dependencies

- Existing mocks: `testutil.MockCharacterSheetRepo`, `testutil.MockCampaignRepo`
- Real: `sheet.CharacterSheetFactory`, `characterclass.BuildSwordsman()`, `characterclass.BuildNinja()`
- Gateway errors: `pgCampaign.ErrCampaignNotFound`, `pgSheet.ErrCharacterSheetNotFound`
