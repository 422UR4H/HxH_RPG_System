# Match Enrollments Listing — Design Spec

## Problem

The match page needs to display the roster of all enrollments for a given match (`pending`, `accepted`, and `rejected`). The existing `GET /matches/{uuid}` returns only match data and is reused across screens, so embedding enrollments would couple consumers and bloat unaffected callers. A dedicated endpoint is required.

Beyond the basic listing, two access controls must apply:

1. **Match-level visibility** — for private matches, only the master and players who participate in the campaign may view the roster.
2. **Per-row data scope** — the master sees the full character sheet summary (base + private fields); other authorized viewers see only base fields.

The privacy rule for private matches is currently inconsistent in `GetMatchUC` (only the master is allowed). This spec retroports the rule there to keep both endpoints aligned.

## Solution

Add `GET /matches/{uuid}/enrollments` returning the match roster with per-row visibility derived from the viewer's relationship to the match. Keep `GET /matches/{uuid}` unchanged in shape but harmonize its private-match access rule.

## Endpoint

```
GET /matches/{uuid}/enrollments
```

Auth: required (existing middleware).

### Authorization (match-level)

The viewer is allowed when **any** of the following holds:

- The match is public (`is_public = true`)
- The viewer is the match's master (`userUUID == match.MasterUUID`)
- The viewer has at least one character sheet linked to the match's campaign (`EXISTS(SELECT 1 FROM character_sheets WHERE player_uuid = $userUUID AND campaign_uuid = $match.CampaignUUID)`)

Otherwise → `403 Forbidden`.

### Visibility (per-row)

Two tiers, decided by a single boolean computed once at the use case (`viewerIsMaster := userUUID == match.MasterUUID`):

| Viewer | Sheet payload per enrollment |
|---|---|
| Match master | Base fields + nested `private` object populated for every row |
| Any other authorized viewer | Base fields only; `private` is `null` for every row |

Rationale for not specializing the player's own row:
- Stable JSON shape simplifies the frontend.
- Robust to fresh sessions, hard refreshes, deep links, multi-tab.
- The base summary is small (~200 bytes); negligible redundancy.
- Keeps the use case logic to a single boolean.

The frontend may choose to discard the redundant data for the requesting player's own row — that is a UI optimization, not an API concern.

### Response

```json
{
  "enrollments": [
    {
      "uuid": "…",
      "status": "pending",
      "created_at": "Mon, 02 Jan 2006 15:04:05 GMT",
      "character_sheet": {
        "uuid": "…",
        "player_uuid": "…",
        "master_uuid": null,
        "campaign_uuid": "…",
        "nick_name": "Gon",
        "story_start_at": "2026-01-01",
        "story_current_at": "2026-01-15",
        "dead_at": null,
        "created_at": "…",
        "updated_at": "…",
        "private": {
          "full_name": "Gon Freecss",
          "alignment": "neutral_good",
          "character_class": "hunter",
          "birthday": "1987-05-05",
          "category_name": "reinforcement",
          "curr_hex_value": 80,
          "level": 5,
          "points": 12,
          "talent_lvl": 3,
          "physicals_lvl": 4,
          "mentals_lvl": 2,
          "spirituals_lvl": 3,
          "skills_lvl": 5,
          "stamina": { "min": 0, "current": 30, "max": 50 },
          "health":  { "min": 0, "current": 40, "max": 60 }
        }
      },
      "player": { "uuid": "…", "nick": "tiago" }
    }
  ]
}
```

`private` is always serialized as `null` for non-master viewers (no `omitempty`) so the JSON shape is stable across roles.

Date formats inherited from existing summary types: enrollment `created_at` uses `http.TimeFormat` (RFC1123 GMT, matching `MatchResponse`); character sheet `created_at`/`updated_at` use RFC3339; `story_start_at`/`story_current_at` use `2006-01-02`; `dead_at` uses RFC3339 — all per the existing `toSummaryBaseResponse` and `ToPrivateSummaryResponse` mappings.

Ordering: `ORDER BY enrollments.created_at ASC`.

### Error Mapping

| Condition | HTTP |
|---|---|
| Unauthenticated | 401 (middleware) |
| Match not found | 404 (`ErrMatchNotFound`) |
| Private match, viewer is neither master nor campaign participant | 403 (`ErrInsufficientPermissions`) |
| Other repository / server error | 500 |

## Schema Changes

### Migration: index for `match_uuid + created_at`

```sql
-- migrations/<timestamp>_add_enrollments_match_uuid_index.sql
-- +goose Up
CREATE INDEX idx_enrollments_match_uuid_created_at
  ON enrollments(match_uuid, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_enrollments_match_uuid_created_at;
```

The composite index covers the listing query's filter and the `ORDER BY`, avoiding a sort step. The existing `idx_enrollments_sheet_match_uuid (character_sheet_uuid, match_uuid)` does not help — its leading column is the sheet, not the match.

