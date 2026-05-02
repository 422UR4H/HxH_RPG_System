# Lobby WebSocket Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the lobby phase of the WebSocket game server so a master can open a lobby, kick players, manage enrollments via REST while in WS, and start the match — persisting state to PostgreSQL.

**Architecture:** Bottom-up. Domain entities and errors first, then repository interfaces + mocks, gateway implementations, domain use cases (TDD), temporal guard on existing enrollment UCs, game server messages + room enhancements, and finally handler/hub/cmd wiring. Each layer is testable independently.

**Tech Stack:** Go 1.23, PostgreSQL (goose migrations), gorilla/websocket, pgx/v5, standard `testing` package.

**Spec:** `docs/superpowers/specs/2026-05-01-lobby-websocket-design.md`

---

## File Map

### Refactoring files (Task 0)

| File | Changes |
|------|---------|
| `migrations/20260501190000_refactor_game_start_at.sql` | Add `game_scheduled_at`, make `game_start_at` nullable, update indexes |
| `internal/domain/entity/match/match.go` | `GameStartAt *time.Time` + `GameScheduledAt time.Time` |
| `internal/domain/entity/match/summary.go` | Same changes as match.go |
| `internal/domain/match/create_match.go` | `GameStartAt` → `GameScheduledAt` in input/validation |
| `internal/domain/match/error.go` | `ErrMinOfGameScheduledAt`, `ErrMaxOfGameScheduledAt` |
| `internal/domain/match/match_uc_test.go` | Update scheduled time field references |
| `internal/gateway/pg/match/create_match.go` | INSERT `game_scheduled_at` instead of `game_start_at` |
| `internal/gateway/pg/match/read_match.go` | SELECT both columns, update all Scan calls |
| `internal/gateway/pg/campaign/read_campaign.go` | SELECT both columns, update Scan |
| `internal/gateway/pg/match/match_integration_test.go` | Update test fixture |
| `internal/gateway/pg/pgtest/setup.go` | Update `InsertTestMatch` |
| `internal/app/api/match/create_match.go` | Request: `game_scheduled_at`, Response: both fields |
| `internal/app/api/match/match_summary_response.go` | Add `GameScheduledAt` + nullable `GameStartAt` |
| `internal/app/api/match/get_match.go` | Response: both fields |
| `internal/app/api/match/create_match_test.go` | Update all body maps + mocks |
| `internal/app/api/match/get_match_test.go` | Update mock fixture |
| `internal/app/api/match/list_matches_test.go` | Update Summary fixtures |
| `internal/app/api/match/list_public_upcoming_matches_test.go` | Update Summary fixture |
| `internal/app/api/match/routes.go` | Update description text |

### New files (Lobby feature)

| File | Responsibility |
|------|---------------|
| `internal/gateway/pg/match/start_match.go` | Gateway: `StartMatch(ctx, matchUUID)` UPDATE |
| `internal/gateway/pg/enrollment/reject_by_player_and_match.go` | Gateway: reject enrollment by playerUUID + matchUUID |
| `internal/gateway/pg/enrollment/reject_pending_enrollments.go` | Gateway: reject all pending enrollments for a match |
| `internal/domain/match/start_match.go` | StartMatchUC domain use case |
| `internal/domain/match/start_match_test.go` | StartMatchUC tests |
| `internal/domain/enrollment/kick_player.go` | KickPlayerUC domain use case |
| `internal/domain/enrollment/kick_player_test.go` | KickPlayerUC tests |

### Modified files (Lobby feature)

| File | Changes |
|------|---------|
| `internal/domain/match/i_repository.go` | Add `StartMatch` method |
| `internal/domain/match/error.go` | Add `ErrMatchAlreadyStarted`, `ErrMatchAlreadyFinished`, `ErrNotMatchMaster` |
| `internal/domain/enrollment/i_repository.go` | Add 3 new methods |
| `internal/domain/enrollment/error.go` | Add `ErrMatchAlreadyStarted`, `ErrMatchAlreadyFinished`, `ErrPlayerNotEnrolled` |
| `internal/domain/enrollment/accept_enrollment.go` | Add temporal guard |
| `internal/domain/enrollment/reject_enrollment.go` | Add temporal guard |
| `internal/domain/enrollment/enrollment_test.go` | Add temporal guard test cases |
| `internal/domain/testutil/mock_match_repo.go` | Add `StartMatchFn` |
| `internal/domain/testutil/mock_enrollment_repo.go` | Add 3 new Fn fields + methods |
| `internal/gateway/pg/enrollment/is_player_enrolled.go` | Add `AND e.status = 'accepted'` to query |
| `internal/app/game/message.go` | Add `MsgTypeKickPlayer`, `MsgTypePlayerKicked`, `KickPlayerPayload`, `PlayerKickedPayload` |
| `internal/app/game/room.go` | Inject UCs, enhance `StartMatch`/`handleClientMessage`/`sendRoomState` |
| `internal/app/game/handler.go` | Expand interfaces, update constructor |
| `internal/app/game/hub.go` | Update `GetOrCreateRoom` signature |
| `internal/app/game/handler_test.go` | Update mocks + constructor calls, add kick/start test cases |
| `cmd/game/main.go` | Wire new UCs + repos |

---

### Task 0: Refactoring — GameStartAt → GameScheduledAt

> **IMPORTANT — Post-refactoring naming convention:**
> After this task, all subsequent tasks use the **new** naming:
> - `GameStartAt *time.Time` = when the match **actually started** (nullable, filled by `StartMatch`)
> - `GameScheduledAt time.Time` = when the master **scheduled** the match (NOT NULL, set at creation)
> - DB column `game_start_at` = nullable TIMESTAMP (actual start)
> - DB column `game_scheduled_at` = NOT NULL TIMESTAMP (scheduled time)
>
> In downstream tasks, wherever the old plan references `StartedAt` or `started_at`, it now corresponds to `GameStartAt` / `game_start_at`. The old `GameStartAt` / `game_start_at` (scheduled time) is now `GameScheduledAt` / `game_scheduled_at`.

**Files:**
- Create: `migrations/20260501190000_refactor_game_start_at.sql`
- Modify: `internal/domain/entity/match/match.go`
- Modify: `internal/domain/entity/match/summary.go`
- Modify: `internal/domain/match/create_match.go`
- Modify: `internal/domain/match/error.go`
- Modify: `internal/domain/match/match_uc_test.go`
- Modify: `internal/gateway/pg/match/create_match.go`
- Modify: `internal/gateway/pg/match/read_match.go`
- Modify: `internal/gateway/pg/campaign/read_campaign.go`
- Modify: `internal/gateway/pg/match/match_integration_test.go`
- Modify: `internal/gateway/pg/pgtest/setup.go`
- Modify: `internal/app/api/match/create_match.go`
- Modify: `internal/app/api/match/match_summary_response.go`
- Modify: `internal/app/api/match/get_match.go`
- Modify: `internal/app/api/match/create_match_test.go`
- Modify: `internal/app/api/match/get_match_test.go`
- Modify: `internal/app/api/match/list_matches_test.go`
- Modify: `internal/app/api/match/list_public_upcoming_matches_test.go`
- Modify: `internal/app/api/match/routes.go`

#### Part A: Migration

- [ ] **Step 1: Create migration file**

Create `migrations/20260501190000_refactor_game_start_at.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE matches ADD COLUMN game_scheduled_at TIMESTAMP;
UPDATE matches SET game_scheduled_at = game_start_at;
ALTER TABLE matches ALTER COLUMN game_scheduled_at SET NOT NULL;

ALTER TABLE matches ALTER COLUMN game_start_at DROP NOT NULL;
UPDATE matches SET game_start_at = NULL;

DROP INDEX IF EXISTS idx_matches_is_public_game_start_master;
DROP INDEX IF EXISTS idx_matches_game_start_at;

CREATE INDEX IF NOT EXISTS idx_matches_is_public_game_scheduled_master ON matches(is_public, game_scheduled_at, master_uuid);
CREATE INDEX IF NOT EXISTS idx_matches_game_scheduled_at ON matches(game_scheduled_at);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

UPDATE matches SET game_start_at = game_scheduled_at WHERE game_start_at IS NULL;
ALTER TABLE matches ALTER COLUMN game_start_at SET NOT NULL;

ALTER TABLE matches DROP COLUMN game_scheduled_at;

DROP INDEX IF EXISTS idx_matches_is_public_game_scheduled_master;
DROP INDEX IF EXISTS idx_matches_game_scheduled_at;

CREATE INDEX IF NOT EXISTS idx_matches_is_public_game_start_master ON matches(is_public, game_start_at, master_uuid);
CREATE INDEX IF NOT EXISTS idx_matches_game_start_at ON matches(game_start_at);

COMMIT;
-- +goose StatementEnd
```

- [ ] **Step 2: Run migration**

Run: `make migrate-up`
Expected: Migration applies successfully.

#### Part B: Entity layer

- [ ] **Step 3: Update Match entity**

In `internal/domain/entity/match/match.go`, change `GameStartAt time.Time` to `GameStartAt *time.Time` and add `GameScheduledAt time.Time`:

```go
type Match struct {
UUID                    uuid.UUID
MasterUUID              uuid.UUID
CampaignUUID            uuid.UUID
Title                   string
BriefInitialDescription string
BriefFinalDescription   *string
Description             string
IsPublic                bool
scenes                  []*scene.Scene
events                  []GameEvent
GameScheduledAt         time.Time
GameStartAt             *time.Time
StoryStartAt            time.Time
StoryEndAt              *time.Time
CreatedAt               time.Time
UpdatedAt               time.Time
}
```

Update `NewMatch` — rename the parameter and set `GameScheduledAt` instead of `GameStartAt`:

```go
func NewMatch(
masterUUID uuid.UUID,
campaignUUID uuid.UUID,
title string,
briefInitialDescription string,
description string,
isPublic bool,
gameScheduledAt time.Time,
storyStartAt time.Time,
) (*Match, error) {
now := time.Now()
return &Match{
UUID:                    uuid.New(),
MasterUUID:              masterUUID,
CampaignUUID:            campaignUUID,
Title:                   title,
BriefInitialDescription: briefInitialDescription,
Description:             description,
IsPublic:                isPublic,
GameScheduledAt:         gameScheduledAt,
StoryStartAt:            storyStartAt,
CreatedAt:               now,
UpdatedAt:               now,
}, nil
}
```

