# Manage Character Sheet Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow the sheet owner to edit and delete their own character sheet from the detail page, via a compact "Gerenciar" button with an inline dropdown menu, gated by the sheet's submission state.

**Architecture:** Backend adds three endpoints (GET extension, PATCH full, DELETE) with strict isFree guards; frontend adds a ManageButton + SheetBottomActions that share layout with the existing campaign button, plus two new edit pages.

**Tech Stack:** Go + Huma + Chi + pgx (backend); React + TypeScript + styled-components + TanStack Query (frontend).

---

## File Map

### Backend — `System_X_System/`

| Action | File |
|--------|------|
| Modify | `internal/application/character_sheet/get_character_sheet.go` |
| Modify | `internal/application/character_sheet/get_character_sheet_test.go` |
| Modify | `internal/application/character_sheet/i_repository.go` |
| Modify | `internal/application/testutil/mock_character_sheet_repo.go` |
| Create | `internal/application/character_sheet/delete_character_sheet.go` |
| Create | `internal/application/character_sheet/delete_character_sheet_test.go` |
| Create | `internal/application/character_sheet/update_character_sheet.go` |
| Create | `internal/application/character_sheet/update_character_sheet_test.go` |
| Modify | `internal/gateway/pg/sheet/update_character_sheet_profile.go` |
| Modify | `internal/gateway/pg/submission/read_submission.go` |
| Create | `internal/gateway/pg/sheet/delete_character_sheet.go` |
| Create | `internal/gateway/pg/sheet/update_character_sheet.go` |
| Modify | `internal/app/api/sheet/get_character_sheet.go` |
| Modify | `internal/app/api/sheet/character_sheet_response.go` |
| Modify | `internal/app/api/sheet/patch_character_sheet_profile.go` |
| Create | `internal/app/api/sheet/delete_character_sheet.go` |
| Create | `internal/app/api/sheet/update_character_sheet.go` |
| Modify | `internal/app/api/sheet/routes.go` |
| Modify | `cmd/api/main.go` |
| Modify | `docs/dev/api/character-sheet.md` |

### Frontend — `System_X_System_React/`

| Action | File |
|--------|------|
| Modify | `src/types/characterSheet.ts` |
| Modify | `src/services/characterSheetsService.ts` |
| Create | `src/hooks/useDeleteCharacterSheet.ts` |
| Create | `src/hooks/useUpdateCharacterSheet.ts` |
| Create | `src/features/sheet/ManageButton.tsx` |
| Create | `src/features/sheet/SheetBottomActions.tsx` |
| Modify | `src/features/sheet/CharacterSheetTemplate.tsx` |
| Modify | `src/pages/CharacterSheetPage.tsx` |
| Create | `src/pages/EditCharacterSheetPage.tsx` |
| Create | `src/pages/EditCharacterSheetProfilePage.tsx` |
| Modify | `src/App.tsx` |

---

## Task 1: Fix ISubmissionLookup + test mock

The interface `ISubmissionLookup` in `get_character_sheet.go` had `ExistsSubmittedCharacterSheet` added to it in a previous session, but the GET use case never calls that method — it only calls `GetSubmissionCampaignUUIDBySheetUUID`. Remove the extra method from the interface; it will live in a focused interface defined by the delete/update use cases.

Also add `SubmissionInfo` and `ISubmissionFetcher` (for the GET handler's optional include).

**Files:**
- Modify: `internal/application/character_sheet/get_character_sheet.go`

- [ ] **Step 1: Update `ISubmissionLookup` and add `ISubmissionFetcher` + `SubmissionInfo`**

Replace the two-method `ISubmissionLookup` block with:

```go
// ISubmissionLookup lets the GET use case check pending submissions for authorization.
type ISubmissionLookup interface {
	GetSubmissionCampaignUUIDBySheetUUID(ctx context.Context, sheetUUID uuid.UUID) (uuid.UUID, error)
}

// SubmissionInfo is returned by ISubmissionFetcher for the optional ?include=submission.
type SubmissionInfo struct {
	CampaignUUID uuid.UUID
	CreatedAt    time.Time
}

// ISubmissionFetcher is satisfied by the submission gateway; used by the HTTP handler only.
type ISubmissionFetcher interface {
	GetSubmissionInfoBySheetUUID(ctx context.Context, sheetUUID uuid.UUID) (*SubmissionInfo, error)
}
```

Add `"time"` to imports in that file.

- [ ] **Step 2: Verify tests compile and pass**

```bash
cd System_X_System && go test ./internal/application/character_sheet/...
```

Expected: all tests pass (the local `mockSubmissionLookup` in the test file already only implements `GetSubmissionCampaignUUIDBySheetUUID`).

- [ ] **Step 3: Commit**

```bash
git add internal/application/character_sheet/get_character_sheet.go
git commit -m "refactor(sheet): slim ISubmissionLookup, add ISubmissionFetcher+SubmissionInfo"
```

---

## Task 2: Add GetSubmissionInfoBySheetUUID to submission gateway

**Files:**
- Modify: `internal/gateway/pg/submission/read_submission.go`

- [ ] **Step 1: Add method to Repository**

Append to `read_submission.go`:

```go
func (r *Repository) GetSubmissionInfoBySheetUUID(
	ctx context.Context, sheetUUID uuid.UUID,
) (*charactersheet.SubmissionInfo, error) {
	const query = `
		SELECT campaign_uuid, created_at
		FROM submissions
		WHERE character_sheet_uuid = $1
	`
	var info charactersheet.SubmissionInfo
	err := r.q.QueryRow(ctx, query, sheetUUID).Scan(&info.CampaignUUID, &info.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get submission info: %w", err)
	}
	return &info, nil
}
```

Add import for `charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"` to the file.

- [ ] **Step 2: Verify compile**

```bash
go build ./internal/gateway/pg/submission/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/gateway/pg/submission/read_submission.go
git commit -m "feat(submission/gateway): add GetSubmissionInfoBySheetUUID"
```

---

## Task 3: Extend GET handler to support `?include=submission`

**Files:**
- Modify: `internal/app/api/sheet/get_character_sheet.go`
- Modify: `internal/app/api/sheet/character_sheet_response.go`

- [ ] **Step 1: Add `SubmissionResponse` to `character_sheet_response.go`**

Add at the bottom of the file, before the closing of the package:

```go
type SubmissionResponse struct {
	CampaignUUID string `json:"campaign_uuid"`
	CreatedAt    string `json:"created_at"`
}
```

- [ ] **Step 2: Add `Submission` field to `CharacterSheetResponse`**

In `character_sheet_response.go`, add the field to `CharacterSheetResponse`:

```go
Submission *SubmissionResponse `json:"submission,omitempty"`
```

Place it after `CampaignUUID`.

- [ ] **Step 3: Update `get_character_sheet.go`**

Replace the entire file content with:

```go
package sheet

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetCharacterSheetRequest struct {
	UUID    string `path:"uuid" required:"true" doc:"UUID of the character sheet"`
	Include string `query:"include" required:"false" doc:"Comma-separated list of includes: submission"`
}

type GetCharacterSheetResponseBody struct {
	CharacterSheet CharacterSheetResponse `json:"character_sheet"`
}

type GetCharacterSheetResponse struct {
	Body   GetCharacterSheetResponseBody `json:"body"`
	Status int                           `json:"status"`
}

func GetCharacterSheetHandler(
	uc cs.IGetCharacterSheet,
	submissionFetcher cs.ISubmissionFetcher,
) func(context.Context, *GetCharacterSheetRequest) (*GetCharacterSheetResponse, error) {

	return func(ctx context.Context, req *GetCharacterSheetRequest) (*GetCharacterSheetResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		charSheetId, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		characterSheet, err := uc.GetCharacterSheet(ctx, charSheetId, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, cs.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		response := NewCharacterSheetResponse(characterSheet)

		if req.Include == "submission" || containsInclude(req.Include, "submission") {
			info, err := submissionFetcher.GetSubmissionInfoBySheetUUID(ctx, charSheetId)
			if err != nil {
				return nil, huma.Error500InternalServerError(err.Error())
			}
			if info != nil {
				response.Submission = &SubmissionResponse{
					CampaignUUID: info.CampaignUUID.String(),
					CreatedAt:    info.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
				}
			}
		}

		return &GetCharacterSheetResponse{
			Body: GetCharacterSheetResponseBody{
				CharacterSheet: *response,
			},
			Status: http.StatusOK,
		}, nil
	}
}

// containsInclude checks if a comma-separated include string contains the target.
func containsInclude(include, target string) bool {
	if include == "" {
		return false
	}
	for _, s := range splitIncludes(include) {
		if s == target {
			return true
		}
	}
	return false
}

func splitIncludes(include string) []string {
	result := []string{}
	start := 0
	for i := 0; i <= len(include); i++ {
		if i == len(include) || include[i] == ',' {
			part := trimSpace(include[start:i])
			if part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
```

- [ ] **Step 4: Update `main.go` — inject `submitRepo` into `GetCharacterSheetHandler`**

Find the `characterSheetsApi` struct literal in `cmd/api/main.go` and update:

```go
GetCharacterSheetHandler: sheetHandler.GetCharacterSheetHandler(getCharacterSheetUC, submitRepo),
```

- [ ] **Step 5: Verify compile**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/app/api/sheet/get_character_sheet.go \
        internal/app/api/sheet/character_sheet_response.go \
        cmd/api/main.go
git commit -m "feat(sheet/api): support ?include=submission on GET /charactersheets/{uuid}"
```

---

## Task 4: Extend `IRepository` and mock with Delete + Update methods

**Files:**
- Modify: `internal/application/character_sheet/i_repository.go`
- Modify: `internal/application/testutil/mock_character_sheet_repo.go`

- [ ] **Step 1: Add methods to `IRepository`**

In `i_repository.go`, add to the interface:

```go
DeleteCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, playerUUID uuid.UUID) error
UpdateCharacterSheet(ctx context.Context, sheet *sheet.CharacterSheet) error
```

- [ ] **Step 2: Add stub methods to mock**

In `mock_character_sheet_repo.go`, add the two new fields and methods:

```go
DeleteCharacterSheetFn func(ctx context.Context, sheetUUID uuid.UUID, playerUUID uuid.UUID) error
UpdateCharacterSheetFn func(ctx context.Context, sheet *sheet.CharacterSheet) error
```

```go
func (m *MockCharacterSheetRepo) DeleteCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, playerUUID uuid.UUID) error {
	if m.DeleteCharacterSheetFn != nil {
		return m.DeleteCharacterSheetFn(ctx, sheetUUID, playerUUID)
	}
	return nil
}

