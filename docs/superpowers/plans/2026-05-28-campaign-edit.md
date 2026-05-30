# Campaign Edit — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement end-to-end campaign editing (PATCH /campaigns/{uuid}) with two runtime modes: free (all fields) before any match starts, restricted (locked fields + non-regression constraint on story_current_at) after.

**Architecture:** Single PATCH endpoint. UC detects mode via `GetCampaignForUpdate` (one query with EXISTS subquery). No extra DB call after update — campaign is patched in memory and returned directly. Frontend hides/disables mode-locked fields proactively based on `campaign.matches.some(m => !!m.gameStartAt)`.

**Tech Stack:** Go 1.23, pgx/v5, huma/v2, humatest; React 18, TypeScript, TanStack Query, styled-components.

---

## File Map

| Action | Path |
|--------|------|
| Modify | `internal/application/campaign/error.go` |
| Modify | `internal/application/campaign/i_repository.go` |
| Create | `internal/application/campaign/update_campaign.go` |
| Modify | `internal/application/testutil/mock_campaign_repo.go` |
| Modify | `internal/application/campaign/campaign_test.go` |
| Create | `internal/gateway/pg/campaign/read_campaign_for_update.go` |
| Create | `internal/gateway/pg/campaign/update_campaign.go` |
| Modify | `internal/gateway/pg/campaign/campaign_integration_test.go` |
| Create | `internal/app/api/campaign/update_campaign.go` |
| Modify | `internal/app/api/campaign/mocks_test.go` |
| Create | `internal/app/api/campaign/update_campaign_test.go` |
| Modify | `internal/app/api/campaign/routes.go` |
| Modify | `cmd/api/main.go` |
| Create | `docs/dev/api/update-campaign.md` |
| Modify | `src/services/campaignService.ts` |
| Modify | `src/types/campaign.ts` |
| Create | `src/hooks/useUpdateCampaign.ts` |
| Create | `src/features/campaign/campaignErrorMessages.ts` |
| Create | `src/pages/EditCampaignPage.tsx` |
| Modify | `src/App.tsx` |

---

## Task 1: Errors, IRepository, and CampaignUpdateContext

**Files:**
- Modify: `internal/application/campaign/error.go`
- Modify: `internal/application/campaign/i_repository.go`
- Create: `internal/application/campaign/update_campaign.go` (struct + interface definitions only)

- [ ] **Add three new errors to `error.go`**

Open `internal/application/campaign/error.go` and append after the last existing error:

```go
ErrCampaignAlreadyEnded        = domain.NewValidationError(errors.New("campaign has already ended"))
ErrLockedAfterMatchStart       = domain.NewValidationError(errors.New("name and story_start_at cannot be changed after a match has started"))
ErrCannotRegressStoryCurrentAt = domain.NewValidationError(errors.New("story_current_at cannot be set to a date earlier than the current value"))
```

- [ ] **Create `update_campaign.go` with input struct, context struct, and interface**

Create `internal/application/campaign/update_campaign.go`:

```go
package campaign

import (
	"context"
	"time"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type IUpdateCampaign interface {
	Update(ctx context.Context, input *UpdateCampaignInput) (*campaignEntity.Campaign, error)
}

// CampaignUpdateContext is returned by GetCampaignForUpdate: all editable fields
// plus validation flags, fetched in a single query.
type CampaignUpdateContext struct {
	MasterUUID              uuid.UUID
	Name                    string
	BriefInitialDescription string
	Description             string
	IsPublic                bool
	CallLink                string
	StoryStartAt            time.Time
	StoryCurrentAt          *time.Time
	StoryEndAt              *time.Time
	HasStartedMatch         bool
}

type UpdateCampaignInput struct {
	CampaignUUID uuid.UUID
	MasterUUID   uuid.UUID
	// Always editable
	BriefInitialDescription *string
	Description             *string
	IsPublic                *bool
	CallLink                *string
	StoryCurrentAt          *time.Time
	// Free mode only (locked after any match starts)
	Name         *string
	StoryStartAt *time.Time
}
```

- [ ] **Add two new methods to `i_repository.go`**

Open `internal/application/campaign/i_repository.go` and add to the `IRepository` interface:

```go
	UpdateCampaign(ctx context.Context, campaign *campaign.Campaign) error
	GetCampaignForUpdate(ctx context.Context, uuid uuid.UUID) (*CampaignUpdateContext, error)
```

The full interface now has 10 methods. No import changes needed (`CampaignUpdateContext` is in the same package).

- [ ] **Verify build**

```bash
go vet ./internal/application/campaign/...
```

