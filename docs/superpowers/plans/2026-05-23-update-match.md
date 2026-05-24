# Update Match (PATCH) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `PATCH /matches/{uuid}` endpoint that allows the match master to update editable fields while `game_start_at IS NULL`. Also create the missing API contract doc `docs/dev/api/match.md`.

**Architecture:** Three layers mirroring `CreateMatch` — HTTP handler in `internal/app/api/match/`, use case in `internal/application/match/`, gateway in `internal/gateway/pg/match/`. Body uses pointer fields for true partial PATCH; UC loads the match, validates only present fields, applies them in-memory, persists via repository whose SQL has `WHERE game_start_at IS NULL` as race guard against concurrent `StartMatch`.

**Tech Stack:** Go 1.24+, huma v2 (HTTP framework), chi router, pgx (Postgres), Postgres integration tests via `pgtest`.

**Spec:** `docs/superpowers/specs/2026-05-23-update-match-design.md`

---

## File Structure

| Status | Path | Responsibility |
|---|---|---|
| New | `internal/application/match/update_match.go` | `UpdateMatchUC` use case, validations, business rules |
| New | `internal/app/api/match/update_match.go` | HTTP handler: parse body, call UC, map errors → status |
| New | `internal/app/api/match/update_match_test.go` | Handler unit tests |
| New | `internal/gateway/pg/match/update_match.go` | `Repository.UpdateMatch` SQL |
| New | `docs/dev/api/match.md` | Full API contract for `/matches` surface |
| Modified | `internal/application/match/i_repository.go` | Add `UpdateMatch` to interface |
| Modified | `internal/application/testutil/mock_match_repo.go` | Add `UpdateMatchFn` + method |
| Modified | `internal/application/match/match_uc_test.go` | Add `TestUpdateMatch` |
| Modified | `internal/app/api/match/routes.go` | Add handler field + `huma.Register` for PATCH |
| Modified | `internal/app/api/match/mocks_test.go` | Add `mockUpdateMatch` |
| Modified | `internal/gateway/pg/match/match_integration_test.go` | Add `TestUpdateMatch` (integration) |
| Modified | `cmd/api/main.go` | Wire `NewUpdateMatchUC` + handler |
| Modified | `docs/documentation-map.yaml` | Map `internal/app/api/match/` → `docs/dev/api/match.md` |

---

### Task 1: Extend `IRepository` interface and mock

Adds the method signature so subsequent tasks compile. No behavior change yet.

**Files:**
- Modify: `internal/application/match/i_repository.go`
- Modify: `internal/application/testutil/mock_match_repo.go`

- [ ] **Step 1: Add `UpdateMatch` to `IRepository`**

Edit `internal/application/match/i_repository.go` — add one line inside the `IRepository` interface (right after `CreateMatch`):

```go
type IRepository interface {
	CreateMatch(ctx context.Context, match *match.Match) error
	UpdateMatch(ctx context.Context, match *match.Match) error
	GetMatch(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
	GetMatchCampaignUUID(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
	StartMatch(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error
	ListParticipantsByMatchUUID(ctx context.Context, matchUUID uuid.UUID) ([]*match.Participant, error)
	ListMatchesByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
	ListPublicUpcomingMatches(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error)
}
```

- [ ] **Step 2: Add `UpdateMatchFn` and method to `MockMatchRepo`**

Edit `internal/application/testutil/mock_match_repo.go`:

In the struct, add `UpdateMatchFn` right after `CreateMatchFn`:

```go
type MockMatchRepo struct {
	CreateMatchFn                        func(ctx context.Context, match *match.Match) error
	UpdateMatchFn                        func(ctx context.Context, match *match.Match) error
	GetMatchFn                           func(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
	// ... rest unchanged
}
```

At the bottom of the file, add the method:

```go
func (m *MockMatchRepo) UpdateMatch(ctx context.Context, mt *match.Match) error {
	if m.UpdateMatchFn != nil {
		return m.UpdateMatchFn(ctx, mt)
	}
	return nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd System_X_System && go build ./...`
Expected: builds without errors.

- [ ] **Step 4: Commit**

```bash
cd System_X_System
git add internal/application/match/i_repository.go internal/application/testutil/mock_match_repo.go
git commit -m "feat(match): add UpdateMatch to repository interface

Extend IRepository with UpdateMatch and wire the mock so subsequent
PATCH endpoint work compiles cleanly."
```

---

### Task 2: Use case `UpdateMatchUC` with tests

TDD. Write the failing tests first, then implement.

**Files:**
- Create: `internal/application/match/update_match.go`
- Modify: `internal/application/match/match_uc_test.go`

- [ ] **Step 1: Add `TestUpdateMatch` to `match_uc_test.go`**

Append to `internal/application/match/match_uc_test.go` (after `TestListPublicUpcomingMatches`):