func (m *MockCharacterSheetRepo) UpdateCharacterSheet(ctx context.Context, s *sheet.CharacterSheet) error {
	if m.UpdateCharacterSheetFn != nil {
		return m.UpdateCharacterSheetFn(ctx, s)
	}
	return nil
}
```

- [ ] **Step 3: Verify compile**

```bash
go build ./internal/application/...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/application/character_sheet/i_repository.go \
        internal/application/testutil/mock_character_sheet_repo.go
git commit -m "feat(sheet): add DeleteCharacterSheet and UpdateCharacterSheet to IRepository + mock"
```

---

## Task 5: Implement `DeleteCharacterSheetUC`

**Files:**
- Create: `internal/application/character_sheet/delete_character_sheet.go`
- Create: `internal/application/character_sheet/delete_character_sheet_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/application/character_sheet/delete_character_sheet_test.go`:

```go
package charactersheet_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/google/uuid"
)

type mockFreeStateChecker struct {
	exists bool
	err    error
}

func (m *mockFreeStateChecker) ExistsSubmittedCharacterSheet(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.exists, m.err
}

func TestDeleteCharacterSheet(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - free sheet deleted", func(t *testing.T) {
		playerUUID := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
			DeleteCharacterSheetFn: func(ctx context.Context, sUUID uuid.UUID, pUUID uuid.UUID) error {
				return nil
			},
		}
		mockChecker := &mockFreeStateChecker{exists: false, err: nil}

		uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
		err := uc.DeleteCharacterSheet(ctx, sheetUUID, playerUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("error - not owner", func(t *testing.T) {
		playerUUID := uuid.New()
		otherUser := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
		}
		mockChecker := &mockFreeStateChecker{}

		uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
		err := uc.DeleteCharacterSheet(ctx, sheetUUID, otherUser)
		if !errors.Is(err, auth.ErrInsufficientPermissions) {
			t.Fatalf("expected ErrInsufficientPermissions, got: %v", err)
		}
	})

	t.Run("error - sheet has submission (not free)", func(t *testing.T) {
		playerUUID := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
		}
		mockChecker := &mockFreeStateChecker{exists: true, err: nil}

		uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
		err := uc.DeleteCharacterSheet(ctx, sheetUUID, playerUUID)
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFreeToManage) {
			t.Fatalf("expected ErrCharacterSheetNotFreeToManage, got: %v", err)
		}
	})

	t.Run("error - sheet not found", func(t *testing.T) {
		sheetUUID := uuid.New()
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, charactersheet.ErrCharacterSheetNotFound
			},
		}
		mockChecker := &mockFreeStateChecker{}

		uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
		err := uc.DeleteCharacterSheet(ctx, sheetUUID, uuid.New())
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got: %v", err)
		}
	})
}
```

- [ ] **Step 2: Run test — verify it fails**

```bash
go test ./internal/application/character_sheet/... -run TestDeleteCharacterSheet -v
```

Expected: compile error — `charactersheet.NewDeleteCharacterSheetUC` not found.

- [ ] **Step 3: Implement `delete_character_sheet.go`**

Create `internal/application/character_sheet/delete_character_sheet.go`:

```go
package charactersheet

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/google/uuid"
)

// IFreeStateChecker checks whether a sheet has a pending submission.
type IFreeStateChecker interface {
	ExistsSubmittedCharacterSheet(ctx context.Context, sheetUUID uuid.UUID) (bool, error)
}

type IDeleteCharacterSheet interface {
	DeleteCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID) error
}

type DeleteCharacterSheetUC struct {
	repo    IRepository
	checker IFreeStateChecker
}

func NewDeleteCharacterSheetUC(repo IRepository, checker IFreeStateChecker) *DeleteCharacterSheetUC {
	return &DeleteCharacterSheetUC{repo: repo, checker: checker}
}

