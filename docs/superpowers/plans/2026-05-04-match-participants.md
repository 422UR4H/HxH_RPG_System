# Match Participants Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an n:m `match_participants` table, populate it on `StartMatch`, and expose `GET /matches/{uuid}/participants` with the same visibility pattern as enrollments.

**Architecture:** A new `Participant` entity lives in `domain/entity/match/`. Two gateway methods are added to the existing `pg/match.Repository`. `StartMatchUC` gains a narrow `IMatchParticipantWriter` dependency and generates the `gameStartAt` timestamp (lifted from the gateway). A new `GetMatchParticipantsUC` mirrors `ListMatchEnrollmentsUC` exactly. `CharacterSheetWithVisibilityResponse` is refactored from `api/match/` to `api/sheet/` for reuse.

**Tech Stack:** Go 1.23, pgx v5, goose migrations, huma v2, table-driven tests with `t.Run`, humatest for handler tests.

**Spec:** `docs/superpowers/specs/2026-05-04-match-participants-design.md`

---

## File Map

| Action | File |
|--------|------|
| Create | `migrations/20260504000000_add_match_participants_table.sql` |
| Create | `internal/domain/entity/match/participant.go` |
| Modify | `internal/domain/match/i_repository.go` — `StartMatch` signature |
| Modify | `internal/domain/match/start_match.go` — add `IMatchParticipantWriter`, lift timestamp |
| Modify | `internal/domain/match/start_match_test.go` — update signatures, add participant cases |
| Create | `internal/domain/testutil/mock_match_participant_repo.go` |
| Modify | `internal/domain/testutil/mock_match_repo.go` — `StartMatch` signature |
| Create | `internal/gateway/pg/match/register_participants.go` |
| Create | `internal/gateway/pg/match/read_participants.go` |
| Modify | `internal/gateway/pg/match/start_match.go` — accept `gameStartAt time.Time` |
| Modify | `internal/gateway/pg/match/match_integration_test.go` — new integration tests |
| Modify | `internal/gateway/pg/pgtest/setup.go` — `InsertTestMatchParticipant`, update `TruncateAll` |
| Create | `internal/domain/match/get_match_participants.go` |
| Create | `internal/domain/match/get_match_participants_test.go` |
| Modify | `internal/app/api/sheet/character_sheet_sumary_response.go` — add `CharacterSheetWithVisibilityResponse` |
| Modify | `internal/app/api/match/list_match_enrollments.go` — remove local type, import from sheet |
| Create | `internal/app/api/match/get_match_participants.go` |
| Modify | `internal/app/api/match/mocks_test.go` — add `mockGetMatchParticipants` |
| Create | `internal/app/api/match/get_match_participants_test.go` |
| Modify | `internal/app/api/match/routes.go` — add handler field + register route |
| Modify | `cmd/api/main.go` — wire `GetMatchParticipantsUC` |
| Modify | `cmd/game/main.go` — pass `matchRepository` as 3rd arg to `NewStartMatchUC` |

---

## Task 1: Migration + pgtest update

**Files:**
- Create: `migrations/20260504000000_add_match_participants_table.sql`
- Modify: `internal/gateway/pg/pgtest/setup.go`

- [ ] **Step 1: Create migration**

Create `migrations/20260504000000_add_match_participants_table.sql`:

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

CREATE TABLE IF NOT EXISTS match_participants (
    id   SERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT gen_random_uuid(),

    match_uuid           UUID NOT NULL REFERENCES matches(uuid),
    character_sheet_uuid UUID NOT NULL REFERENCES character_sheets(uuid),

    joined_at  TIMESTAMP NOT NULL,
    left_at    TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (uuid),
    UNIQUE (match_uuid, character_sheet_uuid)
);
CREATE INDEX idx_match_participants_match_uuid ON match_participants(match_uuid);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP TABLE IF EXISTS match_participants;

COMMIT;
-- +goose StatementEnd
```

- [ ] **Step 2: Add `InsertTestMatchParticipant` and update `TruncateAll`**

In `internal/gateway/pg/pgtest/setup.go`, update `TruncateAll` to include `match_participants` (before `matches` in the list to avoid FK issues):

```go
func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE enrollments, submissions, sessions,
			match_participants, matches, campaigns, scenarios,
			joint_proficiencies, proficiencies, character_profiles,
			character_sheets, users
		CASCADE
	`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
}
```

Then add at the end of the file:

```go
func InsertTestMatchParticipant(
	t *testing.T, pool *pgxpool.Pool,
	matchUUID, sheetUUID string, joinedAt time.Time,
) string {
	t.Helper()
	ctx := context.Background()
	now := time.Now()

	var participantUUID string
	err := pool.QueryRow(ctx,
		`INSERT INTO match_participants
		 (uuid, match_uuid, character_sheet_uuid, joined_at, created_at, updated_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $4) RETURNING uuid`,
		matchUUID, sheetUUID, joinedAt, now,
	).Scan(&participantUUID)
	if err != nil {
		t.Fatalf("failed to insert test match participant: %v", err)
	}
	return participantUUID
}
```

- [ ] **Step 3: Commit**

```bash
git add migrations/20260504000000_add_match_participants_table.sql \
        internal/gateway/pg/pgtest/setup.go
git commit -m "feat(migration): add match_participants table and pgtest helper

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 2: `Participant` entity

**Files:**
- Create: `internal/domain/entity/match/participant.go`

- [ ] **Step 1: Create entity**

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
	Sheet     csEntity.Summary
	JoinedAt  time.Time
	LeftAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/domain/entity/match/...
```
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/domain/entity/match/participant.go
git commit -m "feat(entity): add Participant to match entity package

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 3: `StartMatch` signature cascade

