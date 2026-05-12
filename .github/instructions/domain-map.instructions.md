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
| Match Bounded Context | `internal/domain/match/` |
| Match Entities | `internal/domain/match/entity/` |
| Match Domain Services | `internal/domain/match/service/` |
| Campaign | `internal/domain/entity/campaign/` |
| Scenario | `internal/domain/entity/scenario/` |
| User | `internal/domain/entity/user/` |
| Enums | `internal/domain/entity/enum/` |
| Dice | `internal/domain/entity/die/` |
| Items | `internal/domain/entity/item/` |

## Current State

- ✅ `character_sheet/` — Stable, fully tested
- ✅ `domain/match/` — Bounded context: entities + 3 domain services (Phase 1 complete)
- ⏳ `domain/match/matchsession/` — Pending Phase 2
- ✅ `gateway/` — PostgreSQL repositories (fully implemented)
- ✅ `app/api/` — HTTP handlers (unit tested with humatest)
- ✅ `app/game/` — WebSocket game server (Hub/Room/Client pattern)
- ✅ `application/` — Use cases migrated from domain/ (all features)