func (uc *DeleteCharacterSheetUC) DeleteCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID,
) error {
	playerUUID, err := uc.repo.GetCharacterSheetPlayerUUID(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if playerUUID != userUUID {
		return auth.ErrInsufficientPermissions
	}

	hasSubmission, err := uc.checker.ExistsSubmittedCharacterSheet(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if hasSubmission {
		return ErrCharacterSheetNotFreeToManage
	}

	return uc.repo.DeleteCharacterSheet(ctx, sheetUUID, userUUID)
}
```

Note: The isFree guard checks `ExistsSubmittedCharacterSheet`. When a sheet has `campaign_uuid` set (accepted), there's no submission row — but the UC receives `GetCharacterSheetByUUID` indirectly via `GetCharacterSheetPlayerUUID`. To cover the `campaign_uuid != null` case, the gateway implementation of `DeleteCharacterSheet` should also check `campaign_uuid IS NULL` in the WHERE clause and return `ErrCharacterSheetNotFound` (which the handler maps to 422 via a different error, or we add a separate guard). The cleanest approach: read `GetCharacterSheetRelationshipUUIDs` in the UC to check `campaignUUID == nil`.

Update the UC to add the campaign check:

```go
func (uc *DeleteCharacterSheetUC) DeleteCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID,
) error {
	playerUUID, err := uc.repo.GetCharacterSheetPlayerUUID(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if playerUUID != userUUID {
		return auth.ErrInsufficientPermissions
	}

	rel, err := uc.repo.GetCharacterSheetRelationshipUUIDs(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if rel.CampaignUUID != nil {
		return ErrCharacterSheetNotFreeToManage
	}

	hasSubmission, err := uc.checker.ExistsSubmittedCharacterSheet(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if hasSubmission {
		return ErrCharacterSheetNotFreeToManage
	}

	return uc.repo.DeleteCharacterSheet(ctx, sheetUUID, userUUID)
}
```

Check `csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"` — look at `RelationshipUUIDs` to confirm the `CampaignUUID` field name:

```bash
grep -n "CampaignUUID\|RelationshipUUIDs" /path/to/System_X_System/internal/domain/entity/character_sheet/*.go | head -10
```

Use whatever field name is correct.

- [ ] **Step 4: Add `campaign_uuid` check to the test**

Add a test case in `delete_character_sheet_test.go`:

```go
t.Run("error - sheet has campaign (not free)", func(t *testing.T) {
	playerUUID := uuid.New()
	campaignUUID := uuid.New()
	sheetUUID := uuid.New()

	mockRepo := &testutil.MockCharacterSheetRepo{
		GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
			return playerUUID, nil
		},
		GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
			return csEntity.RelationshipUUIDs{CampaignUUID: &campaignUUID}, nil
		},
	}
	mockChecker := &mockFreeStateChecker{}

	uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
	err := uc.DeleteCharacterSheet(ctx, sheetUUID, playerUUID)
	if !errors.Is(err, charactersheet.ErrCharacterSheetNotFreeToManage) {
		t.Fatalf("expected ErrCharacterSheetNotFreeToManage, got: %v", err)
	}
})
```

Add `csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"` import.

- [ ] **Step 5: Run tests — verify they pass**

```bash
go test ./internal/application/character_sheet/... -run TestDeleteCharacterSheet -v
```

Expected: all 5 cases PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/application/character_sheet/delete_character_sheet.go \
        internal/application/character_sheet/delete_character_sheet_test.go
git commit -m "feat(sheet): implement DeleteCharacterSheetUC with isFree guard"
```

---

## Task 6: Gateway — `DeleteCharacterSheet`

**Files:**
- Create: `internal/gateway/pg/sheet/delete_character_sheet.go`

- [ ] **Step 1: Create the file**

```go
package sheet

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) DeleteCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, playerUUID uuid.UUID,
) error {
	const query = `
		DELETE FROM character_sheets
		WHERE uuid = $1 AND player_uuid = $2
	`
	tag, err := r.q.Exec(ctx, query, sheetUUID, playerUUID)
	if err != nil {
		return fmt.Errorf("failed to delete character sheet: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCharacterSheetNotFound
	}
	return nil
}
```

Note: `ON DELETE CASCADE` is expected on `character_profiles`, `proficiencies`, and `joint_proficiencies`. Verify with:

```bash
grep -r "REFERENCES character_sheets" System_X_System/migrations/ | grep -i cascade
```

If there's no CASCADE, add explicit DELETEs in a transaction instead.

- [ ] **Step 2: Verify compile**

```bash
go build ./internal/gateway/pg/sheet/...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/gateway/pg/sheet/delete_character_sheet.go
git commit -m "feat(sheet/gateway): implement DeleteCharacterSheet"
```

---

## Task 7: HTTP handler — `DELETE /charactersheets/{uuid}`

**Files:**
- Create: `internal/app/api/sheet/delete_character_sheet.go`
- Modify: `internal/app/api/sheet/routes.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Create handler file**

```go
package sheet

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type DeleteCharacterSheetRequest struct {
	UUID string `path:"uuid" required:"true"`
}

type DeleteCharacterSheetResponse struct {
	Status int
}

func DeleteCharacterSheetHandler(
	uc cs.IDeleteCharacterSheet,
) func(context.Context, *DeleteCharacterSheetRequest) (*DeleteCharacterSheetResponse, error) {
	return func(ctx context.Context, req *DeleteCharacterSheetRequest) (*DeleteCharacterSheetResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		sheetUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid uuid")
		}

		err = uc.DeleteCharacterSheet(ctx, sheetUUID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, cs.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, cs.ErrCharacterSheetNotFreeToManage):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		return &DeleteCharacterSheetResponse{Status: http.StatusNoContent}, nil
	}
}
```

- [ ] **Step 2: Register in `routes.go`**

Add to the `Api` struct:

```go
DeleteCharacterSheetHandler Handler[DeleteCharacterSheetRequest, DeleteCharacterSheetResponse]
```

Add to `RegisterRoutes`:

```go
huma.Register(api, huma.Operation{
    Method:      http.MethodDelete,
    Path:        "/charactersheets/{uuid}",
    Description: "Delete a character sheet (owner only, free state)",
    Tags:        []string{"character_sheets"},
    Errors: []int{
        http.StatusBadRequest,
        http.StatusUnauthorized,
        http.StatusNotFound,
        http.StatusForbidden,
        http.StatusUnprocessableEntity,
        http.StatusInternalServerError,
    },
    DefaultStatus: http.StatusNoContent,
}, a.DeleteCharacterSheetHandler)
```

- [ ] **Step 3: Wire in `main.go`**

After `createCharacterSheetUC`:

```go
deleteCharacterSheetUC := cs.NewDeleteCharacterSheetUC(characterSheetRepo, submitRepo)
```

Add to `characterSheetsApi`:

```go
DeleteCharacterSheetHandler: sheetHandler.DeleteCharacterSheetHandler(deleteCharacterSheetUC),
```

- [ ] **Step 4: Verify compile + run all tests**

```bash
go build ./... && go test ./internal/application/character_sheet/...
```

Expected: build succeeds, all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/app/api/sheet/delete_character_sheet.go \
        internal/app/api/sheet/routes.go \
        cmd/api/main.go
git commit -m "feat(sheet/api): DELETE /charactersheets/{uuid}"
```

---

## Task 8: Implement `UpdateCharacterSheetUC`

This use case rebuilds the sheet via factory, then delegates the full DB update to the gateway.

**Files:**
- Create: `internal/application/character_sheet/update_character_sheet.go`
- Create: `internal/application/character_sheet/update_character_sheet_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/application/character_sheet/update_character_sheet_test.go`:

```go
package charactersheet_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/google/uuid"
)

func TestUpdateCharacterSheet(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - free sheet updated", func(t *testing.T) {
		playerUUID := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{CampaignUUID: nil}, nil
			},
			UpdateCharacterSheetFn: func(ctx context.Context, s interface{ GetUUID() uuid.UUID }) error {
				return nil
			},
		}
		mockChecker := &mockFreeStateChecker{exists: false}
		classMap := newTestClassMap()
		factory := newTestFactory()

		uc := charactersheet.NewUpdateCharacterSheetUC(classMap, factory, mockRepo, mockChecker)
		input := newValidCreateInput()
		input.PlayerUUID = &playerUUID

		err := uc.UpdateCharacterSheet(ctx, sheetUUID, playerUUID, input)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("error - not owner", func(t *testing.T) {
		playerUUID := uuid.New()
		otherUser := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
		}
		mockChecker := &mockFreeStateChecker{}
		classMap := newTestClassMap()
		factory := newTestFactory()

		uc := charactersheet.NewUpdateCharacterSheetUC(classMap, factory, mockRepo, mockChecker)
		err := uc.UpdateCharacterSheet(ctx, sheetUUID, otherUser, newValidCreateInput())
		if !errors.Is(err, auth.ErrInsufficientPermissions) {
			t.Fatalf("expected ErrInsufficientPermissions, got: %v", err)
		}
	})

	t.Run("error - sheet not free (has submission)", func(t *testing.T) {
		playerUUID := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{CampaignUUID: nil}, nil
			},
		}
		mockChecker := &mockFreeStateChecker{exists: true}
		classMap := newTestClassMap()
		factory := newTestFactory()

		uc := charactersheet.NewUpdateCharacterSheetUC(classMap, factory, mockRepo, mockChecker)
		input := newValidCreateInput()
		input.PlayerUUID = &playerUUID
		err := uc.UpdateCharacterSheet(ctx, sheetUUID, playerUUID, input)
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFreeToManage) {
			t.Fatalf("expected ErrCharacterSheetNotFreeToManage, got: %v", err)
		}
	})
}
```

Note: `UpdateCharacterSheetFn` in `MockCharacterSheetRepo` takes `*sheet.CharacterSheet` — fix the lambda above to match the actual type in the mock:

```go
UpdateCharacterSheetFn: func(ctx context.Context, s *sheet.CharacterSheet) error {
    return nil
},
```

- [ ] **Step 2: Run test — verify compile error**

```bash
go test ./internal/application/character_sheet/... -run TestUpdateCharacterSheet -v
```

Expected: compile error — `NewUpdateCharacterSheetUC` not found.

- [ ] **Step 3: Implement `update_character_sheet.go`**

```go
package charactersheet

import (
	"context"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/google/uuid"
)

type IUpdateCharacterSheet interface {
	UpdateCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID, input *CreateCharacterSheetInput) error
}

type UpdateCharacterSheetUC struct {
	characterClasses *sync.Map
	factory          *sheet.CharacterSheetFactory
	repo             IRepository
	checker          IFreeStateChecker
}

func NewUpdateCharacterSheetUC(
	charClasses *sync.Map,
	factory *sheet.CharacterSheetFactory,
	repo IRepository,
	checker IFreeStateChecker,
) *UpdateCharacterSheetUC {
	return &UpdateCharacterSheetUC{
		characterClasses: charClasses,
		factory:          factory,
		repo:             repo,
		checker:          checker,
	}
}