## Domain Layer

### New Entity (`internal/domain/entity/enrollment/`)

There is currently no entity package for enrollment. Introduce one mirroring the convention used by `entity/match/summary.go`:

```go
// internal/domain/entity/enrollment/enrollment.go
package enrollment

import (
    "time"

    sheetModel "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
    "github.com/google/uuid"
)

type PlayerRef struct {
    UUID uuid.UUID
    Nick string
}

type Enrollment struct {
    UUID           uuid.UUID
    Status         string
    CreatedAt      time.Time
    // TODO(architecture): CharacterSheetSummary lives in gateway/pg/model — entity should not
    // import outer layers. Tracked for cleanup: move CharacterSheetSummary to
    // domain/entity/character_sheet/summary.go in a follow-up task and update all call sites
    // (use cases under domain/character_sheet/ already import model.CharacterSheetSummary too,
    // so the cleanup is shared, not specific to enrollment).
    CharacterSheet sheetModel.CharacterSheetSummary // includes base + private fields
    Player         PlayerRef
}
```

Rationale: `CharacterSheetSummary` already carries every field the existing `CharacterPrivateSummaryResponse` needs. The use case never strips fields — that decision lives in the handler layer (visibility), keeping the domain layer free of presentation concerns. The architectural violation (entity importing gateway model) is intentional and matches an existing pattern at the use case layer; cleanup is deferred to a separate task.

### Use Case (`internal/domain/match/`)

Decision: place this UC in `domain/match`, not `domain/enrollment`. The endpoint is fundamentally a match-page read whose primary inputs are the match's privacy state and the master/participant relationship; enrollments are aggregated data, not the orchestration subject. Operations that act on a single enrollment (accept/reject) stay in `domain/enrollment`.

