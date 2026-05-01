# Enrollment Accept/Reject Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable the master to accept or reject player enrollment requests in matches, with bidirectional status transitions and idempotent behavior.

**Architecture:** Add a `status` column (`pending`/`accepted`/`rejected`) to the `enrollments` table. Two new use cases (`AcceptEnrollmentUC`, `RejectEnrollmentUC`) mirror the submission accept/reject pattern — each validates master ownership via enrollment → match → campaign chain before updating status. Gateway adds 3 new repo methods. App layer adds 2 new HTTP handlers with path-param-based routing.

**Tech Stack:** Go 1.23, PostgreSQL (goose migrations), Huma v2 + Chi router, pgx v5, standard `testing` package

---

## Phase 1 — Accept Enrollment

### Task 1: Database Migration

**Files:**
- Create: `migrations/20260501180000_add_enrollment_status.sql`

- [ ] **Step 1: Create the migration file**

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE enrollments ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';

ALTER TABLE enrollments DROP CONSTRAINT enrollments_character_sheet_uuid_key;

CREATE UNIQUE INDEX idx_enrollments_active_sheet
ON enrollments (character_sheet_uuid)
WHERE status != 'rejected';

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

DROP INDEX IF EXISTS idx_enrollments_active_sheet;

ALTER TABLE enrollments ADD CONSTRAINT enrollments_character_sheet_uuid_key UNIQUE (character_sheet_uuid);

ALTER TABLE enrollments DROP COLUMN status;

COMMIT;
-- +goose StatementEnd
```

- [ ] **Step 2: Commit**

```bash
git add migrations/20260501180000_add_enrollment_status.sql
git commit -m "feat(enrollment): add status column migration

Adds 'status' (pending/accepted/rejected) column to enrollments table.
Replaces global UNIQUE(character_sheet_uuid) with partial unique index
excluding rejected enrollments.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Gateway — Error Sentinel + Read Status

**Files:**
- Create: `internal/gateway/pg/enrollment/error.go`
- Create: `internal/gateway/pg/enrollment/read_enrollment_status.go`

- [ ] **Step 1: Create the gateway error sentinel**

Create `internal/gateway/pg/enrollment/error.go`:

```go
package enrollment

import "errors"

var (
	ErrEnrollmentNotFound = errors.New("enrollment not found in database")
)
```

- [ ] **Step 2: Create the read enrollment status query**

Create `internal/gateway/pg/enrollment/read_enrollment_status.go`:

