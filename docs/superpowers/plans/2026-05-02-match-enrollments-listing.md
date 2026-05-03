# Match Enrollments Listing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `GET /matches/{uuid}/enrollments` returning a match's roster with per-row visibility (master sees private fields; everyone else sees base only) and align `GET /matches/{uuid}` with the same private-match access rule.

**Architecture:** Use case lives in `domain/match/` (the read is match-page-driven; enrollments are aggregated data). Cross-domain dependencies declared as local interfaces in `domain/match` (Go's "interfaces defined at consumer" idiom) to avoid the cycle that would form if `domain/match` imported `domain/enrollment` (which already imports `domain/match`). The handler decides visibility purely from one boolean flag returned by the UC.

**Tech Stack:** Go 1.23 · `chi` + `huma/v2` (HTTP) · `pgx/v5` (Postgres) · goose (migrations) · standard `testing` package · `humatest` for handler tests · `pgtest` helpers for integration tests.

**Spec reference:** `docs/superpowers/specs/2026-05-02-match-enrollments-listing-design.md` (and `.pt-br.md`).

**Phasing:**
- Phase 1 (Tasks 1–9): Read path without privacy check. Anyone authenticated can call the endpoint and see all enrollments according to the master-vs-other visibility tier.
- Phase 2 (Tasks 10–14): Add the campaign-participant privacy check to both the new endpoint and the existing `GetMatchUC`.
- Phase 3 (Tasks 15–17): Documentation.

---

## Task 0: Branch Setup

**Files:** none (git only)

- [ ] **Step 1: Create feature branch**

```bash
git checkout main
git pull origin main
git checkout -b feat/match-enrollments-listing
```

- [ ] **Step 2: Verify clean state**

```bash
git status
```

Expected: `nothing to commit, working tree clean` (the spec files are already committed on `main`; if not, abort and ensure spec is merged first).

---

## Task 1: Migration — Index for `(match_uuid, created_at)`

**Files:**
- Create: `migrations/20260502120000_add_enrollments_match_uuid_index.sql`

**Why:** The listing query filters by `match_uuid` and orders by `created_at`. The existing `idx_enrollments_sheet_match_uuid (character_sheet_uuid, match_uuid)` doesn't help — its leading column is the sheet, not the match. A composite covers both filter and sort without a sort step.

- [ ] **Step 1: Create migration via goose**

Run: `make migrate-create name=add_enrollments_match_uuid_index`

This produces an empty file under `migrations/`. Identify the generated filename (timestamp prefix differs from the example above — use the actual generated one).

- [ ] **Step 2: Write the migration body**

Replace the generated file's contents with:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE INDEX idx_enrollments_match_uuid_created_at
  ON enrollments(match_uuid, created_at);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_enrollments_match_uuid_created_at;

COMMIT;
-- +goose StatementEnd
```

- [ ] **Step 3: Apply migration locally**

Run: `make migrate-up`
Expected: goose applies the new migration, exit 0.

- [ ] **Step 4: Verify index exists**

Run:
```bash
psql "$DATABASE_URL" -c "\\di+ idx_enrollments_match_uuid_created_at"
```
Expected: row showing the index on table `enrollments`. (If `DATABASE_URL` is not set, use `psql postgres://postgres:postgres@localhost:5432/hxh_rpg`.)

- [ ] **Step 5: Verify rollback works**

Run:
```bash
make migrate-down
make migrate-up
```
Expected: down removes the index, up re-creates it. Both succeed.

- [ ] **Step 6: Commit**

```bash
git add migrations/20260502120000_add_enrollments_match_uuid_index.sql
git commit -m "$(cat <<'EOF'
feat(db): add composite index for enrollments listing by match

Covers GET /matches/{uuid}/enrollments query (filter by match_uuid + ORDER BY created_at).

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

(Replace the filename with the actual goose-generated name if different.)

---

## Task 2: Entity — `Enrollment`

**Files:**
- Create: `internal/domain/entity/enrollment/enrollment.go`

**Why:** No entity package exists for enrollment yet. Mirrors the convention from `entity/match/summary.go`. Uses `gateway/pg/model.CharacterSheetSummary` because that struct already carries every field both base and private summaries need; a TODO is included to track moving the model to the entity layer in a follow-up (existing UCs already import the same model, so the cleanup is repo-wide and not specific to this PR).

- [ ] **Step 1: Create the entity file**

```go
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
	UUID      uuid.UUID
	Status    string
	CreatedAt time.Time
	// TODO(architecture): CharacterSheetSummary lives in gateway/pg/model — entity should not
	// import outer layers. Tracked for cleanup: move CharacterSheetSummary to
	// domain/entity/character_sheet/summary.go in a follow-up task and update all call sites
	// (use cases under domain/character_sheet/ already import model.CharacterSheetSummary too,
	// so the cleanup is shared, not specific to enrollment).
	CharacterSheet sheetModel.CharacterSheetSummary
	Player         PlayerRef
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/domain/entity/enrollment/...`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/enrollment/enrollment.go
git commit -m "$(cat <<'EOF'
feat(entity): add Enrollment aggregate type

Carries enrollment metadata, joined character sheet summary, and player
reference. Used by the new match roster read path.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 3: Gateway — `ListByMatchUUID` (TDD)

**Files:**
- Create: `internal/gateway/pg/enrollment/list_by_match_uuid.go`
- Modify: `internal/gateway/pg/enrollment/enrollment_integration_test.go` (append `TestListByMatchUUID`)

- [ ] **Step 1: Write failing integration test**

Append to the END of `internal/gateway/pg/enrollment/enrollment_integration_test.go`:

```go
func TestListByMatchUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := enrollmentRepo.NewRepository(pool)

	t.Run("lists all statuses ordered by created_at and includes joined sheet+player data", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match 1")

		sheet1 := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Gon")
		sheet2 := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Killua")
		sheet3 := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Kurapika")

		pgtest.InsertTestEnrollment(t, pool, matchUUID, sheet1, "pending")
		pgtest.InsertTestEnrollment(t, pool, matchUUID, sheet2, "accepted")
		pgtest.InsertTestEnrollment(t, pool, matchUUID, sheet3, "rejected")

		got, err := repo.ListByMatchUUID(ctx, uuid.MustParse(matchUUID))
		if err != nil {
			t.Fatalf("ListByMatchUUID() error = %v, want nil", err)
		}
		if len(got) != 3 {
			t.Fatalf("ListByMatchUUID() len = %d, want 3", len(got))
		}

		// Ordered by created_at ASC — fixture insertion order.
		wantNicks := []string{"Gon", "Killua", "Kurapika"}
		wantStatuses := []string{"pending", "accepted", "rejected"}
		for i, e := range got {
			if e.CharacterSheet.NickName != wantNicks[i] {
				t.Errorf("row %d: nick = %q, want %q", i, e.CharacterSheet.NickName, wantNicks[i])
			}
			if e.Status != wantStatuses[i] {
				t.Errorf("row %d: status = %q, want %q", i, e.Status, wantStatuses[i])
			}
			if e.Player.Nick != "player1" {
				t.Errorf("row %d: player nick = %q, want %q", i, e.Player.Nick, "player1")
			}
			if e.Player.UUID.String() != playerUUID {
				t.Errorf("row %d: player uuid = %s, want %s", i, e.Player.UUID, playerUUID)
			}
			if e.CharacterSheet.UUID == uuid.Nil {
				t.Errorf("row %d: character sheet uuid is nil", i)
			}
			if e.CharacterSheet.CampaignUUID == nil {
				t.Errorf("row %d: campaign_uuid is nil, want set", i)
			}
		}
	})

	t.Run("returns empty slice when match has no enrollments", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Test Campaign")
		matchUUID := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match 1")

		got, err := repo.ListByMatchUUID(ctx, uuid.MustParse(matchUUID))
		if err != nil {
			t.Fatalf("ListByMatchUUID() error = %v, want nil", err)
		}
		if len(got) != 0 {
			t.Errorf("ListByMatchUUID() len = %d, want 0", len(got))
		}
	})

	t.Run("does not include enrollments from other matches", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Test Campaign")
		matchA := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match A")
		matchB := pgtest.InsertTestMatch(t, pool, masterUUID, campaignUUID, "Match B")

		sheetA := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Gon")
		sheetB := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Killua")

		pgtest.InsertTestEnrollment(t, pool, matchA, sheetA, "pending")
		pgtest.InsertTestEnrollment(t, pool, matchB, sheetB, "accepted")

		got, err := repo.ListByMatchUUID(ctx, uuid.MustParse(matchA))
		if err != nil {
			t.Fatalf("ListByMatchUUID() error = %v, want nil", err)
		}
		if len(got) != 1 {
			t.Fatalf("ListByMatchUUID() len = %d, want 1", len(got))
		}
		if got[0].CharacterSheet.NickName != "Gon" {
			t.Errorf("got nick %q, want %q", got[0].CharacterSheet.NickName, "Gon")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=integration -v ./internal/gateway/pg/enrollment/ -run TestListByMatchUUID`
Expected: build error / `FAIL`: `repo.ListByMatchUUID undefined`.

- [ ] **Step 3: Implement the gateway method**

Create `internal/gateway/pg/enrollment/list_by_match_uuid.go`:

```go
package enrollment

import (
	"context"
	"fmt"

	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	"github.com/google/uuid"
)

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

	rows, err := r.q.Query(ctx, query, matchUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query enrollments by match: %w", err)
	}
	defer rows.Close()

	out := make([]*enrollmentEntity.Enrollment, 0)
	for rows.Next() {
		var e enrollmentEntity.Enrollment
		var s = &e.CharacterSheet
		err := rows.Scan(
			&e.UUID, &e.Status, &e.CreatedAt,
			&s.ID, &s.UUID, &s.PlayerUUID, &s.MasterUUID, &s.CampaignUUID,
			&s.CategoryName, &s.CurrHexValue,
			&s.Level, &s.Points, &s.TalentLvl, &s.SkillsLvl,
			&s.Health.Min, &s.Health.Curr, &s.Health.Max,
			&s.Stamina.Min, &s.Stamina.Curr, &s.Stamina.Max,
			&s.PhysicalsLvl, &s.MentalsLvl, &s.SpiritualsLvl,
			&s.Aura.Min, &s.Aura.Curr, &s.Aura.Max,
			&s.CreatedAt, &s.UpdatedAt,
			&s.NickName, &s.FullName, &s.Alignment, &s.CharacterClass, &s.Birthday,
			&e.Player.UUID, &e.Player.Nick,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan enrollment row: %w", err)
		}
		out = append(out, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return out, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags=integration -v ./internal/gateway/pg/enrollment/ -run TestListByMatchUUID`
Expected: 3 sub-tests `PASS`, exit 0.

- [ ] **Step 5: Run the full enrollment integration suite to ensure no regression**

Run: `go test -tags=integration ./internal/gateway/pg/enrollment/...`
Expected: `PASS`, exit 0.

- [ ] **Step 6: Commit**

```bash
git add internal/gateway/pg/enrollment/list_by_match_uuid.go internal/gateway/pg/enrollment/enrollment_integration_test.go
git commit -m "$(cat <<'EOF'
feat(gateway): add ListByMatchUUID for enrollment roster

JOINs character_sheets + character_profiles + users to populate the new
Enrollment entity in a single query. Ordered by created_at ASC.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 4: Use Case — `ListMatchEnrollmentsUC` (TDD, no privacy check yet)

**Files:**
- Create: `internal/domain/match/list_match_enrollments.go`
- Create: `internal/domain/match/list_match_enrollments_test.go`

**Why:** Wire the read flow — fetch the match, decide `viewerIsMaster`, then fetch enrollments. The campaign-participant privacy check is added in Phase 2 (Task 12) so this task can ship in isolation if needed.

- [ ] **Step 1: Write the failing UC unit test**

Create `internal/domain/match/list_match_enrollments_test.go`:

```go
package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type mockMatchRepoForList struct {
	getMatchFn func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error)
	domainMatch.IRepository
}

func (m *mockMatchRepoForList) GetMatch(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
	return m.getMatchFn(ctx, id)
}

type mockEnrollmentLister struct {
	fn func(ctx context.Context, matchUUID uuid.UUID) ([]*enrollmentEntity.Enrollment, error)
}

func (m *mockEnrollmentLister) ListByMatchUUID(ctx context.Context, matchUUID uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
	return m.fn(ctx, matchUUID)
}

func TestListMatchEnrollmentsUC(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	otherUserUUID := uuid.New()

	makeMatch := func() *matchEntity.Match {
		return &matchEntity.Match{
			UUID:         matchUUID,
			MasterUUID:   masterUUID,
			CampaignUUID: uuid.New(),
			IsPublic:     true,
		}
	}

	tests := []struct {
		name           string
		userUUID       uuid.UUID
		matchFn        func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error)
		listFn         func(ctx context.Context, matchUUID uuid.UUID) ([]*enrollmentEntity.Enrollment, error)
		wantErr        error
		wantIsMaster   bool
		wantLen        int
	}{
		{
			name:     "master sees ViewerIsMaster=true",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{
					{UUID: uuid.New(), Status: "pending", CreatedAt: time.Now()},
					{UUID: uuid.New(), Status: "accepted", CreatedAt: time.Now()},
				}, nil
			},
			wantIsMaster: true,
			wantLen:      2,
		},
		{
			name:     "non-master on public match sees ViewerIsMaster=false",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{{UUID: uuid.New(), Status: "pending"}}, nil
			},
			wantIsMaster: false,
			wantLen:      1,
		},
		{
			name:     "match not found maps to ErrMatchNotFound",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return nil, matchPg.ErrMatchNotFound
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				t.Fatal("listFn should not be called when match is missing")
				return nil, nil
			},
			wantErr: domainMatch.ErrMatchNotFound,
		},
		{
			name:     "lister returns empty slice and no error",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{}, nil
			},
			wantIsMaster: true,
			wantLen:      0,
		},
		{
			name:     "lister error is propagated",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return nil, errors.New("db down")
			},
			wantErr: errors.New("db down"),
		},
		{
			name:     "match repo error (other than not found) is propagated",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return nil, errors.New("conn refused")
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				t.Fatal("listFn should not be called when match repo errors")
				return nil, nil
			},
			wantErr: errors.New("conn refused"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := domainMatch.NewListMatchEnrollmentsUC(
				&mockMatchRepoForList{getMatchFn: tc.matchFn},
				&mockEnrollmentLister{fn: tc.listFn},
			)

			got, err := uc.List(context.Background(), matchUUID, tc.userUUID)

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				if errors.Is(tc.wantErr, domainMatch.ErrMatchNotFound) {
					if !errors.Is(err, domainMatch.ErrMatchNotFound) {
						t.Fatalf("expected ErrMatchNotFound, got %v", err)
					}
					return
				}
				if err.Error() != tc.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tc.wantErr.Error(), err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ViewerIsMaster != tc.wantIsMaster {
				t.Errorf("ViewerIsMaster = %v, want %v", got.ViewerIsMaster, tc.wantIsMaster)
			}
			if len(got.Enrollments) != tc.wantLen {
				t.Errorf("Enrollments len = %d, want %d", len(got.Enrollments), tc.wantLen)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./internal/domain/match/ -run TestListMatchEnrollmentsUC`
Expected: build error: `NewListMatchEnrollmentsUC undefined`.

- [ ] **Step 3: Implement the use case**

Create `internal/domain/match/list_match_enrollments.go`:

```go
package match

import (
	"context"
	"errors"

	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

// EnrollmentLister is a local interface (defined at the consumer to avoid a
// cycle: domain/enrollment already imports domain/match). The pg enrollment
// repository satisfies it via structural typing.
type EnrollmentLister interface {
	ListByMatchUUID(
		ctx context.Context, matchUUID uuid.UUID,
	) ([]*enrollmentEntity.Enrollment, error)
}

type ListMatchEnrollmentsResult struct {
	Enrollments    []*enrollmentEntity.Enrollment
	ViewerIsMaster bool
}

type IListMatchEnrollments interface {
	List(
		ctx context.Context, matchUUID uuid.UUID, userUUID uuid.UUID,
	) (*ListMatchEnrollmentsResult, error)
}

type ListMatchEnrollmentsUC struct {
	matchRepo        IRepository
	enrollmentLister EnrollmentLister
}

func NewListMatchEnrollmentsUC(
	matchRepo IRepository,
	enrollmentLister EnrollmentLister,
) *ListMatchEnrollmentsUC {
	return &ListMatchEnrollmentsUC{
		matchRepo:        matchRepo,
		enrollmentLister: enrollmentLister,
	}
}

func (uc *ListMatchEnrollmentsUC) List(
	ctx context.Context, matchUUID uuid.UUID, userUUID uuid.UUID,
) (*ListMatchEnrollmentsResult, error) {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	enrollments, err := uc.enrollmentLister.ListByMatchUUID(ctx, matchUUID)
	if err != nil {
		return nil, err
	}

	return &ListMatchEnrollmentsResult{
		Enrollments:    enrollments,
		ViewerIsMaster: match.MasterUUID == userUUID,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./internal/domain/match/ -run TestListMatchEnrollmentsUC`
Expected: all 6 sub-tests `PASS`.

- [ ] **Step 5: Verify no regression on the broader package**

Run: `go test ./internal/domain/match/...`
Expected: `PASS` overall (the existing `match_uc_test.go` continues to pass).

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/list_match_enrollments.go internal/domain/match/list_match_enrollments_test.go
git commit -m "$(cat <<'EOF'
feat(match): add ListMatchEnrollmentsUC

Use case for the match roster read path. Derives ViewerIsMaster (one bool)
to drive per-row visibility downstream in the handler. Privacy check for
private matches is added in a follow-up task.

EnrollmentLister is a local interface defined here at the consumer to avoid
a dependency cycle with domain/enrollment.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 5: Extract `CharacterPrivateOnlyResponse` (refactor existing summary types)

**Files:**
- Modify: `internal/app/api/sheet/character_sheet_sumary_response.go` (extract a private-only struct without the embedded base)
- Modify: any callers of `ToPrivateSummaryResponse` if needed (none expected — the existing flattened type continues to be returned by the existing list endpoint)

**Why:** The new handler nests `private` as a sub-object, but the existing `CharacterPrivateSummaryResponse` flattens private fields into the base. Extract a struct that holds only the private fields, then refactor `CharacterPrivateSummaryResponse` to compose from `CharacterBaseSummaryResponse` + `CharacterPrivateOnlyResponse` so both views share the same source of truth.

- [ ] **Step 1: Refactor the response file**

Replace `internal/app/api/sheet/character_sheet_sumary_response.go` with:

```go
package sheet

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

type CharacterBaseSummaryResponse struct {
	UUID           uuid.UUID  `json:"uuid"`
	PlayerUUID     *uuid.UUID `json:"player_uuid,omitempty"`
	MasterUUID     *uuid.UUID `json:"master_uuid,omitempty"`
	CampaignUUID   *uuid.UUID `json:"campaign_uuid,omitempty"`
	NickName       string     `json:"nick_name"`
	StoryStartAt   *string    `json:"story_start_at,omitempty"`
	StoryCurrentAt *string    `json:"story_current_at,omitempty"`
	DeadAt         *string    `json:"dead_at,omitempty"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
}

// CharacterPrivateOnlyResponse holds the fields that are private to the sheet
// owner (and to the master of a match the sheet is enrolled in). It does NOT
// embed the base — it is meant to be nested under a base-typed parent.
type CharacterPrivateOnlyResponse struct {
	FullName       string    `json:"full_name"`
	Alignment      string    `json:"alignment"`
	CharacterClass string    `json:"character_class"`
	Birthday       string    `json:"birthday"`
	CategoryName   string    `json:"category_name"`
	CurrHexValue   *int      `json:"curr_hex_value,omitempty"`
	Level          int       `json:"level"`
	Points         int       `json:"points"`
	TalentLvl      int       `json:"talent_lvl"`
	PhysicalsLvl   int       `json:"physicals_lvl"`
	MentalsLvl     int       `json:"mentals_lvl"`
	SpiritualsLvl  int       `json:"spirituals_lvl"`
	SkillsLvl      int       `json:"skills_lvl"`
	Stamina        StatusBar `json:"stamina"`
	Health         StatusBar `json:"health"`
}

// CharacterPrivateSummaryResponse is the flat (base + private) shape used by
// existing endpoints (kept for backward compatibility).
type CharacterPrivateSummaryResponse struct {
	CharacterBaseSummaryResponse
	CharacterPrivateOnlyResponse
}

type CharacterPublicSummaryResponse struct {
	CharacterBaseSummaryResponse
}

type StatusBar struct {
	Min     int `json:"min"`
	Current int `json:"current"`
	Max     int `json:"max"`
}

func ToPrivateOnlyResponse(sheet *model.CharacterSheetSummary) CharacterPrivateOnlyResponse {
	stamina := sheet.Stamina
	health := sheet.Health
	return CharacterPrivateOnlyResponse{
		FullName:       sheet.FullName,
		Alignment:      sheet.Alignment,
		CharacterClass: sheet.CharacterClass,
		Birthday:       sheet.Birthday.Format("2006-01-02"),
		CategoryName:   sheet.CategoryName,
		CurrHexValue:   sheet.CurrHexValue,
		Level:          sheet.Level,
		Points:         sheet.Points,
		TalentLvl:      sheet.TalentLvl,
		PhysicalsLvl:   sheet.PhysicalsLvl,
		MentalsLvl:     sheet.MentalsLvl,
		SpiritualsLvl:  sheet.SpiritualsLvl,
		SkillsLvl:      sheet.SkillsLvl,
		Stamina: StatusBar{
			Min:     stamina.Min,
			Current: stamina.Curr,
			Max:     stamina.Max,
		},
		Health: StatusBar{
			Min:     health.Min,
			Current: health.Curr,
			Max:     health.Max,
		},
	}
}

func ToPrivateSummaryResponse(sheet *model.CharacterSheetSummary) CharacterPrivateSummaryResponse {
	return CharacterPrivateSummaryResponse{
		CharacterBaseSummaryResponse: ToBaseSummaryResponse(sheet),
		CharacterPrivateOnlyResponse: ToPrivateOnlyResponse(sheet),
	}
}

func ToPublicSummaryResponse(sheet *model.CharacterSheetSummary) CharacterPublicSummaryResponse {
	return CharacterPublicSummaryResponse{
		CharacterBaseSummaryResponse: ToBaseSummaryResponse(sheet),
	}
}

// ToBaseSummaryResponse exports what was previously the unexported
// `toSummaryBaseResponse` so the new match handler can map base summaries.
func ToBaseSummaryResponse(sheet *model.CharacterSheetSummary) CharacterBaseSummaryResponse {
	var storyStartAtStr, storyCurrentAtStr, deadAtStr *string
	if sheet.StoryStartAt != nil {
		formatted := sheet.StoryStartAt.Format("2006-01-02")
		storyStartAtStr = &formatted
	}
	if sheet.StoryCurrentAt != nil {
		formatted := sheet.StoryCurrentAt.Format("2006-01-02")
		storyCurrentAtStr = &formatted
	}
	if sheet.DeadAt != nil {
		formatted := sheet.DeadAt.Format(time.RFC3339)
		deadAtStr = &formatted
	}
	return CharacterBaseSummaryResponse{
		UUID:           sheet.UUID,
		PlayerUUID:     sheet.PlayerUUID,
		MasterUUID:     sheet.MasterUUID,
		CampaignUUID:   sheet.CampaignUUID,
		NickName:       sheet.NickName,
		StoryStartAt:   storyStartAtStr,
		StoryCurrentAt: storyCurrentAtStr,
		DeadAt:         deadAtStr,
		CreatedAt:      sheet.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      sheet.UpdatedAt.Format(time.RFC3339),
	}
}
```

- [ ] **Step 2: Verify the package and all callers still compile**

Run: `go build ./...`
Expected: no errors.

- [ ] **Step 3: Run the existing sheet handler tests to confirm no regression**

Run: `go test ./internal/app/api/sheet/...`
Expected: `PASS` (no behavioral change — `CharacterPrivateSummaryResponse` is now composed but emits the same JSON because struct embedding flattens).

- [ ] **Step 4: Commit**

```bash
git add internal/app/api/sheet/character_sheet_sumary_response.go
git commit -m "$(cat <<'EOF'
refactor(sheet): split private summary into base + private-only structs

Extracts CharacterPrivateOnlyResponse so other handlers can nest the
private fields under a base-typed parent (rather than flatten). Existing
ToPrivateSummaryResponse output is unchanged (struct embedding preserves
the same JSON shape).

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 6: Handler — `ListMatchEnrollmentsHandler` (TDD)

**Files:**
- Create: `internal/app/api/match/list_match_enrollments.go`
- Create: `internal/app/api/match/list_match_enrollments_test.go`
- Modify: `internal/app/api/match/mocks_test.go` (add `mockListMatchEnrollments`)

- [ ] **Step 1: Add the mock**

Open `internal/app/api/match/mocks_test.go` and append:

```go
type mockListMatchEnrollments struct {
	fn func(ctx context.Context, matchUUID, userUUID uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error)
}

func (m *mockListMatchEnrollments) List(
	ctx context.Context, matchUUID, userUUID uuid.UUID,
) (*domainMatch.ListMatchEnrollmentsResult, error) {
	return m.fn(ctx, matchUUID, userUUID)
}
```

If imports `context`, `domainMatch`, or `uuid` are not yet present, add them. Verify by reading the existing top of the file.

- [ ] **Step 2: Write the failing handler test**

Create `internal/app/api/match/list_match_enrollments_test.go`:

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
	apiMatch "github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestListMatchEnrollmentsHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	now := time.Now()

	makeFixture := func() []*enrollmentEntity.Enrollment {
		return []*enrollmentEntity.Enrollment{
			{
				UUID:      uuid.New(),
				Status:    "pending",
				CreatedAt: now,
				CharacterSheet: model.CharacterSheetSummary{
					UUID:     uuid.New(),
					NickName: "Gon",
					FullName: "Gon Freecss",
					Birthday: now,
				},
				Player: enrollmentEntity.PlayerRef{UUID: uuid.New(), Nick: "tiago"},
			},
		}
	}

	tests := []struct {
		name           string
		ucFn           func(ctx context.Context, matchID, uid uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error)
		wantStatus     int
		wantPrivateNil bool // when status==200, asserts the first row's character_sheet.private nullness
	}{
		{
			name: "200 with private populated when ViewerIsMaster",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return &domainMatch.ListMatchEnrollmentsResult{
					Enrollments:    makeFixture(),
					ViewerIsMaster: true,
				}, nil
			},
			wantStatus:     http.StatusOK,
			wantPrivateNil: false,
		},
		{
			name: "200 with private null when not master",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return &domainMatch.ListMatchEnrollmentsResult{
					Enrollments:    makeFixture(),
					ViewerIsMaster: false,
				}, nil
			},
			wantStatus:     http.StatusOK,
			wantPrivateNil: true,
		},
		{
			name: "200 with empty list",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return &domainMatch.ListMatchEnrollmentsResult{
					Enrollments:    []*enrollmentEntity.Enrollment{},
					ViewerIsMaster: true,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "404 on ErrMatchNotFound",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return nil, domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "403 on ErrInsufficientPermissions",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return nil, domainAuth.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "500 on generic error",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.ListMatchEnrollmentsResult, error) {
				return nil, errors.New("boom")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, api := humatest.New(t)
			handler := apiMatch.ListMatchEnrollmentsHandler(&mockListMatchEnrollments{fn: tc.ucFn})

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/matches/{uuid}/enrollments",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/matches/"+matchUUID.String()+"/enrollments")

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d. Body: %s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if tc.wantStatus != http.StatusOK {
				return
			}
			var body map[string]any
			if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			enrollments, ok := body["enrollments"].([]any)
			if !ok {
				t.Fatal("response missing 'enrollments' array")
			}
			if len(enrollments) == 0 {
				return // empty-list case
			}
			row := enrollments[0].(map[string]any)
			sheet := row["character_sheet"].(map[string]any)
			privateField, present := sheet["private"]
			if !present {
				t.Fatal("character_sheet.private must be present (null or populated), not omitted")
			}
			if tc.wantPrivateNil {
				if privateField != nil {
					t.Errorf("character_sheet.private = %v, want null", privateField)
				}
			} else {
				if privateField == nil {
					t.Error("character_sheet.private = null, want populated object")
				}
			}
		})
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test -v ./internal/app/api/match/ -run TestListMatchEnrollmentsHandler`
Expected: build error: `apiMatch.ListMatchEnrollmentsHandler undefined`.

- [ ] **Step 4: Implement the handler**

Create `internal/app/api/match/list_match_enrollments.go`:

```go
package match

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	apiSheet "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

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
	UUID           uuid.UUID                            `json:"uuid"`
	Status         string                               `json:"status"`
	CreatedAt      string                               `json:"created_at"`
	CharacterSheet CharacterSheetWithVisibilityResponse `json:"character_sheet"`
	Player         PlayerRefResponse                    `json:"player"`
}