Expected: no errors (implementations haven't been added to gateway yet — the mock handles it for now).

---

## Task 2: UpdateCampaignUC implementation + unit tests

**Files:**
- Modify: `internal/application/campaign/update_campaign.go` (add UC body)
- Modify: `internal/application/testutil/mock_campaign_repo.go` (add two new methods)
- Modify: `internal/application/campaign/campaign_test.go` (add TestUpdateCampaign)

- [ ] **Extend the mock to satisfy the updated IRepository**

Open `internal/application/testutil/mock_campaign_repo.go`. Change the imports to:

```go
import (
	"context"
	"time"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)
```

Update the struct alias from `campaign.Campaign` → `campaignEntity.Campaign` and `campaign.Summary` → `campaignEntity.Summary`, etc. throughout the file (every method signature). Then add these two fields to the struct and two new methods at the bottom:

```go
// Add to MockCampaignRepo struct:
GetCampaignForUpdateFn func(ctx context.Context, uuid uuid.UUID) (*campaignApp.CampaignUpdateContext, error)
UpdateCampaignFn       func(ctx context.Context, c *campaignEntity.Campaign) error
```

```go
func (m *MockCampaignRepo) GetCampaignForUpdate(ctx context.Context, id uuid.UUID) (*campaignApp.CampaignUpdateContext, error) {
	if m.GetCampaignForUpdateFn != nil {
		return m.GetCampaignForUpdateFn(ctx, id)
	}
	return nil, nil
}

func (m *MockCampaignRepo) UpdateCampaign(ctx context.Context, c *campaignEntity.Campaign) error {
	if m.UpdateCampaignFn != nil {
		return m.UpdateCampaignFn(ctx, c)
	}
	return nil
}
```

> **Note:** Because `mock_campaign_repo.go` is in `package testutil` and now imports `campaignApp`, you must update every existing method signature that used the old `campaign.*` alias to use `campaignEntity.*`. For example, `CreateCampaign(ctx context.Context, c *campaign.Campaign)` becomes `CreateCampaign(ctx context.Context, c *campaignEntity.Campaign)`.

- [ ] **Write failing tests (add to `campaign_test.go`)**

Open `internal/application/campaign/campaign_test.go` and add after the last test function:

```go
func TestUpdateCampaign(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	campaignUUID := uuid.New()
	now := time.Now()

	baseCtx := func(opts ...func(*campaign.CampaignUpdateContext)) *campaign.CampaignUpdateContext {
		c := &campaign.CampaignUpdateContext{
			MasterUUID:              masterUUID,
			Name:                    "Valid Name",
			BriefInitialDescription: "Brief",
			Description:             "Desc",
			IsPublic:                false,
			CallLink:                "https://discord.gg/abc",
			StoryStartAt:            now,
			HasStartedMatch:         false,
		}
		for _, o := range opts {
			o(c)
		}
		return c
	}

	tests := []struct {
		name    string
		input   *campaign.UpdateCampaignInput
		mock    *testutil.MockCampaignRepo
		wantErr error
		check   func(t *testing.T, result *campaignEntity.Campaign)
	}{
		{
			name: "success_full_free_mode",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				Name:         strPtr("New Name"),
				StoryStartAt: &now,
				IsPublic:     boolPtr(true),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(), nil
				},
			},
			wantErr: nil,
			check: func(t *testing.T, c *campaignEntity.Campaign) {
				if c.Name != "New Name" {
					t.Errorf("Name = %q, want %q", c.Name, "New Name")
				}
				if !c.IsPublic {
					t.Error("IsPublic should be true")
				}
			},
		},
		{
			name: "success_partial_always_editable",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				CallLink:     strPtr("https://meet.new"),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(func(c *campaign.CampaignUpdateContext) { c.HasStartedMatch = true }), nil
				},
			},
			wantErr: nil,
		},
		{
			name: "success_noop_empty_body",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(), nil
				},
			},
			wantErr: nil,
		},
		{
			name: "not_found",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				Name:         strPtr("X"),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return nil, campaignPg.ErrCampaignNotFound
				},
			},
			wantErr: campaign.ErrCampaignNotFound,
		},
		{
			name: "not_owner",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   otherUUID,
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(), nil
				},
			},
			wantErr: campaign.ErrNotCampaignOwner,
		},
		{
			name: "already_ended",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				Name:         strPtr("X"),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					ended := now
					return baseCtx(func(c *campaign.CampaignUpdateContext) { c.StoryEndAt = &ended }), nil
				},
			},
			wantErr: campaign.ErrCampaignAlreadyEnded,
		},
		{
			name: "locked_name_after_match_start",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				Name:         strPtr("Locked"),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(func(c *campaign.CampaignUpdateContext) { c.HasStartedMatch = true }), nil
				},
			},
			wantErr: campaign.ErrLockedAfterMatchStart,
		},
		{
			name: "locked_story_start_at_after_match_start",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				StoryStartAt: &now,
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(func(c *campaign.CampaignUpdateContext) { c.HasStartedMatch = true }), nil
				},
			},
			wantErr: campaign.ErrLockedAfterMatchStart,
		},
		{
			name: "cannot_regress_story_current_at",
			input: func() *campaign.UpdateCampaignInput {
				past := now.AddDate(0, 0, -1)
				return &campaign.UpdateCampaignInput{
					CampaignUUID:   campaignUUID,
					MasterUUID:     masterUUID,
					StoryCurrentAt: &past,
				}
			}(),
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(func(c *campaign.CampaignUpdateContext) {
						c.HasStartedMatch = true
						c.StoryCurrentAt = &now
					}), nil
				},
			},
			wantErr: campaign.ErrCannotRegressStoryCurrentAt,
		},
		{
			name: "story_current_at_null_is_free_in_restricted_mode",
			input: func() *campaign.UpdateCampaignInput {
				past := now.AddDate(0, -1, 0)
				return &campaign.UpdateCampaignInput{
					CampaignUUID:   campaignUUID,
					MasterUUID:     masterUUID,
					StoryCurrentAt: &past,
				}
			}(),
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(func(c *campaign.CampaignUpdateContext) {
						c.HasStartedMatch = true
						c.StoryCurrentAt = nil // currently null → free to set any value
					}), nil
				},
			},
			wantErr: nil,
		},
		{
			name: "name_too_short",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				Name:         strPtr("ab"),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(), nil
				},
			},
			wantErr: campaign.ErrMinNameLength,
		},
		{
			name: "name_too_long",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				Name:         strPtr("this name is way too long for the limit"),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(), nil
				},
			},
			wantErr: campaign.ErrMaxNameLength,
		},
		{
			name: "brief_too_long",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID:            campaignUUID,
				MasterUUID:              masterUUID,
				BriefInitialDescription: strPtr(string(make([]byte, 256))),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(), nil
				},
			},
			wantErr: campaign.ErrMaxBriefDescLength,
		},
		{
			name: "call_link_too_long",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				CallLink:     strPtr(string(make([]byte, 256))),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(), nil
				},
			},
			wantErr: campaign.ErrMaxCallLinkLength,
		},
		{
			name: "repo_error_on_update",
			input: &campaign.UpdateCampaignInput{
				CampaignUUID: campaignUUID,
				MasterUUID:   masterUUID,
				Name:         strPtr("Valid"),
			},
			mock: &testutil.MockCampaignRepo{
				GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaign.CampaignUpdateContext, error) {
					return baseCtx(), nil
				},
				UpdateCampaignFn: func(_ context.Context, _ *campaignEntity.Campaign) error {
					return errors.New("db down")
				},
			},
			wantErr: errors.New("db down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := campaign.NewUpdateCampaignUC(tt.mock)
			result, err := uc.Update(context.Background(), tt.input)

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
			if result == nil {
				t.Fatal("expected non-nil campaign")
			}
			if tt.check != nil {
				tt.check(t, result)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
```

Also add these imports to `campaign_test.go` (merge with existing):
```go
campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
```

- [ ] **Run tests — expect failure** (UC not yet implemented)

```bash
go test ./internal/application/campaign/... -run TestUpdateCampaign -v
```

Expected: FAIL with "undefined: campaign.NewUpdateCampaignUC"

- [ ] **Implement the UC body in `update_campaign.go`**

Append after the type declarations in `internal/application/campaign/update_campaign.go`:

```go
type UpdateCampaignUC struct {
	repo IRepository
}

func NewUpdateCampaignUC(repo IRepository) *UpdateCampaignUC {
	return &UpdateCampaignUC{repo: repo}
}

func (uc *UpdateCampaignUC) Update(
	ctx context.Context, input *UpdateCampaignInput,
) (*campaignEntity.Campaign, error) {
	ctxData, err := uc.repo.GetCampaignForUpdate(ctx, input.CampaignUUID)
	if err != nil {
		if errors.Is(err, campaignPg.ErrCampaignNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}
	if ctxData.MasterUUID != input.MasterUUID {
		return nil, ErrNotCampaignOwner
	}
	if ctxData.StoryEndAt != nil {
		return nil, ErrCampaignAlreadyEnded
	}
	if ctxData.HasStartedMatch && (input.Name != nil || input.StoryStartAt != nil) {
		return nil, ErrLockedAfterMatchStart
	}
	if input.StoryCurrentAt != nil && ctxData.StoryCurrentAt != nil &&
		input.StoryCurrentAt.Before(*ctxData.StoryCurrentAt) {
		return nil, ErrCannotRegressStoryCurrentAt
	}

	c := buildFromContext(input.CampaignUUID, ctxData)

	if input.Name == nil && input.BriefInitialDescription == nil &&
		input.Description == nil && input.IsPublic == nil &&
		input.CallLink == nil && input.StoryCurrentAt == nil &&
		input.StoryStartAt == nil {
		return c, nil
	}

	if input.Name != nil {
		if len(*input.Name) < 5 {
			return nil, ErrMinNameLength
		}
		if len(*input.Name) > 32 {
			return nil, ErrMaxNameLength
		}
		c.Name = *input.Name
	}
	if input.BriefInitialDescription != nil {
		if len(*input.BriefInitialDescription) > 255 {
			return nil, ErrMaxBriefDescLength
		}
		c.BriefInitialDescription = *input.BriefInitialDescription
	}
	if input.Description != nil {
		c.Description = *input.Description
	}
	if input.IsPublic != nil {
		c.IsPublic = *input.IsPublic
	}
	if input.CallLink != nil {
		if len(*input.CallLink) > 255 {
			return nil, ErrMaxCallLinkLength
		}
		c.CallLink = *input.CallLink
	}
	if input.StoryCurrentAt != nil {
		c.StoryCurrentAt = input.StoryCurrentAt
	}
	if input.StoryStartAt != nil {
		c.StoryStartAt = *input.StoryStartAt
	}
	c.UpdatedAt = time.Now()

	if err := uc.repo.UpdateCampaign(ctx, c); err != nil {
		if errors.Is(err, campaignPg.ErrCampaignNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}
	return c, nil
}

func buildFromContext(campaignUUID uuid.UUID, d *CampaignUpdateContext) *campaignEntity.Campaign {
	return &campaignEntity.Campaign{
		UUID:                    campaignUUID,
		MasterUUID:              d.MasterUUID,
		Name:                    d.Name,
		BriefInitialDescription: d.BriefInitialDescription,
		Description:             d.Description,
		IsPublic:                d.IsPublic,
		CallLink:                d.CallLink,
		StoryStartAt:            d.StoryStartAt,
		StoryCurrentAt:          d.StoryCurrentAt,
	}
}
```

Add these imports to `update_campaign.go`:

```go
import (
	"context"
	"errors"
	"time"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)
```

- [ ] **Run tests — expect pass**

```bash
go test ./internal/application/campaign/... -run TestUpdateCampaign -v
```

Expected: all 14 sub-tests PASS.

- [ ] **Run all campaign UC tests to check for regressions**

```bash
go test ./internal/application/campaign/...
```

Expected: PASS.

- [ ] **Commit**

```bash
git add internal/application/campaign/error.go \
        internal/application/campaign/i_repository.go \
        internal/application/campaign/update_campaign.go \
        internal/application/campaign/campaign_test.go \
        internal/application/testutil/mock_campaign_repo.go
git commit -m "$(cat <<'EOF'
feat(campaign): add UpdateCampaignUC with dual-mode edit logic

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 3: Gateway — GetCampaignForUpdate and UpdateCampaign

**Files:**
- Create: `internal/gateway/pg/campaign/read_campaign_for_update.go`
- Create: `internal/gateway/pg/campaign/update_campaign.go`
- Modify: `internal/gateway/pg/campaign/campaign_integration_test.go`

- [ ] **Create `read_campaign_for_update.go`**

```go
package campaign

import (
	"context"
	"errors"
	"fmt"

	appCampaign "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetCampaignForUpdate(
	ctx context.Context, campaignUUID uuid.UUID,
) (*appCampaign.CampaignUpdateContext, error) {
	const query = `
		SELECT
			c.master_uuid,
			c.name,
			COALESCE(c.brief_initial_description, ''),
			COALESCE(c.description, ''),
			c.is_public,
			COALESCE(c.call_link, ''),
			c.story_start_at,
			c.story_current_at,
			c.story_end_at,
			EXISTS(
				SELECT 1 FROM matches m
				WHERE m.campaign_uuid = c.uuid AND m.game_start_at IS NOT NULL
			) AS has_started_match
		FROM campaigns c
		WHERE c.uuid = $1
	`
	var d appCampaign.CampaignUpdateContext
	err := r.q.QueryRow(ctx, query, campaignUUID).Scan(
		&d.MasterUUID,
		&d.Name,
		&d.BriefInitialDescription,
		&d.Description,
		&d.IsPublic,
		&d.CallLink,
		&d.StoryStartAt,
		&d.StoryCurrentAt,
		&d.StoryEndAt,
		&d.HasStartedMatch,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCampaignNotFound
		}
		return nil, fmt.Errorf("failed to get campaign for update: %w", err)
	}
	return &d, nil
}
```

- [ ] **Create `update_campaign.go` (gateway)**

```go
package campaign