func (uc *UpdateCharacterSheetUC) UpdateCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID, input *CreateCharacterSheetInput,
) error {
	playerUUID, err := uc.repo.GetCharacterSheetPlayerUUID(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if playerUUID != userUUID {
		return auth.ErrInsufficientPermissions
	}

	rel, err := uc.repo.GetCharacterSheetRelationshipUUIDs(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if rel.CampaignUUID != nil {
		return ErrCharacterSheetNotFreeToManage
	}

	hasSubmission, err := uc.checker.ExistsSubmittedCharacterSheet(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if hasSubmission {
		return ErrCharacterSheetNotFreeToManage
	}

	class, exists := uc.characterClasses.Load(input.CharacterClass)
	if !exists {
		return NewCharacterClassNotFoundError(input.CharacterClass.String())
	}
	charClass := class.(cc.CharacterClass)

	if err := charClass.ValidateSkills(input.SkillsExps); err != nil {
		return err
	}
	if err := charClass.ValidateProficiencies(input.ProficienciesExps); err != nil {
		return err
	}
	charClass.ApplySkills(input.SkillsExps)
	charClass.ApplyProficiencies(input.ProficienciesExps)

	pUUID := userUUID
	charSheet, err := uc.factory.Build(&pUUID, nil, nil, input.Profile, nil, nil, &charClass)
	if err != nil {
		return err
	}
	if len(input.AttributePoints) > 0 {
		if err := charSheet.ApplyInitialAttributePoints(input.AttributePoints); err != nil {
			return err
		}
	}
	charSheet.UUID = sheetUUID

	return uc.repo.UpdateCharacterSheet(ctx, charSheet)
}
```

- [ ] **Step 4: Run tests — verify all pass**

```bash
go test ./internal/application/character_sheet/... -run TestUpdateCharacterSheet -v
```

Expected: all 3 cases PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/application/character_sheet/update_character_sheet.go \
        internal/application/character_sheet/update_character_sheet_test.go
git commit -m "feat(sheet): implement UpdateCharacterSheetUC with isFree guard + factory rebuild"
```

---

## Task 9: Gateway — `UpdateCharacterSheet`

This mirrors the CREATE transaction but uses UPDATE instead of INSERT. The `charSheetToModel` helper is reused.

**Files:**
- Create: `internal/gateway/pg/sheet/update_character_sheet.go`

- [ ] **Step 1: Create the file**

```go
package sheet

import (
	"context"
	"fmt"
	"time"

	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

func (r *Repository) UpdateCharacterSheet(
	ctx context.Context, sheet *domainSheet.CharacterSheet,
) error {
	m := charSheetToModel(sheet)
	now := time.Now()

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

	const sheetQuery = `
		UPDATE character_sheets SET
			category_name=$1, curr_hex_value=$2, talent_exp=$3,
			level=$4, points=$5, talent_lvl=$6, physicals_lvl=$7, mentals_lvl=$8, spirituals_lvl=$9, skills_lvl=$10,
			health_min_pts=$11, health_curr_pts=$12, health_max_pts=$13,
			stamina_min_pts=$14, stamina_curr_pts=$15, stamina_max_pts=$16,
			aura_min_pts=$17, aura_curr_pts=$18, aura_max_pts=$19,
			resistance_pts=$20, strength_pts=$21, agility_pts=$22, celerity_pts=$23, flexibility_pts=$24,
			dexterity_pts=$25, sense_pts=$26, constitution_pts=$27,
			resilience_pts=$28, adaptability_pts=$29, weighting_pts=$30, creativity_pts=$31,
			resilience_exp=$32, adaptability_exp=$33, weighting_exp=$34, creativity_exp=$35,
			vitality_exp=$36, energy_exp=$37, defense_exp=$38, push_exp=$39, grab_exp=$40,
			carry_exp=$41, velocity_exp=$42, accelerate_exp=$43, brake_exp=$44,
			legerity_exp=$45, repel_exp=$46, feint_exp=$47, acrobatics_exp=$48, evasion_exp=$49,
			sneak_exp=$50, reflex_exp=$51, accuracy_exp=$52, stealth_exp=$53,
			vision_exp=$54, hearing_exp=$55, smell_exp=$56, tact_exp=$57, taste_exp=$58,
			heal_exp=$59, breath_exp=$60, tenacity_exp=$61,
			nen_exp=$62, focus_exp=$63, will_power_exp=$64,
			ten_exp=$65, zetsu_exp=$66, ren_exp=$67, gyo_exp=$68, shu_exp=$69, kou_exp=$70,
			ken_exp=$71, ryu_exp=$72, in_exp=$73, en_exp=$74, aura_control_exp=$75, aop_exp=$76,
			reinforcement_exp=$77, transmutation_exp=$78, materialization_exp=$79,
			specialization_exp=$80, manipulation_exp=$81, emission_exp=$82,
			updated_at=$83
		WHERE uuid=$84 AND player_uuid=$85
	`
	tag, err := tx.Exec(ctx, sheetQuery,
		m.CategoryName, m.CurrHexValue, m.TalentExp,
		m.Level, m.Points, m.TalentLvl, m.PhysicalsLvl, m.MentalsLvl, m.SpiritualsLvl, m.SkillsLvl,
		m.Health.Min, m.Health.Curr, m.Health.Max,
		m.Stamina.Min, m.Stamina.Curr, m.Stamina.Max,
		m.Aura.Min, m.Aura.Curr, m.Aura.Max,
		m.ResistancePts, m.StrengthPts, m.AgilityPts, m.CelerityPts, m.FlexibilityPts,
		m.DexterityPts, m.SensePts, m.ConstitutionPts,
		m.ResiliencePts, m.AdaptabilityPts, m.WeightingPts, m.CreativityPts,
		m.ResilienceExp, m.AdaptabilityExp, m.WeightingExp, m.CreativityExp,
		m.VitalityExp, m.EnergyExp, m.DefenseExp, m.PushExp, m.GrabExp,
		m.CarryExp, m.VelocityExp, m.AccelerateExp, m.BrakeExp,
		m.LegerityExp, m.RepelExp, m.FeintExp, m.AcrobaticsExp, m.EvasionExp,
		m.SneakExp, m.ReflexExp, m.AccuracyExp, m.StealthExp,
		m.VisionExp, m.HearingExp, m.SmellExp, m.TactExp, m.TasteExp,
		m.HealExp, m.BreathExp, m.TenacityExp,
		m.NenExp, m.FocusExp, m.WillPowerExp,
		m.TenExp, m.ZetsuExp, m.RenExp, m.GyoExp, m.ShuExp, m.KouExp,
		m.KenExp, m.RyuExp, m.InExp, m.EnExp, m.AuraControlExp, m.AopExp,
		m.ReinforcementExp, m.TransmutationExp, m.MaterializationExp,
		m.SpecializationExp, m.ManipulationExp, m.EmissionExp,
		now,
		m.UUID, m.PlayerUUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update character sheet: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCharacterSheetNotFound
	}

	const profileQuery = `
		UPDATE character_profiles SET
			nickname=$1, fullname=$2, alignment=$3, character_class=$4,
			long_description=$5, brief_description=$6, birthday=$7, age=$8,
			updated_at=$9
		WHERE character_sheet_uuid=$10
	`
	_, err = tx.Exec(ctx, profileQuery,
		m.Profile.NickName, m.Profile.FullName, m.Profile.Alignment, m.Profile.CharacterClass,
		m.Profile.Description, m.Profile.BriefDescription, m.Profile.Birthday, m.Profile.Age,
		now,
		m.UUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update character profile: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM proficiencies WHERE character_sheet_uuid=$1`, m.UUID)
	if err != nil {
		return fmt.Errorf("failed to delete proficiencies: %w", err)
	}

	const profInsert = `INSERT INTO proficiencies (character_sheet_uuid, weapon, exp) VALUES ($1, $2, $3)`
	for _, p := range m.Proficiencies {
		if _, err = tx.Exec(ctx, profInsert, m.UUID, p.Weapon, p.Exp); err != nil {
			return fmt.Errorf("failed to insert proficiency %s: %w", p.Weapon, err)
		}
	}

	_, err = tx.Exec(ctx, `DELETE FROM joint_proficiencies WHERE character_sheet_uuid=$1`, m.UUID)
	if err != nil {
		return fmt.Errorf("failed to delete joint proficiencies: %w", err)
	}

	const jointInsert = `INSERT INTO joint_proficiencies (character_sheet_uuid, name, weapons, exp) VALUES ($1, $2, $3, $4)`
	for _, jp := range m.JointProficiencies {
		if _, err = tx.Exec(ctx, jointInsert, m.UUID, jp.Name, jp.Weapons, jp.Exp); err != nil {
			return fmt.Errorf("failed to insert joint proficiency %s: %w", jp.Name, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit update transaction: %w", err)
	}
	return nil
}
```

Note: verify the model field names `NenExp`, `FocusExp`, `WillPowerExp`, `AuraControlExp`, `AopExp` exist in `internal/gateway/pg/model/character_sheet.go`. If any differ, adjust the field references.

- [ ] **Step 2: Verify compile**

```bash
go build ./internal/gateway/pg/sheet/...
```

Expected: no errors. If any model field names are wrong, fix them and rerun.

- [ ] **Step 3: Commit**

```bash
git add internal/gateway/pg/sheet/update_character_sheet.go
git commit -m "feat(sheet/gateway): implement UpdateCharacterSheet in transaction"
```

---

## Task 10: HTTP handler — `PATCH /charactersheets/{uuid}` (full edit)

**Files:**
- Create: `internal/app/api/sheet/update_character_sheet.go`
- Modify: `internal/app/api/sheet/routes.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Create handler**

```go
package sheet

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type UpdateCharacterSheetRequest struct {
	UUID string                           `path:"uuid" required:"true"`
	Body CreateCharacterSheetRequestBody
}

type UpdateCharacterSheetResponseBody struct {
	CharacterSheet CharacterSheetResponse `json:"character_sheet"`
}

type UpdateCharacterSheetResponse struct {
	Body   UpdateCharacterSheetResponseBody `json:"body"`
	Status int                              `json:"status"`
}

func UpdateCharacterSheetHandler(
	uc cs.IUpdateCharacterSheet,
	getUC cs.IGetCharacterSheet,
) func(context.Context, *UpdateCharacterSheetRequest) (*UpdateCharacterSheetResponse, error) {
	return func(ctx context.Context, req *UpdateCharacterSheetRequest) (*UpdateCharacterSheetResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		sheetUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid uuid")
		}

		input, err := castRequest(&req.Body)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		if err := uc.UpdateCharacterSheet(ctx, sheetUUID, userUUID, input); err != nil {
			switch {
			case errors.Is(err, cs.ErrCharacterSheetNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, cs.ErrCharacterClassNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, cs.ErrCharacterSheetNotFreeToManage):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domain.ErrValidation):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		charSheet, err := getUC.GetCharacterSheet(ctx, sheetUUID, userUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		return &UpdateCharacterSheetResponse{
			Body:   UpdateCharacterSheetResponseBody{CharacterSheet: *NewCharacterSheetResponse(charSheet)},
			Status: http.StatusOK,
		}, nil
	}
}
```

- [ ] **Step 2: Register in `routes.go`**

Add to `Api` struct:

```go
UpdateCharacterSheetHandler Handler[UpdateCharacterSheetRequest, UpdateCharacterSheetResponse]
```

Add to `RegisterRoutes`:

```go
huma.Register(api, huma.Operation{
    Method:      http.MethodPatch,
    Path:        "/charactersheets/{uuid}",
    Description: "Full update of a character sheet (owner only, free state)",
    Tags:        []string{"character_sheets"},
    Errors: []int{
        http.StatusBadRequest,
        http.StatusUnauthorized,
        http.StatusNotFound,
        http.StatusForbidden,
        http.StatusUnprocessableEntity,
        http.StatusInternalServerError,
    },
    DefaultStatus: http.StatusOK,
}, a.UpdateCharacterSheetHandler)
```

- [ ] **Step 3: Wire in `main.go`**

```go
updateCharacterSheetUC := cs.NewUpdateCharacterSheetUC(
    &dryCharacterClasses,
    characterSheetFactory,
    characterSheetRepo,
    submitRepo,
)
```

Add to `characterSheetsApi`:

```go
UpdateCharacterSheetHandler: sheetHandler.UpdateCharacterSheetHandler(updateCharacterSheetUC, getCharacterSheetUC),
```

- [ ] **Step 4: Verify compile**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/app/api/sheet/update_character_sheet.go \
        internal/app/api/sheet/routes.go \
        cmd/api/main.go
git commit -m "feat(sheet/api): PATCH /charactersheets/{uuid} for full edit"
```

---

## Task 11: Extend `PATCH /profile` with `brief_description`

**Files:**
- Modify: `internal/app/api/sheet/patch_character_sheet_profile.go`
- Modify: `internal/gateway/pg/sheet/update_character_sheet_profile.go`

- [ ] **Step 1: Add `Description` to request body**

In `patch_character_sheet_profile.go`, update the body struct:

```go
type PatchCharacterSheetProfileRequestBody struct {
	AvatarURL   *string `json:"avatar_url,omitempty"`
	CoverURL    *string `json:"cover_url,omitempty"`
	Description *string `json:"brief_description,omitempty"`
}
```

Update the interface and handler call:

```go
type IProfileImageUpdater interface {
	UpdateCharacterSheetProfile(ctx context.Context, sheetUUID, playerUUID uuid.UUID, avatarURL, coverURL, description *string) error
}
```

```go
err = repo.UpdateCharacterSheetProfile(ctx, sheetUUID, userUUID, req.Body.AvatarURL, req.Body.CoverURL, req.Body.Description)
```

- [ ] **Step 2: Update gateway**

In `update_character_sheet_profile.go`, update the signature and query:

```go
func (r *Repository) UpdateCharacterSheetProfile(
	ctx context.Context,
	sheetUUID uuid.UUID,
	playerUUID uuid.UUID,
	avatarURL *string,
	coverURL *string,
	description *string,
) error {
	const query = `
		UPDATE character_profiles cp
		SET avatar_url = $1, cover_url = $2, brief_description = $3, updated_at = $4
		FROM character_sheets cs
		WHERE cp.character_sheet_uuid = cs.uuid
		  AND cs.uuid = $5
		  AND cs.player_uuid = $6
	`
	tag, err := r.q.Exec(ctx, query, avatarURL, coverURL, description, time.Now(), sheetUUID, playerUUID)
	if err != nil {
		return fmt.Errorf("failed to update character sheet profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCharacterSheetNotFound
	}
	return nil
}
```

- [ ] **Step 3: Verify compile**

```bash
go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/app/api/sheet/patch_character_sheet_profile.go \
        internal/gateway/pg/sheet/update_character_sheet_profile.go
git commit -m "feat(sheet/api): add brief_description to PATCH /charactersheets/{uuid}/profile"
```

---

## Task 12: Update API contract doc

**Files:**
- Modify: `docs/dev/api/character-sheet.md`
- Modify: `docs/documentation-map.yaml` (if this file exists; create entry if absent)

- [ ] **Step 1: Add new endpoints to contract doc**

In `docs/dev/api/character-sheet.md`, append sections for the three new/modified endpoints following the existing format in that file. Include: method, path, auth, request body (snake_case), response codes, response body, examples.

The three sections to add:
1. `GET /charactersheets/{uuid}?include=submission` — extended behaviour
2. `PATCH /charactersheets/{uuid}` — full edit
3. `DELETE /charactersheets/{uuid}` — delete

Also update the existing `PATCH /charactersheets/{uuid}/profile` section to document `brief_description`.

- [ ] **Step 2: Commit**

```bash
git add docs/dev/api/character-sheet.md
git commit -m "docs: update character-sheet API contract with manage endpoints"
```

---

## Task 13 (Frontend): Extend types and service

**Files:**
- Modify: `System_X_System_React/src/types/characterSheet.ts`
- Modify: `System_X_System_React/src/services/characterSheetsService.ts`

- [ ] **Step 1: Add `uuid` and `Submission` to `CharacterSheet` type**

In `characterSheet.ts`, add to the `CharacterSheet` type:

```ts
export type Submission = {
  campaignUuid: string;
  createdAt: string;
} | null;

// Inside CharacterSheet:
uuid: string;
submission?: Submission;
```

- [ ] **Step 2: Update service**

In `characterSheetsService.ts`:

**Update `getCharacterSheetDetails`** to pass `?include=submission`:

```ts
getCharacterSheetDetails: (token: string, id: string): Promise<CharacterSheet> =>
  httpClient
    .get<{ character_sheet: CharacterSheet }>(
      `/charactersheets/${id}?include=submission`,
      config(token)
    )
    .then(({ data }) => objToCamelCase<CharacterSheet>(data.character_sheet)),
```

**Add `deleteCharacterSheet`**:

```ts
deleteCharacterSheet: (token: string, uuid: string): Promise<void> =>
  httpClient
    .delete(`/charactersheets/${uuid}`, config(token))
    .then(() => undefined),
```

**Add `updateCharacterSheet`**:

```ts
updateCharacterSheet: (
  token: string,
  uuid: string,
  charSheet: CharacterSheet,
  charClass?: CharacterClass
): Promise<CharacterSheet> => {
  const allowedSkills = new Set(charClass?.distribution?.skillsAllowed ?? []);
  const skillsExps: Record<string, number> = {};
  const allSkills = { ...charSheet.physicalSkills, ...charSheet.spiritualSkills };
  Object.entries(allSkills).forEach(([name, skill]) => {
    const apiKey = name.charAt(0).toUpperCase() + name.slice(1);
    if (skill.exp && skill.exp > 0 && allowedSkills.has(apiKey)) {
      skillsExps[apiKey] = skill.exp;
    }
  });

  const allowedProfs = new Set(charClass?.distribution?.proficienciesAllowed ?? []);
  const proficienciesExps: Record<string, number> = {};
  Object.entries(charSheet.commonProficiencies).forEach(([name, prof]) => {
    if (prof.exp && prof.exp > 0 && allowedProfs.has(name)) proficienciesExps[name] = prof.exp;
  });

  const attributePoints: Record<string, number> = {};
  const allAttrs = { ...charSheet.physicalAttributes, ...charSheet.mentalAttributes };
  Object.entries(allAttrs).forEach(([name, attr]) => {
    const apiKey = name.charAt(0).toUpperCase() + name.slice(1);
    if (attr.points > 0) attributePoints[apiKey] = attr.points;
  });

  return httpClient
    .patch<{ character_sheet: CharacterSheet }>(
      `/charactersheets/${uuid}`,
      {
        profile: objToSnakeCase({
          nickname: charSheet.profile.nickname,
          fullname: charSheet.profile.fullname,
          alignment: charSheet.profile.alignment,
          description: charSheet.profile.description ?? "",
          briefDescription: charSheet.profile.briefDescription,
          birthday: charSheet.profile.birthday,
          age: charSheet.profile.age,
        }),
        character_class: charSheet.characterClass,
        skills_exps: skillsExps,
        proficiencies_exps: proficienciesExps,
        attribute_points: attributePoints,
      },
      config(token)
    )
    .then(({ data }) => objToCamelCase<CharacterSheet>(data.character_sheet));
},
```

**Update `patchCharacterSheetProfile`** to accept `description`:

```ts
patchCharacterSheetProfile: (
  token: string,
  sheetUuid: string,
  avatarUrl?: string | null,
  coverUrl?: string | null,
  briefDescription?: string | null
): Promise<void> =>
  httpClient
    .patch(
      `/charactersheets/${sheetUuid}/profile`,
      objToSnakeCase({ avatarUrl, coverUrl, briefDescription }),
      config(token)
    )
    .then(() => undefined),
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
cd System_X_System_React && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add src/types/characterSheet.ts src/services/characterSheetsService.ts
git commit -m "feat(sheet/types+service): add uuid/submission types, delete/update service methods"
```

---

## Task 14 (Frontend): Add hooks

**Files:**
- Create: `src/hooks/useDeleteCharacterSheet.ts`
- Create: `src/hooks/useUpdateCharacterSheet.ts`

- [ ] **Step 1: Create `useDeleteCharacterSheet`**

```ts
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { characterSheetsService } from "../services/characterSheetsService";

export function useDeleteCharacterSheet(token: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (uuid: string) => characterSheetsService.deleteCharacterSheet(token!, uuid),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["characterSheets", token] });
    },
  });
}
```

- [ ] **Step 2: Create `useUpdateCharacterSheet`**

```ts
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { characterSheetsService } from "../services/characterSheetsService";
import type { CharacterSheet } from "../types/characterSheet";
import type { CharacterClass } from "../types/characterClass";