type CharacterSheetWithVisibilityResponse struct {
	apiSheet.CharacterBaseSummaryResponse
	// Pointer with no `omitempty`: serializes as `null` for non-master viewers.
	Private *apiSheet.CharacterPrivateOnlyResponse `json:"private"`
}

type PlayerRefResponse struct {
	UUID uuid.UUID `json:"uuid"`
	Nick string    `json:"nick"`
}

func ListMatchEnrollmentsHandler(
	uc domainMatch.IListMatchEnrollments,
) func(context.Context, *ListMatchEnrollmentsRequest) (*ListMatchEnrollmentsResponse, error) {
	return func(ctx context.Context, req *ListMatchEnrollmentsRequest) (*ListMatchEnrollmentsResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		result, err := uc.List(ctx, req.UUID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainMatch.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		out := make([]EnrollmentResponse, 0, len(result.Enrollments))
		for _, e := range result.Enrollments {
			out = append(out, toEnrollmentResponse(e, result.ViewerIsMaster))
		}
		return &ListMatchEnrollmentsResponse{
			Body: ListMatchEnrollmentsResponseBody{Enrollments: out},
		}, nil
	}
}

func toEnrollmentResponse(e *enrollmentEntity.Enrollment, viewerIsMaster bool) EnrollmentResponse {
	sheet := CharacterSheetWithVisibilityResponse{
		CharacterBaseSummaryResponse: apiSheet.ToBaseSummaryResponse(&e.CharacterSheet),
		Private:                      nil,
	}
	if viewerIsMaster {
		p := apiSheet.ToPrivateOnlyResponse(&e.CharacterSheet)
		sheet.Private = &p
	}
	return EnrollmentResponse{
		UUID:           e.UUID,
		Status:         e.Status,
		CreatedAt:      e.CreatedAt.Format(http.TimeFormat),
		CharacterSheet: sheet,
		Player: PlayerRefResponse{
			UUID: e.Player.UUID,
			Nick: e.Player.Nick,
		},
	}
}
```

- [ ] **Step 5: Run handler tests to verify they pass**

Run: `go test -v ./internal/app/api/match/ -run TestListMatchEnrollmentsHandler`
Expected: all 6 sub-tests `PASS`.

- [ ] **Step 6: Run the full match handler suite for regression**

Run: `go test ./internal/app/api/match/...`
Expected: `PASS` overall.

- [ ] **Step 7: Commit**

```bash
git add internal/app/api/match/list_match_enrollments.go internal/app/api/match/list_match_enrollments_test.go internal/app/api/match/mocks_test.go
git commit -m "$(cat <<'EOF'
feat(handler): add ListMatchEnrollmentsHandler

Maps the UC result to the wire shape with character_sheet.private always
serialized as null for non-master viewers (no omitempty), keeping the JSON
shape stable across roles.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 7: Routes — Register `GET /matches/{uuid}/enrollments`

**Files:**
- Modify: `internal/app/api/match/routes.go`

- [ ] **Step 1: Add the handler field to the `Api` struct**

Open `internal/app/api/match/routes.go`. In the `Api` struct, add a new field after `ListPublicUpcomingMatchesHandler`:

```go
type Api struct {
	CreateMatchHandler               Handler[CreateMatchRequest, CreateMatchResponse]
	GetMatchHandler                  Handler[GetMatchRequest, GetMatchResponse]
	ListMatchesHandler               Handler[struct{}, ListMatchesResponse]
	ListPublicUpcomingMatchesHandler Handler[struct{}, ListMatchesResponse]
	ListMatchEnrollmentsHandler      Handler[ListMatchEnrollmentsRequest, ListMatchEnrollmentsResponse]
}
```

- [ ] **Step 2: Register the route**

In the same file, after the existing `ListPublicUpcomingMatchesHandler` registration, append:

```go
huma.Register(api, huma.Operation{
	Method:      http.MethodGet,
	Path:        "/matches/{uuid}/enrollments",
	Description: "List enrollments of a match (visibility per row depends on viewer)",
	Tags:        []string{"matches"},
	Errors: []int{
		http.StatusNotFound,
		http.StatusForbidden,
		http.StatusUnauthorized,
		http.StatusInternalServerError,
	},
}, a.ListMatchEnrollmentsHandler)
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./internal/app/api/match/...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/app/api/match/routes.go
git commit -m "$(cat <<'EOF'
feat(routes): register GET /matches/{uuid}/enrollments

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 8: Wiring — `cmd/api/main.go`

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Instantiate the UC and inject the handler**

Open `cmd/api/main.go`. After the line:

```go
listPublicUpcomingMatchesUC := domainMatch.NewListPublicUpcomingMatchesUC(matchRepo)
```

Insert:

```go
listMatchEnrollmentsUC := domainMatch.NewListMatchEnrollmentsUC(matchRepo, enrollmentRepo)
```

Then in the `matchesApi := matchHandler.Api{...}` literal, add the field:

```go
matchesApi := matchHandler.Api{
	CreateMatchHandler:               matchHandler.CreateMatchHandler(createMatchUC),
	GetMatchHandler:                  matchHandler.GetMatchHandler(getMatchUC),
	ListMatchesHandler:               matchHandler.ListMatchesHandler(listMatchesUC),
	ListPublicUpcomingMatchesHandler: matchHandler.ListPublicUpcomingMatchesHandler(listPublicUpcomingMatchesUC),
	ListMatchEnrollmentsHandler:      matchHandler.ListMatchEnrollmentsHandler(listMatchEnrollmentsUC),
}
```

- [ ] **Step 2: Verify the binary builds**

Run: `go build ./cmd/api/`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/api/main.go
git commit -m "$(cat <<'EOF'
chore(wiring): inject ListMatchEnrollments into match Api

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 9: Phase 1 Smoke Test (manual, no commit)

**Files:** none

- [ ] **Step 1: Apply migrations and start the API**

```bash
make migrate-up
make run-dev
```

Expected: server listens on `localhost:5000`.

- [ ] **Step 2: Hit the endpoint**

(Use a session token for an authenticated user — get one via the existing login flow. Replace `<token>` and `<match-uuid>` accordingly.)

```bash
curl -s -H "Authorization: Bearer <token>" \
  http://localhost:5000/matches/<match-uuid>/enrollments | jq
```

Expected:
- HTTP 200 with `{ "enrollments": [...] }`.
- For a master-of-match token: each row's `character_sheet.private` is a populated object.
- For any other token: each row's `character_sheet.private` is exactly `null`.

If the body shows `"private"` omitted instead of `"private": null`, return to Task 6 Step 4 and verify the struct tag is `json:"private"` (no `omitempty`).

- [ ] **Step 3: Stop the server**

`Ctrl-C` in the `make run-dev` window.

---

## Task 10: Gateway — `ExistsSheetInCampaign` (TDD, Phase 2 begins)

**Files:**
- Create: `internal/gateway/pg/sheet/exists_in_campaign.go`
- Modify: `internal/gateway/pg/sheet/sheet_integration_test.go` (append `TestExistsSheetInCampaign`)

- [ ] **Step 1: Write the failing integration test**

Append to `internal/gateway/pg/sheet/sheet_integration_test.go`:

```go
func TestExistsSheetInCampaign(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	ctx := context.Background()
	repo := sheetRepo.NewRepository(pool)

	t.Run("true when player has a sheet in the campaign", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Campaign A")

		// InsertTestCharacterSheet creates a sheet that is not yet linked to a campaign.
		// Insert one and then UPDATE its campaign_uuid via raw SQL to simulate "accepted submission".
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Gon")
		if _, err := pool.Exec(ctx,
			`UPDATE character_sheets SET campaign_uuid = $1 WHERE uuid = $2`,
			campaignUUID, sheetUUID,
		); err != nil {
			t.Fatalf("update campaign_uuid: %v", err)
		}

		got, err := repo.ExistsSheetInCampaign(ctx,
			uuid.MustParse(playerUUID), uuid.MustParse(campaignUUID),
		)
		if err != nil {
			t.Fatalf("ExistsSheetInCampaign() error = %v", err)
		}
		if !got {
			t.Error("ExistsSheetInCampaign() = false, want true")
		}
	})

	t.Run("false when player has no sheet in the campaign", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignUUID := pgtest.InsertTestCampaign(t, pool, masterUUID, "Campaign A")

		got, err := repo.ExistsSheetInCampaign(ctx,
			uuid.MustParse(playerUUID), uuid.MustParse(campaignUUID),
		)
		if err != nil {
			t.Fatalf("ExistsSheetInCampaign() error = %v", err)
		}
		if got {
			t.Error("ExistsSheetInCampaign() = true, want false")
		}
	})

	t.Run("false when player has sheets in other campaigns only", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := pgtest.InsertTestUser(t, pool, "master", "master@test.com", "pass123")
		playerUUID := pgtest.InsertTestUser(t, pool, "player1", "p1@test.com", "pass123")
		campaignA := pgtest.InsertTestCampaign(t, pool, masterUUID, "Campaign A")
		campaignB := pgtest.InsertTestCampaign(t, pool, masterUUID, "Campaign B")

		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &playerUUID, nil, "Gon")
		if _, err := pool.Exec(ctx,
			`UPDATE character_sheets SET campaign_uuid = $1 WHERE uuid = $2`,
			campaignA, sheetUUID,
		); err != nil {
			t.Fatalf("update campaign_uuid: %v", err)
		}

		got, err := repo.ExistsSheetInCampaign(ctx,
			uuid.MustParse(playerUUID), uuid.MustParse(campaignB),
		)
		if err != nil {
			t.Fatalf("ExistsSheetInCampaign() error = %v", err)
		}
		if got {
			t.Error("ExistsSheetInCampaign() = true, want false")
		}
	})
}
```

(Adjust `sheetRepo` to whatever import alias `sheet_integration_test.go` already uses for `internal/gateway/pg/sheet`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=integration -v ./internal/gateway/pg/sheet/ -run TestExistsSheetInCampaign`
Expected: build error: `repo.ExistsSheetInCampaign undefined`.