import (
	"context"
	"fmt"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
)

func (r *Repository) UpdateCampaign(ctx context.Context, c *campaignEntity.Campaign) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	const query = `
		UPDATE campaigns SET
			name = $1,
			brief_initial_description = $2,
			description = $3,
			is_public = $4,
			call_link = $5,
			story_start_at = $6,
			story_current_at = $7,
			updated_at = $8
		WHERE uuid = $9
	`
	result, err := tx.Exec(ctx, query,
		c.Name, c.BriefInitialDescription, c.Description,
		c.IsPublic, c.CallLink, c.StoryStartAt, c.StoryCurrentAt,
		c.UpdatedAt, c.UUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update campaign: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}
	return tx.Commit(ctx)
}
```

- [ ] **Write integration tests (append to `campaign_integration_test.go`)**

```go
func TestGetCampaignForUpdate(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()
	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "master_gfu", "gfu@test.com", "pass"))

	t.Run("not found", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		_, err := repo.GetCampaignForUpdate(ctx, uuid.New())
		if !errors.Is(err, pgCampaign.ErrCampaignNotFound) {
			t.Fatalf("expected ErrCampaignNotFound, got %v", err)
		}
	})

	t.Run("no started match", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		master := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "m_gfu2", "gfu2@test.com", "pass"))
		camp := newTestCampaign(master, nil, "ForUpdate Campaign")
		if err := repo.CreateCampaign(ctx, camp); err != nil {
			t.Fatal(err)
		}
		got, err := repo.GetCampaignForUpdate(ctx, camp.UUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.MasterUUID != master {
			t.Errorf("MasterUUID = %v, want %v", got.MasterUUID, master)
		}
		if got.Name != camp.Name {
			t.Errorf("Name = %q, want %q", got.Name, camp.Name)
		}
		if got.HasStartedMatch {
			t.Error("HasStartedMatch should be false")
		}
	})

	t.Run("has started match", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		master := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "m_gfu3", "gfu3@test.com", "pass"))
		campUUID := pgtest.InsertTestCampaign(t, pool, master.String(), "Has Started Match Camp")
		matchUUID := pgtest.InsertTestMatch(t, pool, master.String(), campUUID, "Match A")
		// Start the match
		_, err := pool.Exec(ctx, `UPDATE matches SET game_start_at = NOW() WHERE uuid = $1`, matchUUID)
		if err != nil {
			t.Fatalf("failed to start match: %v", err)
		}
		campParsedUUID := mustParseUUID(t, campUUID)
		got, err := repo.GetCampaignForUpdate(ctx, campParsedUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got.HasStartedMatch {
			t.Error("HasStartedMatch should be true")
		}
	})
}