export function useUpdateCharacterSheet(token: string | null) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ uuid, charSheet, charClass }: { uuid: string; charSheet: CharacterSheet; charClass?: CharacterClass }) =>
      characterSheetsService.updateCharacterSheet(token!, uuid, charSheet, charClass),
    onSuccess: (_, { uuid }) => {
      queryClient.invalidateQueries({ queryKey: ["characterSheet", token, uuid] });
    },
  });
}
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
npx tsc --noEmit
```

- [ ] **Step 4: Commit**

```bash
git add src/hooks/useDeleteCharacterSheet.ts src/hooks/useUpdateCharacterSheet.ts
git commit -m "feat(hooks): add useDeleteCharacterSheet and useUpdateCharacterSheet"
```

---

## Task 15 (Frontend): `ManageButton` component

**Files:**
- Create: `src/features/sheet/ManageButton.tsx`

- [ ] **Step 1: Create the component**

```tsx
import { useState, useEffect, useRef } from "react";
import styled from "styled-components";

interface ManageButtonProps {
  isFree: boolean;
  isFloating: boolean;
  onEdit: () => void;
  onDelete: () => void;
}

export default function ManageButton({ isFree, isFloating, onEdit, onDelete }: ManageButtonProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  const handleEdit = () => {
    setOpen(false);
    onEdit();
  };

  const handleDelete = () => {
    setOpen(false);
    if (window.confirm("Tem certeza que deseja excluir esta ficha? Esta ação não pode ser desfeita.")) {
      onDelete();
    }
  };

  return (
    <Wrapper ref={ref}>
      {open && (
        <Menu>
          <MenuItem onClick={handleEdit}>✏ Editar</MenuItem>
          {isFree && (
            <MenuItemDanger onClick={handleDelete}>🗑 Excluir</MenuItemDanger>
          )}
        </Menu>
      )}
      <Button $isFloating={isFloating} $open={open} onClick={() => setOpen((v) => !v)}>
        ⚙ Gerenciar {open ? "▴" : "▾"}
      </Button>
    </Wrapper>
  );
}