- [ ] **Step 3: Implement the gateway method**

Create `internal/gateway/pg/sheet/exists_in_campaign.go`:

```go
package sheet

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

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
	if err != nil {
		return false, fmt.Errorf("failed to check sheet in campaign: %w", err)
	}
	return exists, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -tags=integration -v ./internal/gateway/pg/sheet/ -run TestExistsSheetInCampaign`
Expected: 3 sub-tests `PASS`.

- [ ] **Step 5: Run the full sheet integration suite**

Run: `go test -tags=integration ./internal/gateway/pg/sheet/...`
Expected: `PASS`.

- [ ] **Step 6: Commit**

```bash
git add internal/gateway/pg/sheet/exists_in_campaign.go internal/gateway/pg/sheet/sheet_integration_test.go
git commit -m "$(cat <<'EOF'
feat(gateway): add ExistsSheetInCampaign for participation check

Used by the new privacy rule: any player with a sheet linked to the
match's campaign may view the match (and its roster) even when private.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 11: Add `ExistsSheetInCampaign` to `character_sheet.IRepository`

**Files:**
- Modify: `internal/domain/character_sheet/i_repository.go`

**Why:** Spec requires the method to be on `domain/character_sheet/IRepository` (so the existing sheet repo formally exposes it across the domain). Structural typing then makes the same gateway satisfy the local `match.CampaignParticipationChecker` interface introduced in Task 12.

- [ ] **Step 1: Add the method to the interface**

Open `internal/domain/character_sheet/i_repository.go`. Add to the `IRepository` interface:

```go
ExistsSheetInCampaign(
	ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID,
) (bool, error)
```

- [ ] **Step 2: Verify everything still compiles**

Run: `go build ./...`
Expected: no errors. (The gateway already implements the method from Task 10; mocks in unrelated UC tests need to expose this method only if they implement the interface fully — `domain/character_sheet` UC tests use partial mocks, so re-running them confirms.)

- [ ] **Step 3: Run the impacted tests**

Run: `go test ./internal/domain/character_sheet/... ./internal/domain/enrollment/...`
Expected: `PASS`. If any mock fails to satisfy the interface, embed `charactersheet.IRepository` in the mock struct as a fallback (Go's "embed-and-override-the-method" idiom).

- [ ] **Step 4: Commit**

```bash
git add internal/domain/character_sheet/i_repository.go
git commit -m "$(cat <<'EOF'
feat(domain): expose ExistsSheetInCampaign on character_sheet.IRepository

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 12: Privacy retrofit — `ListMatchEnrollmentsUC` (TDD)

