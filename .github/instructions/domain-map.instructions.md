---
applyTo: "internal/**"
---

# Domain Map

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

## Current State

- ✅ `character_sheet/` — Stable, fully tested
- ⚠️ `match/` — Turn/Round system WIP (semantic refactoring, broken test)
- ✅ `gateway/` — PostgreSQL repositories (fully implemented)
- ✅ `app/api/` — HTTP handlers (unit tested with humatest)
- ✅ `app/game/` — WebSocket game server (Hub/Room/Client pattern)
