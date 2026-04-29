# Phase 7 — Remaining Unit Tests Design

**Date:** 2026-04-30
**Status:** Implemented
**Scope:** All remaining untestable packages with testable logic

## Objective

Complete unit test coverage for every package that contains testable logic but
had no tests. This is the final unit-test phase — after this, only the
pre-existing `turn/engine_test.go` failure remains (deferred: semantic
Turn/Round refactoring).

## Packages Tested

| Package | Tests | Description |
|---------|-------|-------------|
| `pkg/auth` | 6 | JWT GenerateToken + ValidateToken (security-critical) |
| `internal/domain` | 4 | Error wrapper helpers (NewValidationError, NewDomainError, NewDBError) |
| `internal/domain/entity/campaign` | 2 | NewCampaign constructor + optional fields |
| `internal/domain/entity/scenario` | 2 | NewScenario constructor + UUID uniqueness |
| `internal/domain/entity/match/scene` | 7 | Scene lifecycle: create, AddTurn, FinishScene, error paths |
| `internal/config` | 3 | LoadCORS env parsing with defaults and custom values |
| `pkg` | 3 | Config.ConnString() with various SSL modes |

**Total: 27 test cases**

## Packages Skipped (with rationale)

| Package | Reason |
|---------|--------|
| `match/turn` | Pre-existing broken test — deferred by user (semantic refactoring WIP) |
| `match/round` | Tightly coupled to Turn refactoring — deferred |
| `match/battle` | Struct-only, no testable logic |
| `entity/user` | Struct + error constants only |
| `domain/session` | Interface definition only |
| `domain/testutil` | Test helper utilities |
| `gateway/pg/*` | Integration tests exist (Phase 5, `integration` build tag) |
| `cmd/*` | Entry points (main.go) — not unit tested |

## Test Patterns

- **Standard library `testing` only** — no frameworks
- **Table-driven tests** with `t.Run()` where multiple cases exist
- **External test packages** (`package X_test`)
- **`t.Setenv()`** for environment variable tests (auto-cleanup)
- **Zero-value struct construction** for `turn.Turn` in scene tests (avoids broken Turn logic)

## Coverage Summary

After this phase, every Go package with testable production logic has at least
basic test coverage, with the sole exception of the Turn/Round area that is
under active semantic refactoring.