**Files:**
- Modify: `internal/domain/match/list_match_enrollments.go` (add `CampaignParticipationChecker` interface, add field to UC, add check)
- Modify: `internal/domain/match/list_match_enrollments_test.go` (add new mock + new test cases)

- [ ] **Step 1: Extend the test file with the new mock and cases**

In `internal/domain/match/list_match_enrollments_test.go`, **add** the new mock type after the existing `mockEnrollmentLister`:

```go
type mockParticipationChecker struct {
	fn func(ctx context.Context, playerUUID, campaignUUID uuid.UUID) (bool, error)
}

func (m *mockParticipationChecker) ExistsSheetInCampaign(
	ctx context.Context, playerUUID, campaignUUID uuid.UUID,
) (bool, error) {
	return m.fn(ctx, playerUUID, campaignUUID)
}
```

Then **replace** the existing `tests` table (and the loop that drives it) with the expanded version below — the constructor signature changes, so every case must pass a `participationChecker`. The unchanged cases use a checker that fails the test if invoked (defensive: master and public-match paths must NOT consult the checker).

```go
func TestListMatchEnrollmentsUC(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	otherUserUUID := uuid.New()
	campaignUUID := uuid.New()

	makeMatch := func(isPublic bool) *matchEntity.Match {
		return &matchEntity.Match{
			UUID:         matchUUID,
			MasterUUID:   masterUUID,
			CampaignUUID: campaignUUID,
			IsPublic:     isPublic,
		}
	}

	checkerNeverCalled := func(t *testing.T) *mockParticipationChecker {
		return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			t.Fatal("participationChecker should NOT be called for this case")
			return false, nil
		}}
	}

	tests := []struct {
		name         string
		userUUID     uuid.UUID
		matchFn      func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error)
		listFn       func(ctx context.Context, matchUUID uuid.UUID) ([]*enrollmentEntity.Enrollment, error)
		checker      func(t *testing.T) *mockParticipationChecker
		wantErr      error
		wantIsMaster bool
		wantLen      int
	}{
		{
			name:     "master on public match — no checker call",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(true), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{{Status: "pending"}}, nil
			},
			checker:      checkerNeverCalled,
			wantIsMaster: true,
			wantLen:      1,
		},
		{
			name:     "master on private match — no checker call",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(false), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{}, nil
			},
			checker:      checkerNeverCalled,
			wantIsMaster: true,
			wantLen:      0,
		},
		{
			name:     "non-master on public match — no checker call",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(true), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{{Status: "accepted"}}, nil
			},
			checker:      checkerNeverCalled,
			wantIsMaster: false,
			wantLen:      1,
		},
		{
			name:     "non-master on private match, participates — allowed",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(false), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{{Status: "accepted"}}, nil
			},
			checker: func(_ *testing.T) *mockParticipationChecker {
				return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
					return true, nil
				}}
			},
			wantIsMaster: false,
			wantLen:      1,
		},
		{
			name:     "non-master on private match, does NOT participate — forbidden",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(false), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				t.Fatal("listFn should not be called when forbidden")
				return nil, nil
			},
			checker: func(_ *testing.T) *mockParticipationChecker {
				return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
					return false, nil
				}}
			},
			wantErr: domainAuth.ErrInsufficientPermissions,
		},
		{
			name:     "checker error is propagated",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(false), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				t.Fatal("listFn should not be called when checker errors")
				return nil, nil
			},
			checker: func(_ *testing.T) *mockParticipationChecker {
				return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
					return false, errors.New("db down")
				}}
			},
			wantErr: errors.New("db down"),
		},
		{
			name:     "match not found maps to ErrMatchNotFound",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return nil, matchPg.ErrMatchNotFound
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				t.Fatal("listFn should not be called when match is missing")
				return nil, nil
			},
			checker: checkerNeverCalled,
			wantErr: domainMatch.ErrMatchNotFound,
		},
		{
			name:     "lister error is propagated (master path)",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(true), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return nil, errors.New("db down")
			},
			checker: checkerNeverCalled,
			wantErr: errors.New("db down"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := domainMatch.NewListMatchEnrollmentsUC(
				&mockMatchRepoForList{getMatchFn: tc.matchFn},
				&mockEnrollmentLister{fn: tc.listFn},
				tc.checker(t),
			)

			got, err := uc.List(context.Background(), matchUUID, tc.userUUID)

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				switch {
				case errors.Is(tc.wantErr, domainMatch.ErrMatchNotFound):
					if !errors.Is(err, domainMatch.ErrMatchNotFound) {
						t.Fatalf("expected ErrMatchNotFound, got %v", err)
					}
				case errors.Is(tc.wantErr, domainAuth.ErrInsufficientPermissions):
					if !errors.Is(err, domainAuth.ErrInsufficientPermissions) {
						t.Fatalf("expected ErrInsufficientPermissions, got %v", err)
					}
				default:
					if err.Error() != tc.wantErr.Error() {
						t.Fatalf("expected error %q, got %q", tc.wantErr.Error(), err.Error())
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ViewerIsMaster != tc.wantIsMaster {
				t.Errorf("ViewerIsMaster = %v, want %v", got.ViewerIsMaster, tc.wantIsMaster)
			}
			if len(got.Enrollments) != tc.wantLen {
				t.Errorf("Enrollments len = %d, want %d", len(got.Enrollments), tc.wantLen)
			}
		})
	}
}
```