Update the `StartMatch` signature across interface, mock, gateway, and test in one atomic step to keep the build green.

**Files:**
- Modify: `internal/domain/match/i_repository.go`
- Modify: `internal/domain/testutil/mock_match_repo.go`
- Modify: `internal/gateway/pg/match/start_match.go`
- Modify: `internal/domain/match/start_match_test.go`

- [ ] **Step 1: Update `IRepository` interface**

In `internal/domain/match/i_repository.go`, change:
```go
StartMatch(ctx context.Context, matchUUID uuid.UUID) error
```
to:
```go
StartMatch(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error
```

Add `"time"` to imports if not already present.

- [ ] **Step 2: Update `MockMatchRepo`**

In `internal/domain/testutil/mock_match_repo.go`, change:

```go
type MockMatchRepo struct {
	CreateMatchFn               func(ctx context.Context, match *match.Match) error
	GetMatchFn                  func(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
	GetMatchCampaignUUIDFn      func(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
	StartMatchFn                func(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error
	ListMatchesByMasterUUIDFn   func(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
	ListPublicUpcomingMatchesFn func(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error)
}
```

And the method:

```go
func (m *MockMatchRepo) StartMatch(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error {
	if m.StartMatchFn != nil {
		return m.StartMatchFn(ctx, matchUUID, gameStartAt)
	}
	return nil
}
```

- [ ] **Step 3: Update gateway `start_match.go`**

Replace the full function in `internal/gateway/pg/match/start_match.go`:

```go
func (r *Repository) StartMatch(
	ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
) error {
	now := time.Now()
	const query = `
		UPDATE matches
		SET game_start_at = $1, updated_at = $2
		WHERE uuid = $3 AND game_start_at IS NULL
	`
	result, err := r.q.Exec(ctx, query, gameStartAt, now, matchUUID)
	if err != nil {
		return fmt.Errorf("failed to start match: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMatchNotFound
	}
	return nil
}
```

- [ ] **Step 4: Fix `start_match_test.go` compile error**

In `internal/domain/match/start_match_test.go`, update the `StartMatchFn` closure in the "repo error on StartMatch" case:

```go
{
    name:       "repo error on StartMatch",
    matchUUID:  matchUUID,
    masterUUID: masterUUID,
    matchMock: &testutil.MockMatchRepo{
        GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
            return &matchEntity.Match{
                UUID:            matchUUID,
                MasterUUID:      masterUUID,
                CampaignUUID:    campaignUUID,
                GameScheduledAt: now,
            }, nil
        },
        StartMatchFn: func(ctx context.Context, id uuid.UUID, gameStartAt time.Time) error {
            return errors.New("db error")
        },
    },
    enrollMock: &testutil.MockEnrollmentRepo{},
    wantErr:    errors.New("db error"),
},
```

- [ ] **Step 5: Verify build passes**

```bash
go build ./...
```
Expected: no output (success).

- [ ] **Step 6: Run existing tests to confirm no regressions**

```bash
go test ./internal/domain/match/...
```
Expected: `ok  github.com/422UR4H/HxH_RPG_System/internal/domain/match`

- [ ] **Step 7: Commit**