```go
func validUpdateMatchInput(matchUUID, masterUUID uuid.UUID) *domainMatch.UpdateMatchInput {
	title := "Updated Title"
	brief := "Updated brief"
	desc := "Updated description"
	isPublic := false
	gameAt := time.Now().Add(48 * time.Hour)
	storyAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	return &domainMatch.UpdateMatchInput{
		MatchUUID:               matchUUID,
		MasterUUID:              masterUUID,
		Title:                   &title,
		BriefInitialDescription: &brief,
		Description:             &desc,
		IsPublic:                &isPublic,
		GameScheduledAt:         &gameAt,
		StoryStartAt:            &storyAt,
	}
}

func loadMatchFn(m *matchEntity.Match) func(context.Context, uuid.UUID) (*matchEntity.Match, error) {
	return func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
		// return a copy so the UC can mutate without affecting later calls in the same test
		copy := *m
		return &copy, nil
	}
}

func TestUpdateMatch(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()

	baseMatch := &matchEntity.Match{
		UUID:                    matchUUID,
		MasterUUID:              masterUUID,
		CampaignUUID:            uuid.New(),
		Title:                   "Original Title",
		BriefInitialDescription: "Original brief",
		Description:             "Original description",
		IsPublic:                true,
		GameScheduledAt:         time.Now().Add(24 * time.Hour),
		StoryStartAt:            time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:               time.Now().Add(-24 * time.Hour),
		UpdatedAt:               time.Now().Add(-24 * time.Hour),
	}

	validCampaign := func() *campaignEntity.Campaign {
		return &campaignEntity.Campaign{
			MasterUUID:   masterUUID,
			StoryStartAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}
	}

	t.Run("success full patch", func(t *testing.T) {
		input := validUpdateMatchInput(matchUUID, masterUUID)
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn:    loadMatchFn(baseMatch),
			UpdateMatchFn: func(_ context.Context, m *matchEntity.Match) error { return nil },
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return validCampaign(), nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)

		got, err := uc.Update(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Title != *input.Title {
			t.Errorf("title = %q, want %q", got.Title, *input.Title)
		}
		if got.IsPublic != *input.IsPublic {
			t.Errorf("isPublic = %v, want %v", got.IsPublic, *input.IsPublic)
		}
	})

	t.Run("success partial patch — title only", func(t *testing.T) {
		title := "Only Title Changed"
		input := &domainMatch.UpdateMatchInput{
			MatchUUID: matchUUID, MasterUUID: masterUUID, Title: &title,
		}
		updateCalled := false
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: loadMatchFn(baseMatch),
			UpdateMatchFn: func(_ context.Context, m *matchEntity.Match) error {
				updateCalled = true
				if m.Title != title {
					t.Errorf("persisted title = %q, want %q", m.Title, title)
				}
				if m.BriefInitialDescription != baseMatch.BriefInitialDescription {
					t.Errorf("brief mutated unexpectedly")
				}
				return nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})

		_, err := uc.Update(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updateCalled {
			t.Fatal("UpdateMatch should have been called")
		}
	})

	t.Run("no-op when all fields nil", func(t *testing.T) {
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID}
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: loadMatchFn(baseMatch),
			UpdateMatchFn: func(_ context.Context, _ *matchEntity.Match) error {
				t.Fatal("UpdateMatch should NOT be called for no-op input")
				return nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})

		got, err := uc.Update(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Title != baseMatch.Title {
			t.Errorf("title changed unexpectedly")
		}
	})

	t.Run("match not found", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return nil, matchPg.ErrMatchNotFound
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), validUpdateMatchInput(matchUUID, masterUUID))
		if !errors.Is(err, domainMatch.ErrMatchNotFound) {
			t.Fatalf("got %v, want ErrMatchNotFound", err)
		}
	})

	t.Run("not master", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		input := validUpdateMatchInput(matchUUID, uuid.New()) // different master
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrNotMatchMaster) {
			t.Fatalf("got %v, want ErrNotMatchMaster", err)
		}
	})

	t.Run("match already started", func(t *testing.T) {
		started := *baseMatch
		now := time.Now()
		started.GameStartAt = &now
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(&started)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), validUpdateMatchInput(matchUUID, masterUUID))
		if !errors.Is(err, domainMatch.ErrMatchAlreadyStarted) {
			t.Fatalf("got %v, want ErrMatchAlreadyStarted", err)
		}
	})

	t.Run("match already finished", func(t *testing.T) {
		finished := *baseMatch
		end := time.Now()
		finished.StoryEndAt = &end
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(&finished)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), validUpdateMatchInput(matchUUID, masterUUID))
		if !errors.Is(err, domainMatch.ErrMatchAlreadyFinished) {
			t.Fatalf("got %v, want ErrMatchAlreadyFinished", err)
		}
	})

	t.Run("title too short", func(t *testing.T) {
		short := "ab"
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, Title: &short}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMinTitleLength) {
			t.Fatalf("got %v, want ErrMinTitleLength", err)
		}
	})

	t.Run("title too long", func(t *testing.T) {
		long := "this title is way too long for the maximum limit"
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, Title: &long}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMaxTitleLength) {
			t.Fatalf("got %v, want ErrMaxTitleLength", err)
		}
	})

	t.Run("brief too long", func(t *testing.T) {
		brief := string(make([]byte, 65))
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, BriefInitialDescription: &brief}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMaxBriefDescLength) {
			t.Fatalf("got %v, want ErrMaxBriefDescLength", err)
		}
	})

	t.Run("game scheduled in past", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, GameScheduledAt: &past}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMinOfGameScheduledAt) {
			t.Fatalf("got %v, want ErrMinOfGameScheduledAt", err)
		}
	})

	t.Run("game scheduled too far", func(t *testing.T) {
		far := time.Now().AddDate(1, 1, 0)
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, GameScheduledAt: &far}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMaxOfGameScheduledAt) {
			t.Fatalf("got %v, want ErrMaxOfGameScheduledAt", err)
		}
	})

	t.Run("story start before campaign start", func(t *testing.T) {
		before := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, StoryStartAt: &before}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return validCampaign(), nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMinOfStoryStartAt) {
			t.Fatalf("got %v, want ErrMinOfStoryStartAt", err)
		}
	})

	t.Run("story start after campaign end", func(t *testing.T) {
		end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		after := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, StoryStartAt: &after}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				c := validCampaign()
				c.StoryEndAt = &end
				return c, nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMaxOfStoryStartAt) {
			t.Fatalf("got %v, want ErrMaxOfStoryStartAt", err)
		}
	})

	t.Run("repo update returns not found — race with start", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: loadMatchFn(baseMatch),
			UpdateMatchFn: func(_ context.Context, _ *matchEntity.Match) error {
				return matchPg.ErrMatchNotFound
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return validCampaign(), nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)
		_, err := uc.Update(context.Background(), validUpdateMatchInput(matchUUID, masterUUID))
		if !errors.Is(err, domainMatch.ErrMatchAlreadyStarted) {
			t.Fatalf("got %v, want ErrMatchAlreadyStarted (race mapping)", err)
		}
	})
}
```