Add these imports to the test file if not already present: `domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./internal/domain/match/ -run TestListMatchEnrollmentsUC`
Expected: build error: `NewListMatchEnrollmentsUC` constructor signature mismatch (only takes 2 args today).

- [ ] **Step 3: Update the use case to add the participation check**

Replace the body of `internal/domain/match/list_match_enrollments.go` with:

```go
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

// CampaignParticipationChecker is a local interface (defined at the consumer)
// satisfied by the pg sheet repository via structural typing.
type CampaignParticipationChecker interface {
	ExistsSheetInCampaign(
		ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID,
	) (bool, error)
}

type ListMatchEnrollmentsResult struct {
	Enrollments    []*enrollmentEntity.Enrollment
	ViewerIsMaster bool
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
) *ListMatchEnrollmentsUC {
	return &ListMatchEnrollmentsUC{
		matchRepo:            matchRepo,
		enrollmentLister:     enrollmentLister,
		participationChecker: participationChecker,
	}
}

func (uc *ListMatchEnrollmentsUC) List(
	ctx context.Context, matchUUID uuid.UUID, userUUID uuid.UUID,
) (*ListMatchEnrollmentsResult, error) {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
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

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -v ./internal/domain/match/ -run TestListMatchEnrollmentsUC`
Expected: all 8 sub-tests `PASS`.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/list_match_enrollments.go internal/domain/match/list_match_enrollments_test.go
git commit -m "$(cat <<'EOF'
feat(match): add campaign-participant privacy check to roster UC