```bash
git add internal/domain/match/i_repository.go \
        internal/domain/testutil/mock_match_repo.go \
        internal/gateway/pg/match/start_match.go \
        internal/domain/match/start_match_test.go
git commit -m "refactor(gateway): lift game_start_at timestamp to StartMatchUC

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 4: `RegisterFromAcceptedEnrollments` gateway (TDD)

**Files:**
- Modify: `internal/gateway/pg/match/match_integration_test.go`
- Create: `internal/gateway/pg/match/register_participants.go`

- [ ] **Step 1: Write the failing integration test**

Add to `internal/gateway/pg/match/match_integration_test.go`:

```go
func TestRegisterFromAcceptedEnrollments(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_reg", "gm_reg@hunter.com", "pass"))
	campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Register Campaign"))
	player1UUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p1_reg", "p1_reg@hunter.com", "pass"))
	player2UUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p2_reg", "p2_reg@hunter.com", "pass"))

	t.Run("registers only accepted enrollments", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_reg2", "gm_reg2@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Reg Campaign 2"))
		player1UUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p1_reg2", "p1_reg2@hunter.com", "pass"))
		player2UUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p2_reg2", "p2_reg2@hunter.com", "pass"))

		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "Reg Session"))
		sheet1UUID := pgtest.InsertTestCharacterSheet(t, pool, &[]string{player1UUID.String()}[0], nil, nil, "Gon")
		sheet2UUID := pgtest.InsertTestCharacterSheet(t, pool, &[]string{player2UUID.String()}[0], nil, nil, "Killua")

		pgtest.InsertTestEnrollment(t, pool, matchUUID.String(), sheet1UUID, "accepted")
		pgtest.InsertTestEnrollment(t, pool, matchUUID.String(), sheet2UUID, "pending")

		gameStartAt := time.Now().Truncate(time.Microsecond)
		if err := repo.RegisterFromAcceptedEnrollments(ctx, matchUUID, gameStartAt); err != nil {
			t.Fatalf("RegisterFromAcceptedEnrollments() unexpected error: %v", err)
		}

		var count int
		err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM match_participants WHERE match_uuid = $1`,
			matchUUID,
		).Scan(&count)
		if err != nil {
			t.Fatalf("count query error: %v", err)
		}
		if count != 1 {
			t.Errorf("participant count = %d, want 1", count)
		}

		var joinedAt time.Time
		err = pool.QueryRow(ctx,
			`SELECT joined_at FROM match_participants WHERE match_uuid = $1`,
			matchUUID,
		).Scan(&joinedAt)
		if err != nil {
			t.Fatalf("joined_at query error: %v", err)
		}
		if !joinedAt.Equal(gameStartAt) {
			t.Errorf("joined_at = %v, want %v", joinedAt, gameStartAt)
		}

		_ = masterUUID
		_ = campaignUUID
		_ = player1UUID
		_ = player2UUID
	})

	t.Run("idempotent on double call", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_idem", "gm_idem@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Idem Campaign"))
		playerUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p_idem", "p_idem@hunter.com", "pass"))

		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "Idem Session"))
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &[]string{playerUUID.String()}[0], nil, nil, "Kurapika")
		pgtest.InsertTestEnrollment(t, pool, matchUUID.String(), sheetUUID, "accepted")

		gameStartAt := time.Now()
		if err := repo.RegisterFromAcceptedEnrollments(ctx, matchUUID, gameStartAt); err != nil {
			t.Fatalf("first call error: %v", err)
		}
		if err := repo.RegisterFromAcceptedEnrollments(ctx, matchUUID, gameStartAt); err != nil {
			t.Fatalf("second call (idempotent) error: %v", err)
		}

		var count int
		if err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM match_participants WHERE match_uuid = $1`, matchUUID,
		).Scan(&count); err != nil {
			t.Fatalf("count query: %v", err)
		}
		if count != 1 {
			t.Errorf("count = %d after double call, want 1", count)
		}

		_ = masterUUID
		_ = campaignUUID
		_ = playerUUID
	})

	_ = masterUUID
	_ = campaignUUID
	_ = player1UUID
	_ = player2UUID
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
go test -tags=integration ./internal/gateway/pg/match/... -run TestRegisterFromAcceptedEnrollments -v
```
Expected: FAIL — `repo.RegisterFromAcceptedEnrollments undefined`

- [ ] **Step 3: Implement**

Create `internal/gateway/pg/match/register_participants.go`:

```go
package match

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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

- [ ] **Step 4: Run test to confirm it passes**

```bash
go test -tags=integration ./internal/gateway/pg/match/... -run TestRegisterFromAcceptedEnrollments -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/gateway/pg/match/register_participants.go \
        internal/gateway/pg/match/match_integration_test.go
git commit -m "feat(gateway): add RegisterFromAcceptedEnrollments to match repository

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 5: `ListParticipantsByMatchUUID` gateway (TDD)

**Files:**
- Modify: `internal/gateway/pg/match/match_integration_test.go`
- Create: `internal/gateway/pg/match/read_participants.go`

- [ ] **Step 1: Write the failing integration test**

Add to `internal/gateway/pg/match/match_integration_test.go`:

```go
func TestListParticipantsByMatchUUID(t *testing.T) {
	pool := pgtest.SetupTestDB(t)
	repo := pgMatch.NewRepository(pool)
	ctx := context.Background()

	t.Run("returns participants with sheet data", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_list", "gm_list@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "List Campaign"))
		playerUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "p_list", "p_list@hunter.com", "pass"))

		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "List Session"))
		sheetUUID := pgtest.InsertTestCharacterSheet(t, pool, &[]string{playerUUID.String()}[0], nil, nil, "Leorio")

		joinedAt := time.Now().Truncate(time.Microsecond)
		pgtest.InsertTestMatchParticipant(t, pool, matchUUID.String(), sheetUUID, joinedAt)

		participants, err := repo.ListParticipantsByMatchUUID(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListParticipantsByMatchUUID() unexpected error: %v", err)
		}
		if len(participants) != 1 {
			t.Fatalf("got %d participants, want 1", len(participants))
		}

		p := participants[0]
		if p.MatchUUID != matchUUID {
			t.Errorf("MatchUUID = %v, want %v", p.MatchUUID, matchUUID)
		}
		if p.Sheet.NickName != "Leorio" {
			t.Errorf("NickName = %q, want %q", p.Sheet.NickName, "Leorio")
		}
		if !p.JoinedAt.Equal(joinedAt) {
			t.Errorf("JoinedAt = %v, want %v", p.JoinedAt, joinedAt)
		}
		if p.LeftAt != nil {
			t.Errorf("LeftAt = %v, want nil", p.LeftAt)
		}

		_ = masterUUID
		_ = campaignUUID
		_ = playerUUID
	})

	t.Run("returns empty slice when no participants", func(t *testing.T) {
		pgtest.TruncateAll(t, pool)

		masterUUID := mustParseUUID(t, pgtest.InsertTestUser(t, pool, "gm_empty", "gm_empty@hunter.com", "pass"))
		campaignUUID := mustParseUUID(t, pgtest.InsertTestCampaign(t, pool, masterUUID.String(), "Empty Campaign"))
		matchUUID := mustParseUUID(t, pgtest.InsertTestMatch(t, pool, masterUUID.String(), campaignUUID.String(), "Empty Session"))

		participants, err := repo.ListParticipantsByMatchUUID(ctx, matchUUID)
		if err != nil {
			t.Fatalf("ListParticipantsByMatchUUID() unexpected error: %v", err)
		}
		if len(participants) != 0 {
			t.Errorf("got %d participants, want 0", len(participants))
		}

		_ = masterUUID
		_ = campaignUUID
	})
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
go test -tags=integration ./internal/gateway/pg/match/... -run TestListParticipantsByMatchUUID -v
```
Expected: FAIL — `repo.ListParticipantsByMatchUUID undefined`

- [ ] **Step 3: Implement**

Create `internal/gateway/pg/match/read_participants.go`:

```go
package match