Note: `GameStartAt` is NOT set in the constructor — it stays `nil` (match hasn't started yet).

- [ ] **Step 4: Update Summary struct**

In `internal/domain/entity/match/summary.go`:

```go
type Summary struct {
UUID                    uuid.UUID
CampaignUUID            uuid.UUID
Title                   string
BriefInitialDescription string
BriefFinalDescription   *string
IsPublic                bool
GameScheduledAt         time.Time
GameStartAt             *time.Time
StoryStartAt            time.Time
StoryEndAt              *time.Time
CreatedAt               time.Time
UpdatedAt               time.Time
}
```

#### Part C: Domain layer

- [ ] **Step 5: Rename domain errors**

In `internal/domain/match/error.go`, rename the GameStartAt error sentinels:

```go
var (
ErrMatchNotFound          = domain.NewValidationError(errors.New("match not found"))
ErrMinTitleLength         = domain.NewValidationError(errors.New("title must be at least 5 characters"))
ErrMaxTitleLength         = domain.NewValidationError(errors.New("title cannot exceed 32 characters"))
ErrMinOfStoryStartAt      = domain.NewValidationError(errors.New("story start date must be after campaign start date"))
ErrMaxOfStoryStartAt      = domain.NewValidationError(errors.New("story start date must be before campaign end date"))
ErrMinOfGameScheduledAt   = domain.NewValidationError(errors.New("game scheduled at cannot be in the past"))
ErrMaxOfGameScheduledAt   = domain.NewValidationError(errors.New("game scheduled at cannot be greater than one year from now"))
ErrMaxBriefDescLength     = domain.NewValidationError(errors.New("brief description cannot exceed 64 characters"))
)
```

- [ ] **Step 6: Update CreateMatchInput and CreateMatchUC**

In `internal/domain/match/create_match.go`:

Change `CreateMatchInput.GameStartAt` → `GameScheduledAt`:

```go
type CreateMatchInput struct {
MasterUUID              uuid.UUID
CampaignUUID            uuid.UUID
Title                   string
BriefInitialDescription string
Description             string
IsPublic                bool
GameScheduledAt         time.Time
StoryStartAt            time.Time
}
```

Update validation in `CreateMatch`:

```go
if input.GameScheduledAt.Before(time.Now()) {
return nil, ErrMinOfGameScheduledAt
}
if input.GameScheduledAt.After(time.Now().AddDate(1, 0, 0)) {
return nil, ErrMaxOfGameScheduledAt
}
```

Update `NewMatch` call:

```go
newMatch, err := match.NewMatch(
input.MasterUUID,
input.CampaignUUID,
input.Title,
input.BriefInitialDescription,
input.Description,
input.IsPublic,
input.GameScheduledAt,
input.StoryStartAt,
)
```

- [ ] **Step 7: Update CreateMatch UC test**

In `internal/domain/match/match_uc_test.go`, update `validCreateMatchInput()`:

```go
func validCreateMatchInput() *domainMatch.CreateMatchInput {
return &domainMatch.CreateMatchInput{
MasterUUID:              uuid.New(),
CampaignUUID:            uuid.New(),
Title:                   "Valid Title",
BriefInitialDescription: "Brief",
Description:             "Full description",
IsPublic:                true,
GameScheduledAt:         time.Now().Add(24 * time.Hour),
StoryStartAt:            time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
}
}
```

Update the "game start at in the past" test case:

```go
{
name: "game scheduled at in the past",
input: func() *domainMatch.CreateMatchInput {
i := validCreateMatchInput()
i.GameScheduledAt = time.Now().Add(-1 * time.Hour)
return i
}(),
matchMock:    &testutil.MockMatchRepo{},
campaignMock: &testutil.MockCampaignRepo{},
wantErr:      domainMatch.ErrMinOfGameScheduledAt,
},
```

Update the "game start at more than 1 year ahead" test case:

```go
{
name: "game scheduled at more than 1 year ahead",
input: func() *domainMatch.CreateMatchInput {
i := validCreateMatchInput()
i.GameScheduledAt = time.Now().AddDate(1, 1, 0)
return i
}(),
matchMock:    &testutil.MockMatchRepo{},
campaignMock: &testutil.MockCampaignRepo{},
wantErr:      domainMatch.ErrMaxOfGameScheduledAt,
},
```

#### Part D: Gateway layer

- [ ] **Step 8: Update CreateMatch gateway**

In `internal/gateway/pg/match/create_match.go`, add `game_scheduled_at` to the INSERT:

```go
const query = `
        INSERT INTO matches (
            uuid, master_uuid, campaign_uuid,
            title, brief_initial_description, description,
            is_public, game_scheduled_at,
            story_start_at, story_end_at,
            created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
        )
    `
_, err = tx.Exec(ctx, query,
match.UUID, match.MasterUUID, match.CampaignUUID,
match.Title, match.BriefInitialDescription, match.Description,
match.IsPublic, match.GameScheduledAt,
match.StoryStartAt, match.StoryEndAt,
match.CreatedAt, match.UpdatedAt,
)
```

Note: `game_start_at` is NOT included — it defaults to NULL (match hasn't started).

- [ ] **Step 9: Update GetMatch gateway**

In `internal/gateway/pg/match/read_match.go`, update `GetMatch` to SELECT both columns:

```go
const query = `
        SELECT 
            uuid, master_uuid, campaign_uuid,
            title, brief_initial_description, brief_final_description, description,
            is_public, game_scheduled_at, game_start_at,
            story_start_at, story_end_at,
            created_at, updated_at
        FROM matches
        WHERE uuid = $1
    `
var m match.Match
err = tx.QueryRow(ctx, query, uuid).Scan(
&m.UUID,
&m.MasterUUID,
&m.CampaignUUID,
&m.Title,
&m.BriefInitialDescription,
&m.BriefFinalDescription,
&m.Description,
&m.IsPublic,
&m.GameScheduledAt,
&m.GameStartAt,
&m.StoryStartAt,
&m.StoryEndAt,
&m.CreatedAt,
&m.UpdatedAt,
)
```

- [ ] **Step 10: Update ListMatchesByMasterUUID gateway**

In `internal/gateway/pg/match/read_match.go`, update `ListMatchesByMasterUUID` SELECT and Scan:

```go
const query = `
        SELECT 
            uuid, campaign_uuid, title,
            brief_initial_description, brief_final_description,
            is_public, game_scheduled_at, game_start_at,
            story_start_at, story_end_at,
            created_at, updated_at
        FROM matches
        WHERE master_uuid = $1
        ORDER BY story_start_at ASC
    `
```

Scan:
```go
err := rows.Scan(
&m.UUID,
&m.CampaignUUID,
&m.Title,
&m.BriefInitialDescription,
&m.BriefFinalDescription,
&m.IsPublic,
&m.GameScheduledAt,
&m.GameStartAt,
&m.StoryStartAt,
&m.StoryEndAt,
&m.CreatedAt,
&m.UpdatedAt,
)
```

- [ ] **Step 11: Update ListPublicUpcomingMatches gateway**

In `internal/gateway/pg/match/read_match.go`, update query to filter by `game_scheduled_at` and SELECT both columns:

```go
const query = `
        SELECT 
            uuid, campaign_uuid, title,
            brief_initial_description, brief_final_description,
            is_public, game_scheduled_at, game_start_at,
            story_start_at, story_end_at,
            created_at, updated_at
        FROM matches
        WHERE is_public = true
        AND game_scheduled_at > $1
        AND master_uuid != $2
        ORDER BY game_scheduled_at ASC
    `
```

Same Scan update as Step 10.

- [ ] **Step 12: Update campaign gateway**

In `internal/gateway/pg/campaign/read_campaign.go`, update the matches query in `GetCampaignByUUID`:

```go
const matchesQuery = `
        SELECT 
            uuid, campaign_uuid,
            title, brief_initial_description, brief_final_description,
            is_public, game_scheduled_at, game_start_at,
            story_start_at, story_end_at,
            created_at, updated_at
        FROM matches
        WHERE campaign_uuid = $1
        ORDER BY story_start_at DESC
    `
```

Update Scan to include both fields:
```go
err := rows.Scan(
&m.UUID,
&m.CampaignUUID,
&m.Title,
&m.BriefInitialDescription,
&m.BriefFinalDescription,
&m.IsPublic,
&m.GameScheduledAt,
&m.GameStartAt,
&m.StoryStartAt,
&m.StoryEndAt,
&m.CreatedAt,
&m.UpdatedAt,
)
```

- [ ] **Step 13: Update integration test fixture**

In `internal/gateway/pg/pgtest/setup.go`, update `InsertTestMatch`:

```go
func InsertTestMatch(t *testing.T, pool *pgxpool.Pool, masterUUID, campaignUUID, title string) string {
t.Helper()
ctx := context.Background()

var matchUUID string
err := pool.QueryRow(ctx,
`INSERT INTO matches (master_uuid, campaign_uuid, title, game_scheduled_at, story_start_at)
 VALUES ($1, $2, $3, NOW() + INTERVAL '1 day', CURRENT_DATE) RETURNING uuid`,
masterUUID, campaignUUID, title,
).Scan(&matchUUID)
if err != nil {
t.Fatalf("failed to insert test match: %v", err)
}
return matchUUID
}
```

- [ ] **Step 14: Update match integration test**

In `internal/gateway/pg/match/match_integration_test.go`, update `newTestMatch`:

```go
func newTestMatch(masterUUID, campaignUUID uuid.UUID, title string, isPublic bool, gameScheduledAt time.Time) *entityMatch.Match {
now := time.Now().Truncate(time.Microsecond)
return &entityMatch.Match{
UUID:                    uuid.New(),
MasterUUID:              masterUUID,
CampaignUUID:            campaignUUID,
Title:                   title,
BriefInitialDescription: "Brief description for " + title,
Description:             "Full description for " + title,
IsPublic:                isPublic,
GameScheduledAt:         gameScheduledAt.Truncate(time.Microsecond),
StoryStartAt:            now,
CreatedAt:               now,
UpdatedAt:               now,
}
}
```

Note: `GameStartAt` is not set — stays nil.

#### Part E: API layer

- [ ] **Step 15: Update CreateMatch API handler**

In `internal/app/api/match/create_match.go`:

Update `CreateMatchRequestBody`:
```go
type CreateMatchRequestBody struct {
CampaignUUID            uuid.UUID `json:"campaign_uuid" required:"true" doc:"UUID of the campaign this match is based on"`
Title                   string    `json:"title" required:"true" maxLength:"32" doc:"Title of the match"`
BriefInitialDescription string    `json:"brief_initial_description" maxLength:"64" doc:"Brief description of the match"`
Description             string    `json:"description" doc:"Full description of the match"`
IsPublic                bool      `json:"is_public" default:"true" doc:"If the match is public or private"`
GameScheduledAt         string    `json:"game_scheduled_at" required:"true" doc:"Date and time when the game is scheduled (ISO 8601)"`
StoryStartAt            string    `json:"story_start_at" required:"true" doc:"Date when the match story starts (YYYY-MM-DD)"`
}
```

Update `MatchResponse` — now has both fields:
```go
type MatchResponse struct {
UUID                    uuid.UUID `json:"uuid"`
CampaignUUID            uuid.UUID `json:"campaign_uuid"`
Title                   string    `json:"title"`
BriefInitialDescription string    `json:"brief_initial_description"`
BriefFinalDescription   *string   `json:"brief_final_description,omitempty"`
Description             string    `json:"description"`
IsPublic                bool      `json:"is_public"`
GameScheduledAt         string    `json:"game_scheduled_at"`
GameStartAt             *string   `json:"game_start_at,omitempty"`
StoryStartAt            string    `json:"story_start_at"`
StoryEndAt              *string   `json:"story_end_at,omitempty"`
CreatedAt               string    `json:"created_at"`
UpdatedAt               string    `json:"updated_at"`
}
```

Update the handler parsing:
```go
gameScheduledAt, err := time.Parse(time.RFC3339, req.Body.GameScheduledAt)
if err != nil {
return nil, huma.Error422UnprocessableEntity(
"invalid game_scheduled_at date format, use ISO 8601. E.g. 2026-06-15T19:30:00Z")
}

input := &domainMatch.CreateMatchInput{
MasterUUID:              userUUID,
CampaignUUID:            req.Body.CampaignUUID,
Title:                   req.Body.Title,
BriefInitialDescription: req.Body.BriefInitialDescription,
Description:             req.Body.Description,
IsPublic:                req.Body.IsPublic,
GameScheduledAt:         gameScheduledAt,
StoryStartAt:            storyStartAt,
}
```

Update response building:
```go
var gameStartAtStr *string
if match.GameStartAt != nil {
formatted := match.GameStartAt.Format(time.RFC3339)
gameStartAtStr = &formatted
}

response := MatchResponse{
UUID:                    match.UUID,
CampaignUUID:            match.CampaignUUID,
Title:                   match.Title,
BriefInitialDescription: match.BriefInitialDescription,
BriefFinalDescription:   match.BriefFinalDescription,
Description:             match.Description,
IsPublic:                match.IsPublic,
GameScheduledAt:         match.GameScheduledAt.Format(time.RFC3339),
GameStartAt:             gameStartAtStr,
StoryStartAt:            match.StoryStartAt.Format("2006-01-02"),
CreatedAt:               match.CreatedAt.Format(http.TimeFormat),
UpdatedAt:               match.UpdatedAt.Format(http.TimeFormat),
}
```

- [ ] **Step 16: Update GetMatch handler**

In `internal/app/api/match/get_match.go`, update response building same as create:

```go
var gameStartAtStr *string
if match.GameStartAt != nil {
formatted := match.GameStartAt.Format(time.RFC3339)
gameStartAtStr = &formatted
}

var storyEndAtStr *string
if match.StoryEndAt != nil {
formattedDate := match.StoryEndAt.Format("2006-01-02")
storyEndAtStr = &formattedDate
}

response := MatchResponse{
UUID:                    match.UUID,
CampaignUUID:            match.CampaignUUID,
Title:                   match.Title,
BriefInitialDescription: match.BriefInitialDescription,
BriefFinalDescription:   match.BriefFinalDescription,
Description:             match.Description,
IsPublic:                match.IsPublic,
GameScheduledAt:         match.GameScheduledAt.Format(time.RFC3339),
GameStartAt:             gameStartAtStr,
StoryStartAt:            match.StoryStartAt.Format("2006-01-02"),
StoryEndAt:              storyEndAtStr,
CreatedAt:               match.CreatedAt.Format(http.TimeFormat),
UpdatedAt:               match.UpdatedAt.Format(http.TimeFormat),
}
```

- [ ] **Step 17: Update MatchSummaryResponse**

In `internal/app/api/match/match_summary_response.go`:

```go
type MatchSummaryResponse struct {
UUID                    uuid.UUID `json:"uuid"`
CampaignUUID            uuid.UUID `json:"campaign_uuid"`
Title                   string    `json:"title"`
BriefInitialDescription string    `json:"brief_initial_description"`
BriefFinalDescription   *string   `json:"brief_final_description,omitempty"`
IsPublic                bool      `json:"is_public"`
GameScheduledAt         string    `json:"game_scheduled_at"`
GameStartAt             *string   `json:"game_start_at,omitempty"`
StoryStartAt            string    `json:"story_start_at"`
StoryEndAt              *string   `json:"story_end_at,omitempty"`
CreatedAt               string    `json:"created_at"`
UpdatedAt               string    `json:"updated_at"`
}

func ToSummaryResponse(m *domainMatch.Summary) MatchSummaryResponse {
var storyEndAtStr *string
if m.StoryEndAt != nil {
formatted := m.StoryEndAt.Format("2006-01-02")
storyEndAtStr = &formatted
}

var gameStartAtStr *string
if m.GameStartAt != nil {
formatted := m.GameStartAt.Format(time.RFC3339)
gameStartAtStr = &formatted
}

return MatchSummaryResponse{
UUID:                    m.UUID,
CampaignUUID:            m.CampaignUUID,
Title:                   m.Title,
BriefInitialDescription: m.BriefInitialDescription,
BriefFinalDescription:   m.BriefFinalDescription,
IsPublic:                m.IsPublic,
GameScheduledAt:         m.GameScheduledAt.Format(time.RFC3339),
GameStartAt:             gameStartAtStr,
StoryStartAt:            m.StoryStartAt.Format("2006-01-02"),
StoryEndAt:              storyEndAtStr,
CreatedAt:               m.CreatedAt.Format(http.TimeFormat),
UpdatedAt:               m.UpdatedAt.Format(http.TimeFormat),
}
}
```

Add `"time"` to imports if not already present.

- [ ] **Step 18: Update routes description**

In `internal/app/api/match/routes.go`, update the public matches route description:

```go
Description: "List all upcoming public matches sorted by game_scheduled_at",
```

- [ ] **Step 19: Update API handler tests**

In `internal/app/api/match/create_match_test.go`, update ALL test body maps:
- Change `"game_start_at"` → `"game_scheduled_at"` in all request bodies
- Update success mock to set `GameScheduledAt` instead of `GameStartAt`:

```go
mockFn: func(ctx context.Context, input *domainMatch.CreateMatchInput) (*matchEntity.Match, error) {
return &matchEntity.Match{
UUID:                    uuid.New(),
CampaignUUID:            input.CampaignUUID,
MasterUUID:              input.MasterUUID,
Title:                   input.Title,
BriefInitialDescription: input.BriefInitialDescription,
Description:             input.Description,
IsPublic:                input.IsPublic,
GameScheduledAt:         input.GameScheduledAt,
StoryStartAt:            input.StoryStartAt,
CreatedAt:               now,
UpdatedAt:               now,
}, nil
},
```

In `internal/app/api/match/get_match_test.go`, update the success mock:

```go
return &matchEntity.Match{
UUID:                    id,
CampaignUUID:            uuid.New(),
MasterUUID:              uid,
Title:                   "My Match",
BriefInitialDescription: "Brief",
Description:             "Full",
IsPublic:                true,
GameScheduledAt:         now,
StoryStartAt:            now,
CreatedAt:               now,
UpdatedAt:               now,
}, nil
```

In `internal/app/api/match/list_matches_test.go`, update Summary fixtures:

```go
{
UUID:                    uuid.New(),
CampaignUUID:            uuid.New(),
Title:                   "Match 1",
BriefInitialDescription: "Brief 1",
IsPublic:                true,
GameScheduledAt:         now,
StoryStartAt:            now,
CreatedAt:               now,
UpdatedAt:               now,
},
{
UUID:                    uuid.New(),
CampaignUUID:            uuid.New(),
Title:                   "Match 2",
BriefInitialDescription: "Brief 2",
IsPublic:                false,
GameScheduledAt:         now,
StoryStartAt:            now,
CreatedAt:               now,
UpdatedAt:               now,
},
```

In `internal/app/api/match/list_public_upcoming_matches_test.go`, update Summary fixture:

```go
{
UUID:                    uuid.New(),
CampaignUUID:            uuid.New(),
Title:                   "Public Match",
BriefInitialDescription: "Upcoming",
IsPublic:                true,
GameScheduledAt:         now.Add(24 * time.Hour),
StoryStartAt:            now,
CreatedAt:               now,
UpdatedAt:               now,
},
```

#### Part F: Verify and commit

- [ ] **Step 20: Run all tests**

Run: `go test ./... -count=1`
Expected: All tests pass (except known broken match/Turn/Round tests).

- [ ] **Step 21: Build both binaries**

Run: `go build ./cmd/api/ && go build ./cmd/game/`
Expected: Both build successfully.

- [ ] **Step 22: Commit**

```bash
git add -A
git commit -m "refactor: rename GameStartAt semantics + add GameScheduledAt

GameStartAt is now *time.Time — tracks when match actually started.
GameScheduledAt is time.Time (NOT NULL) — tracks when master scheduled.
Migration copies existing game_start_at → game_scheduled_at, nullifies.
All layers updated: entity, domain, gateway, API, tests.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 1: Domain errors

**Files:**
- Modify: `internal/domain/match/error.go`
- Modify: `internal/domain/enrollment/error.go`

- [ ] **Step 1: Add match domain errors**

In `internal/domain/match/error.go`, add after the existing errors:

```go
var (
	ErrMatchNotFound        = domain.NewValidationError(errors.New("match not found"))
	ErrMinTitleLength       = domain.NewValidationError(errors.New("title must be at least 5 characters"))
	ErrMaxTitleLength       = domain.NewValidationError(errors.New("title cannot exceed 32 characters"))
	ErrMinOfStoryStartAt    = domain.NewValidationError(errors.New("story start date must be after campaign start date"))
	ErrMaxOfStoryStartAt    = domain.NewValidationError(errors.New("story start date must be before campaign end date"))
	ErrMinOfGameScheduledAt     = domain.NewValidationError(errors.New("game start at cannot be in the past"))
	ErrMaxOfGameScheduledAt     = domain.NewValidationError(errors.New("game start at cannot be greater than one year from now"))
	ErrMaxBriefDescLength   = domain.NewValidationError(errors.New("brief description cannot exceed 64 characters"))
	ErrMatchAlreadyStarted  = domain.NewValidationError(errors.New("match has already started"))
	ErrMatchAlreadyFinished = domain.NewValidationError(errors.New("match has already finished"))
	ErrNotMatchMaster       = domain.NewValidationError(errors.New("user is not the match master"))
)
```

- [ ] **Step 2: Add enrollment domain errors**

In `internal/domain/enrollment/error.go`, add after existing errors:

```go
var (
	ErrCharacterNotInCampaign   = domain.NewValidationError(errors.New("character sheet does not belong to the match's campaign"))
	ErrCharacterAlreadyEnrolled = domain.NewValidationError(errors.New("character sheet is already enrolled in this match"))
	ErrEnrollmentNotFound       = domain.NewValidationError(errors.New("enrollment not found"))
	ErrNotMatchMaster           = domain.NewValidationError(errors.New("user is not the match's campaign master"))
	ErrMatchAlreadyStarted      = domain.NewValidationError(errors.New("match has already started"))
	ErrMatchAlreadyFinished     = domain.NewValidationError(errors.New("match has already finished"))
	ErrPlayerNotEnrolled        = domain.NewValidationError(errors.New("player is not enrolled in this match"))
)
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/domain/...`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/match/error.go internal/domain/enrollment/error.go
git commit -m "feat(domain): add lobby-related domain errors

Match: ErrMatchAlreadyStarted, ErrMatchAlreadyFinished
Enrollment: ErrMatchAlreadyStarted, ErrMatchAlreadyFinished, ErrPlayerNotEnrolled

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Repository interfaces

**Files:**
- Modify: `internal/domain/match/i_repository.go`
- Modify: `internal/domain/enrollment/i_repository.go`

- [ ] **Step 1: Add StartMatch to match IRepository**

In `internal/domain/match/i_repository.go`, add `StartMatch` method:

```go
type IRepository interface {
	CreateMatch(ctx context.Context, match *match.Match) error
	GetMatch(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
	GetMatchCampaignUUID(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
	StartMatch(ctx context.Context, matchUUID uuid.UUID) error
	ListMatchesByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
	ListPublicUpcomingMatches(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error)
}
```

- [ ] **Step 2: Add new methods to enrollment IRepository**

In `internal/domain/enrollment/i_repository.go`, add 3 new methods:

```go
type IRepository interface {
	EnrollCharacterSheet(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheet(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
	GetEnrollmentByUUID(ctx context.Context, enrollmentUUID uuid.UUID) (status string, matchUUID uuid.UUID, err error)
	AcceptEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error
	RejectEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error
	RejectPendingEnrollments(ctx context.Context, matchUUID uuid.UUID) error
	RejectEnrollmentByPlayerAndMatch(ctx context.Context, playerUUID uuid.UUID, matchUUID uuid.UUID) error
}
```

- [ ] **Step 3: Verify build (expect failures)**

Run: `go build ./...`
Expected: Build FAILS because `MockMatchRepo`, `MockEnrollmentRepo`, and gateway `Repository` types don't implement the new interface methods yet. This is expected — we'll fix it in the next task.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/match/i_repository.go internal/domain/enrollment/i_repository.go
git commit -m "feat(domain): expand repository interfaces for lobby

Match: add StartMatch method
Enrollment: add RejectPendingEnrollments, RejectEnrollmentByPlayerAndMatch

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Mocks — update for new interfaces

**Files:**
- Modify: `internal/domain/testutil/mock_match_repo.go`
- Modify: `internal/domain/testutil/mock_enrollment_repo.go`

- [ ] **Step 1: Add StartMatchFn to MockMatchRepo**

In `internal/domain/testutil/mock_match_repo.go`, add to the struct and add the method:

```go
type MockMatchRepo struct {
	CreateMatchFn               func(ctx context.Context, match *match.Match) error
	GetMatchFn                  func(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
	GetMatchCampaignUUIDFn      func(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
	StartMatchFn                func(ctx context.Context, matchUUID uuid.UUID) error
	ListMatchesByMasterUUIDFn   func(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
	ListPublicUpcomingMatchesFn func(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error)
}
```

Add the method:

```go
func (m *MockMatchRepo) StartMatch(ctx context.Context, matchUUID uuid.UUID) error {
	if m.StartMatchFn != nil {
		return m.StartMatchFn(ctx, matchUUID)
	}
	return nil
}
```

- [ ] **Step 2: Add new Fn fields to MockEnrollmentRepo**

In `internal/domain/testutil/mock_enrollment_repo.go`, add to the struct and add methods:

```go
type MockEnrollmentRepo struct {
	EnrollCharacterSheetFn              func(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheetFn      func(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
	GetEnrollmentByUUIDFn               func(ctx context.Context, enrollmentUUID uuid.UUID) (string, uuid.UUID, error)
	AcceptEnrollmentFn                  func(ctx context.Context, enrollmentUUID uuid.UUID) error
	RejectEnrollmentFn                  func(ctx context.Context, enrollmentUUID uuid.UUID) error
	RejectPendingEnrollmentsFn          func(ctx context.Context, matchUUID uuid.UUID) error
	RejectEnrollmentByPlayerAndMatchFn  func(ctx context.Context, playerUUID uuid.UUID, matchUUID uuid.UUID) error
}
```

Add the methods:

```go
func (m *MockEnrollmentRepo) RejectPendingEnrollments(ctx context.Context, matchUUID uuid.UUID) error {
	if m.RejectPendingEnrollmentsFn != nil {
		return m.RejectPendingEnrollmentsFn(ctx, matchUUID)
	}
	return nil
}

func (m *MockEnrollmentRepo) RejectEnrollmentByPlayerAndMatch(ctx context.Context, playerUUID uuid.UUID, matchUUID uuid.UUID) error {
	if m.RejectEnrollmentByPlayerAndMatchFn != nil {
		return m.RejectEnrollmentByPlayerAndMatchFn(ctx, playerUUID, matchUUID)
	}
	return nil
}
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/domain/...`
Expected: Build succeeds (mocks now implement updated interfaces).

- [ ] **Step 4: Commit**

```bash
git add internal/domain/testutil/mock_match_repo.go internal/domain/testutil/mock_enrollment_repo.go
git commit -m "feat(testutil): update mocks for new repository methods

MockMatchRepo: add StartMatchFn
MockEnrollmentRepo: add RejectPendingEnrollmentsFn, RejectEnrollmentByPlayerAndMatchFn

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Gateway — match (StartMatch + update GetMatch scan)

**Files:**
- Create: `internal/gateway/pg/match/start_match.go`
- Modify: `internal/gateway/pg/match/read_match.go:32-57`

- [ ] **Step 1: Create StartMatch gateway**

Create `internal/gateway/pg/match/start_match.go`:

```go
package match

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) StartMatch(
	ctx context.Context, matchUUID uuid.UUID,
) error {
	const query = `
		UPDATE matches
		SET game_start_at = NOW(), updated_at = NOW()
		WHERE uuid = $1 AND game_start_at IS NULL
	`
	result, err := r.q.Exec(ctx, query, matchUUID)
	if err != nil {
		return fmt.Errorf("failed to start match: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return nil
}
```

Note: Returns `ErrMatchNotFound` when `RowsAffected == 0`. This can mean either the match doesn't exist OR it was already started (`game_start_at IS NULL` condition fails). The UC checks existence beforehand, so at the gateway level this is correct.

- [ ] **Step 2: Update GetMatch query to include game_start_at**

In `internal/gateway/pg/match/read_match.go`, update the `GetMatch` method's SQL query and Scan call to include `game_start_at`:

The SELECT should become:
```sql
SELECT 
    uuid, master_uuid, campaign_uuid,
    title, brief_initial_description, brief_final_description, description,
    is_public, game_scheduled_at, game_start_at,
    story_start_at, story_end_at,
    created_at, updated_at
FROM matches
WHERE uuid = $1
```

And the Scan should include `&m.GameStartAt` after `&m.GameScheduledAt`:
```go
err = tx.QueryRow(ctx, query, uuid).Scan(
    &m.UUID,
    &m.MasterUUID,
    &m.CampaignUUID,
    &m.Title,
    &m.BriefInitialDescription,
    &m.BriefFinalDescription,
    &m.Description,
    &m.IsPublic,
    &m.GameScheduledAt,
    &m.GameStartAt,
    &m.StoryStartAt,
    &m.StoryEndAt,
    &m.CreatedAt,
    &m.UpdatedAt,
)
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/gateway/pg/match/...`
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add internal/gateway/pg/match/start_match.go internal/gateway/pg/match/read_match.go
git commit -m "feat(gateway): add StartMatch and update GetMatch for game_start_at

StartMatch sets game_start_at = NOW() atomically (WHERE game_start_at IS NULL).
GetMatch now includes game_start_at in SELECT and Scan.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Gateway — enrollment (new queries + fix IsPlayerEnrolled)

**Files:**
- Create: `internal/gateway/pg/enrollment/reject_pending_enrollments.go`
- Create: `internal/gateway/pg/enrollment/reject_by_player_and_match.go`
- Modify: `internal/gateway/pg/enrollment/is_player_enrolled.go:13-21`

- [ ] **Step 1: Create RejectPendingEnrollments gateway**

Create `internal/gateway/pg/enrollment/reject_pending_enrollments.go`:

```go
package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) RejectPendingEnrollments(
	ctx context.Context, matchUUID uuid.UUID,
) error {
	const query = `
		UPDATE enrollments
		SET status = 'rejected'
		WHERE match_uuid = $1 AND status = 'pending'
	`
	_, err := r.q.Exec(ctx, query, matchUUID)
	if err != nil {
		return fmt.Errorf("failed to reject pending enrollments: %w", err)
	}
	return nil
}
```

Note: No `RowsAffected` check — it's valid to have zero pending enrollments.

- [ ] **Step 2: Create RejectEnrollmentByPlayerAndMatch gateway**

Create `internal/gateway/pg/enrollment/reject_by_player_and_match.go`:

```go
package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) RejectEnrollmentByPlayerAndMatch(
	ctx context.Context, playerUUID uuid.UUID, matchUUID uuid.UUID,
) error {
	const query = `
		UPDATE enrollments
		SET status = 'rejected'
		WHERE match_uuid = $1
		AND status = 'accepted'
		AND character_sheet_uuid IN (
			SELECT uuid FROM character_sheets
			WHERE player_uuid = $2 OR master_uuid = $2
		)
	`
	result, err := r.q.Exec(ctx, query, matchUUID, playerUUID)
	if err != nil {
		return fmt.Errorf("failed to reject enrollment by player and match: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}
```

- [ ] **Step 3: Fix IsPlayerEnrolledInMatch — add status filter**

In `internal/gateway/pg/enrollment/is_player_enrolled.go`, add `AND e.status = 'accepted'` to the query:

```go
func (r *Repository) IsPlayerEnrolledInMatch(
	ctx context.Context, playerUUID, matchUUID uuid.UUID,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM enrollments e
			JOIN character_sheets cs ON cs.uuid = e.character_sheet_uuid
			WHERE e.match_uuid = $1
			AND (cs.player_uuid = $2 OR cs.master_uuid = $2)
			AND e.status = 'accepted'
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, matchUUID, playerUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if player is enrolled in match: %w", err)
	}
	return exists, nil
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./internal/gateway/pg/enrollment/...`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add internal/gateway/pg/enrollment/reject_pending_enrollments.go \
       internal/gateway/pg/enrollment/reject_by_player_and_match.go \
       internal/gateway/pg/enrollment/is_player_enrolled.go
git commit -m "feat(gateway): add enrollment queries for lobby + fix status filter

New: RejectPendingEnrollments, RejectEnrollmentByPlayerAndMatch
Fix: IsPlayerEnrolledInMatch now requires status = 'accepted'

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: StartMatchUC (TDD)

**Files:**
- Create: `internal/domain/match/start_match.go`
- Create: `internal/domain/match/start_match_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/domain/match/start_match_test.go`:

```go
package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

func TestStartMatch(t *testing.T) {
	masterUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()
	otherUUID := uuid.New()
	now := time.Now()
	finishedAt := now.Add(-time.Hour)

	tests := []struct {
		name       string
		matchUUID  uuid.UUID
		masterUUID uuid.UUID
		matchMock  *testutil.MockMatchRepo
		enrollMock *testutil.MockEnrollmentRepo
		wantErr    error
	}{
		{
			name:       "success",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    nil,
		},
		{
			name:       "match not found",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, matchPg.ErrMatchNotFound
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    domainMatch.ErrMatchNotFound,
		},
		{
			name:       "not match master",
			matchUUID:  matchUUID,
			masterUUID: otherUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    domainMatch.ErrNotMatchMaster,
		},
		{
			name:       "match already started",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
						GameStartAt:    &now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    domainMatch.ErrMatchAlreadyStarted,
		},
		{
			name:       "match already finished",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
						StoryEndAt:   &finishedAt,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    domainMatch.ErrMatchAlreadyFinished,
		},
		{
			name:       "repo error on GetMatch",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, errors.New("db error")
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    errors.New("db error"),
		},
		{
			name:       "repo error on StartMatch",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
					}, nil
				},
				StartMatchFn: func(ctx context.Context, id uuid.UUID) error {
					return errors.New("db error")
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    errors.New("db error"),
		},
		{
			name:       "repo error on RejectPendingEnrollments",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{
				RejectPendingEnrollmentsFn: func(ctx context.Context, id uuid.UUID) error {
					return errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := domainMatch.NewStartMatchUC(tt.matchMock, tt.enrollMock)
			err := uc.Start(context.Background(), tt.matchUUID, tt.masterUUID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/ -run TestStartMatch -v`
Expected: FAIL — `NewStartMatchUC` not found.

- [ ] **Step 3: Write the implementation**

Create `internal/domain/match/start_match.go`:

```go
package match

import (
	"context"
	"errors"

	enrollmentDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IStartMatch interface {
	Start(ctx context.Context, matchUUID uuid.UUID, masterUUID uuid.UUID) error
}

type StartMatchUC struct {
	matchRepo      IRepository
	enrollmentRepo enrollmentDomain.IRepository
}

func NewStartMatchUC(
	matchRepo IRepository,
	enrollmentRepo enrollmentDomain.IRepository,
) *StartMatchUC {
	return &StartMatchUC{
		matchRepo:      matchRepo,
		enrollmentRepo: enrollmentRepo,
	}
}

func (uc *StartMatchUC) Start(
	ctx context.Context,
	matchUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return ErrMatchNotFound
		}
		return err
	}

	if match.MasterUUID != masterUUID {
		return ErrNotMatchMaster
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}
	if match.StoryEndAt != nil {
		return ErrMatchAlreadyFinished
	}

	if err := uc.matchRepo.StartMatch(ctx, matchUUID); err != nil {
		return err
	}

	return uc.enrollmentRepo.RejectPendingEnrollments(ctx, matchUUID)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/ -run TestStartMatch -v`
Expected: All 8 test cases PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/start_match.go internal/domain/match/start_match_test.go
git commit -m "feat(domain): add StartMatchUC with TDD

Validates master ownership, not-already-started, not-finished.
Persists game_start_at then rejects pending enrollments.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 7: KickPlayerUC (TDD)

**Files:**
- Create: `internal/domain/enrollment/kick_player.go`
- Create: `internal/domain/enrollment/kick_player_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/domain/enrollment/kick_player_test.go`:

```go
package enrollment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

func TestKickPlayer(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()
	otherUUID := uuid.New()
	now := time.Now()
	startedAt := now.Add(-time.Hour)

	tests := []struct {
		name       string
		matchUUID  uuid.UUID
		playerUUID uuid.UUID
		masterUUID uuid.UUID
		matchMock  *testutil.MockMatchRepo
		enrollMock *testutil.MockEnrollmentRepo
		wantErr    error
	}{
		{
			name:       "success",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    nil,
		},
		{
			name:       "match not found",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, matchPg.ErrMatchNotFound
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    domainMatch.ErrMatchNotFound,
		},
		{
			name:       "not match master",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: otherUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    enrollment.ErrNotMatchMaster,
		},
		{
			name:       "match already started",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
						GameStartAt:    &startedAt,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    enrollment.ErrMatchAlreadyStarted,
		},
		{
			name:       "cannot kick self (master)",
			matchUUID:  matchUUID,
			playerUUID: masterUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    enrollment.ErrNotMatchMaster,
		},
		{
			name:       "player not enrolled",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{
				RejectEnrollmentByPlayerAndMatchFn: func(ctx context.Context, pUUID uuid.UUID, mUUID uuid.UUID) error {
					return errors.New("enrollment not found in database")
				},
			},
			wantErr: enrollment.ErrPlayerNotEnrolled,
		},
		{
			name:       "repo error on reject",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:         matchUUID,
						MasterUUID:   masterUUID,
						CampaignUUID: campaignUUID,
						GameScheduledAt:  now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{
				RejectEnrollmentByPlayerAndMatchFn: func(ctx context.Context, pUUID uuid.UUID, mUUID uuid.UUID) error {
					return errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := enrollment.NewKickPlayerUC(tt.matchMock, tt.enrollMock)
			err := uc.Kick(context.Background(), tt.matchUUID, tt.playerUUID, tt.masterUUID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/enrollment/ -run TestKickPlayer -v`
Expected: FAIL — `NewKickPlayerUC` not found.

- [ ] **Step 3: Write the implementation**

Create `internal/domain/enrollment/kick_player.go`:

```go
package enrollment

import (
	"context"
	"errors"

	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IKickPlayer interface {
	Kick(ctx context.Context, matchUUID uuid.UUID, playerUUID uuid.UUID, masterUUID uuid.UUID) error
}

type KickPlayerUC struct {
	matchRepo matchDomain.IRepository
	repo      IRepository
}

func NewKickPlayerUC(
	matchRepo matchDomain.IRepository,
	repo IRepository,
) *KickPlayerUC {
	return &KickPlayerUC{
		matchRepo: matchRepo,
		repo:      repo,
	}
}

func (uc *KickPlayerUC) Kick(
	ctx context.Context,
	matchUUID uuid.UUID,
	playerUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return matchDomain.ErrMatchNotFound
		}
		return err
	}

	if match.MasterUUID != masterUUID || playerUUID == masterUUID {
		return ErrNotMatchMaster
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}

	err = uc.repo.RejectEnrollmentByPlayerAndMatch(ctx, playerUUID, matchUUID)
	if err != nil {
		if errors.Is(err, enrollmentPg.ErrEnrollmentNotFound) {
			return ErrPlayerNotEnrolled
		}
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/enrollment/ -run TestKickPlayer -v`
Expected: All 7 test cases PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/enrollment/kick_player.go internal/domain/enrollment/kick_player_test.go
git commit -m "feat(domain): add KickPlayerUC with TDD

Master-only, lobby-only kick. Rejects enrollment by player+match.
Validates ownership, started state, self-kick prevention.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 8: Temporal guard on enrollment UCs

**Files:**
- Modify: `internal/domain/enrollment/accept_enrollment.go:36-73`
- Modify: `internal/domain/enrollment/reject_enrollment.go:36-73`
- Modify: `internal/domain/enrollment/enroll_character_sheet.go:40-83`
- Modify: `internal/domain/enrollment/enrollment_test.go`

- [ ] **Step 1: Add temporal guard to AcceptEnrollmentUC**

In `internal/domain/enrollment/accept_enrollment.go`, replace the `Accept` method body. After the idempotent check (`status == "accepted"`) and before the `GetMatchCampaignUUID` call, add a `GetMatch` call for temporal guard:

```go
func (uc *AcceptEnrollmentUC) Accept(
	ctx context.Context,
	enrollmentUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	status, matchUUID, err := uc.repo.GetEnrollmentByUUID(ctx, enrollmentUUID)
	if err == enrollmentPg.ErrEnrollmentNotFound {
		return ErrEnrollmentNotFound
	}
	if err != nil {
		return err
	}
	if status == "accepted" {
		return nil
	}

	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err == matchPg.ErrMatchNotFound {
		return matchDomain.ErrMatchNotFound
	}
	if err != nil {
		return err
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}
	if match.StoryEndAt != nil {
		return ErrMatchAlreadyFinished
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, match.CampaignUUID)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if campaignMasterUUID != masterUUID {
		return ErrNotMatchMaster
	}

	return uc.repo.AcceptEnrollment(ctx, enrollmentUUID)
}
```

Note: This replaces the `GetMatchCampaignUUID` call with `GetMatch` (which returns the full match including CampaignUUID, GameStartAt, StoryEndAt). We read `match.CampaignUUID` directly instead of a separate `GetMatchCampaignUUID` call.

The imports need to include:
```go
import (
	"context"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)
```

Remove the now-unused `enrollmentPg` import.

- [ ] **Step 2: Add temporal guard to RejectEnrollmentUC**

Apply the same pattern to `internal/domain/enrollment/reject_enrollment.go`:

```go
func (uc *RejectEnrollmentUC) Reject(
	ctx context.Context,
	enrollmentUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	status, matchUUID, err := uc.repo.GetEnrollmentByUUID(ctx, enrollmentUUID)
	if err == enrollmentPg.ErrEnrollmentNotFound {
		return ErrEnrollmentNotFound
	}
	if err != nil {
		return err
	}
	if status == "rejected" {
		return nil
	}

	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err == matchPg.ErrMatchNotFound {
		return matchDomain.ErrMatchNotFound
	}
	if err != nil {
		return err
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}
	if match.StoryEndAt != nil {
		return ErrMatchAlreadyFinished
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, match.CampaignUUID)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if campaignMasterUUID != masterUUID {
		return ErrNotMatchMaster
	}

	return uc.repo.RejectEnrollment(ctx, enrollmentUUID)
}
```

Same import changes as Accept: replace `enrollmentPg` with `matchPg` (for the `ErrMatchNotFound` sentinel).

Wait — actually the Accept UC still uses `enrollmentPg.ErrEnrollmentNotFound` at the top of the function for the `GetEnrollmentByUUID` error. So we need `enrollmentPg` import too. Let me correct: Keep `enrollmentPg` AND add `matchPg`:

```go
import (
	"context"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)
```

This applies to both `accept_enrollment.go` and `reject_enrollment.go`.

- [ ] **Step 3: Add temporal guard to EnrollCharacterInMatchUC**

In `internal/domain/enrollment/enroll_character_sheet.go`, replace `GetMatchCampaignUUID` with `GetMatch` and add temporal guard. The `Enroll` method body becomes:

```go
func (uc *EnrollCharacterInMatchUC) Enroll(
	ctx context.Context,
	matchUUID uuid.UUID,
	sheetUUID uuid.UUID,
	playerUUID uuid.UUID,
) error {
	sheetRelationship, err := uc.sheetRepo.GetCharacterSheetRelationshipUUIDs(
		ctx, sheetUUID,
	)
	if err == sheet.ErrCharacterSheetNotFound {
		return charactersheet.ErrCharacterSheetNotFound
	}
	if err != nil {
		return err
	}
	// TODO: treat if the request was made by a master too
	if sheetRelationship.PlayerUUID == nil ||
		*sheetRelationship.PlayerUUID != playerUUID {
		return charactersheet.ErrNotCharacterSheetOwner
	}

	alreadyEnrolled, err := uc.repo.ExistsEnrolledCharacterSheet(
		ctx, sheetUUID, matchUUID,
	)
	if err != nil {
		return err
	}
	if alreadyEnrolled {
		return ErrCharacterAlreadyEnrolled
	}

	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err == matchPg.ErrMatchNotFound {
		return matchDomain.ErrMatchNotFound
	}
	if err != nil {
		return err
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}
	if match.StoryEndAt != nil {
		return ErrMatchAlreadyFinished
	}

	if sheetRelationship.CampaignUUID == nil ||
		*sheetRelationship.CampaignUUID != match.CampaignUUID {
		return ErrCharacterNotInCampaign
	}
	return uc.repo.EnrollCharacterSheet(ctx, matchUUID, sheetUUID)
}
```

Update imports to add `matchPg` and `matchDomain` (remove unused `enrollmentPg` if applicable):
```go
import (
	"context"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
)
```

- [ ] **Step 4: Add temporal guard test cases to enrollment_test.go**

In `internal/domain/enrollment/enrollment_test.go`, add two new test cases to `TestAcceptEnrollment` (after "idempotent when already accepted"):

```go
{
	name:       "match already started",
	enrollUUID: enrollmentUUID,
	masterUUID: masterUUID,
	enrollMock: &testutil.MockEnrollmentRepo{
		GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
			return "pending", matchUUID, nil
		},
	},
	matchMock: &testutil.MockMatchRepo{
		GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
			startedAt := time.Now()
			return &matchEntity.Match{
				UUID:         matchUUID,
				MasterUUID:   masterUUID,
				CampaignUUID: campaignUUID,
				GameScheduledAt:  time.Now(),
				GameStartAt:    &startedAt,
			}, nil
		},
	},
	campaignMock: &testutil.MockCampaignRepo{},
	wantErr:      enrollment.ErrMatchAlreadyStarted,
},
{
	name:       "match already finished",
	enrollUUID: enrollmentUUID,
	masterUUID: masterUUID,
	enrollMock: &testutil.MockEnrollmentRepo{
		GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
			return "pending", matchUUID, nil
		},
	},
	matchMock: &testutil.MockMatchRepo{
		GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
			finishedAt := time.Now()
			return &matchEntity.Match{
				UUID:         matchUUID,
				MasterUUID:   masterUUID,
				CampaignUUID: campaignUUID,
				GameScheduledAt:  time.Now(),
				StoryEndAt:   &finishedAt,
			}, nil
		},
	},
	campaignMock: &testutil.MockCampaignRepo{},
	wantErr:      enrollment.ErrMatchAlreadyFinished,
},
```

Add the same two test cases to `TestRejectEnrollment` (after "idempotent when already rejected").

Note: The existing test cases that use `GetMatchCampaignUUIDFn` need to be updated to use `GetMatchFn` instead, since the UCs now call `GetMatch` rather than `GetMatchCampaignUUID`. This affects:
- `TestAcceptEnrollment`: "success from pending", "success from rejected", "match not found", "campaign not found", "not campaign master", "repo error on accept"
- `TestRejectEnrollment`: same cases
- `TestEnrollCharacterSheet`: "success", "match not found", "character not in campaign"

For each of those test cases, change:
```go
matchMock: &testutil.MockMatchRepo{
	GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
		return campaignUUID, nil
	},
},
```
to:
```go
matchMock: &testutil.MockMatchRepo{
	GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
		return &matchEntity.Match{
			UUID:         matchUUID,
			MasterUUID:   masterUUID,
			CampaignUUID: campaignUUID,
			GameScheduledAt:  time.Now(),
		}, nil
	},
},
```

And change match-not-found cases from:
```go
matchMock: &testutil.MockMatchRepo{
	GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
		return uuid.Nil, matchPg.ErrMatchNotFound
	},
},
```
to:
```go
matchMock: &testutil.MockMatchRepo{
	GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
		return nil, matchPg.ErrMatchNotFound
	},
},
```

Also add imports at the top of `enrollment_test.go`:
```go
import (
	"time"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/domain/enrollment/ -v`
Expected: All test cases PASS (including new temporal guard tests).

- [ ] **Step 6: Commit**

```bash
git add internal/domain/enrollment/accept_enrollment.go \
       internal/domain/enrollment/reject_enrollment.go \
       internal/domain/enrollment/enroll_character_sheet.go \
       internal/domain/enrollment/enrollment_test.go
git commit -m "feat(domain): add temporal guard to enrollment UCs

Accept/Reject/Enroll now call GetMatch and check GameStartAt/StoryEndAt.
Replaces GetMatchCampaignUUID with GetMatch for efficiency.
Tests updated for new mock pattern + new guard test cases.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 9: Game server — messages and payloads

**Files:**
- Modify: `internal/app/game/message.go`

- [ ] **Step 1: Add kick message types and payloads**

In `internal/app/game/message.go`, add the new message types and payloads:

Add to Server → Client constants:
```go
MsgTypePlayerKicked MessageType = "player_kicked"
```

Add to Client → Server constants:
```go
MsgTypeKickPlayer MessageType = "kick_player"
```

Add new payload structs after `ChatPayload`:
```go
type KickPlayerPayload struct {
	PlayerUUID uuid.UUID `json:"player_uuid"`
}

type PlayerKickedPayload struct {
	UUID     uuid.UUID `json:"uuid"`
	Nickname string    `json:"nickname"`
	Reason   string    `json:"reason"`
}
```

The full constants block should be:
```go
const (
	// Server → Client
	MsgTypeRoomState    MessageType = "room_state"
	MsgTypePlayerJoined MessageType = "player_joined"
	MsgTypePlayerLeft   MessageType = "player_left"
	MsgTypePlayerKicked MessageType = "player_kicked"
	MsgTypeMatchStarted MessageType = "match_started"
	MsgTypeChatMessage  MessageType = "chat_message"
	MsgTypeError        MessageType = "error"

	// Client → Server
	MsgTypeStartMatch MessageType = "start_match"
	MsgTypeKickPlayer MessageType = "kick_player"
	MsgTypeChat       MessageType = "chat"
)
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/app/game/...`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add internal/app/game/message.go
git commit -m "feat(game): add kick player message types and payloads

MsgTypeKickPlayer (client→server), MsgTypePlayerKicked (server→client)
KickPlayerPayload, PlayerKickedPayload structs

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 10: Game server — Room enhancements

**Files:**
- Modify: `internal/app/game/room.go`

This task adds domain use cases to Room, rewrites `StartMatch` to call `StartMatchUC`, adds `KickPlayer` handler, and passes `context.Background()` for DB calls inside the goroutine-based `Run()` loop.

- [ ] **Step 1: Update Room struct with UC dependencies**

Add fields to Room struct and update `NewRoom`:

```go
type Room struct {
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	state      RoomState
	clients    map[uuid.UUID]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	mu         sync.RWMutex

	startMatchUC IStartMatch
	kickPlayerUC IKickPlayer
}
```

Add interfaces at the top of the file (after the `RoomState` constants):

```go
type IStartMatch interface {
	Start(ctx context.Context, matchUUID uuid.UUID, masterUUID uuid.UUID) error
}

type IKickPlayer interface {
	Kick(ctx context.Context, matchUUID uuid.UUID, playerUUID uuid.UUID, masterUUID uuid.UUID) error
}
```

Update `NewRoom`:

```go
func NewRoom(
	matchUUID, masterUUID uuid.UUID,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
) *Room {
	return &Room{
		matchUUID:    matchUUID,
		masterUUID:   masterUUID,
		state:        RoomStateLobby,
		clients:      make(map[uuid.UUID]*Client),
		broadcast:    make(chan []byte, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		stop:         make(chan struct{}),
		startMatchUC: startMatchUC,
		kickPlayerUC: kickPlayerUC,
	}
}
```

Add `"context"` to imports.

- [ ] **Step 2: Rewrite StartMatch to use UC**

Replace the `StartMatch` method:

```go
func (r *Room) StartMatch(userUUID uuid.UUID) error {
	if !r.IsMaster(userUUID) {
		return ErrNotMaster
	}
	r.mu.RLock()
	if r.state != RoomStateLobby {
		r.mu.RUnlock()
		return ErrAlreadyPlaying
	}
	r.mu.RUnlock()

	if err := r.startMatchUC.Start(context.Background(), r.matchUUID, userUUID); err != nil {
		return err
	}

	r.mu.Lock()
	r.state = RoomStatePlaying
	r.mu.Unlock()

	msg := NewServerMessage(MsgTypeMatchStarted, struct{}{})
	data, _ := json.Marshal(msg)
	go func() { r.broadcast <- data }()
	return nil
}
```

- [ ] **Step 3: Add KickPlayer handler**

Add a new method to Room:

```go
func (r *Room) KickPlayer(masterUUID uuid.UUID, playerUUID uuid.UUID) error {
	if !r.IsMaster(masterUUID) {
		return ErrNotMaster
	}

	if err := r.kickPlayerUC.Kick(context.Background(), r.matchUUID, playerUUID, masterUUID); err != nil {
		return err
	}

	r.mu.Lock()
	client, ok := r.clients[playerUUID]
	if ok {
		delete(r.clients, playerUUID)
	}
	r.mu.Unlock()

	if ok {
		kickedMsg := NewServerMessage(MsgTypePlayerKicked, PlayerKickedPayload{
			UUID:     playerUUID,
			Nickname: client.nickname,
			Reason:   "kicked by master",
		})

		client.SendMessage(kickedMsg)
		close(client.send)

		data, _ := json.Marshal(kickedMsg)
		r.mu.RLock()
		for _, c := range r.clients {
			select {
			case c.send <- data:
			default:
			}
		}
		r.mu.RUnlock()
	}
	return nil
}
```

- [ ] **Step 4: Add kick_player to handleClientMessage**

In `handleClientMessage`, add a case for `MsgTypeKickPlayer` after the `MsgTypeStartMatch` case:

```go
case MsgTypeKickPlayer:
	var kickPayload KickPlayerPayload
	if err := json.Unmarshal(incoming.Payload, &kickPayload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid kick payload"))
		return
	}
	if err := r.KickPlayer(client.userUUID, kickPayload.PlayerUUID); err != nil {
		client.SendMessage(NewErrorMessage("forbidden", err.Error()))
	}
```

- [ ] **Step 5: Verify build**

Run: `go build ./internal/app/game/...`
Expected: Build FAILS because `NewRoom` signature changed but `Hub.GetOrCreateRoom` still uses old signature. This is expected — we'll fix in the next task.

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/room.go
git commit -m "feat(game): enhance Room with UC deps, kick, and DB-backed start

Room now accepts StartMatchUC and KickPlayerUC.
StartMatch persists to DB. KickPlayer rejects enrollment + disconnects.
handleClientMessage handles kick_player messages.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 11: Game server — Handler, Hub, and wiring

**Files:**
- Modify: `internal/app/game/handler.go`
- Modify: `internal/app/game/hub.go`
- Modify: `cmd/game/main.go`

- [ ] **Step 1: Update Hub.GetOrCreateRoom signature**

In `internal/app/game/hub.go`, update `GetOrCreateRoom` to accept the new UC dependencies:

```go
func (h *Hub) GetOrCreateRoom(
	matchUUID, masterUUID uuid.UUID,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[matchUUID]; ok {
		return room
	}

	room := NewRoom(matchUUID, masterUUID, startMatchUC, kickPlayerUC)
	h.rooms[matchUUID] = room
	go room.Run()
	return room
}
```

Add `IStartMatch` and `IKickPlayer` references — these are already defined in `room.go` which is in the same package, so no extra imports needed.

- [ ] **Step 2: Update Handler with new dependencies**

In `internal/app/game/handler.go`, add UC fields and update constructor:

```go
type Handler struct {
	hub            *Hub
	matchRepo      MatchRepository
	enrollmentRepo EnrollmentChecker
	startMatchUC   IStartMatch
	kickPlayerUC   IKickPlayer
}

func NewHandler(
	hub *Hub,
	matchRepo MatchRepository,
	enrollmentRepo EnrollmentChecker,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
) *Handler {
	return &Handler{
		hub:            hub,
		matchRepo:      matchRepo,
		enrollmentRepo: enrollmentRepo,
		startMatchUC:   startMatchUC,
		kickPlayerUC:   kickPlayerUC,
	}
}
```

Update the `GetOrCreateRoom` call in `HandleWebSocket`:

```go
room := h.hub.GetOrCreateRoom(matchUUID, masterUUID, h.startMatchUC, h.kickPlayerUC)
```

- [ ] **Step 3: Update cmd/game/main.go wiring**

In `cmd/game/main.go`, construct the UCs and pass them to the handler:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
	"github.com/joho/godotenv"
)

func main() {
	// TODO: evaluate to action — consider config/env loading
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	addr := os.Getenv("GAME_SERVER_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	pgPool, err := pgfs.New(ctx, "")
	if err != nil {
		panic(fmt.Errorf("error creating pg pool: %w", err))
	}
	defer pgPool.Close()

	matchRepository := matchPg.NewRepository(pgPool)
	enrollmentRepository := enrollmentPg.NewRepository(pgPool)

	startMatchUC := domainMatch.NewStartMatchUC(matchRepository, enrollmentRepository)
	kickPlayerUC := enrollment.NewKickPlayerUC(matchRepository, enrollmentRepository)

	hub := game.NewHub()
	handler := game.NewHandler(hub, matchRepository, enrollmentRepository, startMatchUC, kickPlayerUC)
	server := game.NewServer(addr, hub, handler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			log.Printf("game server error: %v", err)
		}
	}()

	// TODO: verify this before game testing with other players
	log.Printf("game server running on %s", addr)
	<-quit
	log.Println("shutting down game server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("game server shutdown error: %v", err)
	}
	log.Println("game server stopped")
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./cmd/game/...`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/handler.go internal/app/game/hub.go cmd/game/main.go
git commit -m "feat(game): wire UCs through Handler → Hub → Room

Handler accepts StartMatchUC + KickPlayerUC, passes through Hub.
cmd/game constructs UCs with shared repositories.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 12: Game server tests — update and expand

**Files:**
- Modify: `internal/app/game/handler_test.go`

- [ ] **Step 1: Update test mocks and setupTestServer**

The `setupTestServer` function needs to pass the new UC dependencies. Since game server tests use local mocks (not testutil), we need local mock UCs:

Add mock UCs at the top of the file:

```go
type mockStartMatchUC struct {
	err error
}

func (m *mockStartMatchUC) Start(_ context.Context, _, _ uuid.UUID) error {
	return m.err
}

type mockKickPlayerUC struct {
	err error
}

func (m *mockKickPlayerUC) Kick(_ context.Context, _, _, _ uuid.UUID) error {
	return m.err
}
```

Update `setupTestServer`:

```go
func setupTestServer(masterUUID uuid.UUID, enrolled bool) (*httptest.Server, *game.Hub) {
	hub := game.NewHub()
	go hub.Run()

	matchRepo := &mockMatchRepo{masterUUID: masterUUID}
	enrollmentRepo := &mockEnrollmentChecker{enrolled: enrolled}
	startUC := &mockStartMatchUC{}
	kickUC := &mockKickPlayerUC{}
	handler := game.NewHandler(hub, matchRepo, enrollmentRepo, startUC, kickUC)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)

	server := httptest.NewServer(mux)
	return server, hub
}
```

- [ ] **Step 2: Add kick player test**

Add a test for kick flow:

```go
func TestKickPlayerFlow(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // player_joined

	kickMsg := game.Message{
		Type:    game.MsgTypeKickPlayer,
		Payload: json.RawMessage(`{"player_uuid":"` + playerUUID.String() + `"}`),
	}
	data, _ := json.Marshal(kickMsg)
	if err := masterConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send kick_player: %v", err)
	}

	playerReceived := readMessage(t, playerConn)
	if playerReceived.Type != game.MsgTypePlayerKicked {
		t.Errorf("expected player_kicked, got %s", playerReceived.Type)
	}

	masterReceived := readMessage(t, masterConn)
	if masterReceived.Type != game.MsgTypePlayerKicked {
		t.Errorf("expected master to get player_kicked, got %s", masterReceived.Type)
	}
}
```

- [ ] **Step 3: Add test for player cannot kick**

```go
func TestPlayerCannotKick(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()
	_ = readMessage(t, playerConn) // room_state

	kickMsg := game.Message{
		Type:    game.MsgTypeKickPlayer,
		Payload: json.RawMessage(`{"player_uuid":"` + masterUUID.String() + `"}`),
	}
	data, _ := json.Marshal(kickMsg)
	if err := playerConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send kick_player: %v", err)
	}

	received := readMessage(t, playerConn)
	if received.Type != game.MsgTypeError {
		t.Errorf("expected error, got %s", received.Type)
	}
}
```

- [ ] **Step 4: Run all game server tests**

Run: `go test ./internal/app/game/ -v`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/handler_test.go
git commit -m "test(game): update tests for new handler signature + add kick tests

Update setupTestServer with mock UCs.
Add TestKickPlayerFlow, TestPlayerCannotKick.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 13: Full test suite verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... -count=1`
Expected: All tests pass. The `match/` turn/round tests may be broken (known issue per AGENTS.md) — ignore those.

- [ ] **Step 2: Build both binaries**

Run: `go build ./cmd/api/ && go build ./cmd/game/`
Expected: Both build successfully.

- [ ] **Step 3: Final commit (if any fixes needed)**

If any test failures required fixes, commit them here. Otherwise, no action needed.

---

### Task 14: Documentation check

- [ ] **Step 1: Run documentation impact check**

Use the `check_documentation_impact` tool against the `main` branch.

- [ ] **Step 2: Address any documentation impacts**

If docs need updating, update them. Otherwise, note "no doc updates needed" in the PR description.

- [ ] **Step 3: Final commit**

```bash
git add -A
git commit -m "docs: update documentation for lobby websocket feature

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Deferred: Sheet Data in Room State (follow-up)

The spec includes personalized character sheet data in `room_state` and `player_joined` payloads (master sees private data, players see base data for others + private for own). This requires:

1. `GetAcceptedEnrollmentsWithSheets(ctx, matchUUID)` gateway query
2. `EnrollmentReader` interface on Room
3. Personalized `sendRoomState()` (master vs player views)
4. Personalized `broadcastPlayerJoined()` per recipient
5. Reusing `CharacterBaseSummaryResponse` / `CharacterPrivateSummaryResponse` models

This is visual enrichment — the core lobby flow (connect → kick → start → persist → temporal guard) works without it. It should be implemented as a follow-up after the core flow is tested locally.