func TestUpdateCampaign(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgCampaign.NewRepository(pool)
	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		master := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "m_upd", "upd@test.com", "pass"))
		c := newTestCampaign(master, nil, "Before Update")
		if err := repo.CreateCampaign(ctx, c); err != nil {
			t.Fatal(err)
		}
		c.Name = "After Update"
		c.IsPublic = false
		c.UpdatedAt = time.Now().Truncate(time.Microsecond)

		if err := repo.UpdateCampaign(ctx, c); err != nil {
			t.Fatalf("UpdateCampaign() unexpected error: %v", err)
		}
		got, err := repo.GetCampaignForUpdate(ctx, c.UUID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "After Update" {
			t.Errorf("Name = %q, want %q", got.Name, "After Update")
		}
		if got.IsPublic {
			t.Error("IsPublic should be false")
		}
	})

	t.Run("not found returns ErrCampaignNotFound", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		c := newTestCampaign(masterUUID, nil, "Ghost")
		c.UUID = uuid.New() // not in DB
		c.UpdatedAt = time.Now()
		err := repo.UpdateCampaign(ctx, c)
		if !errors.Is(err, pgCampaign.ErrCampaignNotFound) {
			t.Fatalf("expected ErrCampaignNotFound, got %v", err)
		}
	})
}
```

> **Note:** `masterUUID` in the "not found" sub-test can reuse the variable from `TestGetCampaignForUpdate` since they're in the same package, or just use `uuid.New()` instead.

Replace the reference to `masterUUID` in the "not found" sub-test with `uuid.New()`:

```go
c := newTestCampaign(uuid.New(), nil, "Ghost")
```

- [ ] **Vet with integration tags**

```bash
go vet -tags=integration ./internal/gateway/pg/...
```

Expected: no errors.

- [ ] **Run integration tests**

```bash
go test -tags=integration ./internal/gateway/pg/campaign/...
```

Expected: all tests PASS (requires running DB).

- [ ] **Commit**

```bash
git add internal/gateway/pg/campaign/read_campaign_for_update.go \
        internal/gateway/pg/campaign/update_campaign.go \
        internal/gateway/pg/campaign/campaign_integration_test.go
git commit -m "$(cat <<'EOF'
feat(gateway/campaign): add GetCampaignForUpdate and UpdateCampaign

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 4: HTTP handler, mock, tests, and route

**Files:**
- Create: `internal/app/api/campaign/update_campaign.go`
- Modify: `internal/app/api/campaign/mocks_test.go`
- Create: `internal/app/api/campaign/update_campaign_test.go`
- Modify: `internal/app/api/campaign/routes.go`

- [ ] **Add mock for `IUpdateCampaign` to `mocks_test.go`**

Append to `internal/app/api/campaign/mocks_test.go`:

```go
type mockUpdateCampaign struct {
	fn func(ctx context.Context, input *campaign.UpdateCampaignInput) (*campaignEntity.Campaign, error)
}

func (m *mockUpdateCampaign) Update(ctx context.Context, input *campaign.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
	return m.fn(ctx, input)
}
```

Add `campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"` to the imports if not already present.

- [ ] **Write failing handler tests**