- [ ] **Step 2: Run tests — expect compile failure**

Run: `cd System_X_System && go test ./internal/application/match/... -run TestUpdateMatch`
Expected: build fails with "undefined: domainMatch.UpdateMatchInput / NewUpdateMatchUC / IUpdateMatch".

- [ ] **Step 3: Implement `UpdateMatchUC`**

Create `internal/application/match/update_match.go`:

```go
package match

import (
	"context"
	"time"

	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	pgMatch "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IUpdateMatch interface {
	Update(ctx context.Context, input *UpdateMatchInput) (*match.Match, error)
}

type UpdateMatchInput struct {
	MatchUUID  uuid.UUID
	MasterUUID uuid.UUID

	Title                   *string
	BriefInitialDescription *string
	Description             *string
	IsPublic                *bool
	GameScheduledAt         *time.Time
	StoryStartAt            *time.Time
}

type UpdateMatchUC struct {
	matchRepo    IRepository
	campaignRepo domainCampaign.IRepository
}

func NewUpdateMatchUC(
	matchRepo IRepository,
	campaignRepo domainCampaign.IRepository,
) *UpdateMatchUC {
	return &UpdateMatchUC{matchRepo: matchRepo, campaignRepo: campaignRepo}
}

func (uc *UpdateMatchUC) Update(
	ctx context.Context, input *UpdateMatchInput,
) (*match.Match, error) {
	m, err := uc.matchRepo.GetMatch(ctx, input.MatchUUID)
	if err != nil {
		if err == pgMatch.ErrMatchNotFound {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	if m.MasterUUID != input.MasterUUID {
		return nil, ErrNotMatchMaster
	}
	if m.GameStartAt != nil {
		return nil, ErrMatchAlreadyStarted
	}
	if m.StoryEndAt != nil {
		return nil, ErrMatchAlreadyFinished
	}

	if input.Title == nil && input.BriefInitialDescription == nil &&
		input.Description == nil && input.IsPublic == nil &&
		input.GameScheduledAt == nil && input.StoryStartAt == nil {
		return m, nil
	}

	if input.Title != nil {
		if len(*input.Title) < 5 {
			return nil, ErrMinTitleLength
		}
		if len(*input.Title) > 32 {
			return nil, ErrMaxTitleLength
		}
	}
	if input.BriefInitialDescription != nil && len(*input.BriefInitialDescription) > 64 {
		return nil, ErrMaxBriefDescLength
	}
	if input.GameScheduledAt != nil {
		now := time.Now()
		if input.GameScheduledAt.Before(now) {
			return nil, ErrMinOfGameScheduledAt
		}
		if input.GameScheduledAt.After(now.AddDate(1, 0, 0)) {
			return nil, ErrMaxOfGameScheduledAt
		}
	}
	if input.StoryStartAt != nil {
		campaign, err := uc.campaignRepo.GetCampaignStoryDates(ctx, m.CampaignUUID)
		if err == pgCampaign.ErrCampaignNotFound {
			return nil, domainCampaign.ErrCampaignNotFound
		}
		if err != nil {
			return nil, err
		}
		if input.StoryStartAt.Before(campaign.StoryStartAt) {
			return nil, ErrMinOfStoryStartAt
		}
		if campaign.StoryEndAt != nil && input.StoryStartAt.After(*campaign.StoryEndAt) {
			return nil, ErrMaxOfStoryStartAt
		}
	}

	if input.Title != nil {
		m.Title = *input.Title
	}
	if input.BriefInitialDescription != nil {
		m.BriefInitialDescription = *input.BriefInitialDescription
	}
	if input.Description != nil {
		m.Description = *input.Description
	}
	if input.IsPublic != nil {
		m.IsPublic = *input.IsPublic
	}
	if input.GameScheduledAt != nil {
		m.GameScheduledAt = *input.GameScheduledAt
	}
	if input.StoryStartAt != nil {
		m.StoryStartAt = *input.StoryStartAt
	}
	m.UpdatedAt = time.Now()

	if err := uc.matchRepo.UpdateMatch(ctx, m); err != nil {
		if err == pgMatch.ErrMatchNotFound {
			return nil, ErrMatchAlreadyStarted
		}
		return nil, err
	}
	return m, nil
}
```