const Wrapper = styled.div`
  position: relative;
  flex-shrink: 0;
`;

const Button = styled.button<{ $isFloating: boolean; $open: boolean }>`
  background: #1c1c1c;
  border: 1px solid ${({ $open }) => ($open ? "#ffa216" : "#555")};
  color: white;
  font-family: "Roboto", sans-serif;
  font-size: 18px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
  transition: all 0.2s ease;

  ${({ $isFloating }) =>
    $isFloating
      ? `
        border-radius: 50px;
        padding: 14px 22px;
        box-shadow: 0 4px 10px rgba(0,0,0,0.5);
        &:hover { transform: translateY(-3px); filter: brightness(1.2); }
      `
      : `
        border-radius: 8px;
        height: 100%;
        padding: 0 20px;
        &:hover { filter: brightness(1.2); }
      `}

  &:active { transform: scale(0.98); }
`;

const Menu = styled.div`
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  background: #1c1c1c;
  border: 1px solid #555;
  border-radius: 8px;
  overflow: hidden;
  min-width: 160px;
  box-shadow: 0 -4px 16px rgba(0,0,0,0.6);
  z-index: 20;
`;

const MenuItem = styled.div`
  padding: 12px 18px;
  color: white;
  font-family: "Roboto", sans-serif;
  font-size: 17px;
  font-weight: 600;
  cursor: pointer;
  border-bottom: 1px solid #333;
  &:last-child { border-bottom: none; }
  &:hover { background: #2a2a2a; }
`;

const MenuItemDanger = styled(MenuItem)`
  color: #f38ba8;
`;
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
npx tsc --noEmit
```

- [ ] **Step 3: Commit**

```bash
git add src/features/sheet/ManageButton.tsx
git commit -m "feat(sheet): add ManageButton component with inline dropdown"
```

---

## Task 16 (Frontend): `SheetBottomActions` — replaces `SheetCampaignButton`

**Files:**
- Create: `src/features/sheet/SheetBottomActions.tsx`

The floating detection logic moves here from `SheetCampaignButton`. `SheetCampaignButton.tsx` is left unchanged — it will no longer be imported by `CharacterSheetTemplate` (which switches to `SheetBottomActions`).

- [ ] **Step 1: Create the component**

```tsx
import { useEffect, useState } from "react";
import { useDebounce } from "../../hooks/useDebounce";
import PlusIcon from "../../components/ions/PlusIcon";
import ManageButton from "./ManageButton";
import styled from "styled-components";

interface SheetBottomActionsProps {
  onCampaignClick?: () => void;
  campaignLabel?: string;
  manage?: {
    isFree: boolean;
    onEdit: () => void;
    onDelete: () => void;
  };
}

export default function SheetBottomActions({
  onCampaignClick,
  campaignLabel = "Procurar Campanhas",
  manage,
}: SheetBottomActionsProps) {
  const [isFloating, setIsFloating] = useState(false);
  const [scrollPosition, setScrollPosition] = useState(0);
  const debouncedScroll = useDebounce(scrollPosition, 50);

  useEffect(() => {
    const root = document.getElementById("root");
    if (!root) return;
    const checkScroll = () => setScrollPosition(root.scrollTop);
    checkScroll();
    root.addEventListener("scroll", checkScroll);
    window.addEventListener("resize", checkScroll);
    return () => {
      root.removeEventListener("scroll", checkScroll);
      window.removeEventListener("resize", checkScroll);
    };
  }, []);

  useEffect(() => {
    const root = document.getElementById("root");
    if (!root) return;
    const check = () => {
      setIsFloating(root.scrollTop + root.clientHeight < root.scrollHeight - 30);
    };
    const timers = [0, 50, 150, 300, 500].map((t) => setTimeout(check, t));
    return () => timers.forEach(clearTimeout);
  }, []);

  useEffect(() => {
    const root = document.getElementById("root");
    if (!root) return;
    setIsFloating(debouncedScroll + root.clientHeight < root.scrollHeight - 30);
  }, [debouncedScroll]);

  if (isFloating) {
    return (
      <>
        {manage && (
          <FloatingLeft>
            <ManageButton {...manage} isFloating={true} />
          </FloatingLeft>
        )}
        {onCampaignClick && (
          <FloatingRight onClick={onCampaignClick}>
            <PlusIcon />
            <span>{campaignLabel}</span>
          </FloatingRight>
        )}
      </>
    );
  }

  return (
    <AnchoredWrapper>
      {manage && (
        <ManageButton {...manage} isFloating={false} />
      )}
      {onCampaignClick && (
        <CampaignButton onClick={onCampaignClick}>
          <PlusIcon />
          <span>{campaignLabel}</span>
        </CampaignButton>
      )}
    </AnchoredWrapper>
  );
}

const AnchoredWrapper = styled.div`
  position: absolute;
  bottom: 22px;
  left: 0;
  z-index: 10;
  height: 91px;
  width: 100%;
  display: flex;
  align-items: center;
  padding: 0 3.2%;
  gap: 10px;
  box-sizing: border-box;
`;

const CampaignButton = styled.button`
  flex: 1;
  height: 100%;
  border: none;
  border-radius: 8px;
  background: linear-gradient(to bottom, #ffa216 0%, #ffa216 20%, #e60000 100%);
  color: black;
  font-family: "Roboto", sans-serif;
  font-size: 26px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 15px;
  cursor: pointer;
  transition: all 0.3s ease;
  &:hover { transform: translateY(-5px); filter: brightness(1.1); }
  &:active { transform: scale(0.98); }
`;

const FloatingLeft = styled.div`
  position: fixed;
  bottom: 20px;
  left: 60px;
  z-index: 10;
`;

const FloatingRight = styled.button`
  position: fixed;
  bottom: 20px;
  right: 60px;
  z-index: 10;
  border: none;
  border-radius: 50px;
  padding: 15px 30px 15px 26px;
  background: linear-gradient(to bottom, #ffa216 0%, #ffa216 20%, #e60000 100%);
  color: black;
  font-family: "Roboto", sans-serif;
  font-size: 26px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  box-shadow: 0 4px 10px rgba(0,0,0,0.3);
  transition: all 0.3s ease;
  &:hover { transform: translateY(-5px); filter: brightness(1.1); box-shadow: 0 8px 15px rgba(0,0,0,0.4); }
  &:active { transform: scale(0.98); }
`;
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
npx tsc --noEmit
```

- [ ] **Step 3: Commit**

```bash
git add src/features/sheet/SheetBottomActions.tsx
git commit -m "feat(sheet): add SheetBottomActions (replaces SheetCampaignButton in template)"
```

---

## Task 17 (Frontend): Update `CharacterSheetTemplate`

**Files:**
- Modify: `src/features/sheet/CharacterSheetTemplate.tsx`

- [ ] **Step 1: Add `manage` prop to `Data` interface**

In `CharacterSheetTemplate.tsx`, update the `Data` interface:

```ts
manage?: {
  isFree: boolean;
  onEdit: () => void;
  onDelete: () => void;
};
```

- [ ] **Step 2: Update destructuring in the component**

Add `manage` to the destructured props in `CharacterSheetTemplate`.

- [ ] **Step 3: Replace `SheetCampaignButton` import with `SheetBottomActions`**

Remove:
```ts
import SheetCampaignButton from "./SheetCampaignButton";
```

Add:
```ts
import SheetBottomActions from "./SheetBottomActions";
```

- [ ] **Step 4: Update the bottom button rendering**

Replace the existing `SheetCampaignButton` block with `SheetBottomActions`:

```tsx
{sheetMode.headerMode === "view" && (onCampaignClick || manage) && (
  <SheetBottomActions
    onCampaignClick={onCampaignClick}
    campaignLabel={hasCampaign ? "Ver Campanha" : "Procurar Campanhas"}
    manage={manage}
  />
)}
```

- [ ] **Step 5: Update `$hasCampaignButton` prop**

In `MainContent` and its styled definition, rename `$hasCampaignButton` to `$hasBottomActions` and update the condition:

```tsx
<MainContent $hasBottomActions={sheetMode.headerMode === "view" && !!(onCampaignClick || manage || onAcceptSubmission || onRejectSubmission)}>
```

Update `MainContent` styled-component signature from `$hasCampaignButton` to `$hasBottomActions`.

- [ ] **Step 6: Verify TypeScript compiles**

```bash
npx tsc --noEmit
```

- [ ] **Step 7: Commit**

```bash
git add src/features/sheet/CharacterSheetTemplate.tsx
git commit -m "feat(sheet/template): integrate SheetBottomActions with manage prop"
```

---

## Task 18 (Frontend): Update `CharacterSheetPage`

**Files:**
- Modify: `src/pages/CharacterSheetPage.tsx`

- [ ] **Step 1: Add delete hook and manage logic**

```tsx
import { Navigate, useParams, useNavigate, useLocation } from "react-router-dom";
import CharacterSheetTemplate from "../features/sheet/CharacterSheetTemplate";
import useToken from "../hooks/useToken";
import useUser from "../hooks/useUser";
import type { SheetMode } from "../features/sheet/types/sheetMode";
import { useCharacterSheet } from "../hooks/useCharacterSheet";
import { useAcceptSheetSubmission } from "../hooks/useAcceptSheetSubmission";
import { useRejectSheetSubmission } from "../hooks/useRejectSheetSubmission";
import { useDeleteCharacterSheet } from "../hooks/useDeleteCharacterSheet";

