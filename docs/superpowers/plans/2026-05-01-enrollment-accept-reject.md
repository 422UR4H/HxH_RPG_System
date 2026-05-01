# Enrollment Accept/Reject Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enable the master to accept or reject player enrollment requests in matches, with bidirectional status transitions and idempotent behavior.

**Architecture:** Add a `status` column (`pending`/`accepted`/`rejected`) to the `enrollments` table. Two new use cases (`AcceptEnrollmentUC`, `RejectEnrollmentUC`) mirror the submission accept/reject pattern — each validates master ownership via enrollment → match → campaign chain before updating status. Gateway adds 3 new repo methods. App layer adds 2 new HTTP handlers following REST convention (`/enrollments/{uuid}/accept`, `/enrollments/{uuid}/reject`).

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

### Task 2: Gateway — Error Sentinel + Read Enrollment

**Files:**
- Create: `internal/gateway/pg/enrollment/error.go`
- Create: `internal/gateway/pg/enrollment/read_enrollment.go`

- [ ] **Step 1: Create the gateway error sentinel**

Create `internal/gateway/pg/enrollment/error.go`:

```go
package enrollment

import "errors"

var (
	ErrEnrollmentNotFound = errors.New("enrollment not found in database")
)
```

- [ ] **Step 2: Create the read enrollment query**

Create `internal/gateway/pg/enrollment/read_enrollment.go`:

```go
package enrollment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetEnrollmentByUUID(
	ctx context.Context,
	enrollmentUUID uuid.UUID,
) (string, uuid.UUID, error) {
	const query = `
		SELECT status, match_uuid
		FROM enrollments
		WHERE uuid = $1
	`
	var status string
	var matchUUID uuid.UUID
	err := r.q.QueryRow(ctx, query, enrollmentUUID).Scan(&status, &matchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", uuid.Nil, ErrEnrollmentNotFound
		}
		return "", uuid.Nil, fmt.Errorf("failed to get enrollment: %w", err)
	}
	return status, matchUUID, nil
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/gateway/pg/enrollment/error.go internal/gateway/pg/enrollment/read_enrollment.go
git commit -m "feat(enrollment): add gateway error sentinel and read enrollment query

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
	enrollmentUUID uuid.UUID,
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
		WHERE uuid = $1
	`
	result, err := tx.Exec(ctx, query, enrollmentUUID)
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

Replace `internal/domain/enrollment/error.go` entirely:

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

Replace `internal/domain/enrollment/i_repository.go` entirely:

```go
package enrollment

import (
	"context"

	"github.com/google/uuid"
)

type IRepository interface {
	EnrollCharacterSheet(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheet(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
	GetEnrollmentByUUID(ctx context.Context, enrollmentUUID uuid.UUID) (status string, matchUUID uuid.UUID, err error)
	AcceptEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error
	RejectEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error
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

Replace `internal/domain/testutil/mock_enrollment_repo.go` entirely:

```go
package testutil

import (
	"context"

	"github.com/google/uuid"
)

