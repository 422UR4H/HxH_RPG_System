# Design: List Public Upcoming Campaigns

**Date:** 2026-05-06
**Status:** Approved

## Summary

New endpoint `GET /campaigns/public` for players to discover public campaigns to submit their character sheets. Returns public campaigns not owned by the requesting user, ordered by the nearest upcoming match (`game_scheduled_at ASC NULLS LAST`). Campaigns with no future matches appear at the end with `next_game_scheduled_at: null`.

## Motivation

The existing `GET /campaigns` lists only campaigns where the user is master. Players need a discovery endpoint to find public campaigns and submit their sheets.

## Architecture

Follows the existing campaign listing slice exactly:

```
entity/campaign/public_summary.go     ← extends Summary with NextGameScheduledAt *time.Time
domain/campaign/i_repository.go       ← adds ListPublicUpcomingCampaigns to IRepository
domain/campaign/list_public_upcoming_campaigns.go  ← UC + IListPublicUpcomingCampaigns interface
gateway/pg/campaign/read_campaign.go  ← adds ListPublicUpcomingCampaigns method
migrations/<timestamp>_idx_matches_campaign_uuid_game_scheduled_at.sql  ← DB index
app/api/campaign/list_public_upcoming_campaigns.go  ← handler + response types
app/api/campaign/routes.go            ← GET /campaigns/public
cmd/api/main.go                       ← wiring
```

## Data Model

### Entity extension

```go
// entity/campaign/public_summary.go
type PublicSummary struct {
    Summary
    NextGameScheduledAt *time.Time
}
```

`Summary` is unchanged. `NextGameScheduledAt` is nil for campaigns with no future matches.

## Database

### Query (gateway)

```sql
WITH next_match AS (
    SELECT DISTINCT ON (campaign_uuid)
        campaign_uuid, game_scheduled_at
    FROM matches
    WHERE game_scheduled_at > $2
    ORDER BY campaign_uuid, game_scheduled_at ASC
)
SELECT
    c.uuid, c.scenario_uuid,
    c.name, COALESCE(c.brief_initial_description, ''), c.brief_final_description,
    c.is_public, c.call_link,
    c.story_start_at, c.story_current_at, c.story_end_at,
    c.created_at, c.updated_at,
    nm.game_scheduled_at
FROM campaigns c
LEFT JOIN next_match nm ON nm.campaign_uuid = c.uuid
WHERE c.is_public = true
  AND c.master_uuid != $1
ORDER BY nm.game_scheduled_at ASC NULLS LAST
```

- `$1` = requesting user UUID (excludes own campaigns)
- `$2` = `time.Now()` passed from UC (per gateway-conventions: no `NOW()` in SQL)

### Index (migration)

```sql
-- +goose Up
CREATE INDEX idx_matches_campaign_uuid_game_scheduled_at
    ON matches(campaign_uuid, game_scheduled_at ASC);

-- +goose Down
DROP INDEX IF EXISTS idx_matches_campaign_uuid_game_scheduled_at;
```

Supports the `DISTINCT ON (campaign_uuid) ORDER BY campaign_uuid, game_scheduled_at ASC` in the CTE.

## Domain Use Case

```go
type IListPublicUpcomingCampaigns interface {
    ListPublicUpcomingCampaigns(ctx context.Context, userUUID uuid.UUID) ([]*campaignEntity.PublicSummary, error)
}
```

The UC calls `time.Now()` and delegates to the repo, consistent with `ListPublicUpcomingMatchesUC`.

## HTTP Layer

**Route:** `GET /public/campaigns` (consistent with existing `GET /public/matches`)
**Auth:** required (user UUID from context)
**Tags:** `campaigns`

### Response types

```go
type PublicCampaignSummaryResponse struct {
    CampaignSummaryResponse
    NextGameScheduledAt *string `json:"next_game_scheduled_at,omitempty"`
}

type ListPublicCampaignsResponseBody struct {
    Campaigns []PublicCampaignSummaryResponse `json:"campaigns"`
}

type ListPublicCampaignsResponse struct {
    Body ListPublicCampaignsResponseBody `json:"body"`
}
```

`PublicCampaignSummaryResponse` embeds the existing `CampaignSummaryResponse` unchanged. `NextGameScheduledAt` is formatted as `time.RFC3339` when non-nil (consistent with `game_scheduled_at` in `MatchSummaryResponse`), omitted otherwise.

## Testing

### Handler unit test (`list_public_upcoming_campaigns_test.go`)

Cases:
- `success_with_campaigns` — returns campaigns ordered, next_game_scheduled_at populated
- `success_campaigns_without_future_match` — next_game_scheduled_at is null/omitted
- `success_empty_list` — empty array
- `internal_server_error` — 500 on repo failure

Mock added to `mocks_test.go`: `mockListPublicUpcomingCampaigns`.

### Gateway integration test (`campaign_integration_test.go`)

Cases:
- `returns campaigns with future matches ordered asc` — two campaigns, verifies order
- `campaigns without future match appear last with nil scheduled_at`
- `excludes campaigns owned by requesting user`
- `excludes non-public campaigns`
- `empty` — no public campaigns from other users

## Wiring (`cmd/api/main.go`)

```go
listPublicUpcomingCampaignsUC := domainCampaign.NewListPublicUpcomingCampaignsUC(campaignRepo)

campaignsApi := campaignHandler.Api{
    // ... existing handlers ...
    ListPublicUpcomingCampaignsHandler: campaignHandler.ListPublicUpcomingCampaignsHandler(listPublicUpcomingCampaignsUC),
}
```