function CharacterSheetPage() {
  const { id } = useParams<{ id: string }>();
  const { token } = useToken();
  const { user } = useUser();
  const navigate = useNavigate();
  const location = useLocation();
  const locationState = (location.state as { isPending?: boolean; campaignId?: string } | null);
  const isPending = locationState?.isPending ?? false;
  const campaignId = locationState?.campaignId;

  const sheetMode: SheetMode = {
    headerMode: "view",
    profileMode: "view",
    diagramsMode: "view",
    proficiencyMode: "view",
    skillsMode: "view",
  };

  const { data: charSheet, isLoading, error } = useCharacterSheet(token, id);
  const { mutate: acceptSubmission, isPending: accepting } = useAcceptSheetSubmission(token, campaignId);
  const { mutate: rejectSubmission, isPending: rejecting } = useRejectSheetSubmission(token, campaignId);
  const { mutate: deleteSheet } = useDeleteCharacterSheet(token);

  if (!token || !id) return <Navigate to="/" replace />;

  const isOwner = !!charSheet && !!user && charSheet.playerUuid === user.uuid;
  const isFree = isOwner && !!charSheet && !charSheet.campaignUuid && !charSheet.submission;

  const handleCampaignClick = () => {
    if (charSheet?.campaignUuid) {
      navigate(`/campaigns/${charSheet.campaignUuid}`);
    } else {
      navigate("/campaigns/public", { state: { sheetId: id } });
    }
  };

  const handleEdit = () => {
    if (!charSheet?.uuid) return;
    navigate(isFree ? `/charactersheet/${charSheet.uuid}/edit` : `/charactersheet/${charSheet.uuid}/edit/profile`);
  };

  const handleDelete = () => {
    if (!charSheet?.uuid) return;
    deleteSheet(charSheet.uuid, {
      onSuccess: () => navigate("/", { replace: true }),
    });
  };

  const handleAccept = () => {
    if (!id) return;
    acceptSubmission(id, { onSuccess: () => campaignId && navigate(`/campaigns/${campaignId}`) });
  };

  const handleReject = () => {
    if (!id) return;
    rejectSubmission(id, { onSuccess: () => campaignId && navigate(`/campaigns/${campaignId}`) });
  };

  return (
    <CharacterSheetTemplate
      sheetMode={sheetMode}
      data={{
        charSheet,
        isLoading,
        error: error ? error.message : null,
        onCampaignClick: isOwner && charSheet ? handleCampaignClick : undefined,
        hasCampaign: !!charSheet?.campaignUuid,
        manage: isOwner ? { isFree, onEdit: handleEdit, onDelete: handleDelete } : undefined,
        onAcceptSubmission: !isOwner && isPending && !accepting ? handleAccept : undefined,
        onRejectSubmission: !isOwner && isPending && !rejecting ? handleReject : undefined,
      }}
    />
  );
}

export default CharacterSheetPage;
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
npx tsc --noEmit
```

- [ ] **Step 3: Commit**

```bash
git add src/pages/CharacterSheetPage.tsx
git commit -m "feat(sheet/page): wire ManageButton — isFree logic, edit/delete handlers"
```

---

## Task 19 (Frontend): `EditCharacterSheetPage` (full edit)

**Files:**
- Create: `src/pages/EditCharacterSheetPage.tsx`

- [ ] **Step 1: Create the page**

This page mirrors `CreateCharacterSheetPage` but pre-populates from `useCharacterSheet` and calls `updateCharacterSheet` on submit. Guard: redirect to `/charactersheet/:id` if the sheet is not free.

```tsx
import { useState, useEffect, useRef } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import useToken from "../hooks/useToken";
import useUser from "../hooks/useUser";
import type { SheetMode } from "../features/sheet/types/sheetMode";
import CharacterSheetTemplate from "../features/sheet/CharacterSheetTemplate";
import { useCharacterClasses } from "../hooks/useCharacterClasses";
import { useCharacterSheet } from "../hooks/useCharacterSheet";
import type { CharacterSheet } from "../types/characterSheet";
import { characterSheetsService } from "../services/characterSheetsService";
import { uploadService } from "../services/uploadService";

function EditCharacterSheetPage() {
  const { id } = useParams<{ id: string }>();
  const { token } = useToken();
  const { user } = useUser();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: existingSheet, isLoading: sheetLoading } = useCharacterSheet(token, id);
  const { data: charClasses, isLoading: classesLoading, error: classesError } = useCharacterClasses(token);

  const [charSheet, setCharSheet] = useState<CharacterSheet | undefined>(undefined);
  const [avatarBlob, setAvatarBlob] = useState<Blob | null>(null);
  const [coverBlob, setCoverBlob] = useState<Blob | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const avatarBlobUrlRef = useRef<string | undefined>(undefined);
  const coverBlobUrlRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    if (existingSheet) setCharSheet(existingSheet);
  }, [existingSheet]);

  useEffect(() => {
    return () => {
      if (avatarBlobUrlRef.current) URL.revokeObjectURL(avatarBlobUrlRef.current);
      if (coverBlobUrlRef.current) URL.revokeObjectURL(coverBlobUrlRef.current);
    };
  }, []);

  if (!token || !id) return <Navigate to="/" replace />;

  // Guard: redirect if not owner or not free
  if (existingSheet && user) {
    const isOwner = existingSheet.playerUuid === user.uuid;
    const isFree = !existingSheet.campaignUuid && !existingSheet.submission;
    if (!isOwner || !isFree) return <Navigate to={`/charactersheet/${id}`} replace />;
  }

  const sheetMode: SheetMode = {
    headerMode: "edit",
    profileMode: "create",
    diagramsMode: "create",
    proficiencyMode: "create",
    skillsMode: "view",
  };

  const handleAvatarSelected = (blob: Blob | null, url: string | null) => {
    setAvatarBlob(blob);
    if (avatarBlobUrlRef.current) { URL.revokeObjectURL(avatarBlobUrlRef.current); avatarBlobUrlRef.current = undefined; }
    const previewUrl = blob ? URL.createObjectURL(blob) : url ?? undefined;
    if (blob && previewUrl) avatarBlobUrlRef.current = previewUrl;
    setCharSheet((prev) => prev ? { ...prev, profile: { ...prev.profile, avatarUrl: previewUrl } } : prev);
  };

  const handleCoverSelected = (blob: Blob | null, url: string | null) => {
    setCoverBlob(blob);
    if (coverBlobUrlRef.current) { URL.revokeObjectURL(coverBlobUrlRef.current); coverBlobUrlRef.current = undefined; }
    const previewUrl = blob ? URL.createObjectURL(blob) : url ?? undefined;
    if (blob && previewUrl) coverBlobUrlRef.current = previewUrl;
    setCharSheet((prev) => prev ? { ...prev, profile: { ...prev.profile, coverUrl: previewUrl } } : prev);
  };

  const handleSave = async () => {
    if (!token || !id || !charSheet || isSubmitting) return;
    setSubmitError(null);
    setIsSubmitting(true);
    let resolvedAvatarUrl: string | undefined = avatarBlob ? undefined : charSheet.profile.avatarUrl;
    let resolvedCoverUrl: string | undefined = coverBlob ? undefined : charSheet.profile.coverUrl;
    try {
      const selectedClass = charClasses?.find((cc) => cc.profile.name === charSheet.characterClass);
      await characterSheetsService.updateCharacterSheet(token, id, charSheet, selectedClass);

      if (avatarBlob) {
        const { uploadUrl, publicUrl } = await uploadService.getPresignedUrl(token, "avatar", id);
        await uploadService.uploadToR2(uploadUrl, avatarBlob);
        resolvedAvatarUrl = publicUrl;
      }
      if (coverBlob) {
        const { uploadUrl, publicUrl } = await uploadService.getPresignedUrl(token, "cover", id);
        await uploadService.uploadToR2(uploadUrl, coverBlob);
        resolvedCoverUrl = publicUrl;
      }
      if (resolvedAvatarUrl !== undefined || resolvedCoverUrl !== undefined) {
        await characterSheetsService.patchCharacterSheetProfile(token, id, resolvedAvatarUrl ?? null, resolvedCoverUrl ?? null);
      }

      queryClient.invalidateQueries({ queryKey: ["characterSheet", token, id] });
      navigate(`/charactersheet/${id}`, { replace: true });
    } catch {
      setSubmitError("Erro ao salvar. Tente novamente.");
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!charSheet) return null;

  return (
    <CharacterSheetTemplate
      sheetMode={sheetMode}
      data={{
        charSheet,
        setCharSheet,
        charClasses,
        isLoading: sheetLoading || classesLoading || isSubmitting,
        error: classesError ? classesError.message : null,
        onAvatarSelected: handleAvatarSelected,
        onCoverSelected: handleCoverSelected,
        onCreateSheet: handleSave,
        submitError,
      }}
    />
  );
}