- [ ] **Step 4: Run tests — expect pass**

Run: `cd System_X_System && go test ./internal/application/match/... -run TestUpdateMatch -v`
Expected: all 14 sub-tests PASS.

- [ ] **Step 5: Run full unit suite (regression check)**

Run: `cd System_X_System && go test ./internal/application/match/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd System_X_System
git add internal/application/match/update_match.go internal/application/match/match_uc_test.go
git commit -m "feat(match): UpdateMatchUC with partial PATCH semantics

Validates only present fields, blocks updates once game_start_at or
story_end_at is set, maps race with concurrent StartMatch to
ErrMatchAlreadyStarted."
```

---

### Task 3: Repository `UpdateMatch` SQL + integration test

**Files:**
- Create: `internal/gateway/pg/match/update_match.go`
- Modify: `internal/gateway/pg/match/match_integration_test.go`

- [ ] **Step 1: Write integration test**

Append to `internal/gateway/pg/match/match_integration_test.go` (after `TestListParticipantsByMatchUUID`):

```go
func TestUpdateMatch(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	t.Run("happy path updates fields and bumps updated_at", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_upd", "gm_upd@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Update Campaign"))

		m := newTestMatch(masterUUID, campaignUUID, "Original", true, time.Now().Add(24*time.Hour))
		if err := repo.CreateMatch(ctx, m); err != nil {
			t.Fatalf("CreateMatch() unexpected error: %v", err)
		}

		original, err := repo.GetMatch(ctx, m.UUID)
		if err != nil {
			t.Fatalf("GetMatch() after create: %v", err)
		}
		originalUpdated := original.UpdatedAt

		// Apply changes in memory and persist
		time.Sleep(10 * time.Millisecond) // ensure UpdatedAt differs measurably
		original.Title = "Patched"
		original.BriefInitialDescription = "Patched brief"
		original.Description = "Patched description"
		original.IsPublic = false
		original.GameScheduledAt = time.Now().Add(72 * time.Hour).Truncate(time.Microsecond)
		original.StoryStartAt = time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
		original.UpdatedAt = time.Now().Truncate(time.Microsecond)

		if err := repo.UpdateMatch(ctx, original); err != nil {
			t.Fatalf("UpdateMatch() unexpected error: %v", err)
		}

		got, err := repo.GetMatch(ctx, m.UUID)
		if err != nil {
			t.Fatalf("GetMatch() after update: %v", err)
		}
		if got.Title != "Patched" {
			t.Errorf("Title = %q, want %q", got.Title, "Patched")
		}
		if got.IsPublic != false {
			t.Errorf("IsPublic = %v, want false", got.IsPublic)
		}
		if !got.UpdatedAt.After(originalUpdated) {
			t.Errorf("UpdatedAt = %v, want after %v", got.UpdatedAt, originalUpdated)
		}
	})

	t.Run("race guard — already started rejects update", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_race", "gm_race@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Race Campaign"))

		m := newTestMatch(masterUUID, campaignUUID, "Will Start", true, time.Now().Add(24*time.Hour))
		if err := repo.CreateMatch(ctx, m); err != nil {
			t.Fatalf("CreateMatch() unexpected error: %v", err)
		}
		if err := repo.StartMatch(ctx, m.UUID, time.Now()); err != nil {
			t.Fatalf("StartMatch() unexpected error: %v", err)
		}

		m.Title = "Should not persist"
		m.UpdatedAt = time.Now()
		err := repo.UpdateMatch(ctx, m)
		if !errors.Is(err, pgMatch.ErrMatchNotFound) {
			t.Errorf("UpdateMatch() after start: got %v, want ErrMatchNotFound", err)
		}
	})

	t.Run("unknown uuid returns not found", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)
		m := &entityMatch.Match{
			UUID:                    uuid.New(),
			Title:                   "Ghost",
			BriefInitialDescription: "x",
			Description:             "x",
			IsPublic:                true,
			GameScheduledAt:         time.Now(),
			StoryStartAt:            time.Now(),
			UpdatedAt:               time.Now(),
		}
		err := repo.UpdateMatch(ctx, m)
		if !errors.Is(err, pgMatch.ErrMatchNotFound) {
			t.Errorf("UpdateMatch() for unknown uuid: got %v, want ErrMatchNotFound", err)
		}
	})
}
```

- [ ] **Step 2: Run integration test — expect compile failure**

Run: `cd System_X_System && go test -tags=integration ./internal/gateway/pg/match/... -run TestUpdateMatch`
Expected: build fails with "repo.UpdateMatch undefined".

- [ ] **Step 3: Implement `UpdateMatch` repository method**

Create `internal/gateway/pg/match/update_match.go`:

```go
package match

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
)

func (r *Repository) UpdateMatch(ctx context.Context, m *match.Match) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx) // no-op after Commit
	}()

	const query = `
		UPDATE matches SET
			title = $1,
			brief_initial_description = $2,
			description = $3,
			is_public = $4,
			game_scheduled_at = $5,
			story_start_at = $6,
			updated_at = $7
		WHERE uuid = $8 AND game_start_at IS NULL
	`
	result, err := tx.Exec(ctx, query,
		m.Title, m.BriefInitialDescription, m.Description,
		m.IsPublic, m.GameScheduledAt, m.StoryStartAt,
		m.UpdatedAt, m.UUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update match: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return tx.Commit(ctx)
}
```

