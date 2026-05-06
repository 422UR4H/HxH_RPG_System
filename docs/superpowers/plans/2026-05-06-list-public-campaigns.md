# List Public Upcoming Campaigns — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /public/campaigns` that returns public campaigns from other masters ordered by nearest upcoming match (`game_scheduled_at ASC NULLS LAST`), so players can discover campaigns to submit their character sheets.

**Architecture:** New slice following the existing `ListPublicUpcomingMatches` pattern. Extends `campaign.Summary` with a new `PublicSummary` entity that adds the optional `NextGameScheduledAt *time.Time`. The gateway uses a CTE with `DISTINCT ON` to find the nearest upcoming match per campaign, joined via `LEFT JOIN` so campaigns with no future match appear last.

**Tech Stack:** Go 1.23, pgx/v5, huma/v2, humatest, goose migrations.

---

## File Map

| Action | File |
|--------|------|
| Create | `internal/domain/entity/campaign/public_summary.go` |
| Modify | `internal/domain/campaign/i_repository.go` |
| Modify | `internal/domain/testutil/mock_campaign_repo.go` |
| Create | `internal/domain/campaign/list_public_upcoming_campaigns.go` |
| Modify | `internal/domain/campaign/campaign_test.go` |
| Create | `migrations/<timestamp>_idx_matches_campaign_uuid_game_scheduled_at.sql` |
| Modify | `internal/gateway/pg/campaign/read_campaign.go` |
| Modify | `internal/gateway/pg/campaign/campaign_integration_test.go` |
| Create | `internal/app/api/campaign/list_public_upcoming_campaigns.go` |
| Modify | `internal/app/api/campaign/mocks_test.go` |
| Create | `internal/app/api/campaign/list_public_upcoming_campaigns_test.go` |
| Modify | `internal/app/api/campaign/routes.go` |
| Modify | `cmd/api/main.go` |

---

## Task 1: Entity — `PublicSummary`

**Files:**
- Create: `internal/domain/entity/campaign/public_summary.go`

- [ ] **Step 1: Create the file**

```go
package campaign

import "time"

type PublicSummary struct {
	Summary
	NextGameScheduledAt *time.Time
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/domain/entity/campaign/...
```

Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/campaign/public_summary.go
git commit -m "feat: add PublicSummary entity extending campaign Summary with NextGameScheduledAt

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: Repository interface + testutil mock

**Files:**
- Modify: `internal/domain/campaign/i_repository.go`
- Modify: `internal/domain/testutil/mock_campaign_repo.go`

- [ ] **Step 1: Add method to `IRepository`**

In `internal/domain/campaign/i_repository.go`, add the new method to the interface:

```go
package campaign

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateCampaign(ctx context.Context, campaign *campaign.Campaign) error
	GetCampaign(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	GetCampaignMasterUUID(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCampaignStoryDates(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	CountCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) (int, error)
	ListCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*campaign.Summary, error)
	ListPublicUpcomingCampaigns(ctx context.Context, after time.Time, userUUID uuid.UUID) ([]*campaign.PublicSummary, error)
}
```

- [ ] **Step 2: Add method to `MockCampaignRepo`**

In `internal/domain/testutil/mock_campaign_repo.go`, add the new field and method:

```go
package testutil

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type MockCampaignRepo struct {
	CreateCampaignFn                  func(ctx context.Context, campaign *campaign.Campaign) error
	GetCampaignFn                     func(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	GetCampaignMasterUUIDFn           func(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCampaignStoryDatesFn           func(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	CountCampaignsByMasterUUIDFn      func(ctx context.Context, masterUUID uuid.UUID) (int, error)
	ListCampaignsByMasterUUIDFn       func(ctx context.Context, masterUUID uuid.UUID) ([]*campaign.Summary, error)
	ListPublicUpcomingCampaignsFn     func(ctx context.Context, after time.Time, userUUID uuid.UUID) ([]*campaign.PublicSummary, error)
}

func (m *MockCampaignRepo) CreateCampaign(ctx context.Context, c *campaign.Campaign) error {
	if m.CreateCampaignFn != nil {
		return m.CreateCampaignFn(ctx, c)
	}
	return nil
}

func (m *MockCampaignRepo) GetCampaign(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	if m.GetCampaignFn != nil {
		return m.GetCampaignFn(ctx, id)
	}
	return nil, nil
}

func (m *MockCampaignRepo) GetCampaignMasterUUID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	if m.GetCampaignMasterUUIDFn != nil {
		return m.GetCampaignMasterUUIDFn(ctx, id)
	}
	return uuid.Nil, nil
}

func (m *MockCampaignRepo) GetCampaignStoryDates(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	if m.GetCampaignStoryDatesFn != nil {
		return m.GetCampaignStoryDatesFn(ctx, id)
	}
	return nil, nil
}

func (m *MockCampaignRepo) CountCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) (int, error) {
	if m.CountCampaignsByMasterUUIDFn != nil {
		return m.CountCampaignsByMasterUUIDFn(ctx, masterUUID)
	}
	return 0, nil
}

func (m *MockCampaignRepo) ListCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*campaign.Summary, error) {
	if m.ListCampaignsByMasterUUIDFn != nil {
		return m.ListCampaignsByMasterUUIDFn(ctx, masterUUID)
	}
	return nil, nil
}

func (m *MockCampaignRepo) ListPublicUpcomingCampaigns(ctx context.Context, after time.Time, userUUID uuid.UUID) ([]*campaign.PublicSummary, error) {
	if m.ListPublicUpcomingCampaignsFn != nil {
		return m.ListPublicUpcomingCampaignsFn(ctx, after, userUUID)
	}
	return nil, nil
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./internal/domain/...
```

Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add internal/domain/campaign/i_repository.go internal/domain/testutil/mock_campaign_repo.go
git commit -m "feat: add ListPublicUpcomingCampaigns to campaign IRepository and mock

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3: Domain use case (TDD)

**Files:**
- Create: `internal/domain/campaign/list_public_upcoming_campaigns.go`
- Modify: `internal/domain/campaign/campaign_test.go`

- [ ] **Step 1: Write the failing test**

Add `TestListPublicUpcomingCampaigns` at the bottom of `internal/domain/campaign/campaign_test.go`:

```go
func TestListPublicUpcomingCampaigns(t *testing.T) {
	userUUID := uuid.New()

	tests := []struct {
		name    string
		mock    *testutil.MockCampaignRepo
		wantErr error
		wantLen int
	}{
		{
			name: "success with results",
			mock: &testutil.MockCampaignRepo{
				ListPublicUpcomingCampaignsFn: func(ctx context.Context, after time.Time, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
					return []*campaignEntity.PublicSummary{
						{Summary: campaignEntity.Summary{Name: "C1"}},
						{Summary: campaignEntity.Summary{Name: "C2"}},
					}, nil
				},
			},
			wantLen: 2,
		},
		{
			name: "success empty",
			mock: &testutil.MockCampaignRepo{
				ListPublicUpcomingCampaignsFn: func(ctx context.Context, after time.Time, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
					return []*campaignEntity.PublicSummary{}, nil
				},
			},
			wantLen: 0,
		},
		{
			name: "repo error",
			mock: &testutil.MockCampaignRepo{
				ListPublicUpcomingCampaignsFn: func(ctx context.Context, after time.Time, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := campaign.NewListPublicUpcomingCampaignsUC(tt.mock)
			result, err := uc.ListPublicUpcomingCampaigns(context.Background(), userUUID)

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
			if len(result) != tt.wantLen {
				t.Errorf("expected %d results, got %d", tt.wantLen, len(result))
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/domain/campaign/... -run TestListPublicUpcomingCampaigns -v
```

Expected: FAIL — `campaign.NewListPublicUpcomingCampaignsUC undefined`.

- [ ] **Step 3: Implement the use case**

Create `internal/domain/campaign/list_public_upcoming_campaigns.go`:

```go
package campaign

import (
	"context"
	"time"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type IListPublicUpcomingCampaigns interface {
	ListPublicUpcomingCampaigns(ctx context.Context, userUUID uuid.UUID) ([]*campaignEntity.PublicSummary, error)
}

type ListPublicUpcomingCampaignsUC struct {
	repo IRepository
}

func NewListPublicUpcomingCampaignsUC(repo IRepository) *ListPublicUpcomingCampaignsUC {
	return &ListPublicUpcomingCampaignsUC{repo: repo}
}

func (uc *ListPublicUpcomingCampaignsUC) ListPublicUpcomingCampaigns(
	ctx context.Context, userUUID uuid.UUID,
) ([]*campaignEntity.PublicSummary, error) {
	now := time.Now()
	return uc.repo.ListPublicUpcomingCampaigns(ctx, now, userUUID)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/domain/campaign/... -run TestListPublicUpcomingCampaigns -v
```

Expected: PASS for all 3 sub-tests.

- [ ] **Step 5: Run full domain/campaign tests to check for regressions**

```bash
go test ./internal/domain/campaign/...
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/campaign/list_public_upcoming_campaigns.go internal/domain/campaign/campaign_test.go
git commit -m "feat: add ListPublicUpcomingCampaigns use case

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: Migration — DB index

**Files:**
- Create: `migrations/<timestamp>_idx_matches_campaign_uuid_game_scheduled_at.sql`

- [ ] **Step 1: Create the migration file**

```bash
make migrate-create name=idx_matches_campaign_uuid_game_scheduled_at
```

Expected: creates `migrations/<timestamp>_idx_matches_campaign_uuid_game_scheduled_at.sql`.

- [ ] **Step 2: Fill in the migration SQL**

Replace the generated file content with:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE INDEX idx_matches_campaign_uuid_game_scheduled_at
    ON matches(campaign_uuid, game_scheduled_at ASC);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_matches_campaign_uuid_game_scheduled_at;

COMMIT;
-- +goose StatementEnd
```

- [ ] **Step 3: Apply the migration (requires local DB)**

```bash
make migrate-up
```

Expected: `OK   <timestamp>_idx_matches_campaign_uuid_game_scheduled_at.sql`.

- [ ] **Step 4: Commit**

```bash
git add migrations/
git commit -m "feat: add index on matches(campaign_uuid, game_scheduled_at) for public campaign listing

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5: Gateway method (TDD — integration)

**Files:**
- Modify: `internal/gateway/pg/campaign/read_campaign.go`
- Modify: `internal/gateway/pg/campaign/campaign_integration_test.go`

- [ ] **Step 1: Write the failing integration test**

Add `TestListPublicUpcomingCampaigns` and a local helper `insertMatchWithScheduledAt` to `internal/gateway/pg/campaign/campaign_integration_test.go`:

```go
func insertMatchWithScheduledAt(
	t *testing.T, pool *pgxpool.Pool,
	masterUUID, campaignUUID uuid.UUID,
	title string, scheduledAt time.Time,
) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO matches (master_uuid, campaign_uuid, title, game_scheduled_at, story_start_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		masterUUID, campaignUUID, title, scheduledAt, time.Now(),
	)
	if err != nil {
		t.Fatalf("insertMatchWithScheduledAt: %v", err)
	}
}