Create `internal/app/api/campaign/update_campaign_test.go`:

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
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestUpdateCampaignHandler(t *testing.T) {
	userUUID := uuid.New()
	campaignUUID := uuid.New()
	now := time.Now()

	baseResp := func(name string) *campaignEntity.Campaign {
		return &campaignEntity.Campaign{
			UUID:                    campaignUUID,
			MasterUUID:              userUUID,
			Name:                    name,
			BriefInitialDescription: "brief",
			Description:             "full",
			IsPublic:                true,
			CallLink:                "https://discord.gg/abc",
			StoryStartAt:            now,
			UpdatedAt:               now,
		}
	}

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error)
		wantStatus int
	}{
		{
			name: "success_full_patch",
			body: map[string]any{
				"name":                      "New Name",
				"brief_initial_description": "new brief",
				"description":               "new desc",
				"is_public":                 false,
				"call_link":                 "https://meet.new",
				"story_start_at":            "2026-07-20",
				"story_current_at":          "2026-07-20T10:00:00Z",
			},
			mockFn: func(_ context.Context, input *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				if input.Name == nil || *input.Name != "New Name" {
					t.Errorf("name not forwarded: %+v", input.Name)
				}
				return baseResp("New Name"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success_partial_always_editable",
			body: map[string]any{"call_link": "https://new-link.com"},
			mockFn: func(_ context.Context, input *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				if input.Name != nil {
					t.Errorf("name should be nil, got %v", *input.Name)
				}
				return baseResp("Original"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success_empty_body_noop",
			body: map[string]any{},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return baseResp("Original"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid_story_start_at",
			body: map[string]any{"story_start_at": "not-a-date"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				t.Fatal("UC should not be called when date parsing fails")
				return nil, nil
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid_story_current_at",
			body: map[string]any{"story_current_at": "not-a-date"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				t.Fatal("UC should not be called when date parsing fails")
				return nil, nil
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "not_found",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_owner",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrNotCampaignOwner
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "already_ended",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrCampaignAlreadyEnded
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "locked_after_match_start",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrLockedAfterMatchStart
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "cannot_regress_story_current_at",
			body: map[string]any{"story_current_at": "2024-01-01T00:00:00Z"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, campaignUC.ErrCannotRegressStoryCurrentAt
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "validation_error",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, domain.ErrValidation
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_error",
			body: map[string]any{"name": "x"},
			mockFn: func(_ context.Context, _ *campaignUC.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
				return nil, errors.New("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			mock := &mockUpdateCampaign{fn: tt.mockFn}
			handler := campaign.UpdateCampaignHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodPatch,
				Path:   "/campaigns/{uuid}",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PatchCtx(ctx, "/campaigns/"+campaignUUID.String(), tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("unmarshal failed: %v", err)
				}
				c, ok := result["campaign"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'campaign' field")
				}
				if c["master_uuid"] != userUUID.String() {
					t.Errorf("master_uuid = %v, want %v", c["master_uuid"], userUUID.String())
				}
			}
		})
	}
}
```

- [ ] **Run tests — expect failure**

```bash
go test ./internal/app/api/campaign/... -run TestUpdateCampaignHandler -v
```

Expected: FAIL with "undefined: campaign.UpdateCampaignHandler"

- [ ] **Create the handler `update_campaign.go`**

Create `internal/app/api/campaign/update_campaign.go`:

```go
package campaign

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type UpdateCampaignRequestBody struct {
	Name                    *string `json:"name,omitempty" doc:"Campaign name (5-32 characters)"`
	BriefInitialDescription *string `json:"brief_initial_description,omitempty" doc:"Brief description (max 255 characters)"`
	Description             *string `json:"description,omitempty" doc:"Full description"`
	IsPublic                *bool   `json:"is_public,omitempty" doc:"Public/private flag"`
	CallLink                *string `json:"call_link,omitempty" doc:"Call link URL (max 255 characters)"`
	StoryStartAt            *string `json:"story_start_at,omitempty" doc:"YYYY-MM-DD (locked after any match starts)"`
	StoryCurrentAt          *string `json:"story_current_at,omitempty" doc:"ISO 8601 date-time (cannot regress after match starts)"`
}

type UpdateCampaignRequest struct {
	UUID uuid.UUID                 `path:"uuid" required:"true" doc:"UUID of the campaign to update"`
	Body UpdateCampaignRequestBody `json:"body"`
}

type CampaignEditResponse struct {
	UUID                    uuid.UUID `json:"uuid"`
	MasterUUID              uuid.UUID `json:"master_uuid"`
	Name                    string    `json:"name"`
	BriefInitialDescription string    `json:"brief_initial_description"`
	Description             string    `json:"description"`
	IsPublic                bool      `json:"is_public"`
	CallLink                string    `json:"call_link"`
	StoryStartAt            string    `json:"story_start_at"`
	StoryCurrentAt          *string   `json:"story_current_at,omitempty"`
	UpdatedAt               string    `json:"updated_at"`
}

type UpdateCampaignResponseBody struct {
	Campaign CampaignEditResponse `json:"campaign"`
}

type UpdateCampaignResponse struct {
	Body UpdateCampaignResponseBody `json:"body"`
}

func UpdateCampaignHandler(
	uc campaignUC.IUpdateCampaign,
) func(context.Context, *UpdateCampaignRequest) (*UpdateCampaignResponse, error) {
	return func(ctx context.Context, req *UpdateCampaignRequest) (*UpdateCampaignResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		input := &campaignUC.UpdateCampaignInput{
			CampaignUUID:            req.UUID,
			MasterUUID:              userUUID,
			BriefInitialDescription: req.Body.BriefInitialDescription,
			Description:             req.Body.Description,
			IsPublic:                req.Body.IsPublic,
			CallLink:                req.Body.CallLink,
			Name:                    req.Body.Name,
		}

		if req.Body.StoryStartAt != nil {
			t, err := time.Parse("2006-01-02", *req.Body.StoryStartAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(
					"invalid story_start_at date format, use YYYY-MM-DD")
			}
			input.StoryStartAt = &t
		}
		if req.Body.StoryCurrentAt != nil {
			t, err := time.Parse(time.RFC3339, *req.Body.StoryCurrentAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(
					"invalid story_current_at format, use ISO 8601. E.g. 2026-06-15T19:30:00Z")
			}
			input.StoryCurrentAt = &t
		}

		c, err := uc.Update(ctx, input)
		if err != nil {
			switch {
			case errors.Is(err, campaignUC.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaignUC.ErrNotCampaignOwner):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, campaignUC.ErrCampaignAlreadyEnded),
				errors.Is(err, campaignUC.ErrLockedAfterMatchStart),
				errors.Is(err, campaignUC.ErrCannotRegressStoryCurrentAt):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			case errors.Is(err, domain.ErrValidation):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		var storyCurrentAtStr *string
		if c.StoryCurrentAt != nil {
			s := c.StoryCurrentAt.Format(time.RFC3339)
			storyCurrentAtStr = &s
		}

		return &UpdateCampaignResponse{
			Body: UpdateCampaignResponseBody{
				Campaign: CampaignEditResponse{
					UUID:                    c.UUID,
					MasterUUID:              c.MasterUUID,
					Name:                    c.Name,
					BriefInitialDescription: c.BriefInitialDescription,
					Description:             c.Description,
					IsPublic:                c.IsPublic,
					CallLink:                c.CallLink,
					StoryStartAt:            c.StoryStartAt.Format("2006-01-02"),
					StoryCurrentAt:          storyCurrentAtStr,
					UpdatedAt:               c.UpdatedAt.Format(http.TimeFormat),
				},
			},
		}, nil
	}
}
```

- [ ] **Run tests — expect pass**

```bash
go test ./internal/app/api/campaign/... -run TestUpdateCampaignHandler -v
```

Expected: all 12 sub-tests PASS.

- [ ] **Register route in `routes.go`**

In `internal/app/api/campaign/routes.go`, add the field to `Api`:

```go
UpdateCampaignHandler Handler[UpdateCampaignRequest, UpdateCampaignResponse]
```

Add to `RegisterRoutes` after the CreateCampaign registration:

```go
huma.Register(api, huma.Operation{
    Method:      http.MethodPatch,
    Path:        "/campaigns/{uuid}",
    Description: "Update a campaign (master only; name and story_start_at locked after match starts)",
    Tags:        []string{"campaigns"},
    Errors: []int{
        http.StatusNotFound,
        http.StatusBadRequest,
        http.StatusForbidden,
        http.StatusUnauthorized,
        http.StatusUnprocessableEntity,
        http.StatusInternalServerError,
    },
}, a.UpdateCampaignHandler)
```

- [ ] **Run all campaign handler tests**

```bash
go test ./internal/app/api/campaign/...
```

Expected: PASS.

- [ ] **Commit**

```bash
git add internal/app/api/campaign/update_campaign.go \
        internal/app/api/campaign/update_campaign_test.go \
        internal/app/api/campaign/mocks_test.go \
        internal/app/api/campaign/routes.go
git commit -m "$(cat <<'EOF'
feat(api/campaign): add PATCH /campaigns/{uuid} handler and route

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 5: Wire in cmd/api/main.go and verify build

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Add the UC instantiation and wire to the handler**

In `cmd/api/main.go`, after `deleteCampaignUC := campaign.NewDeleteCampaignUC(campaignRepo)` add:

```go
updateCampaignUC := campaign.NewUpdateCampaignUC(campaignRepo)
```

In `campaignsApi := campaignHandler.Api{...}`, add:

```go
UpdateCampaignHandler: campaignHandler.UpdateCampaignHandler(updateCampaignUC),
```

- [ ] **Build to verify**

```bash
go build ./cmd/api/...
```

Expected: exits 0 with no output.

- [ ] **Run all tests**

```bash
go test ./...
```

Expected: PASS (integration tests are skipped without the build tag).

- [ ] **Commit**

```bash
git add cmd/api/main.go
git commit -m "$(cat <<'EOF'
feat(cmd/api): wire UpdateCampaignUC into campaign handler

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 6: API contract documentation

**Files:**
- Create: `docs/dev/api/update-campaign.md`

- [ ] **Write the contract**

Create `docs/dev/api/update-campaign.md`:

```markdown
# PATCH /campaigns/{uuid}

Update a campaign's editable fields. Two runtime modes apply based on whether
any match in the campaign has been started (`game_start_at IS NOT NULL`).

## Auth
JWT required (`Authorization: Bearer <token>`). Only the campaign's master may call this endpoint.

## Path Parameters
| Param | Type | Description |
|-------|------|-------------|
| uuid  | UUID | Campaign UUID |

## Request Body (all fields optional)

```json
{
  "name": "string (5-32 chars)",
  "brief_initial_description": "string (max 255)",
  "description": "string",
  "is_public": true,
  "call_link": "string (max 255)",
  "story_start_at": "YYYY-MM-DD",
  "story_current_at": "ISO 8601 (e.g. 2026-07-20T10:00:00Z)"
}
```

Empty body is a valid noop (returns current state).

## Field Availability by Mode

| Field | Free (no match started) | Restricted (match started) |
|-------|------------------------|---------------------------|
| `name` | ✅ editable | ❌ locked |
| `story_start_at` | ✅ editable | ❌ locked |
| `story_current_at` | ✅ any value | ✅ cannot go earlier than current value |
| `brief_initial_description` | ✅ | ✅ |
| `description` | ✅ | ✅ |
| `is_public` | ✅ | ✅ |
| `call_link` | ✅ | ✅ |

**`story_current_at` non-regression:** if the campaign already has a `story_current_at` value, the new value must be ≥ the current one. If the current value is null, any value is accepted.

## Success Response `200 OK`

```json
{
  "campaign": {
    "uuid": "uuid",
    "master_uuid": "uuid",
    "name": "string",
    "brief_initial_description": "string",
    "description": "string",
    "is_public": true,
    "call_link": "string",
    "story_start_at": "YYYY-MM-DD",
    "story_current_at": "ISO 8601 | omitted if null",
    "updated_at": "RFC 1123"
  }
}
```

## Error Responses

| Status | Condition |
|--------|-----------|
| 403 | Caller is not the campaign master |
| 404 | Campaign not found |
| 422 | Campaign has ended (story_end_at set) |
| 422 | `name` or `story_start_at` sent after match has started |
| 422 | `story_current_at` would go back in time |
| 422 | Validation error (name length, brief desc length, call link length) |
| 422 | Invalid date format |
| 500 | Unexpected server error |
```

- [ ] **Commit**

```bash
git add docs/dev/api/update-campaign.md
git commit -m "$(cat <<'EOF'
docs: add API contract for PATCH /campaigns/{uuid}

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 7: Frontend — service, type, and hook

**Files:**
- Modify: `src/types/campaign.ts`
- Modify: `src/services/campaignService.ts`
- Create: `src/hooks/useUpdateCampaign.ts`

All changes are in the `System_X_System_React/` repository.

- [ ] **Add `CampaignEditResult` type to `types/campaign.ts`**

Append at the end of `src/types/campaign.ts`:

```ts
export interface CampaignEditResult {
  uuid: string;
  masterUuid: string;
  name: string;
  briefInitialDescription: string;
  description: string;
  isPublic: boolean;
  callLink: string;
  storyStartAt: string;
  storyCurrentAt?: string;
  updatedAt: string;
}
```

- [ ] **Add `updateCampaign` to `campaignService.ts`**

Open `src/services/campaignService.ts`. Add `CampaignEditResult` to the import from `../types/campaign` and add the method:

```ts
import type { CampaignMaster, CampaignEditResult } from "../types/campaign";
```

```ts
updateCampaign: (token: string, id: string, data: object): Promise<CampaignEditResult> =>
  httpClient
    .patch<{ campaign: CampaignEditResult }>(`/campaigns/${id}`, objToSnakeCase(data), config(token))
    .then(({ data }) => objToCamelCase<CampaignEditResult>(data.campaign)),
```

- [ ] **Create `useUpdateCampaign.ts`**

Create `src/hooks/useUpdateCampaign.ts`:

```ts
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { campaignService } from "../services/campaignService";

export function useUpdateCampaign(token: string | null, campaignId: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: object) =>
      campaignService.updateCampaign(token!, campaignId!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ["campaignDetails", token, campaignId],
      });
    },
  });
}
```

- [ ] **Verify TypeScript**

```bash
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Commit (frontend repo)**

```bash
git add src/types/campaign.ts src/services/campaignService.ts src/hooks/useUpdateCampaign.ts
git commit -m "$(cat <<'EOF'
feat(campaign): add updateCampaign service method and useUpdateCampaign hook

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 8: Frontend — EditCampaignPage and error messages

**Files:**
- Create: `src/features/campaign/campaignErrorMessages.ts`
- Create: `src/pages/EditCampaignPage.tsx`

- [ ] **Create `campaignErrorMessages.ts`**

Create `src/features/campaign/campaignErrorMessages.ts`:

```ts
const campaignValidationMessages: Record<string, string> = {
  "validation error: name must be at least 5 characters":
    "O nome deve ter pelo menos 5 caracteres.",
  "validation error: name cannot exceed 32 characters":
    "O nome não pode ter mais de 32 caracteres.",
  "validation error: brief description cannot exceed 255 characters":
    "A descrição breve não pode ter mais de 255 caracteres.",
  "validation error: call link cannot exceed 255 characters":
    "O link da chamada não pode ter mais de 255 caracteres.",
  "validation error: name and story_start_at cannot be changed after a match has started":
    "Nome e data de início não podem ser alterados após uma partida iniciar.",
  "validation error: story_current_at cannot be set to a date earlier than the current value":
    "A data atual da história não pode ser anterior ao valor atual.",
};

export function getCampaignValidationMessage(detail: string): string | undefined {
  return campaignValidationMessages[detail];
}
```

- [ ] **Create `EditCampaignPage.tsx`**

Create `src/pages/EditCampaignPage.tsx`:

```tsx
import { useState, useEffect } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import useToken from "../hooks/useToken";
import useUser from "../hooks/useUser";
import useForm from "../hooks/useForm";
import { useCampaignDetails } from "../hooks/useCampaignDetails";
import { useUpdateCampaign } from "../hooks/useUpdateCampaign";
import { isApiError } from "../services/httpClient";
import { getCampaignValidationMessage } from "../features/campaign/campaignErrorMessages";
import CreateFormTemplate from "../components/templates/CreateFormTemplate";
import FormField from "../components/molecules/FormField";
import FormRow from "../components/molecules/FormRow";
import FormCheckbox from "../components/molecules/FormCheckbox";
import FormInput from "../components/ions/FormInput";
import FormTextArea from "../components/ions/FormTextArea";
import RulesSidebar from "../components/organisms/RulesSidebar";
import RuleSection from "../components/molecules/RuleSection";
import { LoadingContainer, ErrorContainer } from "../components/atoms/PageStates";

interface CampaignFormData {
  name: string;
  briefInitialDescription: string;
  description: string;
  isPublic: boolean;
  callLink: string;
  storyStartAt: string;
  storyCurrentAt: string;
}

// ISO 8601 → "YYYY-MM-DDTHH:mm" for datetime-local input
function toDateTimeLocal(iso: string): string {
  return iso.replace("Z", "").substring(0, 16);
}

function getErrorMessage(err: unknown): string {
  if (isApiError(err, 403)) return "Apenas o mestre pode editar esta campanha.";
  if (isApiError(err, 404)) return "Campanha não encontrada.";
  if (isApiError(err, 422)) {
    const detail = (err as any).response?.data?.detail as string | undefined;
    if (detail?.toLowerCase().includes("already ended"))
      return "A campanha já foi encerrada e não pode ser editada.";
    return (
      getCampaignValidationMessage(detail ?? "") ||
      "Dados inválidos. Verifique os campos e tente novamente."
    );
  }
  return "Erro ao salvar campanha. Tente novamente.";
}

export default function EditCampaignPage() {
  const { id } = useParams<{ id: string }>();
  const { token } = useToken();
  const { user } = useUser();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const [initialized, setInitialized] = useState(false);

  const { data: campaign, isPending, isError } = useCampaignDetails(token, id);
  const { form, handleForm, setForm } = useForm<CampaignFormData>({
    name: "",
    briefInitialDescription: "",
    description: "",
    isPublic: true,
    callLink: "",
    storyStartAt: "",
    storyCurrentAt: "",
  });

  const hasStartedMatch = campaign?.matches?.some((m) => !!m.gameStartAt) ?? false;

  // Min value for story_current_at input in restricted mode
  const storyCurrentAtMin =
    hasStartedMatch && campaign?.storyCurrentAt
      ? toDateTimeLocal(campaign.storyCurrentAt)
      : undefined;

  useEffect(() => {
    if (campaign && !initialized) {
      setForm({
        name: campaign.name,
        briefInitialDescription: campaign.briefInitialDescription,
        description: campaign.description,
        isPublic: campaign.isPublic,
        callLink: campaign.callLink,
        storyStartAt: campaign.storyStartAt,
        storyCurrentAt: campaign.storyCurrentAt
          ? toDateTimeLocal(campaign.storyCurrentAt)
          : "",
      });
      setInitialized(true);
    }
  }, [campaign, initialized, setForm]);

  const { mutate: updateCampaign, isPending: isSubmitting } = useUpdateCampaign(token, id);

  if (!token) return <Navigate to="/" replace />;
  if (isPending) return <LoadingContainer>Carregando campanha...</LoadingContainer>;
  if (isError) return <ErrorContainer>Falha ao carregar campanha.</ErrorContainer>;
  if (!campaign) return <ErrorContainer>Campanha não encontrada.</ErrorContainer>;

  const isMaster = campaign.masterUuid === user?.uuid;
  if (!isMaster) return <Navigate to={`/campaigns/${id}`} replace />;
  if (campaign.storyEndAt) return <Navigate to={`/campaigns/${id}`} replace />;

  const handleTogglePublic = () => setForm({ ...form, isPublic: !form.isPublic });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.briefInitialDescription) {
      setError("A descrição breve é obrigatória.");
      return;
    }
    setError(null);

    const data: Record<string, unknown> = {
      briefInitialDescription: form.briefInitialDescription,
      description: form.description || undefined,
      isPublic: form.isPublic,
      callLink: form.callLink || undefined,
    };
    if (form.storyCurrentAt) {
      data.storyCurrentAt = `${form.storyCurrentAt}:00Z`;
    }
    if (!hasStartedMatch) {
      if (form.name) data.name = form.name;
      if (form.storyStartAt) data.storyStartAt = form.storyStartAt;
    }

    updateCampaign(data, {
      onSuccess: () => navigate(-1),
      onError: (err) => setError(getErrorMessage(err)),
    });
  };

  return (
    <CreateFormTemplate
      title="EDITAR CAMPANHA"
      error={error}
      onSubmit={handleSubmit}
      submitLabel="Salvar Alterações"
      submittingLabel="Salvando..."
      isSubmitting={isSubmitting}
      onCancel={() => navigate(-1)}
      rulesContent={
        <RulesSidebar
          title="Regras da Campanha"
          footer="Mais opções de configuração serão adicionadas em breve."
        >
          <RuleSection title="Configurações Gerais">
            As regras da campanha serão configuradas aqui.
          </RuleSection>
          <RuleSection title="Sistema de Combate">
            Configure o sistema de combate da sua campanha.
          </RuleSection>
          <RuleSection title="Progressão de Personagens">
            Define como os personagens evoluem durante a campanha.
          </RuleSection>
          <RuleSection title="Nen & Habilidades">
            Configure as regras para uso e desenvolvimento de Nen.
          </RuleSection>
        </RulesSidebar>
      }
    >
      {!hasStartedMatch && (
        <FormField label="Nome da Campanha" htmlFor="name">
          <FormInput
            id="name"
            name="name"
            value={form.name}
            onChange={handleForm}
            placeholder="Nome da campanha (5-32 caracteres)"
            autoComplete="off"
            required
          />
        </FormField>
      )}

      <FormField label="Descrição Breve" htmlFor="briefInitialDescription">
        <FormTextArea
          id="briefInitialDescription"
          name="briefInitialDescription"
          value={form.briefInitialDescription}
          onChange={handleForm}
          placeholder="Uma breve descrição inicial da campanha"
          $resize="none"
          rows={2}
          required
        />
      </FormField>

      <FormField label="Descrição Completa (Opcional)" htmlFor="description">
        <FormTextArea
          id="description"
          name="description"
          value={form.description}
          onChange={handleForm}
          placeholder="Detalhes adicionais da campanha"
          rows={3}
        />
      </FormField>

      <FormField label="Link da Chamada" htmlFor="callLink">
        <FormInput
          id="callLink"
          name="callLink"
          value={form.callLink}
          onChange={handleForm}
          placeholder="Link para chamada de vídeo/áudio"
          autoComplete="off"
        />
      </FormField>

      <FormRow>
        {!hasStartedMatch && (
          <FormField label="Data de Início da História" htmlFor="storyStartAt">
            <FormInput
              id="storyStartAt"
              name="storyStartAt"
              type="date"
              value={form.storyStartAt}
              onChange={handleForm}
              required
            />
          </FormField>
        )}

        <FormField
          label="Data Atual na História (Opcional)"
          htmlFor="storyCurrentAt"
          helpText={
            hasStartedMatch && storyCurrentAtMin
              ? "Não pode ser anterior ao valor atual"
              : "Data e hora atuais dentro do universo da história"
          }
        >
          <FormInput
            id="storyCurrentAt"
            name="storyCurrentAt"
            type="datetime-local"
            value={form.storyCurrentAt}
            onChange={handleForm}
            min={storyCurrentAtMin}
          />
        </FormField>

        <FormCheckbox
          id="isPublic"
          name="isPublic"
          groupLabel="Visibilidade"
          label="Campanha Pública"
          checked={form.isPublic}
          onChange={handleTogglePublic}
          helpText="Campanhas públicas podem ser vistas por todos os usuários"
        />
      </FormRow>
    </CreateFormTemplate>
  );
}
```

- [ ] **Verify TypeScript**

```bash
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Commit**

```bash
git add src/features/campaign/campaignErrorMessages.ts src/pages/EditCampaignPage.tsx
git commit -m "$(cat <<'EOF'
feat(campaign): add EditCampaignPage with dual-mode form

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 9: Wire route in App.tsx

**Files:**
- Modify: `src/App.tsx`

- [ ] **Add import and route**

In `src/App.tsx`, add the import:

```ts
import EditCampaignPage from "./pages/EditCampaignPage";
```

Add the route after `<Route path="/campaigns/:id" element={<CampaignPage />} />`:

```tsx
<Route path="/campaigns/:id/edit" element={<EditCampaignPage />} />
```

- [ ] **Verify TypeScript**

```bash
npx tsc --noEmit
```

Expected: no errors.

- [ ] **Smoke test manually**

Start the dev server (`npm run dev`), navigate to a campaign you master, click Edit. Verify:
- Free mode (no match started): all fields visible including name and story_start_at
- Restricted mode (match started): name and story_start_at hidden; story_current_at shows with min constraint
- Saving works; page navigates back
- Non-master user gets redirected to campaign page

- [ ] **Commit**

```bash
git add src/App.tsx
git commit -m "$(cat <<'EOF'
feat(app): register /campaigns/:id/edit route

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Self-Review

**Spec coverage check:**

| Spec requirement | Covered in task |
|-----------------|----------------|
| Single PATCH endpoint | Task 4 (handler), Task 5 (wire) |
| `GetCampaignForUpdate` one-query with EXISTS | Task 3 |
| Free mode: all fields editable | Task 2 (UC logic) |
| Restricted mode: name/story_start_at locked | Task 2 (UC logic + tests) |
| story_current_at non-regression when current is not null | Task 2 (UC logic + tests) |
| story_current_at free when current is null | Task 2 (test: `story_current_at_null_is_free_in_restricted_mode`) |
| brief_final_description/story_end_at NOT editable | Not in any input struct ✅ |
| ErrCampaignAlreadyEnded | Task 1 (error), Task 2 (UC), Task 4 (handler) |
| ErrLockedAfterMatchStart | Task 1 (error), Task 2 (UC), Task 4 (handler) |
| ErrCannotRegressStoryCurrentAt | Task 1 (error), Task 2 (UC), Task 4 (handler) |
| Handler returns CampaignEditResponse (lightweight) | Task 4 |
| Gateway UpdateCampaign with unconditional rollback defer | Task 3 |
| Gateway GetCampaignForUpdate integration tests | Task 3 |
| Frontend hides locked fields proactively | Task 8 |
| Frontend min attribute for story_current_at | Task 8 |
| Frontend invalidates campaignDetails on success | Task 7 (hook) |
| API contract doc | Task 6 |

No gaps found.

**Type consistency check:**
- `CampaignUpdateContext` defined in Task 1, used in Tasks 2, 3 ✅
- `IUpdateCampaign` defined in Task 1, implemented in Task 2, mocked in Task 4 ✅
- `UpdateCampaignRequest/Response` defined and tested in Task 4 ✅
- `CampaignEditResult` defined in Task 7, used by service and (indirectly) hook ✅