```go
package enrollment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetEnrollmentStatus(
	ctx context.Context,
	characterSheetUUID uuid.UUID,
	matchUUID uuid.UUID,
) (string, error) {
	const query = `
		SELECT status
		FROM enrollments
		WHERE character_sheet_uuid = $1 AND match_uuid = $2
	`
	var status string
	err := r.q.QueryRow(ctx, query, characterSheetUUID, matchUUID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrEnrollmentNotFound
		}
		return "", fmt.Errorf("failed to get enrollment status: %w", err)
	}
	return status, nil
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/gateway/pg/enrollment/error.go internal/gateway/pg/enrollment/read_enrollment_status.go
git commit -m "feat(enrollment): add gateway error sentinel and read status query

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Gateway — Accept Enrollment

**Files:**
- Create: `internal/gateway/pg/enrollment/accept_enrollment.go`

- [ ] **Step 1: Create the accept enrollment repository method**

Create `internal/gateway/pg/enrollment/accept_enrollment.go`:

```go
package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) AcceptEnrollment(
	ctx context.Context,
	characterSheetUUID uuid.UUID,
	matchUUID uuid.UUID,
) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	const query = `
		UPDATE enrollments SET status = 'accepted'
		WHERE character_sheet_uuid = $1 AND match_uuid = $2
	`
	result, err := tx.Exec(ctx, query, characterSheetUUID, matchUUID)
	if err != nil {
		return fmt.Errorf("failed to accept enrollment: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/gateway/pg/enrollment/accept_enrollment.go
git commit -m "feat(enrollment): add accept enrollment gateway method

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Domain — Error + Repository Interface Update

**Files:**
- Modify: `internal/domain/enrollment/error.go`
- Modify: `internal/domain/enrollment/i_repository.go`

- [ ] **Step 1: Add new domain errors**

In `internal/domain/enrollment/error.go`, add the new errors after the existing ones:

```go
package enrollment

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCharacterNotInCampaign   = domain.NewValidationError(errors.New("character sheet does not belong to the match's campaign"))
	ErrCharacterAlreadyEnrolled = domain.NewValidationError(errors.New("character sheet is already enrolled in this match"))
	ErrEnrollmentNotFound       = domain.NewValidationError(errors.New("enrollment not found"))
	ErrNotMatchMaster           = domain.NewValidationError(errors.New("user is not the match's campaign master"))
)
```

- [ ] **Step 2: Expand the repository interface**

In `internal/domain/enrollment/i_repository.go`, add the 3 new methods:

```go
package enrollment

import (
	"context"

	"github.com/google/uuid"
)

type IRepository interface {
	EnrollCharacterSheet(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheet(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
	GetEnrollmentStatus(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (string, error)
	AcceptEnrollment(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) error
	RejectEnrollment(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) error
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/domain/enrollment/error.go internal/domain/enrollment/i_repository.go
git commit -m "feat(enrollment): add domain errors and expand repository interface

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Domain — Update Testutil Mock

**Files:**
- Modify: `internal/domain/testutil/mock_enrollment_repo.go`

- [ ] **Step 1: Add mock methods for the 3 new interface methods**

Replace the entire file with:

```go
package testutil

import (
	"context"

	"github.com/google/uuid"
)

type MockEnrollmentRepo struct {
	EnrollCharacterSheetFn         func(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheetFn func(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
	GetEnrollmentStatusFn          func(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (string, error)
	AcceptEnrollmentFn             func(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) error
	RejectEnrollmentFn             func(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) error
}

func (m *MockEnrollmentRepo) EnrollCharacterSheet(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error {
	if m.EnrollCharacterSheetFn != nil {
		return m.EnrollCharacterSheetFn(ctx, matchUUID, characterSheetUUID)
	}
	return nil
}

func (m *MockEnrollmentRepo) ExistsEnrolledCharacterSheet(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error) {
	if m.ExistsEnrolledCharacterSheetFn != nil {
		return m.ExistsEnrolledCharacterSheetFn(ctx, characterSheetUUID, matchUUID)
	}
	return false, nil
}

func (m *MockEnrollmentRepo) GetEnrollmentStatus(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (string, error) {
	if m.GetEnrollmentStatusFn != nil {
		return m.GetEnrollmentStatusFn(ctx, characterSheetUUID, matchUUID)
	}
	return "", nil
}

func (m *MockEnrollmentRepo) AcceptEnrollment(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) error {
	if m.AcceptEnrollmentFn != nil {
		return m.AcceptEnrollmentFn(ctx, characterSheetUUID, matchUUID)
	}
	return nil
}

func (m *MockEnrollmentRepo) RejectEnrollment(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) error {
	if m.RejectEnrollmentFn != nil {
		return m.RejectEnrollmentFn(ctx, characterSheetUUID, matchUUID)
	}
	return nil
}
```

- [ ] **Step 2: Verify existing tests still compile**

Run: `go test ./internal/domain/enrollment/... -count=1`
Expected: All existing tests PASS (mocks now satisfy the expanded interface).

- [ ] **Step 3: Commit**

```bash
git add internal/domain/testutil/mock_enrollment_repo.go
git commit -m "feat(enrollment): expand enrollment mock with accept/reject methods

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: Domain — AcceptEnrollmentUC (TDD)

**Files:**
- Create: `internal/domain/enrollment/accept_enrollment.go`
- Modify: `internal/domain/enrollment/enrollment_test.go`

- [ ] **Step 1: Write the failing tests for AcceptEnrollmentUC**

Append to `internal/domain/enrollment/enrollment_test.go`:

```go
func TestAcceptEnrollment(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	matchUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name         string
		sheetUUID    uuid.UUID
		matchUUID    uuid.UUID
		masterUUID   uuid.UUID
		enrollMock   *testutil.MockEnrollmentRepo
		matchMock    *testutil.MockMatchRepo
		campaignMock *testutil.MockCampaignRepo
		wantErr      error
	}{
		{
			name:       "success from pending",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: nil,
		},
		{
			name:       "success from rejected",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "rejected", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: nil,
		},
		{
			name:       "idempotent when already accepted",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "accepted", nil
				},
			},
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      nil,
		},
		{
			name:       "enrollment not found",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "", enrollmentPg.ErrEnrollmentNotFound
				},
			},
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      enrollment.ErrEnrollmentNotFound,
		},
		{
			name:       "match not found",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, matchPg.ErrMatchNotFound
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      domainMatch.ErrMatchNotFound,
		},
		{
			name:       "campaign not found",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, campaignPg.ErrCampaignNotFound
				},
			},
			wantErr: domainCampaign.ErrCampaignNotFound,
		},
		{
			name:       "not campaign master",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: otherUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: enrollment.ErrNotMatchMaster,
		},
		{
			name:       "repo error on accept",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
				AcceptEnrollmentFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) error {
					return errors.New("db error")
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := enrollment.NewAcceptEnrollmentUC(tt.enrollMock, tt.matchMock, tt.campaignMock)
			err := uc.Accept(context.Background(), tt.sheetUUID, tt.matchUUID, tt.masterUUID)

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

The test file needs these additional imports (add to the existing import block):

```go
import (
	// ... existing imports ...
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/enrollment/... -run TestAcceptEnrollment -count=1`
Expected: FAIL — `NewAcceptEnrollmentUC` undefined.

- [ ] **Step 3: Implement AcceptEnrollmentUC**

Create `internal/domain/enrollment/accept_enrollment.go`:

```go
package enrollment

import (
	"context"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IAcceptEnrollment interface {
	Accept(ctx context.Context, sheetUUID uuid.UUID, matchUUID uuid.UUID, masterUUID uuid.UUID) error
}

type AcceptEnrollmentUC struct {
	repo         IRepository
	matchRepo    matchDomain.IRepository
	campaignRepo campaignDomain.IRepository
}

func NewAcceptEnrollmentUC(
	repo IRepository,
	matchRepo matchDomain.IRepository,
	campaignRepo campaignDomain.IRepository,
) *AcceptEnrollmentUC {
	return &AcceptEnrollmentUC{
		repo:         repo,
		matchRepo:    matchRepo,
		campaignRepo: campaignRepo,
	}
}

func (uc *AcceptEnrollmentUC) Accept(
	ctx context.Context,
	sheetUUID uuid.UUID,
	matchUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	status, err := uc.repo.GetEnrollmentStatus(ctx, sheetUUID, matchUUID)
	if err == enrollmentPg.ErrEnrollmentNotFound {
		return ErrEnrollmentNotFound
	}
	if err != nil {
		return err
	}
	if status == "accepted" {
		return nil
	}

	// TODO: check if match has already started (temporal guard)
	campaignUUID, err := uc.matchRepo.GetMatchCampaignUUID(ctx, matchUUID)
	if err == matchPg.ErrMatchNotFound {
		return matchDomain.ErrMatchNotFound
	}
	if err != nil {
		return err
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, campaignUUID)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if campaignMasterUUID != masterUUID {
		return ErrNotMatchMaster
	}

	return uc.repo.AcceptEnrollment(ctx, sheetUUID, matchUUID)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/enrollment/... -count=1 -v`
Expected: ALL tests PASS (both existing `TestEnrollCharacterSheet` and new `TestAcceptEnrollment`).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/enrollment/accept_enrollment.go internal/domain/enrollment/enrollment_test.go
git commit -m "feat(enrollment): add AcceptEnrollmentUC with tests

Implements accept enrollment use case with master ownership validation
and idempotent behavior. Table-driven tests cover all paths:
pending→accepted, rejected→accepted, idempotent, not found, not master.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 7: App — Accept Enrollment Handler (TDD)

**Files:**
- Create: `internal/app/api/enrollment/accept_enrollment.go`
- Create: `internal/app/api/enrollment/accept_enrollment_test.go`
- Modify: `internal/app/api/enrollment/mocks_test.go`

- [ ] **Step 1: Add the UC mock to mocks_test.go**

Append to `internal/app/api/enrollment/mocks_test.go`:

```go
type mockAcceptEnrollment struct {
	fn func(ctx context.Context, sheetUUID, matchUUID, masterUUID uuid.UUID) error
}

func (m *mockAcceptEnrollment) Accept(ctx context.Context, sheetUUID, matchUUID, masterUUID uuid.UUID) error {
	return m.fn(ctx, sheetUUID, matchUUID, masterUUID)
}
```

- [ ] **Step 2: Write the failing handler tests**

Create `internal/app/api/enrollment/accept_enrollment_test.go`:

```go
package enrollment_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/enrollment"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainEnrollment "github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestAcceptEnrollmentHandler(t *testing.T) {
	masterUUID := uuid.New()
	sheetUUID := uuid.New()
	matchUUID := uuid.New()

	tests := []struct {
		name          string
		pathSheetUUID string
		pathMatchUUID string
		mockFn        func(ctx context.Context, sheetUUID, matchUUID, masterUUID uuid.UUID) error
		wantStatus    int
	}{
		{
			name:          "success",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "invalid sheet uuid",
			pathSheetUUID: "not-a-valid-uuid",
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:          "invalid match uuid",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: "not-a-valid-uuid",
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:          "enrollment not found",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return domainEnrollment.ErrEnrollmentNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:          "match not found",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:          "campaign not found",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return domainCampaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:          "not match master",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return domainEnrollment.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:          "generic error",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return errors.New("unexpected database error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockAcceptEnrollment{fn: tt.mockFn}
			handler := enrollment.AcceptEnrollmentHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodPost,
				Path:   "/enrollments/{sheet_uuid}/{match_uuid}/accept",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, masterUUID)
			resp := api.PostCtx(ctx, "/enrollments/"+tt.pathSheetUUID+"/"+tt.pathMatchUUID+"/accept")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/app/api/enrollment/... -run TestAcceptEnrollmentHandler -count=1`
Expected: FAIL — `AcceptEnrollmentHandler` undefined.

- [ ] **Step 4: Implement the handler**

Create `internal/app/api/enrollment/accept_enrollment.go`:

```go
package enrollment

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainEnrollment "github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type AcceptEnrollmentRequest struct {
	SheetUUID string `path:"sheet_uuid" required:"true" doc:"enrolled character sheet UUID"`
	MatchUUID string `path:"match_uuid" required:"true" doc:"match UUID"`
}

type AcceptEnrollmentResponse struct {
	Status int `json:"status"`
}

func AcceptEnrollmentHandler(
	uc domainEnrollment.IAcceptEnrollment,
) func(context.Context, *AcceptEnrollmentRequest) (*AcceptEnrollmentResponse, error) {

	return func(ctx context.Context, req *AcceptEnrollmentRequest) (*AcceptEnrollmentResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		sheetUUID, err := uuid.Parse(req.SheetUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid sheet UUID")
		}

		matchUUID, err := uuid.Parse(req.MatchUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid match UUID")
		}

		err = uc.Accept(ctx, sheetUUID, matchUUID, masterUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainEnrollment.ErrEnrollmentNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainMatch.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainCampaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainEnrollment.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &AcceptEnrollmentResponse{
			Status: http.StatusOK,
		}, nil
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/app/api/enrollment/... -count=1 -v`
Expected: ALL tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/api/enrollment/accept_enrollment.go internal/app/api/enrollment/accept_enrollment_test.go internal/app/api/enrollment/mocks_test.go
git commit -m "feat(enrollment): add accept enrollment handler with tests

POST /enrollments/{sheet_uuid}/{match_uuid}/accept
Maps domain errors to appropriate HTTP status codes.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 8: Wiring — Accept Enrollment (Routes + main.go)

**Files:**
- Modify: `internal/app/api/enrollment/routes.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Add the accept handler to routes.go**

Replace the entire `internal/app/api/enrollment/routes.go`:

```go
package enrollment

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	EnrollCharacterHandler  Handler[EnrollCharacterRequest, EnrollCharacterResponse]
	AcceptEnrollmentHandler Handler[AcceptEnrollmentRequest, AcceptEnrollmentResponse]
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/enrollments/charactersheets/enroll",
		Description: "Enroll a character sheet in a match",
		Tags:        []string{"enrollments"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusConflict,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, a.EnrollCharacterHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/enrollments/{sheet_uuid}/{match_uuid}/accept",
		Description: "Accept a character sheet enrollment in a match",
		Tags:        []string{"enrollments"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusInternalServerError,
		},
	}, a.AcceptEnrollmentHandler)
}
```

- [ ] **Step 2: Wire the UC in main.go**

In `cmd/api/main.go`, after the existing `enrollCharacterSheetUC` instantiation (around line 178-185), add the accept UC and update the Api struct:

Find this block:
```go
	enrollCharacterSheetUC := domainEnrollment.NewEnrollCharacterInMatchUC(
		enrollmentRepo,
		matchRepo,
		characterSheetRepo,
	)
	enrollmentApi := enrollmentHandler.Api{
		EnrollCharacterHandler: enrollmentHandler.EnrollCharacterHandler(enrollCharacterSheetUC),
	}
```

Replace with:
```go
	enrollCharacterSheetUC := domainEnrollment.NewEnrollCharacterInMatchUC(
		enrollmentRepo,
		matchRepo,
		characterSheetRepo,
	)
	acceptEnrollmentUC := domainEnrollment.NewAcceptEnrollmentUC(
		enrollmentRepo,
		matchRepo,
		campaignRepo,
	)
	enrollmentApi := enrollmentHandler.Api{
		EnrollCharacterHandler:  enrollmentHandler.EnrollCharacterHandler(enrollCharacterSheetUC),
		AcceptEnrollmentHandler: enrollmentHandler.AcceptEnrollmentHandler(acceptEnrollmentUC),
	}
```

- [ ] **Step 3: Verify build succeeds**

Run: `go build ./...`
Expected: Build succeeds with no errors.

- [ ] **Step 4: Run all tests**

Run: `go test ./internal/domain/enrollment/... ./internal/app/api/enrollment/... -count=1 -v`
Expected: ALL tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/api/enrollment/routes.go cmd/api/main.go
git commit -m "feat(enrollment): wire accept enrollment endpoint

Registers POST /enrollments/{sheet_uuid}/{match_uuid}/accept
and wires AcceptEnrollmentUC in main.go.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Phase 2 — Reject Enrollment

### Task 9: Gateway — Reject Enrollment

**Files:**
- Create: `internal/gateway/pg/enrollment/reject_enrollment.go`

- [ ] **Step 1: Create the reject enrollment repository method**

Create `internal/gateway/pg/enrollment/reject_enrollment.go`:

```go
package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) RejectEnrollment(
	ctx context.Context,
	characterSheetUUID uuid.UUID,
	matchUUID uuid.UUID,
) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	const query = `
		UPDATE enrollments SET status = 'rejected'
		WHERE character_sheet_uuid = $1 AND match_uuid = $2
	`
	result, err := tx.Exec(ctx, query, characterSheetUUID, matchUUID)
	if err != nil {
		return fmt.Errorf("failed to reject enrollment: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/gateway/pg/enrollment/reject_enrollment.go
git commit -m "feat(enrollment): add reject enrollment gateway method

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 10: Domain — RejectEnrollmentUC (TDD)

**Files:**
- Create: `internal/domain/enrollment/reject_enrollment.go`
- Modify: `internal/domain/enrollment/enrollment_test.go`

- [ ] **Step 1: Write the failing tests for RejectEnrollmentUC**

Append to `internal/domain/enrollment/enrollment_test.go`:

```go
func TestRejectEnrollment(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	matchUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name         string
		sheetUUID    uuid.UUID
		matchUUID    uuid.UUID
		masterUUID   uuid.UUID
		enrollMock   *testutil.MockEnrollmentRepo
		matchMock    *testutil.MockMatchRepo
		campaignMock *testutil.MockCampaignRepo
		wantErr      error
	}{
		{
			name:       "success from pending",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: nil,
		},
		{
			name:       "success from accepted",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "accepted", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: nil,
		},
		{
			name:       "idempotent when already rejected",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "rejected", nil
				},
			},
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      nil,
		},
		{
			name:       "enrollment not found",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "", enrollmentPg.ErrEnrollmentNotFound
				},
			},
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      enrollment.ErrEnrollmentNotFound,
		},
		{
			name:       "match not found",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, matchPg.ErrMatchNotFound
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      domainMatch.ErrMatchNotFound,
		},
		{
			name:       "campaign not found",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, campaignPg.ErrCampaignNotFound
				},
			},
			wantErr: domainCampaign.ErrCampaignNotFound,
		},
		{
			name:       "not campaign master",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: otherUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: enrollment.ErrNotMatchMaster,
		},
		{
			name:       "repo error on reject",
			sheetUUID:  sheetUUID,
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentStatusFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (string, error) {
					return "pending", nil
				},
				RejectEnrollmentFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) error {
					return errors.New("db error")
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := enrollment.NewRejectEnrollmentUC(tt.enrollMock, tt.matchMock, tt.campaignMock)
			err := uc.Reject(context.Background(), tt.sheetUUID, tt.matchUUID, tt.masterUUID)

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

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/enrollment/... -run TestRejectEnrollment -count=1`
Expected: FAIL — `NewRejectEnrollmentUC` undefined.

- [ ] **Step 3: Implement RejectEnrollmentUC**

Create `internal/domain/enrollment/reject_enrollment.go`:

```go
package enrollment

import (
	"context"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IRejectEnrollment interface {
	Reject(ctx context.Context, sheetUUID uuid.UUID, matchUUID uuid.UUID, masterUUID uuid.UUID) error
}

type RejectEnrollmentUC struct {
	repo         IRepository
	matchRepo    matchDomain.IRepository
	campaignRepo campaignDomain.IRepository
}

func NewRejectEnrollmentUC(
	repo IRepository,
	matchRepo matchDomain.IRepository,
	campaignRepo campaignDomain.IRepository,
) *RejectEnrollmentUC {
	return &RejectEnrollmentUC{
		repo:         repo,
		matchRepo:    matchRepo,
		campaignRepo: campaignRepo,
	}
}

func (uc *RejectEnrollmentUC) Reject(
	ctx context.Context,
	sheetUUID uuid.UUID,
	matchUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	status, err := uc.repo.GetEnrollmentStatus(ctx, sheetUUID, matchUUID)
	if err == enrollmentPg.ErrEnrollmentNotFound {
		return ErrEnrollmentNotFound
	}
	if err != nil {
		return err
	}
	if status == "rejected" {
		return nil
	}

	// TODO: check if match has already started (temporal guard)
	campaignUUID, err := uc.matchRepo.GetMatchCampaignUUID(ctx, matchUUID)
	if err == matchPg.ErrMatchNotFound {
		return matchDomain.ErrMatchNotFound
	}
	if err != nil {
		return err
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, campaignUUID)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if campaignMasterUUID != masterUUID {
		return ErrNotMatchMaster
	}

	return uc.repo.RejectEnrollment(ctx, sheetUUID, matchUUID)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/enrollment/... -count=1 -v`
Expected: ALL tests PASS (TestEnrollCharacterSheet, TestAcceptEnrollment, TestRejectEnrollment).

- [ ] **Step 5: Commit**

```bash
git add internal/domain/enrollment/reject_enrollment.go internal/domain/enrollment/enrollment_test.go
git commit -m "feat(enrollment): add RejectEnrollmentUC with tests

Implements reject enrollment use case with master ownership validation
and idempotent behavior. Table-driven tests cover all paths:
pending→rejected, accepted→rejected, idempotent, not found, not master.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 11: App — Reject Enrollment Handler (TDD)

**Files:**
- Create: `internal/app/api/enrollment/reject_enrollment.go`
- Create: `internal/app/api/enrollment/reject_enrollment_test.go`
- Modify: `internal/app/api/enrollment/mocks_test.go`

- [ ] **Step 1: Add the UC mock to mocks_test.go**

Append to `internal/app/api/enrollment/mocks_test.go`:

```go
type mockRejectEnrollment struct {
	fn func(ctx context.Context, sheetUUID, matchUUID, masterUUID uuid.UUID) error
}

func (m *mockRejectEnrollment) Reject(ctx context.Context, sheetUUID, matchUUID, masterUUID uuid.UUID) error {
	return m.fn(ctx, sheetUUID, matchUUID, masterUUID)
}
```

- [ ] **Step 2: Write the failing handler tests**

Create `internal/app/api/enrollment/reject_enrollment_test.go`:

```go
package enrollment_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/enrollment"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainEnrollment "github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestRejectEnrollmentHandler(t *testing.T) {
	masterUUID := uuid.New()
	sheetUUID := uuid.New()
	matchUUID := uuid.New()

	tests := []struct {
		name          string
		pathSheetUUID string
		pathMatchUUID string
		mockFn        func(ctx context.Context, sheetUUID, matchUUID, masterUUID uuid.UUID) error
		wantStatus    int
	}{
		{
			name:          "success",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:          "invalid sheet uuid",
			pathSheetUUID: "not-a-valid-uuid",
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:          "invalid match uuid",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: "not-a-valid-uuid",
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:          "enrollment not found",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return domainEnrollment.ErrEnrollmentNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:          "match not found",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:          "campaign not found",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return domainCampaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:          "not match master",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return domainEnrollment.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:          "generic error",
			pathSheetUUID: sheetUUID.String(),
			pathMatchUUID: matchUUID.String(),
			mockFn: func(ctx context.Context, sUUID, mUUID, masterUUID uuid.UUID) error {
				return errors.New("unexpected database error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockRejectEnrollment{fn: tt.mockFn}
			handler := enrollment.RejectEnrollmentHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodPost,
				Path:   "/enrollments/{sheet_uuid}/{match_uuid}/reject",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, masterUUID)
			resp := api.PostCtx(ctx, "/enrollments/"+tt.pathSheetUUID+"/"+tt.pathMatchUUID+"/reject")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/app/api/enrollment/... -run TestRejectEnrollmentHandler -count=1`
Expected: FAIL — `RejectEnrollmentHandler` undefined.

- [ ] **Step 4: Implement the handler**

Create `internal/app/api/enrollment/reject_enrollment.go`:

```go
package enrollment

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainEnrollment "github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type RejectEnrollmentRequest struct {
	SheetUUID string `path:"sheet_uuid" required:"true" doc:"enrolled character sheet UUID to reject"`
	MatchUUID string `path:"match_uuid" required:"true" doc:"match UUID"`
}

type RejectEnrollmentResponse struct {
	Status int `json:"status"`
}

func RejectEnrollmentHandler(
	uc domainEnrollment.IRejectEnrollment,
) func(context.Context, *RejectEnrollmentRequest) (*RejectEnrollmentResponse, error) {

	return func(ctx context.Context, req *RejectEnrollmentRequest) (*RejectEnrollmentResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		sheetUUID, err := uuid.Parse(req.SheetUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid sheet UUID")
		}

		matchUUID, err := uuid.Parse(req.MatchUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid match UUID")
		}

		err = uc.Reject(ctx, sheetUUID, matchUUID, masterUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainEnrollment.ErrEnrollmentNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainMatch.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainCampaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainEnrollment.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &RejectEnrollmentResponse{
			Status: http.StatusOK,
		}, nil
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/app/api/enrollment/... -count=1 -v`
Expected: ALL tests PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/api/enrollment/reject_enrollment.go internal/app/api/enrollment/reject_enrollment_test.go internal/app/api/enrollment/mocks_test.go
git commit -m "feat(enrollment): add reject enrollment handler with tests

POST /enrollments/{sheet_uuid}/{match_uuid}/reject
Maps domain errors to appropriate HTTP status codes.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 12: Wiring — Reject Enrollment (Routes + main.go)

**Files:**
- Modify: `internal/app/api/enrollment/routes.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Add the reject handler to routes.go**

In `internal/app/api/enrollment/routes.go`, add `RejectEnrollmentHandler` to the `Api` struct and register the route:

Update the `Api` struct:
```go
type Api struct {
	EnrollCharacterHandler  Handler[EnrollCharacterRequest, EnrollCharacterResponse]
	AcceptEnrollmentHandler Handler[AcceptEnrollmentRequest, AcceptEnrollmentResponse]
	RejectEnrollmentHandler Handler[RejectEnrollmentRequest, RejectEnrollmentResponse]
}
```

Add after the accept registration in `RegisterRoutes`:
```go
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/enrollments/{sheet_uuid}/{match_uuid}/reject",
		Description: "Reject a character sheet enrollment in a match",
		Tags:        []string{"enrollments"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusInternalServerError,
		},
	}, a.RejectEnrollmentHandler)
```

- [ ] **Step 2: Wire the UC in main.go**

In `cmd/api/main.go`, add the reject UC after `acceptEnrollmentUC`:

```go
	rejectEnrollmentUC := domainEnrollment.NewRejectEnrollmentUC(
		enrollmentRepo,
		matchRepo,
		campaignRepo,
	)
```

And update the `enrollmentApi` struct:
```go
	enrollmentApi := enrollmentHandler.Api{
		EnrollCharacterHandler:  enrollmentHandler.EnrollCharacterHandler(enrollCharacterSheetUC),
		AcceptEnrollmentHandler: enrollmentHandler.AcceptEnrollmentHandler(acceptEnrollmentUC),
		RejectEnrollmentHandler: enrollmentHandler.RejectEnrollmentHandler(rejectEnrollmentUC),
	}
```

- [ ] **Step 3: Verify build succeeds**

Run: `go build ./...`
Expected: Build succeeds with no errors.

- [ ] **Step 4: Run all enrollment tests**

Run: `go test ./internal/domain/enrollment/... ./internal/app/api/enrollment/... -count=1 -v`
Expected: ALL tests PASS.

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -count=1`
Expected: ALL tests PASS (no regressions). The known broken match test may fail — that is pre-existing.

- [ ] **Step 6: Commit**

```bash
git add internal/app/api/enrollment/routes.go cmd/api/main.go
git commit -m "feat(enrollment): wire reject enrollment endpoint

Registers POST /enrollments/{sheet_uuid}/{match_uuid}/reject
and wires RejectEnrollmentUC in main.go.

Completes both accept and reject enrollment flows end-to-end.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
