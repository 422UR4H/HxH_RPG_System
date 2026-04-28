# Design: Remaining Domain Entity Tests

**Date:** 2026-04-28
**Status:** Complete
**Scope:** Add comprehensive test coverage for die, item, character_class, enum, and match domain entity packages

## Problem Statement

After completing character_sheet tests (169+ tests across 8 packages), the remaining domain entity packages had zero test coverage. These packages contain critical game mechanics: dice rolling, weapon systems, character class validation, enum parsing, and combat action ordering.

## Approach

Bottom-up testing organized by dependency graph:

- **Phase 1 (Independent):** die → item → character_class → enum (no cross-domain dependencies)
- **Phase 2 (Match domain):** match/action → match (depends on die, enum)

Excluded from scope:
- `match/scene` — imports `turn` package (broken Turn/Round semantic refactoring)
- `match/battle/blow` — struct with no methods
- `match/round` — imports `turn` package

## Testing Strategy

### Principles Applied

- Standard library `testing` only, table-driven with `t.Run()`
- External test packages (`package foo_test`)
- Copy-safety validation (e.g., `Weapon.GetDice()` returns copy)
- Factory completeness checks (all enums built)
- Randomness testing via range validation over multiple iterations
- Heap invariant verification for PriorityQueue

### Coverage Summary

| Package | File | Tests | Key Behaviors |
|---------|------|-------|---------------|
| die | die_test.go | 3 | Construction, Roll range [1,N], state tracking |
| item | weapon_test.go | 5 | Getters, copy safety, penalty/stamina logic |
| item | weapons_manager_test.go | 5 | CRUD, delegates, error propagation |
| item | weapons_factory_test.go | 2 | Builds all 40 weapons, property verification |
| character_class | character_class_test.go | 7 | Validate/Apply skills & proficiencies, Distribution |
| character_class | character_class_factory_test.go | 3 | Builds all 12 classes, skills, distribution |
| enum | enum_test.go | 7 | NameFrom parsers, DieSides, collection lengths |
| match/action | priority_queue_test.go | 6 | Max-heap ordering, Insert, Extract, Peek, ExtractByID |
| match/action | action_test.go | 4 | Construction, UUID uniqueness, RollContext |
| match | match_test.go | 2 | NewMatch fields, AddScene/GetScenes |
| match | game_event_test.go | 1 (3 sub) | Categories, nil defaults, date change |

**Total: 45 test functions across 11 files**

## Key Discoveries

1. **RollContext.GetDiceResult** has an unused parameter `d die.Die` — it sums all dice in the context regardless of the parameter
2. **CharacterClassFactory** builds 12 classes, but enum returns 16 names (4 are commented out: Athlete, Tribal, Experiment, Circus)
3. **Weapon.GetDice()** returns copy via `copy()` — consistent with owner's preference for safe returns
4. **PriorityQueue** inverts `Less()` to create max-heap — higher speed = higher priority
5. **GameEvent** has all unexported fields with no getters — tested via construction verification

## Documentation Created

Game documentation (PT-BR):
- `docs/game/dados.md` — Dice mechanics
- `docs/game/armas.md` — Weapon system (all 40 weapons)
- `docs/game/classes.md` — Character class system
- `docs/game/combate/acoes.md` — Combat actions, priority queue, match/events

## Commits

1. `ef0aded` — test(die): Die tests
2. `69a3f88` — test(item): Weapon, WeaponsManager, Factory tests
3. `ef603de` — test(character_class): CharacterClass, Factory tests
4. `a36df75` — test(enum): Enum parser and collection tests
5. `55c4ef5` — test(match): Action, PriorityQueue, Match, GameEvent tests