- [ ] **Step 4: Run integration test — expect pass**

Run: `cd System_X_System && go test -tags=integration ./internal/gateway/pg/match/... -run TestUpdateMatch -v`
Expected: 3 sub-tests PASS.

- [ ] **Step 5: Run `go vet` for the changed packages**

Run: `cd System_X_System && go vet ./internal/gateway/pg/match/... ./internal/application/match/...`
Expected: no warnings.

- [ ] **Step 6: Commit**

```bash
cd System_X_System
git add internal/gateway/pg/match/update_match.go internal/gateway/pg/match/match_integration_test.go
git commit -m "feat(match): UpdateMatch repository with race guard

UPDATE … WHERE uuid = \$X AND game_start_at IS NULL guarantees a concurrent
StartMatch wins, surfacing ErrMatchNotFound to the UC for race mapping."
```

---

### Task 4: HTTP handler `UpdateMatchHandler` + unit tests

**Files:**
- Create: `internal/app/api/match/update_match.go`
- Create: `internal/app/api/match/update_match_test.go`
- Modify: `internal/app/api/match/mocks_test.go`

- [ ] **Step 1: Add `mockUpdateMatch` to `mocks_test.go`**

Append to `internal/app/api/match/mocks_test.go`:

```go
type mockUpdateMatch struct {
	fn func(ctx context.Context, input *domainMatch.UpdateMatchInput) (*matchEntity.Match, error)
}

func (m *mockUpdateMatch) Update(
	ctx context.Context, input *domainMatch.UpdateMatchInput,
) (*matchEntity.Match, error) {
	return m.fn(ctx, input)
}
```

- [ ] **Step 2: Write handler test**

Create `internal/app/api/match/update_match_test.go`:

```go
package match_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestUpdateMatchHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	now := time.Now()

	baseResp := func(title string) *matchEntity.Match {
		return &matchEntity.Match{
			UUID:                    matchUUID,
			MasterUUID:              userUUID,
			CampaignUUID:            uuid.New(),
			Title:                   title,
			BriefInitialDescription: "brief",
			Description:             "full",
			IsPublic:                true,
			GameScheduledAt:         now,
			StoryStartAt:            now,
			CreatedAt:               now,
			UpdatedAt:               now,
		}
	}

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *domainMatch.UpdateMatchInput) (*matchEntity.Match, error)
		wantStatus int
	}{
		{
			name: "success_full_patch",
			body: map[string]any{
				"title":                     "Patched Title",
				"brief_initial_description": "Patched brief",
				"description":               "Patched desc",
				"is_public":                 false,
				"game_scheduled_at":         "2026-07-20T19:30:00Z",
				"story_start_at":            "2026-07-20",
			},
			mockFn: func(_ context.Context, input *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				if input.Title == nil || *input.Title != "Patched Title" {
					t.Errorf("title not forwarded: %+v", input.Title)
				}
				return baseResp("Patched Title"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success_partial_patch_title_only",
			body: map[string]any{"title": "Only Title"},
			mockFn: func(_ context.Context, input *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				if input.BriefInitialDescription != nil {
					t.Errorf("brief should be nil, got %+v", *input.BriefInitialDescription)
				}
				return baseResp("Only Title"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "success_empty_body_is_noop",
			body: map[string]any{},
			mockFn: func(_ context.Context, _ *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				return baseResp("Original"), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "invalid_game_scheduled_at",
			body: map[string]any{"game_scheduled_at": "not-a-date"},
			mockFn: func(_ context.Context, _ *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				t.Fatal("UC should not be called when date parsing fails")
				return nil, nil
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid_story_start_at",
			body: map[string]any{"story_start_at": "not-a-date"},
			mockFn: func(_ context.Context, _ *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				t.Fatal("UC should not be called when date parsing fails")
				return nil, nil
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "match_not_found",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "not_master",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, domainMatch.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "already_started",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, domainMatch.ErrMatchAlreadyStarted
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "already_finished",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, domainMatch.ErrMatchAlreadyFinished
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "validation_error",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, domain.ErrValidation
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_server_error",
			body: map[string]any{"title": "anything"},
			mockFn: func(_ context.Context, _ *domainMatch.UpdateMatchInput) (*matchEntity.Match, error) {
				return nil, errors.New("db down")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)
			mock := &mockUpdateMatch{fn: tt.mockFn}
			handler := match.UpdateMatchHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodPatch,
				Path:   "/matches/{uuid}",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PatchCtx(ctx, "/matches/"+matchUUID.String(), tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				matchData, ok := result["match"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'match' field")
				}
				if matchData["master_uuid"] != userUUID.String() {
					t.Errorf("master_uuid = %v, want %v", matchData["master_uuid"], userUUID.String())
				}
			}
		})
	}
}
```

- [ ] **Step 3: Run tests — expect compile failure**

Run: `cd System_X_System && go test ./internal/app/api/match/... -run TestUpdateMatchHandler`
Expected: build fails with "match.UpdateMatchHandler undefined".

- [ ] **Step 4: Implement `UpdateMatchHandler`**

Create `internal/app/api/match/update_match.go`:

