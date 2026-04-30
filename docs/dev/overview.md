# Architecture Overview

## Layer Diagram

```
┌─────────────────────────────────────────────────┐
│                  cmd/ (entry points)            │
│         api/main.go    game/main.go             │
└────────────────────┬────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────┐
│               internal/app/ (presentation)      │
│         api/ (REST handlers)                    │
│         game/ (WebSocket handlers)              │
└────────────────────┬────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────┐
│            internal/domain/ (use cases)         │
│      character_sheet/  match/  campaign/ ...    │
│      auth/  session/  enrollment/  scenario/    │
│                                                 │
│      "Engines" = domain services that           │
│      orchestrate entity interactions            │
└────────┬───────────────────────────┬────────────┘
         │                           │
┌────────▼────────────┐   ┌─────────▼────────────┐
│  internal/domain/   │   │   internal/gateway/   │
│    entity/ (model)  │   │     pg/ (repos)       │
│  Pure value objects │   │  PostgreSQL via pgx   │
│  & entities, no I/O │   │                       │
└─────────────────────┘   └───────────────────────┘
```

## Package Dependency Rules

1. `entity/` imports nothing from the project (only stdlib + enums)
2. `domain/` (use cases) imports `entity/` for domain types
3. `gateway/` imports `entity/` for repository interfaces
4. `app/` imports `domain/` (use cases) — never directly accesses entities or gateway
5. `cmd/` wires everything together (dependency injection)

## Experience Cascade Pattern

The core mechanic is XP flowing upward through the entity hierarchy:

```
Skill.CascadeUpgradeTrigger(values)
  → Attribute.CascadeUpgrade(values)
    → Ability.CascadeUpgrade(values)
      → CharacterExp.CascadeUpgrade(values)
```

Each layer:
1. Increases its own exp (`exp.IncreasePoints(values.GetExp())`)
2. Calls the next layer's `CascadeUpgrade`
3. Records its state in the `UpgradeCascade` struct (level, exp, test values)

The `UpgradeCascade` struct acts as a collector, accumulating cascade data for all entities touched during an XP grant. This data is returned to the caller for display/logging.

### Status Bar Upgrade

After any XP cascade, `status.Manager.Upgrade()` is called to recalculate HP/SP/AP based on new levels:

```
HP = HP_BASE(20) + int(float64(vitality.GetLevel() + resistance.GetValue()) × physicals.GetBonus())
SP = SP_COEF(10) × int(float64(energy.GetLevel() + resistance.GetValue()) × physicals.GetBonus())
AP = int(AP_COEF(10) × float64(mop.GetLevel() + conscience.GetLevel()) × float64(int(spirituals.GetBonus())))
```

## Engine / Domain Service Pattern

"Engines" in `internal/domain/` orchestrate interactions between entities:

- `internal/domain/character_sheet/` — Character sheet use cases (create, level up, distribute points)
- `internal/domain/match/` — Combat/match orchestration (turn/round flow)
- `internal/domain/campaign/` — Campaign management
- `internal/domain/scenario/` — Scenario lifecycle

These are evolving toward formal Domain Services with clearer naming.

## Key Design Decisions

### Interfaces
- Go implicit interfaces: defined where consumed, not where implemented
- `ICascadeUpgrade` and `ITriggerCascadeExp` are the cascade entry points
- `IGameAttribute`, `IDistributableAttribute`, `ISkill`, `IProficiency` define access patterns

### Value Semantics
- Most entities are value types stored in maps
- Known bug: map[key]Struct returns a copy — mutations on that copy don't persist
- Future fix: use pointer maps (`map[key]*Struct`) for mutable entities

### Factory Pattern
- `CharacterSheetFactory` builds the full entity graph from config coefficients
- Constructs all attributes, skills, abilities, principles, status bars
- Applies character class bonuses via `Wrap()`

## Entry Points

### REST API (`cmd/api/main.go`)
- Character CRUD
- Campaign management
- User authentication

### WebSocket Game Server (`cmd/game/main.go`)
- Real-time match/combat flow via WebSocket
- Hub/Room/Client architecture (one room per match)
- Lobby → Playing → Closed state machine
- Turn/Round orchestration (future, over same WS infra)
- Action/Reaction dispatching (future)

### Match Runtime Concepts

A running match organizes into:
- **Scenes** — categorized as `roleplay` or `battle` (for history/readability only)
- **Turns** — mode `free` (no time pressure) or `race` (speed priority queue)
- **Rounds** — one character's action + triggered reactions

Scene category does NOT determine turn mode. The Turn Engine is agnostic to
scene category by design. See `docs/game/cenas-e-turnos.md` for details.

## Dev Documentation Index

Detailed technical documentation for each subsystem (PT-BR with English code references):

### Character Sheet (`internal/domain/entity/character_sheet/`)

| Document | Covers |
|----------|--------|
| [`experience.md`](character-sheet/experience.md) | ExpTable, CharacterExp, cascade flow, XP coefficients |
| [`abilities-attributes.md`](character-sheet/abilities-attributes.md) | Abilities, PrimaryAttribute, MiddleAttribute, SpiritualAttribute, Talent |
| [`skills-proficiencies.md`](character-sheet/skills-proficiencies.md) | CommonSkill, JointSkill, Proficiency, JointProficiency, Managers |
| [`spiritual.md`](character-sheet/spiritual.md) | NenPrinciple, NenCategory, Hexagon, Hatsu, PrinciplesManager |
| [`status.md`](character-sheet/status.md) | StatusBars (HP/SP/AP), formulas, upgrade mechanics |
| [`factory.md`](character-sheet/factory.md) | CharacterSheetFactory, class application, validation |

### Items & Dice (`internal/domain/entity/`)

| Document | Covers |
|----------|--------|
| [`weapons-dice.md`](weapons-dice.md) | Weapon properties, dice system, damage calculation |

### Auth & Sessions

| Document | Covers |
|----------|--------|
| [`auth.md`](auth.md) | User registration, login, JWT, session management |

### Campaigns & Scenarios (`internal/domain/entity/campaign/`, `scenario/`)

| Document | Covers |
|----------|--------|
| [`campaigns-scenarios.md`](campaigns-scenarios.md) | Campaign lifecycle, Scenario, Submission flow |
| [`enrollment.md`](enrollment.md) | Match enrollment, character sheet validation |

### Match Runtime (`internal/domain/entity/match/`)

| Document | Covers |
|----------|--------|
| [`match/scenes.md`](match/scenes.md) | Scene entity, categories (roleplay/battle), lifecycle |
| [`match/turns-rounds.md`](match/turns-rounds.md) | Turn/Round engines, free vs race modes, semantic refactoring |
| [`match/actions.md`](match/actions.md) | Action structure, PriorityQueue, Attack, RollContext |

### WebSocket Game Server (`internal/app/game/`)

| Document | Covers |
|----------|--------|
| [`websocket.md`](websocket.md) | Hub/Room/Client architecture, message protocol, state machine |