To avoid a dependency cycle (`domain/enrollment` already imports `domain/match`), the enrollment-listing dependency is declared as a local interface in `domain/match` (Go's "interfaces are defined where they're consumed" idiom). The same gateway struct satisfies both interfaces via structural typing.

```go
// internal/domain/match/list_match_enrollments.go
package match

import (
    "context"
    "errors"

    "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
    enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
    matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
    "github.com/google/uuid"
)

type EnrollmentLister interface {
    ListByMatchUUID(
        ctx context.Context, matchUUID uuid.UUID,
    ) ([]*enrollmentEntity.Enrollment, error)
}

type CampaignParticipationChecker interface {
    ExistsSheetInCampaign(
        ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID,
    ) (bool, error)
}

type ListMatchEnrollmentsResult struct {
    Enrollments     []*enrollmentEntity.Enrollment
    ViewerIsMaster  bool
}

type IListMatchEnrollments interface {
    List(
        ctx context.Context, matchUUID uuid.UUID, userUUID uuid.UUID,
    ) (*ListMatchEnrollmentsResult, error)
}

type ListMatchEnrollmentsUC struct {
    matchRepo            IRepository
    enrollmentLister     EnrollmentLister
    participationChecker CampaignParticipationChecker
}

func NewListMatchEnrollmentsUC(
    matchRepo IRepository,
    enrollmentLister EnrollmentLister,
    participationChecker CampaignParticipationChecker,
) *ListMatchEnrollmentsUC { /* ... */ }

func (uc *ListMatchEnrollmentsUC) List(
    ctx context.Context, matchUUID uuid.UUID, userUUID uuid.UUID,
) (*ListMatchEnrollmentsResult, error) {
    match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
    if errors.Is(err, matchPg.ErrMatchNotFound) {
        return nil, ErrMatchNotFound
    }
    if err != nil {
        return nil, err
    }

    viewerIsMaster := match.MasterUUID == userUUID
    if !match.IsPublic && !viewerIsMaster {
        ok, err := uc.participationChecker.ExistsSheetInCampaign(
            ctx, userUUID, match.CampaignUUID,
        )
        if err != nil {
            return nil, err
        }
        if !ok {
            return nil, auth.ErrInsufficientPermissions
        }
    }

    enrollments, err := uc.enrollmentLister.ListByMatchUUID(ctx, matchUUID)
    if err != nil {
        return nil, err
    }
    return &ListMatchEnrollmentsResult{
        Enrollments:    enrollments,
        ViewerIsMaster: viewerIsMaster,
    }, nil
}
```

### `GetMatchUC` retrofit (same PR, separate commit)

Update the existing private-match check to align with the new rule:

```go
// before
if match.MasterUUID != userUUID && !match.IsPublic {
    return nil, auth.ErrInsufficientPermissions
}

// after
if !match.IsPublic && match.MasterUUID != userUUID {
    ok, err := uc.participationChecker.ExistsSheetInCampaign(
        ctx, userUUID, match.CampaignUUID,
    )
    if err != nil { return nil, err }
    if !ok { return nil, auth.ErrInsufficientPermissions }
}
```

`GetMatchUC` gains a constructor parameter `participationChecker CampaignParticipationChecker` (same local interface). Wiring updates in `cmd/api/main.go`. All `GetMatchUC` tests and call sites updated.

## Gateway Layer

### `internal/gateway/pg/enrollment/list_by_match_uuid.go`

```go
func (r *Repository) ListByMatchUUID(
    ctx context.Context, matchUUID uuid.UUID,
) ([]*enrollmentEntity.Enrollment, error) {
    const query = `
        SELECT
            e.uuid, e.status, e.created_at,
            cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
            cs.category_name, cs.curr_hex_value,
            cs.level, cs.points, cs.talent_lvl, cs.skills_lvl,
            cs.health_min_pts, cs.health_curr_pts, cs.health_max_pts,
            cs.stamina_min_pts, cs.stamina_curr_pts, cs.stamina_max_pts,
            cs.physicals_lvl, cs.mentals_lvl, cs.spirituals_lvl,
            cs.aura_min_pts, cs.aura_curr_pts, cs.aura_max_pts,
            cs.created_at, cs.updated_at,
            cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday,
            u.uuid, u.nick
        FROM enrollments e
        JOIN character_sheets cs   ON cs.uuid = e.character_sheet_uuid
        JOIN character_profiles cp ON cp.character_sheet_uuid = cs.uuid
        JOIN users u               ON u.uuid = cs.player_uuid
        WHERE e.match_uuid = $1
        ORDER BY e.created_at ASC
    `
    // scan into *enrollmentEntity.Enrollment, return slice
}
```

Notes:
- INNER JOIN on `users` is safe: every enrollment requires `cs.player_uuid != nil` (enforced by `EnrollCharacterInMatchUC`).
- Returns an empty slice (not error) when the match exists but has no enrollments.
- Reuses the field set already selected by `ListCharacterSheetsByPlayerUUID` for consistency with the existing private summary mapping.

### `internal/gateway/pg/sheet/exists_in_campaign.go`

```go
func (r *Repository) ExistsSheetInCampaign(
    ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID,
) (bool, error) {
    const query = `
        SELECT EXISTS (
            SELECT 1 FROM character_sheets
            WHERE player_uuid = $1 AND campaign_uuid = $2
        )
    `
    var exists bool
    err := r.q.QueryRow(ctx, query, playerUUID, campaignUUID).Scan(&exists)
    return exists, err
}
```

Add to `domain/character_sheet/i_repository.go` (sheet repo also satisfies `match.CampaignParticipationChecker` via this method).

## App / Handler Layer

### `internal/app/api/match/list_match_enrollments.go`

```go
type ListMatchEnrollmentsRequest struct {
    UUID uuid.UUID `path:"uuid" required:"true" doc:"UUID of the match"`
}

type ListMatchEnrollmentsResponse struct {
    Body ListMatchEnrollmentsResponseBody `json:"body"`
}

type ListMatchEnrollmentsResponseBody struct {
    Enrollments []EnrollmentResponse `json:"enrollments"`
}

type EnrollmentResponse struct {
    UUID           uuid.UUID                              `json:"uuid"`
    Status         string                                 `json:"status"`
    CreatedAt      string                                 `json:"created_at"`
    CharacterSheet CharacterSheetWithVisibilityResponse   `json:"character_sheet"`
    Player         PlayerRefResponse                      `json:"player"`
}

type CharacterSheetWithVisibilityResponse struct {
    sheetHandler.CharacterBaseSummaryResponse
    Private *sheetHandler.CharacterPrivateOnlyResponse `json:"private"`
}

type PlayerRefResponse struct {
    UUID uuid.UUID `json:"uuid"`
    Nick string    `json:"nick"`
}
```

The existing `CharacterPrivateSummaryResponse` flattens base + private fields. Extract the private-only fields into `CharacterPrivateOnlyResponse` (struct without the embedded base) so the handler can nest it cleanly. Keep `CharacterPrivateSummaryResponse` intact for existing callers; the new struct is a subset, defined alongside.

Handler logic:
1. Decode UUID from path.
2. Read `userUUID` from context.
3. Call UC; map domain errors → HTTP codes.
4. Build response: for each enrollment, populate base summary; if `result.ViewerIsMaster` is true, also populate the nested `private`.

### `internal/app/api/match/routes.go`

Add the `ListMatchEnrollmentsHandler` field to the `Api` struct and register:

```go
huma.Register(api, huma.Operation{
    Method:      http.MethodGet,
    Path:        "/matches/{uuid}/enrollments",
    Description: "List enrollments of a match (visibility per row depends on viewer)",
    Tags:        []string{"matches"},
    Errors: []int{
        http.StatusUnauthorized,
        http.StatusForbidden,
        http.StatusNotFound,
        http.StatusInternalServerError,
    },
}, a.ListMatchEnrollmentsHandler)
```

## Wiring (`cmd/api/main.go`)

```go
listMatchEnrollmentsUC := domainMatch.NewListMatchEnrollmentsUC(
    matchRepo,
    enrollmentRepo,        // satisfies match.EnrollmentLister
    characterSheetRepo,    // satisfies match.CampaignParticipationChecker
)

// also pass characterSheetRepo into the (refactored) GetMatchUC constructor:
getMatchUC := domainMatch.NewGetMatchUC(matchRepo, characterSheetRepo)

matchesApi := matchHandler.Api{
    // existing fields...
    ListMatchEnrollmentsHandler: matchHandler.ListMatchEnrollmentsHandler(listMatchEnrollmentsUC),
}
```

## Testing

Per the project's TDD-by-layer strategy:

### Use case (`internal/domain/match/list_match_enrollments_test.go`)
Unit tests with mocks for `IRepository`, `EnrollmentLister`, `CampaignParticipationChecker`. Cases:
- Master viewer on private match → `ViewerIsMaster=true`, no participation check called
- Master viewer on public match → `ViewerIsMaster=true`
- Non-master on public match → `ViewerIsMaster=false`, no participation check called
- Non-master on private match, participates → `ViewerIsMaster=false`
- Non-master on private match, does not participate → `ErrInsufficientPermissions`
- Match not found → `ErrMatchNotFound`
- `EnrollmentLister` returns empty → empty slice, no error
- `EnrollmentLister` returns error → propagated
- `participationChecker` returns error → propagated

### `GetMatchUC` (existing test file, expand)
Add cases for the new participation-based path:
- Non-master on private match, participates → success
- Non-master on private match, does not participate → `ErrInsufficientPermissions`

### Gateway — enrollment (`internal/gateway/pg/enrollment/enrollment_integration_test.go`)
Add `TestListByMatchUUID` with sub-tests:
- Lists all statuses including rejected
- Ordered by `created_at` ASC
- Empty slice when match has no enrollments
- JOIN materializes player nick + sheet base + sheet private fields
- Different match's enrollments are not included

### Gateway — sheet (`internal/gateway/pg/sheet/sheet_integration_test.go`)
Add `TestExistsSheetInCampaign` with:
- True when player has at least one sheet in the campaign
- False when player has no sheet in the campaign
- False when player has sheets in other campaigns only

### Handler (`internal/app/api/match/list_match_enrollments_test.go`)
Humatest-based, with mock UC. Cases:
- 200 with `private` populated for all rows when `ViewerIsMaster=true`
- 200 with `private` null/omitted for all rows when `ViewerIsMaster=false`
- 200 with empty list
- 404 on `ErrMatchNotFound`
- 403 on `ErrInsufficientPermissions`
- 500 on generic error

Mocks added to `mocks_test.go` in the match handler package.

## Documentation

Per `docs-workflow.instructions.md` (PT-BR for `docs/dev/`):

1. **New** `docs/dev/match/roster.md` — describes the listing endpoint, authorization rule (public / master / campaign participant), per-row visibility tiers, response shape, why the schema is stable across viewers.
2. **Update** `docs/dev/enrollment.md` — add a short §7 "Listagem por Match" cross-referencing `roster.md`, summarizing the cross-domain nature of the read (entity in `domain/entity/enrollment`, use case in `domain/match`).
3. **Update** `docs/documentation-map.yaml` — add mappings:
    - `internal/domain/match/list_match_enrollments.go` → `docs/dev/match/roster.md`
    - `internal/gateway/pg/enrollment/list_by_match_uuid.go` → `docs/dev/match/roster.md` + `docs/dev/enrollment.md`
    - `internal/domain/entity/enrollment/` → `docs/dev/enrollment.md`

## Implementation Phases

**Phase 1 — Read path skeleton (no privacy retrofit yet)**
Migration + entity + gateway methods + use case (without privacy check) + handler with both visibility tiers + tests + wiring + docs.

**Phase 2 — Privacy retrofit**
Add the participation check to `ListMatchEnrollmentsUC` and to `GetMatchUC` (separate commit). Update tests of both UCs.

Splitting the privacy retrofit lets reviewers focus on the data-shape work first and the cross-cutting authorization change second.

## Out of Scope

- Filtering / pagination query params (e.g. `?status=accepted`). Defer until the page actually needs it.
- A composite "match page" endpoint bundling match + enrollments + scene state. Two parallel calls are sufficient at the current scale; revisit only if measured latency becomes a problem.
- Allowing the master to enroll a sheet on behalf of a player (existing TODO in `EnrollCharacterInMatchUC`). Independent of this work.
- Player-facing notifications when their enrollment status changes. Out of scope for a read endpoint.
