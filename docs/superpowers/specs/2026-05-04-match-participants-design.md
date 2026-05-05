# Match Participants — Design Spec

**Date:** 2026-05-04  
**Branch:** `refactor/layer-isolation-pg-model`  
**Status:** Approved

## Context

The match detail page (front-end, out of scope) needs the `CharacterSheet` summaries of participants. This requires an n:m relationship table between matches and character sheets, populated when the match starts and updated as the match progresses.

The existing `enrollments` table is the pre-match approval flow. `match_participants` is the in-match participation record — a different domain concept with its own lifecycle.

**Future note (out of scope):** `match_participants` will also serve as the seed for loading full `CharacterSheet` objects into a `MatchSession` in-memory struct (`app/game/`) when the WS game layer is implemented.

---

## Decisions

| Topic | Decision |
|-------|----------|
| Endpoint | Separate `GET /matches/{uuid}/participants` — not embedded in `GetMatch` |
| Visibility | Same `ViewerIsMaster` pattern as `list_match_enrollments` |
| Participation tracking | Three timestamps: `joined_at`, `left_at`, `died_at` |
| StartMatch integration | Approach A — atomic gateway op inside `StartMatchUC` |
| `Participant` entity | `domain/entity/match/participant.go` — same package, no new package |
| Gateway | New files in existing `pg/match/` package — same `Repository` |
| `CharacterSheetWithVisibilityResponse` | Move from `api/match/` → `api/sheet/` (used by two handlers) |
| `Match` entity | Unchanged — no `CharacterSheets` field |
| `game_start_at` timestamp | Generated in `StartMatchUC` (not in gateway) — passed to both `StartMatch` and `RegisterFromAcceptedEnrollments` |

---

## Schema

```sql
CREATE TABLE match_participants (
    id   SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),

    match_uuid           UUID NOT NULL REFERENCES matches(uuid),
    character_sheet_uuid UUID NOT NULL REFERENCES character_sheets(uuid),

    -- Timestamp-based participation tracking:
    --   joined_at vs match.game_start_at → whether they joined late
    --   left_at IS NULL + match.story_end_at IS NOT NULL → completed normally
    --   death: character_sheets.dead_at is the single source of truth;
    --          a participant with sheet.dead_at != nil died in this match
    --          (dead characters cannot enroll in future matches, so the inference is safe)
    joined_at TIMESTAMP NOT NULL,
    left_at   TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (uuid),
    UNIQUE (match_uuid, character_sheet_uuid)
);
CREATE INDEX idx_match_participants_match_uuid ON match_participants(match_uuid);
```

`UNIQUE (match_uuid, character_sheet_uuid)` — a sheet participates at most once per match. Unlike `enrollments`, no conditional unique needed.

---

## Domain Entity

**`internal/domain/entity/match/participant.go`** — same package as `Match`, `Summary`, `GameEvent`.

