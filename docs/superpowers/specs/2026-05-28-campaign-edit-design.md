# Campaign Edit — Design Spec

**Date:** 2026-05-28  
**Status:** approved

## Context

Implement end-to-end campaign editing, mirroring the existing match edit flow. Campaign editing has two runtime modes determined by whether any match in the campaign has started (`game_start_at IS NOT NULL`).

## Modes

| Mode | Trigger | Editable fields |
|------|---------|----------------|
| **Free** | No match started | `name`, `story_start_at`, `story_current_at` (free) + always-editable |
| **Restricted** | At least one match started | `story_current_at` (no regression) + always-editable |

**Always editable:** `brief_initial_description`, `description`, `is_public`, `call_link`

**Never editable via this endpoint:** `brief_final_description`, `story_end_at` (system-set on campaign close)

### `story_current_at` constraint in restricted mode

- If current value is `NULL`: free to set any value
- If current value is not `NULL`: new value must be `>= current story_current_at` (cannot rewind)
- `story_end_at` of matches is NOT a constraint — master may keep campaign date behind matches

## Architecture

Single `PATCH /campaigns/{uuid}` endpoint. The UC detects the mode internally and applies the appropriate validation. Frontend preemptively hides/disables mode-locked fields to avoid unnecessary requests.

## Backend

### `campaign.IRepository` — new methods

```go
UpdateCampaign(ctx context.Context, campaign *campaign.Campaign) error
GetCampaignForUpdate(ctx context.Context, uuid uuid.UUID) (*CampaignUpdateContext, error)
```

`CampaignUpdateContext` struct (fetched in a single query with EXISTS subquery):
```go
type CampaignUpdateContext struct {
    MasterUUID      uuid.UUID
    StoryCurrentAt  *time.Time
    StoryEndAt      *time.Time
    HasStartedMatch bool
}
```

SQL pattern for `GetCampaignForUpdate`:
```sql
SELECT master_uuid, story_current_at, story_end_at,
       EXISTS(SELECT 1 FROM matches WHERE campaign_uuid = $1 AND game_start_at IS NOT NULL)
FROM campaigns WHERE uuid = $1
```

### Use Case: `update_campaign.go`

```go
type UpdateCampaignInput struct {
    CampaignUUID uuid.UUID
    MasterUUID   uuid.UUID
    // Always editable
    BriefInitialDescription *string
    Description             *string
    IsPublic                *bool
    CallLink                *string
    StoryCurrentAt          *time.Time
    // Free mode only
    Name         *string
    StoryStartAt *time.Time
}
```

Logic:
1. `GetCampaignForUpdate` → check ownership (`ErrNotCampaignOwner`)
2. If `StoryEndAt != nil` → `ErrCampaignAlreadyEnded`
3. If `HasStartedMatch && (Name != nil || StoryStartAt != nil)` → `ErrLockedAfterMatchStart`
4. If `StoryCurrentAt != nil && currentStoryCurrentAt != nil && *StoryCurrentAt < *currentStoryCurrentAt` → `ErrCannotRegressStoryCurrentAt`
5. Validate: name 5–32 chars, brief desc ≤255, call_link ≤255
6. Apply patch in-memory, `updated_at = time.Now()`
7. `UpdateCampaign`
8. Fetch and return full `*campaign.Campaign` via `GetCampaign` for rich response

### New Errors (`campaign/error.go`)

```go
ErrCampaignAlreadyEnded        // campaign story_end_at is set
ErrLockedAfterMatchStart       // name/story_start_at sent in restricted mode
ErrCannotRegressStoryCurrentAt // story_current_at would go back in time
```

### Gateway: `update_campaign.go`

```sql
UPDATE campaigns SET
    name = $1, brief_initial_description = $2, description = $3,
    is_public = $4, call_link = $5,
    story_start_at = $6, story_current_at = $7, updated_at = $8
WHERE uuid = $9
```

No `WHERE story_end_at IS NULL` guard — UC validates before calling. 0 rows affected → `ErrCampaignNotFound`.

### Handler: PATCH `/campaigns/{uuid}`

Request body (all optional):
```json
{
  "name": "...",
  "brief_initial_description": "...",
  "description": "...",
  "is_public": true,
  "call_link": "...",
  "story_start_at": "YYYY-MM-DD",
  "story_current_at": "ISO 8601"
}
```

Date parsing mirrors match handler: `story_start_at` → `time.Parse("2006-01-02", ...)`, `story_current_at` → `time.Parse(time.RFC3339, ...)`.

Response: `CampaignMasterResponse` (reuses existing type via `GetCampaign` after update).

Error mapping:
| UC error | HTTP |
|---|---|
| `ErrCampaignNotFound` | 404 |
| `ErrNotCampaignOwner` | 403 |
| `ErrCampaignAlreadyEnded` | 422 |
| `ErrLockedAfterMatchStart` | 422 |
| `ErrCannotRegressStoryCurrentAt` | 422 |
| `domain.ErrValidation` | 422 |
| other | 500 |

### Tests

- Handler unit tests (humatest): mirror `update_match_test.go` — success full patch, success partial patch, success empty body noop, invalid date formats, each UC error → correct HTTP code
- UC unit tests: ownership, ended campaign, locked fields in restricted mode, `story_current_at` regression, validations
- Gateway integration test: `TestUpdateCampaign` in `campaign_integration_test.go`
- `GetCampaignForUpdate` integration test: has_started_match true/false

## Frontend

### `campaignService.ts`

```ts
updateCampaign: (token: string, id: string, data: object): Promise<CampaignMaster> =>
  httpClient
    .patch<{ campaign: CampaignMaster }>(`/campaigns/${id}`, objToSnakeCase(data), config(token))
    .then(({ data }) => objToCamelCase<CampaignMaster>(data.campaign)),
```

### `useUpdateCampaign.ts`

```ts
export function useUpdateCampaign(token: string | null, campaignId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: object) => campaignService.updateCampaign(token!, campaignId!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["campaignDetails", token, campaignId] });
    },
  });
}
```

### `EditCampaignPage.tsx`

Route: `/campaigns/:id/edit` (already wired in `CampaignPage.handleEdit`).

Behavior:
- Fetches campaign via `useCampaignDetails`
- Redirects if not master or if `campaign.storyEndAt` is set
- Computes `hasStartedMatch = campaign.matches?.some(m => !!m.gameStartAt)`
- **Free mode**: shows all fields (name, story_start_at, story_current_at free + always-editable)
- **Restricted mode**: hides name and story_start_at; story_current_at visible with `min` attribute set to current value (enforces non-regression client-side)
- On submit: sends only changed fields as optional payload (or all fields — to align with match pattern)
- `story_start_at`: `type="date"` (YYYY-MM-DD)
- `story_current_at`: `type="datetime-local"` (ISO 8601)

### `features/campaign/campaignErrorMessages.ts`

Maps API error detail strings to PT-BR user messages (mirrors `matchErrorMessages.ts`).

### `App.tsx`

```tsx
<Route path="/campaigns/:id/edit" element={<EditCampaignPage />} />
```

## API Contract

Separate file: `docs/dev/api/update-campaign.md`
