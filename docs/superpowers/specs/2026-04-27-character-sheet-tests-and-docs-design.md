# Design: Character Sheet Tests & Game Documentation

**Date:** 2026-04-27
**Status:** Approved
**Scope:** Add comprehensive test coverage for the character_sheet domain entities + create scalable game documentation

## Problem Statement

The HxH RPG System backend has a rich domain model for character sheets with ~50+ entity types, a sophisticated experience cascade system, a Nen spiritual system, and combat mechanics. Currently there is only 1 test file (turn engine, currently broken). Before any refactoring can happen safely (experience flow simplification, engines → domain services, OOP → Go-idiomatic patterns), we need a comprehensive test safety net.

Additionally, the RPG game rules exist only implicitly in the code — they need to be documented for players (in Portuguese) and for AI agents navigating the codebase.

## Approach: Tests + Docs Together (Approach A)

Write tests bottom-up for each sub-package, documenting the corresponding game rules as each package is understood through testing. This maximizes context efficiency (write docs while the code is fresh in context) and ensures every package exits "complete" (tested + documented).

## Documentation Structure

```
docs/
├── game/                              # Game rules (PT-BR, for players + AI)
│   ├── glossario.md                   # Glossary: PT-BR ↔ EN keyword mapping
│   ├── ficha-de-personagem/           # Character sheet rules
│   │   ├── habilidades.md             # Abilities
│   │   ├── atributos.md               # Attributes
│   │   ├── pericias.md                # Skills
│   │   ├── proficiencias.md           # Proficiencies
│   │   ├── sistema-nen.md             # Nen: Principles, Categories, Hexagon
│   │   ├── experiencia.md             # XP system and evolution cascade
│   │   └── status.md                  # Status bars (HP, Stamina, Aura)
│   └── classes/                       # One doc per class (future)
├── architecture/                      # Technical docs (EN, for devs + AI)
│   └── overview.md                    # Architecture overview
AGENTS.md                             # Root: AI agent briefing document
```

### Documentation Conventions

- Game docs in Portuguese with English terms in parentheses: "Vitalidade (Vitality)"
- First occurrence in each document: full bilingual form
- Subsequent occurrences: Portuguese only
- All game keywords (abilities, skills, attributes, proficiencies, categories, classes, principles, status, weapons, turn modes) follow this convention
- Glossary is the single source of truth for translations
- User may suggest alternative translations; glossary and all references must be updated accordingly

## Testing Strategy

### Principles

- **Standard library only**: `testing` package, no external test frameworks
- **Table-driven tests**: Idiomatic Go convention using `[]struct` + `t.Run()`
- **Manual mocks**: When needed, hand-written in Go (no mock frameworks)
- **Test helpers (fixtures)**: Factory functions that build pre-configured test objects
- **Mixed strategy**: Bottom-up unit tests first, then cascade/integration tests using real composed objects (no mocks for cascade)

### Test File Location

Each `_test.go` lives next to its source file (Go convention):
```
internal/domain/entity/character_sheet/<package>/<file>_test.go
```

### Bottom-Up Order (following dependency graph)

1. **`experience/`** — ExpTable → Exp → CharacterExp
   - Foundation of the entire progression system
   - Tests: level-up thresholds, XP accumulation, cascade triggers

2. **`ability/`** — Talent → Ability → Manager
   - Depends on: experience
   - Tests: bonus calculation, cascade upgrade propagation, character points

3. **`status/`** — Bar → HealthBar/StaminaBar/AuraBar → Manager
   - Depends on: experience (for upgrade mechanics)
   - Tests: bar increase/decrease, boundaries (min/max), upgrade on level-up

4. **`attribute/`** — PrimaryAttribute → MiddleAttribute → Manager → CharacterAttributes
   - Depends on: experience, ability
   - Tests: physical/mental/spiritual separation, distributable points, power calculation

5. **`skill/`** — CommonSkill → SpecialSkill → JointSkill → Manager → CharacterSkills
   - Depends on: attribute, ability, experience
   - Tests: value-for-test calculation, joint skill composition, cascade triggers

6. **`proficiency/`** — Proficiency → JointProficiency → Manager
   - Depends on: skill, ability, experience
   - Tests: weapon-specific values, joint proficiency buffs, cascade triggers

7. **`spiritual/`** — NenPrinciple → NenCategory → NenHexagon → Manager
   - Depends on: attribute, experience
   - Tests: Nen principle levels, category percentages, hexagon value distribution

8. **`sheet/`** — CharacterProfile → CharacterSheet (aggregate root)
   - Depends on: ALL of the above
   - Tests: profile validation, aggregate cascade tests (XP in skill → ability level up → character level up)

### Cascade Tests (at sheet level)

After all sub-packages are unit-tested, write integration-style tests at the CharacterSheet level that exercise the full experience cascade:
- Skill XP → Ability level up → Character level up
- Attribute distribution → Skill value changes
- Nen hexagon changes → Category percentage effects
- Status bar upgrades on level-up

These use real composed objects (no mocks) to validate the system works end-to-end.

## AGENTS.md Structure

Compact document (~200-400 lines) at project root:
1. Project overview — What it is, HxH RPG context
2. Architecture — Layers (entity → usecase → app → gateway), package conventions
3. Domain map — Where to find each concept
4. Code conventions — Go idiomatic, engines as domain services, implicit interfaces, no test frameworks
5. Quick glossary — Reference to `docs/game/glossario.md` + top 20 inline terms
6. Current state — What's stable, what's WIP (Turn/Round), what needs refactoring
7. How to test — `go test ./...`, table-driven conventions
8. How to build — `make build`, `make run-dev`

## Workflow Per Sub-Package

```
1. Read the sub-package code
2. Write tests (table-driven, reusable helpers)
3. Run tests → ensure they pass (covering CURRENT behavior)
4. Document the discovered rules in docs/game/
5. Update glossary if new terms appear
6. Move to next sub-package
```

## Scope Boundaries

### In scope
- Tests for all 8 character_sheet sub-packages
- Cascade tests at CharacterSheet aggregate level
- Game rules documentation (docs/game/)
- Glossary (docs/game/glossario.md)
- Architecture overview (docs/architecture/overview.md)
- AGENTS.md

### Out of scope
- ❌ Code refactoring of any kind
- ❌ Turn/Round/Action packages (currently broken/WIP)
- ❌ Usecase layer tests
- ❌ Gateway/repository layer tests
- ❌ App/API layer tests
- ❌ Third-party test packages
- ❌ Character class entity tests (not part of character_sheet sub-packages)
- ❌ Die, Item, Match entity tests

## Success Criteria

1. All 8 character_sheet sub-packages have comprehensive test files
2. `go test ./internal/domain/entity/character_sheet/...` passes with 0 failures
3. Cascade tests at sheet level validate end-to-end XP propagation
4. Glossary covers all game keywords with PT-BR ↔ EN mapping
5. Each sub-package has corresponding game rules documentation
6. AGENTS.md provides complete project context for AI agents
7. Architecture overview documents the layer structure