Private matches are now visible to the master OR any player who has at
least one character sheet linked to the campaign. Public matches and
master access remain unchanged.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 13: Retrofit `GetMatchUC` with the same privacy rule

**Files:**
- Modify: `internal/domain/match/get_match.go`
- Modify: `internal/domain/match/match_uc_test.go` (or wherever `TestGetMatchUC` lives — locate via `grep -l "TestGetMatchUC" internal/domain/match`)

- [ ] **Step 1: Locate the existing GetMatch UC test**

Run: `grep -rln "TestGetMatchUC\|TestGetMatch" internal/domain/match/`
Expected: shows the file containing the existing UC test (likely `match_uc_test.go`).

- [ ] **Step 2: Extend the test file with the participation-check cases**

The mock `mockParticipationChecker` is already defined in `list_match_enrollments_test.go` (Task 12). Since both test files share `package match_test`, you can reuse it directly — no redefinition.

The existing `TestGetMatch*` test must be updated because the constructor signature changes from 1 to 2 args. Two sub-changes:

**(a) Update every existing case** to pass a "never called" participation checker (defensive — public-match and master paths must NOT consult the checker). Replace the existing UC construction (which currently looks like `domainMatch.NewGetMatchUC(matchRepo)`) with:

```go
checker := &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
    t.Fatal("participationChecker should NOT be called for this case")
    return false, nil
}}
uc := domainMatch.NewGetMatchUC(matchRepo, checker)
```

If the existing test is table-driven, add a per-case `checker` field of type `func(*testing.T) *mockParticipationChecker` analogous to Task 12, and default each existing case to `checkerNeverCalled` (define it once locally if not already shared).

**(b) Append the three new sub-tests below.** They build a private match with `MasterUUID != userUUID`, swap the checker behavior per case, and assert the outcome. Adapt the variable names (`masterUUID`, `userUUID`, `matchRepo`, `matchUUID`) to whatever the existing test already declares.

