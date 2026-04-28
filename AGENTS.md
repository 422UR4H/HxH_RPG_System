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

## Code Conventions

- **Go idiomatic:** Implicit interfaces, short variable names in context, error wrapping with `%w`
- **Engines as domain services:** Logic extracted from entities lives in "engine" packages under `internal/domain/`. These correlate entities that are themselves dry (or nearly dry) per Go convention.
- **No test frameworks:** Standard library `testing` only. Table-driven tests with `t.Run()`.
- **External test packages:** Tests use `package foo_test` to test exported API only.
- **Experience cascade pattern:** XP flows upward: skill → attribute → ability → character. Each layer calls `CascadeUpgrade`/`CascadeUpgradeTrigger` on the layer above.
- **DDD-lite:** Value objects, entities, domain services (engines), use cases, repository interfaces.
- **Specs:** Design specs in `docs/superpowers/specs/` must have both EN and PT-BR versions (`.pt-br.md` suffix). Both versions committed together.
- **Game docs:** PT-BR documentation in `docs/game/ficha-de-personagem/`.

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

## Current State

- ✅ `character_sheet/` — Stable, fully tested (experience, ability, attribute, skill, proficiency, spiritual, status, sheet)
- ⚠️ `match/` — Turn/Round system WIP (semantic refactoring in progress, broken test)
- ⚠️ `domain/` services (engines) — Pending rename to domain services pattern
- 🔲 `gateway/` — PostgreSQL repositories (basic structure)
- 🔲 `app/` — HTTP handlers (basic structure)

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

1. **Value-copy bug in `attribute.Manager.IncreasePointsForPrimary()`**: `GetPrimary()` returns a value copy, so `IncreasePoints()` modifies the copy only. Changes don't persist.
2. **Value-copy bug in `spiritual.Manager.IncreaseExpByPrinciple()`**: Same pattern — map value copy means exp doesn't persist.
3. **Inverted nil check in `spiritual.Manager.InitNenHexagon()`** (line 30): checks `nenHexagon != nil` instead of `m.nenHexagon != nil`.
4. **Turn/Round test broken**: Semantic refactoring in progress.