export default EditCharacterSheetPage;
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
npx tsc --noEmit
```

- [ ] **Step 3: Commit**

```bash
git add src/pages/EditCharacterSheetPage.tsx
git commit -m "feat(sheet): add EditCharacterSheetPage (full edit, isFree guard)"
```

---

## Task 20 (Frontend): `EditCharacterSheetProfilePage` (partial edit)

**Files:**
- Create: `src/pages/EditCharacterSheetProfilePage.tsx`

- [ ] **Step 1: Create the page**

```tsx
import { useState, useRef, useEffect } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import useToken from "../hooks/useToken";
import useUser from "../hooks/useUser";
import { useCharacterSheet } from "../hooks/useCharacterSheet";
import { characterSheetsService } from "../services/characterSheetsService";
import { uploadService } from "../services/uploadService";
import BackButton from "../components/ions/BackButton";
import styled from "styled-components";

function EditCharacterSheetProfilePage() {
  const { id } = useParams<{ id: string }>();
  const { token } = useToken();
  const { user } = useUser();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: charSheet, isLoading } = useCharacterSheet(token, id);

  const [briefDescription, setBriefDescription] = useState("");
  const [avatarBlob, setAvatarBlob] = useState<Blob | null>(null);
  const [coverBlob, setCoverBlob] = useState<Blob | null>(null);
  const [avatarPreview, setAvatarPreview] = useState<string | undefined>(undefined);
  const [coverPreview, setCoverPreview] = useState<string | undefined>(undefined);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const avatarUrlRef = useRef<string | undefined>(undefined);
  const coverUrlRef = useRef<string | undefined>(undefined);

  useEffect(() => {
    if (charSheet) {
      setBriefDescription(charSheet.profile.briefDescription ?? "");
      setAvatarPreview(charSheet.profile.avatarUrl);
      setCoverPreview(charSheet.profile.coverUrl);
    }
  }, [charSheet]);

  useEffect(() => {
    return () => {
      if (avatarUrlRef.current) URL.revokeObjectURL(avatarUrlRef.current);
      if (coverUrlRef.current) URL.revokeObjectURL(coverUrlRef.current);
    };
  }, []);

  if (!token || !id) return <Navigate to="/" replace />;

  if (charSheet && user) {
    if (charSheet.playerUuid !== user.uuid) return <Navigate to={`/charactersheet/${id}`} replace />;
  }

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setAvatarBlob(file);
    if (avatarUrlRef.current) URL.revokeObjectURL(avatarUrlRef.current);
    const url = URL.createObjectURL(file);
    avatarUrlRef.current = url;
    setAvatarPreview(url);
  };

  const handleCoverChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setCoverBlob(file);
    if (coverUrlRef.current) URL.revokeObjectURL(coverUrlRef.current);
    const url = URL.createObjectURL(file);
    coverUrlRef.current = url;
    setCoverPreview(url);
  };

  const handleSave = async () => {
    if (!token || !id || isSubmitting) return;
    setSubmitError(null);
    setIsSubmitting(true);
    let resolvedAvatar: string | null | undefined = avatarBlob ? undefined : (charSheet?.profile.avatarUrl ?? null);
    let resolvedCover: string | null | undefined = coverBlob ? undefined : (charSheet?.profile.coverUrl ?? null);
    try {
      if (avatarBlob) {
        const { uploadUrl, publicUrl } = await uploadService.getPresignedUrl(token, "avatar", id);
        await uploadService.uploadToR2(uploadUrl, avatarBlob);
        resolvedAvatar = publicUrl;
      }
      if (coverBlob) {
        const { uploadUrl, publicUrl } = await uploadService.getPresignedUrl(token, "cover", id);
        await uploadService.uploadToR2(uploadUrl, coverBlob);
        resolvedCover = publicUrl;
      }
      await characterSheetsService.patchCharacterSheetProfile(
        token, id,
        resolvedAvatar ?? null,
        resolvedCover ?? null,
        briefDescription || null,
      );
      queryClient.invalidateQueries({ queryKey: ["characterSheet", token, id] });
      navigate(`/charactersheet/${id}`, { replace: true });
    } catch {
      setSubmitError("Erro ao salvar. Tente novamente.");
    } finally {
      setIsSubmitting(false);
    }
  };

  if (isLoading) return <Container><p style={{ color: "white" }}>Carregando...</p></Container>;

  return (
    <Container>
      <BackButton />
      <Title>Editar Perfil</Title>

      <Section>
        <Label>Avatar</Label>
        {avatarPreview && <Preview src={avatarPreview} alt="avatar" />}
        <FileInput type="file" accept="image/*" onChange={handleAvatarChange} />
      </Section>

      <Section>
        <Label>Capa</Label>
        {coverPreview && <CoverPreview src={coverPreview} alt="capa" />}
        <FileInput type="file" accept="image/*" onChange={handleCoverChange} />
      </Section>

      <Section>
        <Label>Descrição breve</Label>
        <Textarea
          value={briefDescription}
          onChange={(e) => setBriefDescription(e.target.value)}
          maxLength={255}
          placeholder="Descreva brevemente seu personagem..."
          rows={4}
        />
        <CharCount>{briefDescription.length}/255</CharCount>
      </Section>

      {submitError && <ErrorText>{submitError}</ErrorText>}

      <SaveButton onClick={handleSave} disabled={isSubmitting}>
        {isSubmitting ? "Salvando..." : "Salvar"}
      </SaveButton>
    </Container>
  );
}

export default EditCharacterSheetProfilePage;

const Container = styled.div`
  max-width: 600px;
  margin: 0 auto;
  padding: 30px 20px;
  color: white;
  background: black;
  min-height: 100vh;
`;

const Title = styled.h1`
  font-family: "Roboto", sans-serif;
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 30px;
`;

const Section = styled.div`
  margin-bottom: 24px;
`;

const Label = styled.label`
  display: block;
  font-family: "Roboto", sans-serif;
  font-size: 16px;
  color: #aaa;
  margin-bottom: 8px;
`;

const Preview = styled.img`
  display: block;
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
  margin-bottom: 8px;
  border: 2px solid #555;
`;

const CoverPreview = styled.img`
  display: block;
  width: 100%;
  height: 120px;
  border-radius: 8px;
  object-fit: cover;
  margin-bottom: 8px;
  border: 2px solid #555;
`;

const FileInput = styled.input`
  color: white;
  font-family: "Roboto", sans-serif;
  font-size: 14px;
`;

const Textarea = styled.textarea`
  width: 100%;
  background: #1a1a1a;
  border: 1px solid #555;
  border-radius: 8px;
  color: white;
  font-family: "Roboto", sans-serif;
  font-size: 16px;
  padding: 12px;
  resize: vertical;
  box-sizing: border-box;
  &:focus { outline: none; border-color: #ffa216; }
`;

const CharCount = styled.div`
  font-size: 12px;
  color: #666;
  text-align: right;
  margin-top: 4px;
`;

const ErrorText = styled.p`
  color: #f38ba8;
  font-size: 14px;
  margin-bottom: 16px;
`;

const SaveButton = styled.button`
  width: 100%;
  padding: 16px;
  background: linear-gradient(to bottom, #ffa216 0%, #ffa216 20%, #e60000 100%);
  border: none;
  border-radius: 8px;
  color: black;
  font-family: "Roboto", sans-serif;
  font-size: 20px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
  &:hover:not(:disabled) { filter: brightness(1.1); transform: translateY(-2px); }
  &:disabled { opacity: 0.6; cursor: not-allowed; }
`;
```

- [ ] **Step 2: Verify TypeScript compiles**

```bash
npx tsc --noEmit
```

- [ ] **Step 3: Commit**

```bash
git add src/pages/EditCharacterSheetProfilePage.tsx
git commit -m "feat(sheet): add EditCharacterSheetProfilePage (partial edit: avatar + cover + description)"
```

---

## Task 21 (Frontend): Add routes in `App.tsx`

**Files:**
- Modify: `src/App.tsx`

- [ ] **Step 1: Import new pages**

```ts
import EditCharacterSheetPage from "./pages/EditCharacterSheetPage";
import EditCharacterSheetProfilePage from "./pages/EditCharacterSheetProfilePage";
```

- [ ] **Step 2: Add two protected routes**

Find where `/charactersheet/:id` is defined and add the two new routes after it:

```tsx
<Route path="/charactersheet/:id/edit" element={<EditCharacterSheetPage />} />
<Route path="/charactersheet/:id/edit/profile" element={<EditCharacterSheetProfilePage />} />
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
npx tsc --noEmit
```

- [ ] **Step 4: Commit**

```bash
git add src/App.tsx
git commit -m "feat(routes): add /charactersheet/:id/edit and /charactersheet/:id/edit/profile routes"
```

---

## Self-Review Checklist

- [x] **Spec coverage**: All endpoints (GET extend, PATCH full, PATCH profile extend, DELETE), all frontend components (ManageButton, SheetBottomActions, two edit pages), all routes — all covered.
- [x] **No placeholders**: All steps have concrete code. Gateway UPDATE query has all field names from `charSheetToModel`.
- [x] **Type consistency**: `IFreeStateChecker` defined once in `delete_character_sheet.go`, used in `update_character_sheet.go` (same package). `SubmissionInfo` defined in `get_character_sheet.go`, imported by gateway. `ManageButtonProps.isFloating` matches what `SheetBottomActions` passes.
- [x] **Note on model field names**: Task 9 explicitly calls out to verify `NenExp`, `FocusExp`, `WillPowerExp`, `AuraControlExp`, `AopExp` in the model struct before committing the gateway UPDATE query.