```go
t.Run("non-master on private match, participates — allowed", func(t *testing.T) {
    matchUUID := uuid.New()
    masterUUID := uuid.New()
    userUUID := uuid.New()
    campaignUUID := uuid.New()

    matchRepo := &mockMatchRepo{getMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
        return &matchEntity.Match{
            UUID:         matchUUID,
            MasterUUID:   masterUUID,
            CampaignUUID: campaignUUID,
            IsPublic:     false,
        }, nil
    }}
    checker := &mockParticipationChecker{fn: func(_ context.Context, p, c uuid.UUID) (bool, error) {
        if p != userUUID || c != campaignUUID {
            t.Errorf("checker called with (%v,%v), want (%v,%v)", p, c, userUUID, campaignUUID)
        }
        return true, nil
    }}
    uc := domainMatch.NewGetMatchUC(matchRepo, checker)

    got, err := uc.GetMatch(context.Background(), matchUUID, userUUID)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got.UUID != matchUUID {
        t.Errorf("got match uuid %v, want %v", got.UUID, matchUUID)
    }
})

t.Run("non-master on private match, does NOT participate — forbidden", func(t *testing.T) {
    matchUUID := uuid.New()
    masterUUID := uuid.New()
    userUUID := uuid.New()

    matchRepo := &mockMatchRepo{getMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
        return &matchEntity.Match{
            UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: uuid.New(), IsPublic: false,
        }, nil
    }}
    checker := &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
        return false, nil
    }}
    uc := domainMatch.NewGetMatchUC(matchRepo, checker)

    _, err := uc.GetMatch(context.Background(), matchUUID, userUUID)
    if !errors.Is(err, domainAuth.ErrInsufficientPermissions) {
        t.Fatalf("got %v, want ErrInsufficientPermissions", err)
    }
})

t.Run("non-master on private match, checker errors — propagated", func(t *testing.T) {
    matchUUID := uuid.New()
    masterUUID := uuid.New()
    userUUID := uuid.New()

    matchRepo := &mockMatchRepo{getMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
        return &matchEntity.Match{
            UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: uuid.New(), IsPublic: false,
        }, nil
    }}
    wantErr := errors.New("db down")
    checker := &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
        return false, wantErr
    }}
    uc := domainMatch.NewGetMatchUC(matchRepo, checker)

    _, err := uc.GetMatch(context.Background(), matchUUID, userUUID)
    if err == nil || err.Error() != wantErr.Error() {
        t.Fatalf("got %v, want %v", err, wantErr)
    }
})
```

Adapt the mock type name (`mockMatchRepo` above) to whatever the existing test uses — likely `mockMatchRepository` or similar. Locate it via:

```bash
grep -n "mockMatch\|mock.*Match.*Repo" internal/domain/match/match_uc_test.go
```

Add imports at the top of the test file if missing: `domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"`, `errors`, `matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"`.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test -v ./internal/domain/match/ -run TestGetMatch`
Expected: build / signature errors.

- [ ] **Step 4: Update `GetMatchUC` to accept and use the checker**

Replace `internal/domain/match/get_match.go` with:

```go
package match

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IGetMatch interface {
	GetMatch(
		ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID,
	) (*matchEntity.Match, error)
}

type GetMatchUC struct {
	repo                 IRepository
	participationChecker CampaignParticipationChecker
}

func NewGetMatchUC(
	repo IRepository,
	participationChecker CampaignParticipationChecker,
) *GetMatchUC {
	return &GetMatchUC{
		repo:                 repo,
		participationChecker: participationChecker,
	}
}