```go
package match

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type UpdateMatchRequestBody struct {
	Title                   *string `json:"title,omitempty" minLength:"5" maxLength:"32" doc:"New title (5-32 chars)"`
	BriefInitialDescription *string `json:"brief_initial_description,omitempty" maxLength:"64" doc:"New brief description"`
	Description             *string `json:"description,omitempty" doc:"New full description"`
	IsPublic                *bool   `json:"is_public,omitempty" doc:"New public/private flag"`
	GameScheduledAt         *string `json:"game_scheduled_at,omitempty" doc:"ISO 8601 date-time"`
	StoryStartAt            *string `json:"story_start_at,omitempty" doc:"YYYY-MM-DD"`
}

type UpdateMatchRequest struct {
	UUID uuid.UUID              `path:"uuid" required:"true" doc:"UUID of the match to update"`
	Body UpdateMatchRequestBody `json:"body"`
}

type UpdateMatchResponseBody struct {
	Match MatchResponse `json:"match"`
}

type UpdateMatchResponse struct {
	Body UpdateMatchResponseBody `json:"body"`
}

func UpdateMatchHandler(
	uc domainMatch.IUpdateMatch,
) func(context.Context, *UpdateMatchRequest) (*UpdateMatchResponse, error) {

	return func(ctx context.Context, req *UpdateMatchRequest) (*UpdateMatchResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		input := &domainMatch.UpdateMatchInput{
			MatchUUID:               req.UUID,
			MasterUUID:              userUUID,
			Title:                   req.Body.Title,
			BriefInitialDescription: req.Body.BriefInitialDescription,
			Description:             req.Body.Description,
			IsPublic:                req.Body.IsPublic,
		}

		if req.Body.GameScheduledAt != nil {
			t, err := time.Parse(time.RFC3339, *req.Body.GameScheduledAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(
					"invalid game_scheduled_at date format, use ISO 8601. E.g. 2026-06-15T19:30:00Z")
			}
			input.GameScheduledAt = &t
		}
		if req.Body.StoryStartAt != nil {
			t, err := time.Parse("2006-01-02", *req.Body.StoryStartAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(
					"invalid story_start_at date format, use YYYY-MM-DD")
			}
			input.StoryStartAt = &t
		}

		m, err := uc.Update(ctx, input)
		if err != nil {
			switch {
			case errors.Is(err, domainMatch.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainCampaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainMatch.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, domainMatch.ErrMatchAlreadyStarted),
				errors.Is(err, domainMatch.ErrMatchAlreadyFinished):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			case errors.Is(err, domain.ErrValidation):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		var gameStartAtStr *string
		if m.GameStartAt != nil {
			s := m.GameStartAt.Format(time.RFC3339)
			gameStartAtStr = &s
		}
		var storyEndAtStr *string
		if m.StoryEndAt != nil {
			s := m.StoryEndAt.Format("2006-01-02")
			storyEndAtStr = &s
		}

		response := MatchResponse{
			UUID:                    m.UUID,
			MasterUUID:              m.MasterUUID,
			CampaignUUID:            m.CampaignUUID,
			Title:                   m.Title,
			BriefInitialDescription: m.BriefInitialDescription,
			BriefFinalDescription:   m.BriefFinalDescription,
			Description:             m.Description,
			IsPublic:                m.IsPublic,
			GameScheduledAt:         m.GameScheduledAt.Format(time.RFC3339),
			GameStartAt:             gameStartAtStr,
			StoryStartAt:            m.StoryStartAt.Format("2006-01-02"),
			StoryEndAt:              storyEndAtStr,
			CreatedAt:               m.CreatedAt.Format(http.TimeFormat),
			UpdatedAt:               m.UpdatedAt.Format(http.TimeFormat),
		}
		return &UpdateMatchResponse{
			Body: UpdateMatchResponseBody{Match: response},
		}, nil
	}
}
```

- [ ] **Step 5: Run handler tests — expect pass**

Run: `cd System_X_System && go test ./internal/app/api/match/... -run TestUpdateMatchHandler -v`
Expected: 11 sub-tests PASS.

- [ ] **Step 6: Run full handler suite (regression)**

Run: `cd System_X_System && go test ./internal/app/api/match/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
cd System_X_System
git add internal/app/api/match/update_match.go internal/app/api/match/update_match_test.go internal/app/api/match/mocks_test.go
git commit -m "feat(match): UpdateMatchHandler with partial-PATCH parsing

Parses date fields only when present, maps domain errors to HTTP
404/403/422 according to the contract."
```

---

### Task 5: Register route + wire in `main.go`

