# HxH RPG System — Agent Guide

## Project Overview

Go 1.23 backend for a Hunter × Hunter tabletop RPG. Module: `github.com/422UR4H/HxH_RPG_System`. PostgreSQL (goose migrations). Entry points: REST API (`cmd/api/`) and WebSocket game server (`cmd/game/`).

## Architecture

```
internal/
├── app/       ← HTTP/WS handlers (api/ + game/)
├── domain/    ← Use cases + domain services (engines)
│   └── entity/ ← Pure domain model (no I/O)
├── gateway/   ← PostgreSQL repositories (pg/)
└── config/    ← Configuration loading
```

Dependency: entity ← domain ← app, entity ← gateway. Entities never import outer layers.

## Code Conventions

- **NEVER remove TODO comments** — intentional markers by the owner
- Go idiomatic: implicit interfaces, short var names, error wrapping `%w`
- **User vs Player vs Master:** `User` = generic auth entity. Use `Player`/`Master` for role-specific contexts.
- Engines = domain services under `internal/domain/` correlating entities
- XP cascade: skill → attribute → ability → character (`CascadeUpgrade`/`CascadeUpgradeTrigger`)
- DDD-lite: value objects, entities, domain services, use cases, repository interfaces

## Testing

- Standard `testing` only, no frameworks. Table-driven with `t.Run()`.
- External test packages: `package foo_test`
- Mocks: `mocks_test.go` per handler package (Go idiomatic)
- Create documentation alongside tests during all development work
- **Every feature must have integration tests** (not just unit tests)
- TDD strategy per layer: see `integration-tests.instructions.md` (loaded for `internal/**`)

## Git Workflow

- **Always PRs** — never merge directly to `main`
- Branch: `feat/`, `fix/`, `docs/`, `refactor/`
- Commits: include `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`
- Specs EN + PT-BR versions in same commit

## Agent Model Strategy

| Role | Model |
|------|-------|
| Orchestration/Planning | Opus (main) |
| Code Implementation | Sonnet 4.6 (always override) |
| Code Review/Critique | Sonnet 4.6+ (auto) |
| Exploration/Commands | Haiku 4.5 |

Never accept Haiku default for code-writing sub-agents. When in doubt, prefer Sonnet.

## Commands

**Prefer CI over local runs** (saves tokens). Local only for TDD iteration or debugging.

```bash
# CI (default):
rtk gh run list --workflow=ci.yml --limit=1   # check status
rtk gh run view <run-id> --log-failed         # failure logs

# Local (when needed):
go test ./...                                         # all tests
go test -tags=integration ./internal/gateway/pg/...   # integration tests
make build / make run-dev
make migrate-up / migrate-down / migrate-create name=X
```

## Scoped Instructions

Context-specific content lives in `.github/instructions/` (loaded only when relevant):
- `domain-map.instructions.md` — entity paths and current state (when working on `internal/`)
- `docs-workflow.instructions.md` — documentation maintenance rules (when working on `docs/`)
- `glossary.instructions.md` — EN↔PT-BR terminology (when working on `docs/game/`)
- `integration-tests.instructions.md` — test patterns, helpers, DB setup (when working on `internal/gateway/pg/`)
- `gateway-conventions.instructions.md` — SQL/repository patterns (when working on `internal/gateway/`)

## Known Issues

- `match/` Turn/Round test broken — semantic refactoring in progress