func (uc *GetMatchUC) GetMatch(
	ctx context.Context, id uuid.UUID, userUUID uuid.UUID,
) (*matchEntity.Match, error) {
	match, err := uc.repo.GetMatch(ctx, id)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	if !match.IsPublic && match.MasterUUID != userUUID {
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
	return match, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -v ./internal/domain/match/`
Expected: all `TestGetMatch*` and `TestListMatchEnrollmentsUC` cases `PASS`.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/get_match.go internal/domain/match/match_uc_test.go
git commit -m "$(cat <<'EOF'
feat(match): allow campaign participants to view private matches

GetMatchUC now grants access to any player who has a sheet linked to the
match's campaign — aligning with the rule introduced for the new roster
endpoint. Master and public-match access unchanged.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

(If the test file is named differently, adjust the `git add` accordingly.)

---

## Task 14: Wiring update — pass `characterSheetRepo` to both UCs

**Files:**
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Update the constructor calls**

In `cmd/api/main.go`, change:

```go
getMatchUC := domainMatch.NewGetMatchUC(matchRepo)
```
to:
```go
getMatchUC := domainMatch.NewGetMatchUC(matchRepo, characterSheetRepo)
```

And change:
```go
listMatchEnrollmentsUC := domainMatch.NewListMatchEnrollmentsUC(matchRepo, enrollmentRepo)
```
to:
```go
listMatchEnrollmentsUC := domainMatch.NewListMatchEnrollmentsUC(matchRepo, enrollmentRepo, characterSheetRepo)
```

- [ ] **Step 2: Verify the binary builds**

Run: `go build ./cmd/api/`
Expected: no errors.

- [ ] **Step 3: Verify the full test suite still passes**

Run: `go test ./...`
Expected: `PASS` overall.

- [ ] **Step 4: Commit**

```bash
git add cmd/api/main.go
git commit -m "$(cat <<'EOF'
chore(wiring): pass characterSheetRepo to GetMatchUC and ListMatchEnrollmentsUC

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 15: Documentation — `docs/dev/match/roster.md`

**Files:**
- Create: `docs/dev/match/roster.md`

- [ ] **Step 1: Write the doc (PT-BR per docs-workflow)**

Create `docs/dev/match/roster.md`:

```markdown
# Roster da Match — Listagem de Inscrições

Documentação técnica do endpoint `GET /matches/{uuid}/enrollments`, suas
regras de autorização e a estratégia de visibilidade por linha aplicada ao
character sheet.

---

## 1. Endpoint

```
GET /matches/{uuid}/enrollments
```

Retorna o roster completo da partida (`pending` + `accepted` + `rejected`,
sem filtros, ordenado por `created_at ASC`).

## 2. Autorização

O visualizador autenticado é liberado quando **qualquer** das condições é
verdadeira:

1. A match é pública (`matches.is_public = true`).
2. O visualizador é o mestre da match (`userUUID == match.MasterUUID`).
3. O visualizador possui ao menos uma `character_sheets` cuja
   `campaign_uuid` é igual à `campaign_uuid` da match.

Caso nenhuma seja satisfeita → `403 Forbidden` (`ErrInsufficientPermissions`).

A mesma regra foi retroportada para `GET /matches/{uuid}` (`GetMatchUC`),
mantendo os dois endpoints consistentes — quem pode ver a match também pode
ver seu roster, e vice-versa.

## 3. Visibilidade por linha (master vs demais)

A camada de domínio devolve um booleano único `ViewerIsMaster` no resultado
do UC. O handler usa esse bool para decidir, **igual para todas as linhas**,
se anexa o sub-objeto `private` (campos sensíveis do character sheet).

| Visualizador | `character_sheet.private` em todas as linhas |
|---|---|
| Mestre da match | objeto populado |
| Demais (jogadores autorizados) | `null` (sempre serializado, sem `omitempty`) |

### Por que não tratar a linha do próprio jogador de forma especial?

Schema estável é mais valioso do que economizar ~200 bytes por linha. A
estratégia "incluir base sempre, anexar private só para o mestre" mantém:

- Shape único do JSON entre papéis.
- Robustez a sessões novas, hard refresh, deep link, multi-tab (não
  pressupõe estado client-side).
- Lógica do UC reduzida a um booleano (sem decisão por linha).

O frontend pode descartar a linha do próprio quando já tem a ficha local —
é otimização de UI, não responsabilidade da API.

## 4. Forma do response

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
        "campaign_uuid": "…",
        "nick_name": "Gon",
        "story_start_at": "2026-01-01",
        "created_at": "...",
        "updated_at": "...",
        "private": {
          "full_name": "Gon Freecss",
          "level": 5,
          "stamina": { "min": 0, "current": 30, "max": 50 }
        }
      },
      "player": { "uuid": "…", "nick": "tiago" }
    }
  ]
}
```

Formatos de data herdados dos summary types existentes
(`internal/app/api/sheet/character_sheet_sumary_response.go`):
`http.TimeFormat` para `enrollments[].created_at` (igual ao `MatchResponse`),
RFC3339 para `created_at`/`updated_at` do sheet, `2006-01-02` para
`story_start_at`/`story_current_at`, RFC3339 para `dead_at`.

## 5. Camadas

| Camada | Local | Responsabilidade |
|---|---|---|
| Entity | `internal/domain/entity/enrollment/enrollment.go` | Agregado `Enrollment` (uuid, status, created_at, character sheet summary, player ref) |
| Use case | `internal/domain/match/list_match_enrollments.go` | Orquestra match repo + enrollment lister + participation check; devolve `ViewerIsMaster` |
| Gateway (enrollment) | `internal/gateway/pg/enrollment/list_by_match_uuid.go` | SQL com JOIN em `character_sheets`, `character_profiles` e `users` |
| Gateway (sheet) | `internal/gateway/pg/sheet/exists_in_campaign.go` | Check de participação |
| Handler | `internal/app/api/match/list_match_enrollments.go` | Mapeia para o shape acima; popula `private` quando `ViewerIsMaster` |
| Routes | `internal/app/api/match/routes.go` | Registro em `/matches/{uuid}/enrollments` |

### Por que o use case fica em `domain/match` e não em `domain/enrollment`?

A leitura é semanticamente "roster da match" — entradas primárias são
estado de privacidade da match e relação mestre/participante. Inscrições
são dados agregados, não o sujeito da orquestração. Operações que **agem
sobre** uma inscrição (accept/reject) permanecem em `domain/enrollment`.

Para evitar ciclo (`domain/enrollment` já importa `domain/match`), as
dependências de listagem e participação são declaradas como interfaces
locais em `domain/match`:

- `EnrollmentLister.ListByMatchUUID(...)` — satisfeita pelo
  `*pg/enrollment.Repository` via structural typing.
- `CampaignParticipationChecker.ExistsSheetInCampaign(...)` — satisfeita
  pelo `*pg/sheet.Repository` via structural typing.

## 6. Índice

```sql
CREATE INDEX idx_enrollments_match_uuid_created_at
  ON enrollments(match_uuid, created_at);
```

Cobre o filtro por `match_uuid` e o `ORDER BY created_at` em uma única
estrutura — evita sort step. O índice existente
`idx_enrollments_sheet_match_uuid (character_sheet_uuid, match_uuid)` não
ajuda porque a coluna líder é o sheet.
```

- [ ] **Step 2: Commit**

```bash
git add docs/dev/match/roster.md
git commit -m "$(cat <<'EOF'
docs(dev): add match/roster.md describing the listing endpoint

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 16: Documentation — update `docs/dev/enrollment.md`

**Files:**
- Modify: `docs/dev/enrollment.md`

- [ ] **Step 1: Append §7 "Listagem por Match" before the "## Referências de Código" section**

Open `docs/dev/enrollment.md` and locate the line `## Referências de Código`. Insert the following block immediately **before** it:

```markdown
---

## 7. Listagem de Inscrições por Match

A leitura "todas as inscrições de uma match" é exposta via `GET /matches/{uuid}/enrollments`.
A documentação detalhada (autorização, visibilidade por linha, formato de
response) vive em [`match/roster.md`](./match/roster.md).

**Pontos cross-domain relevantes para esta área:**

- A entidade agregadora `Enrollment` foi adicionada em
  `internal/domain/entity/enrollment/enrollment.go` — antes desta task não
  existia entity package para enrollment. O agregado carrega uuid, status,
  `created_at`, summary completo do character sheet e referência do player.
- O use case que orquestra a leitura vive em
  `internal/domain/match/list_match_enrollments.go` (não em
  `domain/enrollment/`). Razão: a operação é semanticamente "roster da
  match"; enrollments são dados agregados, não sujeito da orquestração.
  As operações que agem sobre uma inscrição (accept/reject/kick)
  permanecem em `domain/enrollment/`.
- O gateway `ListByMatchUUID` em `internal/gateway/pg/enrollment/` faz
  JOIN com `character_sheets`, `character_profiles` e `users` para
  popular o agregado em uma única query.
```

Then add the following row to the **Referências de Código** table:

| `internal/gateway/pg/enrollment/list_by_match_uuid.go` | Listagem cross-domain por match (JOIN com sheet + player) |

- [ ] **Step 2: Commit**

```bash
git add docs/dev/enrollment.md
git commit -m "$(cat <<'EOF'
docs(dev): cross-link enrollment.md to new match/roster.md

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 17: Documentation map — update `docs/documentation-map.yaml`

**Files:**
- Modify: `docs/documentation-map.yaml`

- [ ] **Step 1: Add the new mappings**

Open `docs/documentation-map.yaml`. Append (in the `mappings:` list, alphabetically near the other `internal/domain/match/` and `internal/gateway/pg/enrollment/` entries):

```yaml
  # ─── Match: Roster (Enrollments Listing) ───
  - code_path: internal/domain/match/list_match_enrollments.go
    dev_docs:
      - path: docs/dev/match/roster.md
        confidence: directly_affected
    notes: Use case for the match roster read path; privacy and visibility tiers

  - code_path: internal/domain/entity/enrollment/
    dev_docs:
      - path: docs/dev/enrollment.md
        confidence: directly_affected
      - path: docs/dev/match/roster.md
        confidence: possibly_affected
    notes: Enrollment aggregate (uuid, status, created_at, character sheet summary, player ref)

  - code_path: internal/gateway/pg/enrollment/list_by_match_uuid.go
    dev_docs:
      - path: docs/dev/match/roster.md
        confidence: directly_affected
      - path: docs/dev/enrollment.md
        confidence: possibly_affected
    notes: Cross-domain JOIN (enrollments + character_sheets + character_profiles + users)

  - code_path: internal/gateway/pg/sheet/exists_in_campaign.go
    dev_docs:
      - path: docs/dev/match/roster.md
        confidence: possibly_affected
    notes: Participation check used by private-match privacy rule
```

- [ ] **Step 2: Commit**

```bash
git add docs/documentation-map.yaml
git commit -m "$(cat <<'EOF'
docs(map): register new code paths for match roster listing

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 18: Final Verification & PR

**Files:** none

- [ ] **Step 1: Run the full unit + integration test suite**

```bash
go test ./...
go test -tags=integration ./internal/gateway/pg/...
```
Expected: both `PASS`.

- [ ] **Step 2: Run vet (including integration build tag)**

```bash
go vet ./...
go vet -tags=integration ./internal/gateway/pg/...
```
Expected: no warnings.

- [ ] **Step 3: Phase 2 smoke test**

Repeat Task 9 with two extra scenarios:

- Token of a user **with** a character sheet in the match's campaign requesting a **private** match → 200 with `private = null` for all rows.
- Token of a user **without** a sheet in the campaign requesting a **private** match → 403.

- [ ] **Step 4: Push branch and open PR**

```bash
git push -u origin feat/match-enrollments-listing
gh pr create --title "feat: GET /matches/{uuid}/enrollments + privacy alignment" --body "$(cat <<'EOF'
## Summary
- Adds `GET /matches/{uuid}/enrollments` returning a match's roster with per-row visibility (master sees private fields; everyone else sees base only).
- Aligns `GetMatchUC` with the same private-match access rule (master OR campaign participant).
- Introduces the `entity.Enrollment` aggregate and the composite index `(match_uuid, created_at)` on `enrollments`.

## Test plan
- [ ] `go test ./...`
- [ ] `go test -tags=integration ./internal/gateway/pg/...`
- [ ] Manual smoke: master token → private fields populated; non-master token → `private: null` for all rows.
- [ ] Manual smoke: private match + participant token → 200; private match + non-participant → 403.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 5: Confirm CI passes**

```bash
sleep 30 && gh run list --workflow=ci.yml --limit=1
```
Expected: status `completed`, conclusion `success`. If failure, run `gh run view <run-id> --log-failed` to triage.

---

## Notes for the Implementer

- **Token discipline:** Per the project memory, never bash-read `.github/instructions/` or `CLAUDE.md` files — they auto-load as system reminders. Always use `rtk` for bash; if a normal command returns empty, debug via `rtk proxy <cmd>`, never `/usr/bin/<cmd>`.
- **Commit hygiene:** Each task ends with one commit. If a step fails, fix and commit anew (do not amend).
- **TDD discipline:** Write the test, verify it fails, implement, verify it passes — in that order. Never implement before seeing a red test.
- **Mock pattern:** The match handler package uses one shared `mocks_test.go`. Append new mocks there; do not create per-file mock files.
- **Task 5 sanity:** The refactor preserves the JSON output of `CharacterPrivateSummaryResponse` because struct embedding flattens — but if any handler test starts failing on JSON shape, the embedding order or field names diverged. Diff against the pre-refactor file before chasing other causes.