import (
	"context"
	"fmt"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

func (r *Repository) ListParticipantsByMatchUUID(
	ctx context.Context, matchUUID uuid.UUID,
) ([]*matchEntity.Participant, error) {
	const query = `
		SELECT
			mp.uuid, mp.match_uuid,
			mp.joined_at, mp.left_at,
			mp.created_at, mp.updated_at,
			cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
			cs.category_name, cs.curr_hex_value,
			COALESCE(cs.level, 0), COALESCE(cs.points, 0),
			COALESCE(cs.talent_lvl, 0), COALESCE(cs.skills_lvl, 0),
			COALESCE(cs.health_min_pts, 0), COALESCE(cs.health_curr_pts, 0), COALESCE(cs.health_max_pts, 0),
			COALESCE(cs.stamina_min_pts, 0), COALESCE(cs.stamina_curr_pts, 0), COALESCE(cs.stamina_max_pts, 0),
			COALESCE(cs.physicals_lvl, 0), COALESCE(cs.mentals_lvl, 0), COALESCE(cs.spirituals_lvl, 0),
			COALESCE(cs.aura_min_pts, 0), COALESCE(cs.aura_curr_pts, 0), COALESCE(cs.aura_max_pts, 0),
			cs.created_at, cs.updated_at,
			cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday,
			cs.story_start_at, cs.story_current_at, cs.dead_at
		FROM match_participants mp
		JOIN character_sheets cs   ON cs.uuid = mp.character_sheet_uuid
		JOIN character_profiles cp ON cp.character_sheet_uuid = cs.uuid
		LEFT JOIN users u          ON u.uuid = cs.player_uuid
		WHERE mp.match_uuid = $1
		ORDER BY mp.joined_at ASC
	`

	rows, err := r.q.Query(ctx, query, matchUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query participants: %w", err)
	}
	defer rows.Close()

	out := make([]*matchEntity.Participant, 0)
	for rows.Next() {
		var p matchEntity.Participant
		s := &p.Sheet
		err := rows.Scan(
			&p.UUID, &p.MatchUUID,
			&p.JoinedAt, &p.LeftAt,
			&p.CreatedAt, &p.UpdatedAt,
			&s.ID, &s.UUID, &s.PlayerUUID, &s.MasterUUID, &s.CampaignUUID,
			&s.CategoryName, &s.CurrHexValue,
			&s.Level, &s.Points, &s.TalentLvl, &s.SkillsLvl,
			&s.Health.Min, &s.Health.Curr, &s.Health.Max,
			&s.Stamina.Min, &s.Stamina.Curr, &s.Stamina.Max,
			&s.PhysicalsLvl, &s.MentalsLvl, &s.SpiritualsLvl,
			&s.Aura.Min, &s.Aura.Curr, &s.Aura.Max,
			&s.CreatedAt, &s.UpdatedAt,
			&s.NickName, &s.FullName, &s.Alignment, &s.CharacterClass, &s.Birthday,
			&s.StoryStartAt, &s.StoryCurrentAt, &s.DeadAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan participant row: %w", err)
		}
		out = append(out, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return out, nil
}
```

- [ ] **Step 4: Run test to confirm it passes**

```bash
go test -tags=integration ./internal/gateway/pg/match/... -run TestListParticipantsByMatchUUID -v
```
Expected: PASS

- [ ] **Step 5: Run all match integration tests**

```bash
go test -tags=integration ./internal/gateway/pg/match/... -v
```
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/gateway/pg/match/read_participants.go \
        internal/gateway/pg/match/match_integration_test.go \
        internal/gateway/pg/pgtest/setup.go
git commit -m "feat(gateway): add ListParticipantsByMatchUUID to match repository

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 6: Extend `StartMatchUC` + update game server wiring

**Files:**
- Create: `internal/domain/testutil/mock_match_participant_repo.go`
- Modify: `internal/domain/match/start_match.go`
- Modify: `internal/domain/match/start_match_test.go`
- Modify: `cmd/game/main.go`

- [ ] **Step 1: Create `MockMatchParticipantWriter`**

Create `internal/domain/testutil/mock_match_participant_repo.go`:

```go
package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MockMatchParticipantWriter struct {
	RegisterFromAcceptedEnrollmentsFn func(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error
}

func (m *MockMatchParticipantWriter) RegisterFromAcceptedEnrollments(
	ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
) error {
	if m.RegisterFromAcceptedEnrollmentsFn != nil {
		return m.RegisterFromAcceptedEnrollmentsFn(ctx, matchUUID, gameStartAt)
	}
	return nil
}
```

- [ ] **Step 2: Write the failing unit tests for new cases**

In `internal/domain/match/start_match_test.go`, update the test table to include `participantMock` and two new cases. Replace the entire `TestStartMatch` function:

```go
func TestStartMatch(t *testing.T) {
	masterUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()
	otherUUID := uuid.New()
	now := time.Now()
	finishedAt := now.Add(-time.Hour)

	tests := []struct {
		name            string
		matchUUID       uuid.UUID
		masterUUID      uuid.UUID
		matchMock       *testutil.MockMatchRepo
		enrollMock      *testutil.MockEnrollmentRepo
		participantMock *testutil.MockMatchParticipantWriter
		wantErr         error
	}{
		{
			name:            "success",
			matchUUID:       matchUUID,
			masterUUID:      masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         nil,
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
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         domainMatch.ErrMatchNotFound,
		},
		{
			name:       "not match master",
			matchUUID:  matchUUID,
			masterUUID: otherUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         domainMatch.ErrNotMatchMaster,
		},
		{
			name:       "match already started",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
						GameStartAt:     &now,
					}, nil
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         domainMatch.ErrMatchAlreadyStarted,
		},
		{
			name:       "match already finished",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
						StoryEndAt:      &finishedAt,
					}, nil
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         domainMatch.ErrMatchAlreadyFinished,
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
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         errors.New("db error"),
		},
		{
			name:       "repo error on StartMatch",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
				StartMatchFn: func(ctx context.Context, id uuid.UUID, gameStartAt time.Time) error {
					return errors.New("db error")
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         errors.New("db error"),
		},
		{
			name:       "repo error on RejectPendingEnrollments",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{
				RejectPendingEnrollmentsFn: func(ctx context.Context, id uuid.UUID) error {
					return errors.New("db error")
				},
			},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         errors.New("db error"),
		},
		{
			name:       "repo error on RegisterFromAcceptedEnrollments",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{
				RegisterFromAcceptedEnrollmentsFn: func(ctx context.Context, id uuid.UUID, gameStartAt time.Time) error {
					return errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := domainMatch.NewStartMatchUC(tt.matchMock, tt.enrollMock, tt.participantMock)
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

- [ ] **Step 3: Run to confirm it fails**

```bash
go test ./internal/domain/match/... -run TestStartMatch -v
```
Expected: FAIL — `NewStartMatchUC` has wrong number of arguments.

- [ ] **Step 4: Implement — update `start_match.go`**

Replace the full content of `internal/domain/match/start_match.go`:

```go
package match

import (
	"context"
	"time"

	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IStartMatch interface {
	Start(ctx context.Context, matchUUID uuid.UUID, masterUUID uuid.UUID) error
}

// IEnrollmentRepository is a narrow interface to avoid import cycles with the enrollment package.
type IEnrollmentRepository interface {
	RejectPendingEnrollments(ctx context.Context, matchUUID uuid.UUID) error
}

type IMatchParticipantWriter interface {
	RegisterFromAcceptedEnrollments(
		ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
	) error
}

type StartMatchUC struct {
	matchRepo       IRepository
	enrollmentRepo  IEnrollmentRepository
	participantRepo IMatchParticipantWriter
}

func NewStartMatchUC(
	matchRepo IRepository,
	enrollmentRepo IEnrollmentRepository,
	participantRepo IMatchParticipantWriter,
) *StartMatchUC {
	return &StartMatchUC{
		matchRepo:       matchRepo,
		enrollmentRepo:  enrollmentRepo,
		participantRepo: participantRepo,
	}
}

func (uc *StartMatchUC) Start(
	ctx context.Context,
	matchUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if err == matchPg.ErrMatchNotFound {
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

- [ ] **Step 5: Fix `cmd/game/main.go` wiring**

In `cmd/game/main.go`, change line 44:
```go
// before
startMatchUC := domainMatch.NewStartMatchUC(matchRepository, enrollmentRepository)
// after
startMatchUC := domainMatch.NewStartMatchUC(matchRepository, enrollmentRepository, matchRepository)
```

`matchRepository` satisfies `IMatchParticipantWriter` via structural typing (it now has `RegisterFromAcceptedEnrollments`).

- [ ] **Step 6: Run test to confirm it passes**

```bash
go test ./internal/domain/match/... -run TestStartMatch -v
```
Expected: PASS

- [ ] **Step 7: Verify full build**

```bash
go build ./...
```
Expected: no output.

- [ ] **Step 8: Commit**

```bash
git add internal/domain/match/start_match.go \
        internal/domain/match/start_match_test.go \
        internal/domain/testutil/mock_match_participant_repo.go \
        cmd/game/main.go
git commit -m "feat(domain): extend StartMatchUC to register participants on start

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 7: `GetMatchParticipantsUC` (TDD)

**Files:**
- Create: `internal/domain/match/get_match_participants_test.go`
- Create: `internal/domain/match/get_match_participants.go`

- [ ] **Step 1: Write the failing unit tests**

Create `internal/domain/match/get_match_participants_test.go`:

```go
package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type mockParticipantReader struct {
	fn func(ctx context.Context, matchUUID uuid.UUID) ([]*matchEntity.Participant, error)
}

func (m *mockParticipantReader) ListParticipantsByMatchUUID(
	ctx context.Context, matchUUID uuid.UUID,
) ([]*matchEntity.Participant, error) {
	return m.fn(ctx, matchUUID)
}

func TestGetMatchParticipants(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()

	privateMatch := &matchEntity.Match{
		UUID:         matchUUID,
		MasterUUID:   masterUUID,
		CampaignUUID: campaignUUID,
		IsPublic:     false,
	}
	publicMatch := &matchEntity.Match{
		UUID:         matchUUID,
		MasterUUID:   masterUUID,
		CampaignUUID: campaignUUID,
		IsPublic:     true,
	}
	twoParticipants := []*matchEntity.Participant{
		{UUID: uuid.New(), MatchUUID: matchUUID},
		{UUID: uuid.New(), MatchUUID: matchUUID},
	}

	checkerNeverCalled := func(t *testing.T) *mockParticipationChecker {
		return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			t.Fatal("participationChecker should NOT be called")
			return false, nil
		}}
	}

	tests := []struct {
		name              string
		userUUID          uuid.UUID
		matchMock         *testutil.MockMatchRepo
		participantMock   *mockParticipantReader
		checker           func(t *testing.T) *mockParticipationChecker
		wantErr           error
		wantLen           int
		wantViewerMaster  bool
	}{
		{
			name:     "success as master — returns participants with ViewerIsMaster true",
			userUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return privateMatch, nil
				},
			},
			participantMock: &mockParticipantReader{
				fn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return twoParticipants, nil
				},
			},
			checker:          checkerNeverCalled,
			wantLen:          2,
			wantViewerMaster: true,
		},
		{
			name:     "success as player on public match",
			userUUID: otherUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return publicMatch, nil
				},
			},
			participantMock: &mockParticipantReader{
				fn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return twoParticipants, nil
				},
			},
			checker:          checkerNeverCalled,
			wantLen:          2,
			wantViewerMaster: false,
		},
		{
			name:     "private match — non-participant gets forbidden",
			userUUID: otherUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return privateMatch, nil
				},
			},
			participantMock: &mockParticipantReader{
				fn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					t.Fatal("participant reader should NOT be called")
					return nil, nil
				},
			},
			checker: func(_ *testing.T) *mockParticipationChecker {
				return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
					return false, nil
				}}
			},
			wantErr: auth.ErrInsufficientPermissions,
		},
		{
			name:     "match not found",
			userUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return nil, matchPg.ErrMatchNotFound
				},
			},
			participantMock: &mockParticipantReader{
				fn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					t.Fatal("should not be called")
					return nil, nil
				},
			},
			checker: checkerNeverCalled,
			wantErr: domainMatch.ErrMatchNotFound,
		},
		{
			name:     "participant repo error propagated",
			userUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return privateMatch, nil
				},
			},
			participantMock: &mockParticipantReader{
				fn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return nil, errors.New("db error")
				},
			},
			checker: checkerNeverCalled,
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := domainMatch.NewGetMatchParticipantsUC(
				tt.matchMock,
				tt.participantMock,
				tt.checker(t),
			)
			result, err := uc.Get(context.Background(), matchUUID, tt.userUUID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected %q, got %q", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Participants) != tt.wantLen {
				t.Errorf("len(Participants) = %d, want %d", len(result.Participants), tt.wantLen)
			}
			if result.ViewerIsMaster != tt.wantViewerMaster {
				t.Errorf("ViewerIsMaster = %v, want %v", result.ViewerIsMaster, tt.wantViewerMaster)
			}
		})
	}
}
```

- [ ] **Step 2: Run to confirm it fails**

```bash
go test ./internal/domain/match/... -run TestGetMatchParticipants -v
```
Expected: FAIL — `domainMatch.NewGetMatchParticipantsUC undefined`

- [ ] **Step 3: Implement**

Create `internal/domain/match/get_match_participants.go`:

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

type GetMatchParticipantsUC struct {
	matchRepo            IRepository
	participantRepo      IMatchParticipantReader
	participationChecker CampaignParticipationChecker
}

func NewGetMatchParticipantsUC(
	matchRepo IRepository,
	participantRepo IMatchParticipantReader,
	participationChecker CampaignParticipationChecker,
) *GetMatchParticipantsUC {
	return &GetMatchParticipantsUC{
		matchRepo:            matchRepo,
		participantRepo:      participantRepo,
		participationChecker: participationChecker,
	}
}

func (uc *GetMatchParticipantsUC) Get(
	ctx context.Context, matchUUID, userUUID uuid.UUID,
) (*GetMatchParticipantsResult, error) {
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

	participants, err := uc.participantRepo.ListParticipantsByMatchUUID(ctx, matchUUID)
	if err != nil {
		return nil, err
	}

	return &GetMatchParticipantsResult{
		Participants:   participants,
		ViewerIsMaster: viewerIsMaster,
	}, nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/domain/match/... -v
```
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/get_match_participants.go \
        internal/domain/match/get_match_participants_test.go
git commit -m "feat(domain): add GetMatchParticipantsUC

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 8: Move `CharacterSheetWithVisibilityResponse` to `api/sheet/`

**Files:**
- Modify: `internal/app/api/sheet/character_sheet_sumary_response.go`
- Modify: `internal/app/api/match/list_match_enrollments.go`

- [ ] **Step 1: Add type to `api/sheet/`**

In `internal/app/api/sheet/character_sheet_sumary_response.go`, add at the end of the file (after the existing functions):

```go
// CharacterSheetWithVisibilityResponse is the API shape used wherever a
// character sheet is returned with visibility-gated private fields.
// Private is nil for non-master viewers; the field is intentionally not
// omitempty so consumers always receive the key (as null).
type CharacterSheetWithVisibilityResponse struct {
	CharacterBaseSummaryResponse
	Private *CharacterPrivateOnlyResponse `json:"private"`
}
```

- [ ] **Step 2: Update `list_match_enrollments.go`**

In `internal/app/api/match/list_match_enrollments.go`:

1. Remove the local `CharacterSheetWithVisibilityResponse` type definition (the struct and its comment).

2. Change `EnrollmentResponse.CharacterSheet` field type from the local type to `apiSheet.CharacterSheetWithVisibilityResponse`:

```go
type EnrollmentResponse struct {
	UUID           uuid.UUID                                     `json:"uuid"`
	Status         string                                        `json:"status"`
	CreatedAt      string                                        `json:"created_at"`
	CharacterSheet apiSheet.CharacterSheetWithVisibilityResponse `json:"character_sheet"`
	Player         PlayerRefResponse                             `json:"player"`
}
```

3. Update `toEnrollmentResponse` to use the fully-qualified type:

```go
func toEnrollmentResponse(e *enrollmentEntity.Enrollment, viewerIsMaster bool) EnrollmentResponse {
	sheet := apiSheet.CharacterSheetWithVisibilityResponse{
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

- [ ] **Step 3: Build and run enrollment handler tests**

```bash
go build ./... && go test ./internal/app/api/match/... -v
```
Expected: build succeeds, all tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/app/api/sheet/character_sheet_sumary_response.go \
        internal/app/api/match/list_match_enrollments.go
git commit -m "refactor(api): move CharacterSheetWithVisibilityResponse to api/sheet

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Task 9: `GetMatchParticipantsHandler` + route + wiring

**Files:**
- Modify: `internal/app/api/match/mocks_test.go`
- Create: `internal/app/api/match/get_match_participants_test.go`
- Create: `internal/app/api/match/get_match_participants.go`
- Modify: `internal/app/api/match/routes.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Add mock to `mocks_test.go`**

In `internal/app/api/match/mocks_test.go`, add:

```go
type mockGetMatchParticipants struct {
	fn func(ctx context.Context, matchUUID, userUUID uuid.UUID) (*domainMatch.GetMatchParticipantsResult, error)
}

func (m *mockGetMatchParticipants) Get(
	ctx context.Context, matchUUID, userUUID uuid.UUID,
) (*domainMatch.GetMatchParticipantsResult, error) {
	return m.fn(ctx, matchUUID, userUUID)
}
```

Also add the missing import `matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"` if needed by the tests below.

- [ ] **Step 2: Write failing handler tests**

Create `internal/app/api/match/get_match_participants_test.go`:

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
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestGetMatchParticipantsHandler(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	now := time.Now()

	makeFixture := func() []*matchEntity.Participant {
		return []*matchEntity.Participant{
			{
				UUID:      uuid.New(),
				MatchUUID: matchUUID,
				JoinedAt:  now,
				Sheet: csEntity.Summary{
					UUID:     uuid.New(),
					NickName: "Gon",
					FullName: "Gon Freecss",
					Birthday: now,
				},
			},
		}
	}

	tests := []struct {
		name           string
		ucFn           func(ctx context.Context, matchID, uid uuid.UUID) (*domainMatch.GetMatchParticipantsResult, error)
		wantStatus     int
		wantPrivateNil bool
	}{
		{
			name: "200 with private populated when ViewerIsMaster",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.GetMatchParticipantsResult, error) {
				return &domainMatch.GetMatchParticipantsResult{
					Participants:   makeFixture(),
					ViewerIsMaster: true,
				}, nil
			},
			wantStatus:     http.StatusOK,
			wantPrivateNil: false,
		},
		{
			name: "200 with private null when not master",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.GetMatchParticipantsResult, error) {
				return &domainMatch.GetMatchParticipantsResult{
					Participants:   makeFixture(),
					ViewerIsMaster: false,
				}, nil
			},
			wantStatus:     http.StatusOK,
			wantPrivateNil: true,
		},
		{
			name: "200 with empty list",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.GetMatchParticipantsResult, error) {
				return &domainMatch.GetMatchParticipantsResult{
					Participants:   []*matchEntity.Participant{},
					ViewerIsMaster: true,
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "404 when match not found",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.GetMatchParticipantsResult, error) {
				return nil, domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "403 when insufficient permissions",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.GetMatchParticipantsResult, error) {
				return nil, domainAuth.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "500 on unexpected error",
			ucFn: func(_ context.Context, _, _ uuid.UUID) (*domainMatch.GetMatchParticipantsResult, error) {
				return nil, errors.New("unexpected")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t, huma.DefaultConfig("Test", "1.0.0"))

			uc := &mockGetMatchParticipants{fn: tt.ucFn}
			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/matches/{uuid}/participants",
			}, apiMatch.GetMatchParticipantsHandler(uc))

			req, _ := http.NewRequestWithContext(
				context.WithValue(context.Background(), auth.UserIDKey, userUUID),
				http.MethodGet,
				"/matches/"+matchUUID.String()+"/participants",
				nil,
			)

			resp := humatest.NewTestResponse(t)
			api.Adapter().ServeHTTP(resp, req)

			if resp.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var body struct {
					Participants []struct {
						Sheet struct {
							Private *struct{} `json:"private"`
						} `json:"character_sheet"`
					} `json:"participants"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if len(body.Participants) > 0 {
					gotPrivateNil := body.Participants[0].Sheet.Private == nil
					if gotPrivateNil != tt.wantPrivateNil {
						t.Errorf("private nil = %v, want %v", gotPrivateNil, tt.wantPrivateNil)
					}
				}
			}
		})
	}
}
```

- [ ] **Step 3: Run to confirm it fails**

```bash
go test ./internal/app/api/match/... -run TestGetMatchParticipantsHandler -v
```
Expected: FAIL — `apiMatch.GetMatchParticipantsHandler undefined`

- [ ] **Step 4: Implement the handler**

Create `internal/app/api/match/get_match_participants.go`:

```go
package match

import (
	"context"
	"errors"
	"time"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	apiSheet "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetMatchParticipantsRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"UUID of the match"`
}

type GetMatchParticipantsResponseBody struct {
	Participants []ParticipantResponse `json:"participants"`
}

type GetMatchParticipantsResponse struct {
	Body GetMatchParticipantsResponseBody
}

type ParticipantResponse struct {
	UUID     uuid.UUID                                     `json:"uuid"`
	JoinedAt string                                        `json:"joined_at"`
	LeftAt   *string                                       `json:"left_at,omitempty"`
	Sheet    apiSheet.CharacterSheetWithVisibilityResponse `json:"character_sheet"`
}

func GetMatchParticipantsHandler(
	uc domainMatch.IGetMatchParticipants,
) func(context.Context, *GetMatchParticipantsRequest) (*GetMatchParticipantsResponse, error) {
	return func(ctx context.Context, req *GetMatchParticipantsRequest) (*GetMatchParticipantsResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		result, err := uc.Get(ctx, req.UUID, userUUID)
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

		out := make([]ParticipantResponse, 0, len(result.Participants))
		for _, p := range result.Participants {
			out = append(out, toParticipantResponse(p, result.ViewerIsMaster))
		}
		return &GetMatchParticipantsResponse{
			Body: GetMatchParticipantsResponseBody{Participants: out},
		}, nil
	}
}

func toParticipantResponse(p *matchEntity.Participant, viewerIsMaster bool) ParticipantResponse {
	sheet := apiSheet.CharacterSheetWithVisibilityResponse{
		CharacterBaseSummaryResponse: apiSheet.ToBaseSummaryResponse(&p.Sheet),
		Private:                      nil,
	}
	if viewerIsMaster {
		priv := apiSheet.ToPrivateOnlyResponse(&p.Sheet)
		sheet.Private = &priv
	}

	var leftAtStr *string
	if p.LeftAt != nil {
		s := p.LeftAt.Format(time.RFC3339)
		leftAtStr = &s
	}

	return ParticipantResponse{
		UUID:     p.UUID,
		JoinedAt: p.JoinedAt.Format(time.RFC3339),
		LeftAt:   leftAtStr,
		Sheet:    sheet,
	}
}
```

- [ ] **Step 5: Register route in `routes.go`**

In `internal/app/api/match/routes.go`, add the handler field to the `Api` struct:

```go
type Api struct {
	CreateMatchHandler               Handler[CreateMatchRequest, CreateMatchResponse]
	GetMatchHandler                  Handler[GetMatchRequest, GetMatchResponse]
	ListMatchesHandler               Handler[struct{}, ListMatchesResponse]
	ListPublicUpcomingMatchesHandler Handler[struct{}, ListMatchesResponse]
	ListMatchEnrollmentsHandler      Handler[ListMatchEnrollmentsRequest, ListMatchEnrollmentsResponse]
	GetMatchParticipantsHandler      Handler[GetMatchParticipantsRequest, GetMatchParticipantsResponse]
}
```

At the end of `RegisterRoutes`, add:

```go
huma.Register(api, huma.Operation{
    Method:      http.MethodGet,
    Path:        "/matches/{uuid}/participants",
    Description: "List participants of a match (visibility per row depends on viewer)",
    Tags:        []string{"matches"},
    Errors: []int{
        http.StatusNotFound,
        http.StatusForbidden,
        http.StatusUnauthorized,
        http.StatusInternalServerError,
    },
}, a.GetMatchParticipantsHandler)
```

- [ ] **Step 6: Wire in `cmd/api/main.go`**

In `cmd/api/main.go`, after the existing match UC lines (around line 151), add:

```go
getMatchParticipantsUC := domainMatch.NewGetMatchParticipantsUC(matchRepo, matchRepo, characterSheetRepo)
```

`matchRepo` satisfies both `IRepository` and `IMatchParticipantReader` via structural typing. `characterSheetRepo` satisfies `CampaignParticipationChecker` (it already does for `listMatchEnrollmentsUC`).

Then update the `matchesApi` struct literal to include the new handler:

```go
matchesApi := matchHandler.Api{
    CreateMatchHandler:               matchHandler.CreateMatchHandler(createMatchUC),
    GetMatchHandler:                  matchHandler.GetMatchHandler(getMatchUC),
    ListMatchesHandler:               matchHandler.ListMatchesHandler(listMatchesUC),
    ListPublicUpcomingMatchesHandler: matchHandler.ListPublicUpcomingMatchesHandler(listPublicUpcomingMatchesUC),
    ListMatchEnrollmentsHandler:      matchHandler.ListMatchEnrollmentsHandler(listMatchEnrollmentsUC),
    GetMatchParticipantsHandler:      matchHandler.GetMatchParticipantsHandler(getMatchParticipantsUC),
}
```

- [ ] **Step 7: Run handler tests**

```bash
go test ./internal/app/api/match/... -run TestGetMatchParticipantsHandler -v
```
Expected: PASS

- [ ] **Step 8: Full build and all tests**

```bash
go build ./... && go test ./...
```
Expected: build succeeds, all non-integration tests PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/app/api/match/get_match_participants.go \
        internal/app/api/match/get_match_participants_test.go \
        internal/app/api/match/mocks_test.go \
        internal/app/api/match/routes.go \
        cmd/api/main.go
git commit -m "feat(api): add GET /matches/{uuid}/participants endpoint

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Final verification

- [ ] **Run all integration tests**

```bash
go test -tags=integration ./internal/gateway/pg/...
```
Expected: all PASS

- [ ] **Run all unit tests**

```bash
go test ./...
```
Expected: all PASS

- [ ] **Verify vet with integration tag**

```bash
go vet -tags=integration ./internal/gateway/pg/...
```
Expected: no output
