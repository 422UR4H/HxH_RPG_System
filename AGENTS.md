# HxH RPG System — Agent Guide

## Project Overview

Go backend for a Hunter × Hunter tabletop RPG platform. The system handles all mechanical calculations (experience, levels, stats, combat) so players can focus on roleplay.

- **Language:** Go 1.23
- **Module:** `github.com/422UR4H/HxH_RPG_System`
- **Database:** PostgreSQL (via goose migrations)
- **Entry points:** REST API (`cmd/api/`) and WebSocket game server (`cmd/game/`)

## Architecture (4 Layers)

```
internal/
├── app/          ← HTTP/WS handlers (presentation)
│   ├── api/      ← REST controllers
│   └── game/     ← WebSocket game server
├── domain/       ← Use cases + domain services (engines)
├── entity/       ← Pure domain model (no I/O)
│   └── (inside internal/domain/entity/)
├── gateway/      ← PostgreSQL repositories
│   └── pg/       ← pgx-based implementations
└── config/       ← Configuration loading
```

**Dependency rules:** entity ← domain ← app, entity ← gateway. Entities never import outer layers.

## Domain Map

| Concept | Path |
|---------|------|
| Character Sheet | `internal/domain/entity/character_sheet/` |
| Experience System | `internal/domain/entity/character_sheet/experience/` |
| Abilities | `internal/domain/entity/character_sheet/ability/` |
| Attributes | `internal/domain/entity/character_sheet/attribute/` |
| Skills (Perícias) | `internal/domain/entity/character_sheet/skill/` |
| Proficiencies | `internal/domain/entity/character_sheet/proficiency/` |
| Spiritual/Nen | `internal/domain/entity/character_sheet/spiritual/` |
| Status Bars | `internal/domain/entity/character_sheet/status/` |
| Sheet (integration) | `internal/domain/entity/character_sheet/sheet/` |
| Character Classes | `internal/domain/entity/character_class/` |
| Match/Combat | `internal/domain/entity/match/` |
| Campaign | `internal/domain/entity/campaign/` |
| Scenario | `internal/domain/entity/scenario/` |
| User | `internal/domain/entity/user/` |
| Enums | `internal/domain/entity/enum/` |
| Dice | `internal/domain/entity/die/` |
| Items | `internal/domain/entity/item/` |

## Documentation Structure

| Directory | Purpose | Audience | Language |
|-----------|---------|----------|----------|
| `docs/game/` | Game rules, mechanics, player-facing content (like a RPG rulebook) | Players & Masters | PT-BR |
| `docs/architecture/` | Technical design, entity flows, data models, integration details | Developers | EN |
| `docs/superpowers/specs/` | Feature design specs (formal, timestamped) | Developers | EN + PT-BR |
| `docs/superpowers/plans/` | Implementation plans | Developers | EN |
| `AGENTS.md` | Quick-reference guide for AI agents | AI Agents | EN |

**Key rule:** Game docs (`docs/game/`) must contain ONLY game rules and mechanics — no implementation details, no code references, no software entities. Think of it as content that could be printed in a RPG rulebook. Technical details about how the software implements these rules go in `docs/architecture/`.

## Code Conventions

- **NEVER remove TODO comments:** TODOs in source code are intentional markers written by the owner. They must be preserved in ALL edits, regardless of context.
- **Go idiomatic:** Implicit interfaces, short variable names in context, error wrapping with `%w`
- **Entity naming — User vs Player vs Master:** `User` is the generic identity entity (authentication, account). `Player` and `Master` are specific domain roles. Use the specific name (`player`, `master`) in code unless the context truly applies to both roles equally. Example: `IsPlayerEnrolledInMatch`, not `IsUserEnrolledInMatch`.
- **Engines as domain services:** Logic extracted from entities lives in "engine" packages under `internal/domain/`. These correlate entities that are themselves dry (or nearly dry) per Go convention.
- **No test frameworks:** Standard library `testing` only. Table-driven tests with `t.Run()`.
- **External test packages:** Tests use `package foo_test` to test exported API only.
- **Experience cascade pattern:** XP flows upward: skill → attribute → ability → character. Each layer calls `CascadeUpgrade`/`CascadeUpgradeTrigger` on the layer above.
- **DDD-lite:** Value objects, entities, domain services (engines), use cases, repository interfaces.
- **Specs:** Design specs in `docs/superpowers/specs/` must have both EN and PT-BR versions (`.pt-br.md` suffix). Both versions committed together.
- **Game docs:** PT-BR documentation in `docs/game/` — pure game rules for players (no implementation details). Separate technical docs for developers live in `docs/architecture/`.

## Git Workflow

- **Always use Pull Requests:** Never merge directly to `main`. Work on feature branches, push to origin, create a PR on GitHub, review the diff, then merge.
- **Branch naming:** `feat/<description>`, `fix/<description>`, `docs/<description>`, `refactor/<description>`
- **PR flow:** Create branch → work → push → create PR (with description of changes) → owner reviews diff → merge
- **Commits:** Include `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>` trailer
- **Specs committed together:** EN + PT-BR versions in same commit

## Quick Glossary (EN → PT-BR)

| English | Português |
|---------|-----------|
| Character Sheet | Ficha de Personagem |
| Experience / XP | Experiência / XP |
| Level | Nível |
| Ability | Habilidade (Physicals, Mentals, Spirituals, Skills) |
| Attribute | Atributo |
| Skill (Perícia) | Perícia |
| Proficiency | Proficiência (armas) |
| Status Bar | Barra de Status (HP, SP, AP) |
| Talent | Talento |
| Nen Principle | Princípio Nen |
| Nen Category | Categoria Nen |
| Hexagon | Hexágono |
| Hatsu | Hatsu |
| Buff | Buff (temporário) |
| Cascade | Cascata (fluxo de XP) |
| Turn | Turno |
| Round | Round (ação de um personagem) |
| Action | Ação |
| Reaction | Reação |
| Campaign | Campanha |
| Scenario | Cenário |
| Scene | Cena (roleplay ou battle) |
| Free Turn | Turno Livre (sem disputa de tempo) |
| Race Turn | Turno Disputado (ordem por velocidade) |
| Lobby | Sala de Espera (pré-partida) |
| Room | Sala (instância WS de uma partida) |
| Hub | Hub (gerenciador de rooms) |
| Master | Mestre (quem conduz a partida) |
| Player | Jogador (quem joga um personagem) |
| User | Usuário (entidade genérica de autenticação) |

## Current State

- ✅ `character_sheet/` — Stable, fully tested (experience, ability, attribute, skill, proficiency, spiritual, status, sheet)
- ⚠️ `match/` — Turn/Round system WIP (semantic refactoring in progress, broken test)
- ⚠️ `domain/` services (engines) — Pending rename to domain services pattern
- ✅ `gateway/` — PostgreSQL repositories (fully implemented, integration tested)
- ✅ `app/api/` — HTTP handlers (fully implemented, unit tested with humatest)
- ✅ `app/game/` — WebSocket game server (Hub/Room/Client pattern, unit + integration tested)

## Commands

```bash
# Run all tests
go test ./...

# Run character sheet tests only
go test ./internal/domain/entity/character_sheet/...

# Build
make build

# Run dev server
make run-dev

# Database migrations
make migrate-up
make migrate-down
make migrate-create name=add_users_table
```

## Known Issues / Bugs

1. **Turn/Round test broken**: Semantic refactoring in progress.