**Files:**
- Modify: `internal/app/api/match/routes.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Add field to `Api` struct in `routes.go`**

Edit `internal/app/api/match/routes.go`. Add the field after `CreateMatchHandler`:

```go
type Api struct {
	CreateMatchHandler               Handler[CreateMatchRequest, CreateMatchResponse]
	UpdateMatchHandler               Handler[UpdateMatchRequest, UpdateMatchResponse]
	GetMatchHandler                  Handler[GetMatchRequest, GetMatchResponse]
	ListMatchesHandler               Handler[struct{}, ListMatchesResponse]
	ListPublicUpcomingMatchesHandler Handler[struct{}, ListMatchesResponse]
	ListMatchEnrollmentsHandler      Handler[ListMatchEnrollmentsRequest, ListMatchEnrollmentsResponse]
	GetMatchParticipantsHandler      Handler[GetMatchParticipantsRequest, GetMatchParticipantsResponse]
}
```

- [ ] **Step 2: Register the PATCH route**

In the same file, add a new `huma.Register` block inside `RegisterRoutes` (right after the POST `/matches` block):

```go
huma.Register(api, huma.Operation{
	Method:      http.MethodPatch,
	Path:        "/matches/{uuid}",
	Description: "Update a match (master only, before game starts)",
	Tags:        []string{"matches"},
	Errors: []int{
		http.StatusNotFound,
		http.StatusBadRequest,
		http.StatusForbidden,
		http.StatusUnauthorized,
		http.StatusUnprocessableEntity,
		http.StatusInternalServerError,
	},
}, a.UpdateMatchHandler)
```

- [ ] **Step 3: Wire UC and handler in `cmd/api/main.go`**

Edit `cmd/api/main.go`. After the `createMatchUC` line (around line 180), add:

```go
updateMatchUC := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)
```

In the `matchesApi := matchHandler.Api{` block (around line 187), add the field after `CreateMatchHandler`:

```go
matchesApi := matchHandler.Api{
	CreateMatchHandler:               matchHandler.CreateMatchHandler(createMatchUC),
	UpdateMatchHandler:               matchHandler.UpdateMatchHandler(updateMatchUC),
	GetMatchHandler:                  matchHandler.GetMatchHandler(getMatchUC),
	// ... rest unchanged
}
```

- [ ] **Step 4: Verify build**

Run: `cd System_X_System && go build ./...`
Expected: builds clean.

- [ ] **Step 5: Run full vet + unit tests**

Run: `cd System_X_System && go vet ./... && go test ./...`
Expected: no warnings, all tests PASS.

- [ ] **Step 6: Commit**

```bash
cd System_X_System
git add internal/app/api/match/routes.go cmd/api/main.go
git commit -m "feat(match): wire PATCH /matches/{uuid} route

Registers the route under matches tag and instantiates UpdateMatchUC
with matchRepo + campaignRepo."
```

---

### Task 6: API contract docs + documentation-map mapping

**Files:**
- Create: `docs/dev/api/match.md`
- Modify: `docs/documentation-map.yaml`

- [ ] **Step 1: Write `docs/dev/api/match.md`**

Create `docs/dev/api/match.md` with the full Match REST surface:

````markdown
# Match API

## POST /matches — Criar partida

**Auth:** JWT (master da campanha)

### Request

```json
{
  "campaign_uuid": "uuid-v4",
  "title": "Greed Island - Session 1",
  "brief_initial_description": "Os heróis chegam à ilha (máx 64 chars)",
  "description": "Descrição completa apenas para o mestre",
  "is_public": true,
  "game_scheduled_at": "2026-06-15T19:30:00Z",
  "story_start_at": "2026-06-15"
}
```

| Campo | Regra |
|---|---|
| `title` | obrigatório, 5–32 chars |
| `brief_initial_description` | ≤ 64 chars |
| `game_scheduled_at` | ISO 8601, futuro, ≤ +1 ano |
| `story_start_at` | YYYY-MM-DD, dentro da janela da campanha |
| `is_public` | default `true` |

### Respostas

| Status | Situação |
|---|---|
| 201 | Partida criada, retorna `{ "match": MatchResponse }` |
| 400 | Body malformado |
| 401 | Sem JWT |
| 403 | Usuário não é o mestre da campanha |
| 404 | Campanha não encontrada |
| 422 | Validação (título fora do tamanho, datas fora da janela, etc.) |
| 500 | Erro interno |

---

## GET /matches/{uuid} — Obter partida

**Auth:** JWT obrigatório

### Visibilidade

- **Mestre da partida:** sempre vê.
- **Partida pública:** qualquer usuário autenticado vê.
- **Partida privada:** apenas participantes (jogadores com personagem na campanha) veem; demais recebem 403.

### Response 200

```json
{
  "match": {
    "uuid": "...",
    "master_uuid": "...",
    "campaign_uuid": "...",
    "title": "Greed Island - Session 1",
    "brief_initial_description": "...",
    "brief_final_description": null,
    "description": "...",
    "is_public": true,
    "game_scheduled_at": "2026-06-15T19:30:00Z",
    "game_start_at": null,
    "story_start_at": "2026-06-15",
    "story_end_at": null,
    "created_at": "...",
    "updated_at": "..."
  }
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Partida retornada |
| 400 | UUID inválido |
| 403 | Partida privada e usuário não é mestre nem participante |
| 404 | Partida não encontrada |
| 500 | Erro interno |

---

## PATCH /matches/{uuid} — Editar partida

**Auth:** JWT (apenas o mestre)

**Pré-condição:** `game_start_at IS NULL && story_end_at IS NULL`. Após
`StartMatch` ou encerramento, qualquer PATCH retorna 422.

### Request — todos os campos opcionais

```json
{
  "title": "Novo título",
  "brief_initial_description": "Novo brief",
  "description": "Nova descrição",
  "is_public": false,
  "game_scheduled_at": "2026-07-20T19:30:00Z",
  "story_start_at": "2026-07-20"
}
```

| Campo | Regra (se enviado) |
|---|---|
| `title` | 5–32 chars |
| `brief_initial_description` | ≤ 64 chars |
| `game_scheduled_at` | ISO 8601, futuro, ≤ +1 ano |
| `story_start_at` | YYYY-MM-DD, dentro da janela da campanha |

`campaign_uuid` é imutável — não enviar.

Body vazio (`{}`) é no-op idempotente, retorna 200 com a partida atual.

### Response 200

Mesmo formato de `GET /matches/{uuid}` — partida com campos atualizados.

### Erros

| Status | Situação |
|---|---|
| 200 | Atualizado (ou no-op) |
| 400 | Body malformado |
| 401 | Sem JWT |
| 403 | Usuário não é o mestre |
| 404 | Partida ou campanha não encontrada |
| 422 | Validação (tamanho, data fora de janela) **ou** partida já iniciada/encerrada |
| 500 | Erro interno |

---

## GET /matches — Listar partidas do mestre

**Auth:** JWT obrigatório

Retorna todas as partidas em que o usuário é mestre, ordenadas por `story_start_at` ASC.

### Response 200

```json
{
  "matches": [
    {
      "uuid": "...",
      "campaign_uuid": "...",
      "title": "...",
      "brief_initial_description": "...",
      "brief_final_description": null,
      "is_public": true,
      "game_scheduled_at": "...",
      "game_start_at": null,
      "story_start_at": "...",
      "story_end_at": null,
      "created_at": "...",
      "updated_at": "..."
    }
  ]
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Lista (pode ser vazia) |
| 400 | Token inválido |
| 401 | Sem JWT |
| 500 | Erro interno |

---

## GET /public/matches — Partidas públicas futuras

**Auth:** JWT obrigatório

Retorna partidas públicas com `game_scheduled_at > now()`, ordenadas
ASC, **excluindo** as do próprio mestre autenticado.

### Response 200

Mesmo formato de `GET /matches`.

### Erros

| Status | Situação |
|---|---|
| 200 | Lista (pode ser vazia) |
| 401 | Sem JWT |
| 500 | Erro interno |

---

## GET /matches/{uuid}/enrollments — Inscrições

**Auth:** JWT obrigatório

Visibilidade por linha:

- **Mestre da partida:** vê resumo privado de cada ficha inscrita.
- **Dono da ficha inscrita:** vê o resumo privado da própria; demais linhas, resumo base.
- **Outros:** apenas resumo base de todas as fichas.

### Response 200

```json
{
  "enrollments": [
    {
      "uuid": "...",
      "status": "pending | accepted | rejected",
      "created_at": "...",
      "character_sheet": {
        "uuid": "...",
        "nick_name": "Gon",
        "avatar_url": "...",
        "cover_url": "...",
        "private": { /* opcional, ver Character Sheet API */ }
      },
      "player": { "uuid": "...", "nick": "..." }
    }
  ]
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Lista de enrollments |
| 403 | Acesso negado |
| 404 | Partida não encontrada |
| 500 | Erro interno |

---

## GET /matches/{uuid}/participants — Participantes

**Auth:** JWT obrigatório

Lista personagens que entraram na partida (snapshot a partir de `StartMatch`).

### Response 200

```json
{
  "participants": [
    {
      "uuid": "...",
      "joined_at": "...",
      "left_at": null,
      "character_sheet": {
        "uuid": "...",
        "nick_name": "Killua",
        "avatar_url": "...",
        "cover_url": "...",
        "private": { /* opcional */ }
      }
    }
  ]
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Lista (pode ser vazia) |
| 403 | Acesso negado |
| 404 | Partida não encontrada |
| 500 | Erro interno |
````

- [ ] **Step 2: Update `docs/documentation-map.yaml`**

Open `docs/documentation-map.yaml`. Locate the `internal/app/api/` mapping (around line 302). Add a new mapping for the match REST surface above it (before the generic catch-all `internal/app/api/`):

```yaml
  # ─── Match: REST API ───
  - code_path: internal/app/api/match/
    dev_docs:
      - path: docs/dev/api/match.md
        confidence: directly_affected
    notes: Match REST surface — create/get/update/list/enrollments/participants
```

- [ ] **Step 3: Verify YAML is valid**

Run: `cd System_X_System && python3 -c "import yaml; yaml.safe_load(open('docs/documentation-map.yaml'))" && echo OK`
Expected: prints `OK`.

- [ ] **Step 4: Sanity-check docs file**

Run: `cd System_X_System && test -f docs/dev/api/match.md && echo "doc present" && wc -l docs/dev/api/match.md`
Expected: prints `doc present` and a line count > 150.

- [ ] **Step 5: Commit**

```bash
cd System_X_System
git add docs/dev/api/match.md docs/documentation-map.yaml
git commit -m "docs(api): contract for /matches surface

Documents create, get, update (new PATCH), list, public, enrollments
and participants. Maps internal/app/api/match/ in documentation-map."
```

---

### Task 7: Final smoke

- [ ] **Step 1: Full unit + vet**

Run: `cd System_X_System && go vet ./... && go test ./...`
Expected: no failures.

- [ ] **Step 2: Integration tests**

Run: `cd System_X_System && go test -tags=integration ./internal/gateway/pg/match/... -v`
Expected: all match integration tests PASS, including new `TestUpdateMatch`.

- [ ] **Step 3: Visual sanity on OpenAPI**

Start the server (`go run ./cmd/api` in a separate terminal) and `curl localhost:5000/openapi | jq '.paths["/matches/{uuid}"]'` — expect to see both `get` and `patch` keys.

- [ ] **Step 4: Done — no commit**

This task only verifies; no new files.