func TestListPublicUpcomingCampaigns(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()
	now := time.Now()

	t.Run("returns campaigns ordered by nearest future match asc", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		requester := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "req", "req@hunter.com", "pass"))
		master := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm", "gm@hunter.com", "pass"))

		cFar := newTestCampaign(master, nil, "Far Campaign")
		cNear := newTestCampaign(master, nil, "Near Campaign")
		if err := repo.CreateCampaign(ctx, cFar); err != nil {
			t.Fatalf("CreateCampaign(cFar): %v", err)
		}
		if err := repo.CreateCampaign(ctx, cNear); err != nil {
			t.Fatalf("CreateCampaign(cNear): %v", err)
		}
		insertMatchWithScheduledAt(t, pool, master, cFar.UUID, "Far Match", now.Add(48*time.Hour))
		insertMatchWithScheduledAt(t, pool, master, cNear.UUID, "Near Match", now.Add(24*time.Hour))

		list, err := repo.ListPublicUpcomingCampaigns(ctx, now, requester)
		if err != nil {
			t.Fatalf("ListPublicUpcomingCampaigns() unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 campaigns, got %d", len(list))
		}
		if list[0].UUID != cNear.UUID {
			t.Errorf("list[0] = %v, want Near Campaign %v", list[0].UUID, cNear.UUID)
		}
		if list[1].UUID != cFar.UUID {
			t.Errorf("list[1] = %v, want Far Campaign %v", list[1].UUID, cFar.UUID)
		}
		if list[0].NextGameScheduledAt == nil {
			t.Error("list[0].NextGameScheduledAt is nil, want non-nil")
		}
	})

	t.Run("campaign without future match appears last with nil next_game_scheduled_at", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		requester := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "req", "req@hunter.com", "pass"))
		master := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm", "gm@hunter.com", "pass"))

		cWithMatch := newTestCampaign(master, nil, "Has Future Match")
		cNoMatch := newTestCampaign(master, nil, "No Future Match")
		if err := repo.CreateCampaign(ctx, cWithMatch); err != nil {
			t.Fatalf("CreateCampaign(cWithMatch): %v", err)
		}
		if err := repo.CreateCampaign(ctx, cNoMatch); err != nil {
			t.Fatalf("CreateCampaign(cNoMatch): %v", err)
		}
		insertMatchWithScheduledAt(t, pool, master, cWithMatch.UUID, "Future Match", now.Add(24*time.Hour))
		insertMatchWithScheduledAt(t, pool, master, cNoMatch.UUID, "Past Match", now.Add(-24*time.Hour))

		list, err := repo.ListPublicUpcomingCampaigns(ctx, now, requester)
		if err != nil {
			t.Fatalf("ListPublicUpcomingCampaigns() unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 campaigns, got %d", len(list))
		}
		if list[0].UUID != cWithMatch.UUID {
			t.Errorf("list[0] = %v, want %v (campaign with future match)", list[0].UUID, cWithMatch.UUID)
		}
		if list[0].NextGameScheduledAt == nil {
			t.Error("list[0].NextGameScheduledAt is nil, want non-nil")
		}
		if list[1].UUID != cNoMatch.UUID {
			t.Errorf("list[1] = %v, want %v (campaign without future match)", list[1].UUID, cNoMatch.UUID)
		}
		if list[1].NextGameScheduledAt != nil {
			t.Errorf("list[1].NextGameScheduledAt = %v, want nil", list[1].NextGameScheduledAt)
		}
	})

	t.Run("excludes campaigns owned by requesting user", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		requester := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "req", "req@hunter.com", "pass"))
		master := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm", "gm@hunter.com", "pass"))

		cOwn := newTestCampaign(requester, nil, "Own Campaign")
		cOther := newTestCampaign(master, nil, "Other Campaign")
		if err := repo.CreateCampaign(ctx, cOwn); err != nil {
			t.Fatalf("CreateCampaign(cOwn): %v", err)
		}
		if err := repo.CreateCampaign(ctx, cOther); err != nil {
			t.Fatalf("CreateCampaign(cOther): %v", err)
		}

		list, err := repo.ListPublicUpcomingCampaigns(ctx, now, requester)
		if err != nil {
			t.Fatalf("ListPublicUpcomingCampaigns() unexpected error: %v", err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 campaign, got %d", len(list))
		}
		if list[0].UUID != cOther.UUID {
			t.Errorf("list[0] = %v, want Other Campaign %v", list[0].UUID, cOther.UUID)
		}
	})

	t.Run("excludes non-public campaigns", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		requester := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "req", "req@hunter.com", "pass"))
		master := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm", "gm@hunter.com", "pass"))

		cPrivate := newTestCampaign(master, nil, "Private Campaign")
		cPrivate.IsPublic = false
		if err := repo.CreateCampaign(ctx, cPrivate); err != nil {
			t.Fatalf("CreateCampaign(cPrivate): %v", err)
		}

		list, err := repo.ListPublicUpcomingCampaigns(ctx, now, requester)
		if err != nil {
			t.Fatalf("ListPublicUpcomingCampaigns() unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("expected empty list, got %d campaigns", len(list))
		}
	})

	t.Run("empty when no public campaigns from other users", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		requester := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "req", "req@hunter.com", "pass"))

		list, err := repo.ListPublicUpcomingCampaigns(ctx, now, requester)
		if err != nil {
			t.Fatalf("ListPublicUpcomingCampaigns() unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("expected empty list, got %d campaigns", len(list))
		}
	})
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
go test -tags=integration ./internal/gateway/pg/campaign/... -run TestListPublicUpcomingCampaigns -v
```

Expected: FAIL — `repo.ListPublicUpcomingCampaigns undefined`.

- [ ] **Step 3: Implement the gateway method**

Add to the bottom of `internal/gateway/pg/campaign/read_campaign.go`:

```go
func (r *Repository) ListPublicUpcomingCampaigns(
	ctx context.Context, after time.Time, userUUID uuid.UUID,
) ([]*campaign.PublicSummary, error) {

	tx, err := r.q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	const query = `
		WITH next_match AS (
			SELECT DISTINCT ON (campaign_uuid)
				campaign_uuid,
				game_scheduled_at
			FROM matches
			WHERE game_scheduled_at > $1
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
		  AND c.master_uuid != $2
		ORDER BY nm.game_scheduled_at ASC NULLS LAST
	`
	rows, err := tx.Query(ctx, query, after, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch public upcoming campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []*campaign.PublicSummary
	for rows.Next() {
		var c campaign.PublicSummary
		err := rows.Scan(
			&c.UUID,
			&c.ScenarioUUID,
			&c.Name,
			&c.BriefInitialDescription,
			&c.BriefFinalDescription,
			&c.IsPublic,
			&c.CallLink,
			&c.StoryStartAt,
			&c.StoryCurrentAt,
			&c.StoryEndAt,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.NextGameScheduledAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan public campaign summary: %w", err)
		}
		campaigns = append(campaigns, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over public campaigns: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return campaigns, nil
}
```

Note: `time` is already imported; also import `"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"` is already the `campaign` alias in that file — verify the existing imports cover `time` and `uuid`.

- [ ] **Step 4: Run integration tests to verify they pass**

```bash
go test -tags=integration ./internal/gateway/pg/campaign/... -run TestListPublicUpcomingCampaigns -v
```

Expected: all 5 sub-tests PASS.

- [ ] **Step 5: Run all gateway campaign integration tests**

```bash
go test -tags=integration ./internal/gateway/pg/campaign/...
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/gateway/pg/campaign/read_campaign.go internal/gateway/pg/campaign/campaign_integration_test.go
git commit -m "feat: implement ListPublicUpcomingCampaigns gateway method with integration tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6: HTTP handler (TDD — unit)

**Files:**
- Create: `internal/app/api/campaign/list_public_upcoming_campaigns.go`
- Modify: `internal/app/api/campaign/mocks_test.go`
- Create: `internal/app/api/campaign/list_public_upcoming_campaigns_test.go`

- [ ] **Step 1: Add mock to `mocks_test.go`**

Append to `internal/app/api/campaign/mocks_test.go`:

```go
type mockListPublicUpcomingCampaigns struct {
	fn func(ctx context.Context, userUUID uuid.UUID) ([]*campaignEntity.PublicSummary, error)
}

func (m *mockListPublicUpcomingCampaigns) ListPublicUpcomingCampaigns(ctx context.Context, userUUID uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
	return m.fn(ctx, userUUID)
}
```

Also add the import for `PublicSummary` — `campaignEntity` is already imported in that file as `"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"`.

- [ ] **Step 2: Write the failing test**

Create `internal/app/api/campaign/list_public_upcoming_campaigns_test.go`:

```go
package campaign_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestListPublicUpcomingCampaignsHandler(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()
	nextScheduled := now.Add(24 * time.Hour)

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error)
		wantStatus int
		wantCount  int
	}{
		{
			name: "success_with_upcoming_match",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
				return []*campaignEntity.PublicSummary{
					{
						Summary: campaignEntity.Summary{
							UUID:                    uuid.New(),
							Name:                    "Public Campaign",
							BriefInitialDescription: "Brief",
							IsPublic:                true,
							CallLink:                "https://meet.example.com/1",
							StoryStartAt:            now,
							CreatedAt:               now,
							UpdatedAt:               now,
						},
						NextGameScheduledAt: &nextScheduled,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name: "success_without_future_match",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
				return []*campaignEntity.PublicSummary{
					{
						Summary: campaignEntity.Summary{
							UUID:                    uuid.New(),
							Name:                    "No Schedule Campaign",
							BriefInitialDescription: "Brief",
							IsPublic:                true,
							CallLink:                "https://meet.example.com/2",
							StoryStartAt:            now,
							CreatedAt:               now,
							UpdatedAt:               now,
						},
						NextGameScheduledAt: nil,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name: "success_empty_list",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
				return []*campaignEntity.PublicSummary{}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
				return nil, errors.New("db connection failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockListPublicUpcomingCampaigns{fn: tt.mockFn}
			handler := campaign.ListPublicUpcomingCampaignsHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/public/campaigns",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/public/campaigns")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				campaigns, ok := result["campaigns"].([]any)
				if !ok {
					t.Fatal("response missing 'campaigns' field")
				}
				if len(campaigns) != tt.wantCount {
					t.Errorf("got %d campaigns, want %d", len(campaigns), tt.wantCount)
				}
			}
		})
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/app/api/campaign/... -run TestListPublicUpcomingCampaignsHandler -v
```

Expected: FAIL — `campaign.ListPublicUpcomingCampaignsHandler undefined`.

- [ ] **Step 4: Implement the handler**

Create `internal/app/api/campaign/list_public_upcoming_campaigns.go`:

```go
package campaign

import (
	"context"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

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

func ListPublicUpcomingCampaignsHandler(
	uc domainCampaign.IListPublicUpcomingCampaigns,
) func(context.Context, *struct{}) (*ListPublicCampaignsResponse, error) {

	return func(ctx context.Context, _ *struct{}) (*ListPublicCampaignsResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		campaigns, err := uc.ListPublicUpcomingCampaigns(ctx, userUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		responses := make([]PublicCampaignSummaryResponse, 0, len(campaigns))
		for _, c := range campaigns {
			var storyCurrentAtStr *string
			if c.StoryCurrentAt != nil {
				formatted := c.StoryCurrentAt.Format("2006-01-02")
				storyCurrentAtStr = &formatted
			}
			var storyEndAtStr *string
			if c.StoryEndAt != nil {
				formatted := c.StoryEndAt.Format("2006-01-02")
				storyEndAtStr = &formatted
			}
			var nextGameScheduledAtStr *string
			if c.NextGameScheduledAt != nil {
				formatted := c.NextGameScheduledAt.Format(time.RFC3339)
				nextGameScheduledAtStr = &formatted
			}
			responses = append(responses, PublicCampaignSummaryResponse{
				CampaignSummaryResponse: CampaignSummaryResponse{
					UUID:                    c.UUID,
					Name:                    c.Name,
					BriefInitialDescription: c.BriefInitialDescription,
					BriefFinalDescription:   c.BriefFinalDescription,
					IsPublic:                c.IsPublic,
					CallLink:                c.CallLink,
					StoryStartAt:            c.StoryStartAt.Format("2006-01-02"),
					StoryCurrentAt:          storyCurrentAtStr,
					StoryEndAt:              storyEndAtStr,
					CreatedAt:               c.CreatedAt.Format(http.TimeFormat),
					UpdatedAt:               c.UpdatedAt.Format(http.TimeFormat),
				},
				NextGameScheduledAt: nextGameScheduledAtStr,
			})
		}

		return &ListPublicCampaignsResponse{
			Body: ListPublicCampaignsResponseBody{
				Campaigns: responses,
			},
		}, nil
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
go test ./internal/app/api/campaign/... -run TestListPublicUpcomingCampaignsHandler -v
```

Expected: all 4 sub-tests PASS.

- [ ] **Step 6: Run all campaign handler tests for regressions**

```bash
go test ./internal/app/api/campaign/...
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/app/api/campaign/list_public_upcoming_campaigns.go \
        internal/app/api/campaign/list_public_upcoming_campaigns_test.go \
        internal/app/api/campaign/mocks_test.go
git commit -m "feat: add ListPublicUpcomingCampaigns HTTP handler with unit tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 7: Routes + `Api` struct

**Files:**
- Modify: `internal/app/api/campaign/routes.go`

- [ ] **Step 1: Add handler field to `Api` and register the route**

Replace the content of `internal/app/api/campaign/routes.go`:

```go
package campaign

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	CreateCampaignHandler                Handler[CreateCampaignRequest, CreateCampaignResponse]
	GetCampaignHandler                   Handler[GetCampaignRequest, GetCampaignResponse]
	ListCampaignsHandler                 Handler[struct{}, ListCampaignsResponse]
	ListPublicUpcomingCampaignsHandler   Handler[struct{}, ListPublicCampaignsResponse]
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/campaigns",
		Description: "Create a new campaign from a scenario",
		Tags:        []string{"campaigns"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, a.CreateCampaignHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/campaigns/{uuid}",
		Description: "Get a campaign by UUID",
		Tags:        []string{"campaigns"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusForbidden,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.GetCampaignHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/campaigns",
		Description: "List all user's campaigns",
		Tags:        []string{"campaigns"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.ListCampaignsHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/public/campaigns",
		Description: "List public campaigns from other masters ordered by nearest upcoming match",
		Tags:        []string{"campaigns"},
		Errors: []int{
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.ListPublicUpcomingCampaignsHandler)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/app/api/campaign/...
```

Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/app/api/campaign/routes.go
git commit -m "feat: register GET /public/campaigns route for public campaign listing

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 8: Main wiring

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Instantiate UC and wire handler**

Find the `campaignsApi` block in `cmd/api/main.go` (around line 137–145) and update it:

```go
createCampaignUC := domainCampaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)
getCampaignUC := domainCampaign.NewGetCampaignUC(campaignRepo)
listCampaignsUC := domainCampaign.NewListCampaignsUC(campaignRepo)
listPublicUpcomingCampaignsUC := domainCampaign.NewListPublicUpcomingCampaignsUC(campaignRepo)

campaignsApi := campaignHandler.Api{
    CreateCampaignHandler:              campaignHandler.CreateCampaignHandler(createCampaignUC),
    GetCampaignHandler:                 campaignHandler.GetCampaignHandler(getCampaignUC),
    ListCampaignsHandler:               campaignHandler.ListCampaignsHandler(listCampaignsUC),
    ListPublicUpcomingCampaignsHandler: campaignHandler.ListPublicUpcomingCampaignsHandler(listPublicUpcomingCampaignsUC),
}
```

- [ ] **Step 2: Build to verify wiring**

```bash
go build ./cmd/api/...
```

Expected: no output, exit 0.

- [ ] **Step 3: Run all non-integration tests**

```bash
go test ./...
```

Expected: all pass (excluding integration tag, which requires DB).

- [ ] **Step 4: Commit**

```bash
git add cmd/api/main.go
git commit -m "feat: wire ListPublicUpcomingCampaigns use case into API server

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 9: Final verification

- [ ] **Step 1: Run all unit tests**

```bash
go test ./...
```

Expected: all pass.

- [ ] **Step 2: Run all integration tests**

```bash
go test -tags=integration ./internal/gateway/pg/...
```

Expected: all pass (requires local DB with migration applied).

- [ ] **Step 3: Vet with integration tag**

```bash
go vet -tags=integration ./...
```

Expected: no output, exit 0.