```go
package match

import (
    "time"
    csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
    "github.com/google/uuid"
)

type Participant struct {
    UUID      uuid.UUID
    MatchUUID uuid.UUID
    Sheet     csEntity.Summary  // Sheet.DeadAt is the single source of truth for death
    JoinedAt  time.Time
    LeftAt    *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

No derived methods on the entity — `JoinedLate` and `IsActive` are presentation concerns computed by the handler from timestamps.

---

## Use Cases

### `StartMatchUC` — extend

**`IRepository.StartMatch` signature change** (aligns with gateway-conventions — Go owns timestamps):

```go
// internal/domain/match/i_repository.go
StartMatch(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error
```

**New narrow interface in `start_match.go`:**

```go
type IMatchParticipantWriter interface {
    RegisterFromAcceptedEnrollments(
        ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
    ) error
}
```

**Updated `Start()` flow:**

```go
func (uc *StartMatchUC) Start(ctx context.Context, matchUUID, masterUUID uuid.UUID) error {
    // ... existing validations (GetMatch, MasterUUID check, AlreadyStarted, AlreadyFinished) ...

    gameStartAt := time.Now()
    if err := uc.matchRepo.StartMatch(ctx, matchUUID, gameStartAt); err != nil {
        return err
    }
    if err := uc.enrollmentRepo.RejectPendingEnrollments(ctx, matchUUID); err != nil {
        return err
    }
    return uc.participantRepo.RegisterFromAcceptedEnrollments(ctx, matchUUID, gameStartAt)
}
```

### `GetMatchParticipantsUC` — new

**`internal/domain/match/get_match_participants.go`**

Mirrors `ListMatchEnrollmentsUC` exactly in authorization logic.

```go
type IMatchParticipantReader interface {
    ListParticipantsByMatchUUID(
        ctx context.Context, matchUUID uuid.UUID,
    ) ([]*matchEntity.Participant, error)
}

type GetMatchParticipantsResult struct {
    Participants   []*matchEntity.Participant
    ViewerIsMaster bool
}

type IGetMatchParticipants interface {
    Get(ctx context.Context, matchUUID, userUUID uuid.UUID) (*GetMatchParticipantsResult, error)
}
```

Authorization: same as `ListMatchEnrollmentsUC` — fetch match, check `MasterUUID == userUUID`, if private and not master → `ExistsSheetInCampaign`. Uses the already-declared `CampaignParticipationChecker` interface.

---

## Gateway — `pg/match/`

New files inside the existing package (no new package, same `Repository`):

### `pg/match/register_participants.go`

```go
func (r *Repository) RegisterFromAcceptedEnrollments(
    ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
) error {
    now := time.Now()
    // INSERT ... SELECT is atomic in PostgreSQL — no explicit transaction needed.
    const query = `
        INSERT INTO match_participants
            (uuid, match_uuid, character_sheet_uuid, joined_at, created_at, updated_at)
        SELECT gen_random_uuid(), match_uuid, character_sheet_uuid, $2, $3, $3
        FROM enrollments
        WHERE match_uuid = $1 AND status = 'accepted'
        ON CONFLICT (match_uuid, character_sheet_uuid) DO NOTHING
    `
    _, err := r.q.Exec(ctx, query, matchUUID, gameStartAt, now)
    if err != nil {
        return fmt.Errorf("failed to register match participants: %w", err)
    }
    return nil
}
```

`ON CONFLICT DO NOTHING` — idempotent in case of double-call.

### `pg/match/read_participants.go`

`ListParticipantsByMatchUUID` — JOIN `character_sheets`, `character_profiles`, `LEFT JOIN users` (NPCs are master-owned sheets with no `player_uuid`; enrollment gateway uses INNER JOIN and would silently exclude them — participants must use LEFT JOIN).

Scan includes `cs.story_start_at`, `cs.story_current_at`, `cs.dead_at` — fields that `ToBaseSummaryResponse` uses and that the enrollment gateway currently omits. Participant summaries must be complete.

`cs.dead_at` is the single source of truth for death: if a participant's `sheet.DeadAt != nil`, they died in this match. No `died_at` column on `match_participants`.

### `pg/match/start_match.go` — updated

Receives `gameStartAt time.Time` as parameter instead of generating `time.Now()` internally.

---

## API Handler

### `CharacterSheetWithVisibilityResponse` — move

From `internal/app/api/match/list_match_enrollments.go` → `internal/app/api/sheet/` (where all sheet presentation types live). Used by both enrollment and participant handlers.

### New files in `internal/app/api/match/`

- `get_match_participants.go`
- `get_match_participants_test.go`

### Response shape

```go
type ParticipantResponse struct {
    UUID     uuid.UUID                                     `json:"uuid"`
    JoinedAt string                                        `json:"joined_at"`
    LeftAt   *string                                       `json:"left_at,omitempty"`
    Sheet    apiSheet.CharacterSheetWithVisibilityResponse `json:"character_sheet"`
}
```

`JoinedLate` omitted — front-end computes from `joined_at` vs `match.game_start_at` (already in `GET /matches/{uuid}`).
`DiedAt` omitted — death is surfaced via `sheet.character_sheet.dead_at` (already in `CharacterBaseSummaryResponse`).

Handler mirrors `ListMatchEnrollmentsHandler`: same error switch, same `ViewerIsMaster` → `private: null` vs full data.

### Route

`GET /matches/{uuid}/participants` registered in `internal/app/api/match/routes.go`.

### Wiring (`api.go`)

`pg/match.Repository` satisfies both `IMatchParticipantWriter` and `IMatchParticipantReader` via structural typing — one instance injected into both UCs.

---

## Tests

| Layer | Type | File |
|-------|------|------|
| `pg/match/` — register | Integration | `match_integration_test.go` |
| `pg/match/` — read participants | Integration | `match_integration_test.go` |
| `domain/match/StartMatchUC` | Unit (mock) | `start_match_test.go` |
| `domain/match/GetMatchParticipantsUC` | Unit (mock) | new `get_match_participants_test.go` |
| `app/api/match/` handler | Unit (humatest) | `get_match_participants_test.go` |

New pgtest helper: `InsertTestMatchParticipant(t, pool, matchUUID, sheetUUID string, joinedAt time.Time) string`

---

## Scope Boundary

| In scope | Out of scope |
|----------|-------------|
| `match_participants` migration | `MatchSession` in `app/game/` |
| `Participant` entity | Loading full `CharacterSheet` objects into WS Room |
| `StartMatchUC` extension | Mid-match join/leave/death update endpoints |
| `GetMatchParticipantsUC` + handler | |
| `CharacterSheetWithVisibilityResponse` refactor | |