type MockEnrollmentRepo struct {
	EnrollCharacterSheetFn         func(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheetFn func(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
	GetEnrollmentByUUIDFn          func(ctx context.Context, enrollmentUUID uuid.UUID) (string, uuid.UUID, error)
	AcceptEnrollmentFn             func(ctx context.Context, enrollmentUUID uuid.UUID) error
	RejectEnrollmentFn             func(ctx context.Context, enrollmentUUID uuid.UUID) error
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

func (m *MockEnrollmentRepo) GetEnrollmentByUUID(ctx context.Context, enrollmentUUID uuid.UUID) (string, uuid.UUID, error) {
	if m.GetEnrollmentByUUIDFn != nil {
		return m.GetEnrollmentByUUIDFn(ctx, enrollmentUUID)
	}
	return "", uuid.Nil, nil
}

func (m *MockEnrollmentRepo) AcceptEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error {
	if m.AcceptEnrollmentFn != nil {
		return m.AcceptEnrollmentFn(ctx, enrollmentUUID)
	}
	return nil
}

func (m *MockEnrollmentRepo) RejectEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error {
	if m.RejectEnrollmentFn != nil {
		return m.RejectEnrollmentFn(ctx, enrollmentUUID)
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

Append the `TestAcceptEnrollment` function to `internal/domain/enrollment/enrollment_test.go`.
Also add these imports to the existing import block (if not already present):

```go
import (
	// ... existing imports ...
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
)
```

The test function:

```go
func TestAcceptEnrollment(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	enrollmentUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name         string
		enrollUUID   uuid.UUID
		masterUUID   uuid.UUID
		enrollMock   *testutil.MockEnrollmentRepo
		matchMock    *testutil.MockMatchRepo
		campaignMock *testutil.MockCampaignRepo
		wantErr      error
	}{
		{
			name:       "success from pending",
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "rejected", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "accepted", matchUUID, nil
				},
			},
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      nil,
		},
		{
			name:       "enrollment not found",
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "", uuid.Nil, enrollmentPg.ErrEnrollmentNotFound
				},
			},
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      enrollment.ErrEnrollmentNotFound,
		},
		{
			name:       "match not found",
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: otherUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
				},
				AcceptEnrollmentFn: func(ctx context.Context, id uuid.UUID) error {
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
			err := uc.Accept(context.Background(), tt.enrollUUID, tt.masterUUID)

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
	Accept(ctx context.Context, enrollmentUUID uuid.UUID, masterUUID uuid.UUID) error
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

	return uc.repo.AcceptEnrollment(ctx, enrollmentUUID)
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
	fn func(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error
}

func (m *mockAcceptEnrollment) Accept(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error {
	return m.fn(ctx, enrollmentUUID, masterUUID)
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
	enrollmentUUID := uuid.New()

	tests := []struct {
		name       string
		pathUUID   string
		mockFn     func(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error
		wantStatus int
	}{
		{
			name:     "success",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "invalid uuid in path",
			pathUUID: "not-a-valid-uuid",
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "enrollment not found",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return domainEnrollment.ErrEnrollmentNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "match not found",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "campaign not found",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return domainCampaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "not match master",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return domainEnrollment.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:     "generic error",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
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
				Path:   "/enrollments/{uuid}/accept",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, masterUUID)
			resp := api.PostCtx(ctx, "/enrollments/"+tt.pathUUID+"/accept")

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
	EnrollmentUUID string `path:"uuid" required:"true" doc:"enrollment UUID"`
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

		enrollmentUUID, err := uuid.Parse(req.EnrollmentUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid enrollment UUID")
		}

		err = uc.Accept(ctx, enrollmentUUID, masterUUID)
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

POST /enrollments/{uuid}/accept
Maps domain errors to appropriate HTTP status codes.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 8: Wiring — Accept Enrollment (Routes + main.go)

**Files:**
- Modify: `internal/app/api/enrollment/routes.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Add the accept handler to routes.go**

Replace `internal/app/api/enrollment/routes.go` entirely:

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
		Path:        "/enrollments/{uuid}/accept",
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

In `cmd/api/main.go`, find this block (around line 178-185):

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

Registers POST /enrollments/{uuid}/accept and wires
AcceptEnrollmentUC in main.go.

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
	enrollmentUUID uuid.UUID,
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
		WHERE uuid = $1
	`
	result, err := tx.Exec(ctx, query, enrollmentUUID)
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

Append the `TestRejectEnrollment` function to `internal/domain/enrollment/enrollment_test.go`:

```go
func TestRejectEnrollment(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	enrollmentUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name         string
		enrollUUID   uuid.UUID
		masterUUID   uuid.UUID
		enrollMock   *testutil.MockEnrollmentRepo
		matchMock    *testutil.MockMatchRepo
		campaignMock *testutil.MockCampaignRepo
		wantErr      error
	}{
		{
			name:       "success from pending",
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "accepted", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "rejected", matchUUID, nil
				},
			},
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      nil,
		},
		{
			name:       "enrollment not found",
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "", uuid.Nil, enrollmentPg.ErrEnrollmentNotFound
				},
			},
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      enrollment.ErrEnrollmentNotFound,
		},
		{
			name:       "match not found",
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: otherUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
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
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
				},
				RejectEnrollmentFn: func(ctx context.Context, id uuid.UUID) error {
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
			err := uc.Reject(context.Background(), tt.enrollUUID, tt.masterUUID)

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
	Reject(ctx context.Context, enrollmentUUID uuid.UUID, masterUUID uuid.UUID) error
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

	return uc.repo.RejectEnrollment(ctx, enrollmentUUID)
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
	fn func(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error
}

func (m *mockRejectEnrollment) Reject(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error {
	return m.fn(ctx, enrollmentUUID, masterUUID)
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
	enrollmentUUID := uuid.New()

	tests := []struct {
		name       string
		pathUUID   string
		mockFn     func(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error
		wantStatus int
	}{
		{
			name:     "success",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:     "invalid uuid in path",
			pathUUID: "not-a-valid-uuid",
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "enrollment not found",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return domainEnrollment.ErrEnrollmentNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "match not found",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return domainMatch.ErrMatchNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "campaign not found",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return domainCampaign.ErrCampaignNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "not match master",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
				return domainEnrollment.ErrNotMatchMaster
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:     "generic error",
			pathUUID: enrollmentUUID.String(),
			mockFn: func(ctx context.Context, eUUID, mUUID uuid.UUID) error {
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
				Path:   "/enrollments/{uuid}/reject",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, masterUUID)
			resp := api.PostCtx(ctx, "/enrollments/"+tt.pathUUID+"/reject")

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
	EnrollmentUUID string `path:"uuid" required:"true" doc:"enrollment UUID to reject"`
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

		enrollmentUUID, err := uuid.Parse(req.EnrollmentUUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid enrollment UUID")
		}

		err = uc.Reject(ctx, enrollmentUUID, masterUUID)
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

POST /enrollments/{uuid}/reject
Maps domain errors to appropriate HTTP status codes.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 12: Wiring — Reject Enrollment (Routes + main.go)

**Files:**
- Modify: `internal/app/api/enrollment/routes.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Add the reject handler to routes.go**

In `internal/app/api/enrollment/routes.go`, update the `Api` struct and add the reject route registration.

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
		Path:        "/enrollments/{uuid}/reject",
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

Registers POST /enrollments/{uuid}/reject and wires
RejectEnrollmentUC in main.go.

Completes both accept and reject enrollment flows end-to-end.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
