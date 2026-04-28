# Domain Use Cases Tests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add comprehensive test coverage for 16 domain use cases across 6 packages using manual mocks.

**Architecture:** Each domain package defines a repository interface (IRepository). Tests will use hand-written mock implementations that return configurable results. Mocks live in a shared `internal/domain/testutil/` package to avoid duplication across test files.

**Tech Stack:** Go 1.23, standard library `testing` only, table-driven tests with `t.Run()`, external test packages.

**Excluded from this plan:** `character_sheet` use cases (Create/Get) — these are 200+ lines each with massive model transformations and will get their own dedicated plan.

---

## File Structure

```
internal/domain/
├── testutil/                          # NEW: Shared mock implementations
│   ├── mock_auth_repo.go             # Mock for auth.IRepository
│   ├── mock_session_repo.go          # Mock for session.IRepository
│   ├── mock_scenario_repo.go         # Mock for scenario.IRepository
│   ├── mock_campaign_repo.go         # Mock for campaign.IRepository
│   ├── mock_match_repo.go           # Mock for match.IRepository
│   ├── mock_submission_repo.go       # Mock for submission.IRepository
│   ├── mock_enrollment_repo.go       # Mock for enrollment.IRepository
│   ├── mock_character_sheet_repo.go  # Mock for charactersheet.IRepository (subset)
│   └── doc.go                        # Package documentation
├── scenario/
│   └── scenario_test.go              # NEW: Tests for all 3 scenario UCs
├── campaign/
│   └── campaign_test.go              # NEW: Tests for all 3 campaign UCs
├── match/
│   └── match_test.go                 # NEW: Tests for all 4 match UCs
├── auth/
│   └── auth_test.go                  # NEW: Tests for Login + Register UCs
├── submission/
│   └── submission_test.go            # NEW: Tests for all 3 submission UCs
└── enrollment/
    └── enrollment_test.go            # NEW: Tests for Enroll UC
```

---

### Task 1: Create Mock Infrastructure

**Files:**
- Create: `internal/domain/testutil/doc.go`
- Create: `internal/domain/testutil/mock_scenario_repo.go`
- Create: `internal/domain/testutil/mock_campaign_repo.go`
- Create: `internal/domain/testutil/mock_match_repo.go`
- Create: `internal/domain/testutil/mock_auth_repo.go`
- Create: `internal/domain/testutil/mock_session_repo.go`
- Create: `internal/domain/testutil/mock_submission_repo.go`
- Create: `internal/domain/testutil/mock_enrollment_repo.go`
- Create: `internal/domain/testutil/mock_character_sheet_repo.go`

- [ ] **Step 1: Create doc.go**

```go
// Package testutil provides manual mock implementations of domain repository
// interfaces for use in unit tests. Each mock allows configuring return values
// and errors per method call.
package testutil
```

- [ ] **Step 2: Create mock_scenario_repo.go**

```go
package testutil

import (
	"context"

	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

type MockScenarioRepo struct {
	CreateScenarioFn          func(ctx context.Context, scenario *scenarioEntity.Scenario) error
	GetScenarioFn             func(ctx context.Context, uuid uuid.UUID) (*scenarioEntity.Scenario, error)
	ExistsScenarioFn          func(ctx context.Context, uuid uuid.UUID) (bool, error)
	ExistsScenarioWithNameFn  func(ctx context.Context, name string) (bool, error)
	ListScenariosByUserUUIDFn func(ctx context.Context, userUUID uuid.UUID) ([]*scenarioEntity.Summary, error)
}

func (m *MockScenarioRepo) CreateScenario(ctx context.Context, scenario *scenarioEntity.Scenario) error {
	if m.CreateScenarioFn != nil {
		return m.CreateScenarioFn(ctx, scenario)
	}
	return nil
}

func (m *MockScenarioRepo) GetScenario(ctx context.Context, id uuid.UUID) (*scenarioEntity.Scenario, error) {
	if m.GetScenarioFn != nil {
		return m.GetScenarioFn(ctx, id)
	}
	return nil, nil
}

func (m *MockScenarioRepo) ExistsScenario(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.ExistsScenarioFn != nil {
		return m.ExistsScenarioFn(ctx, id)
	}
	return false, nil
}

func (m *MockScenarioRepo) ExistsScenarioWithName(ctx context.Context, name string) (bool, error) {
	if m.ExistsScenarioWithNameFn != nil {
		return m.ExistsScenarioWithNameFn(ctx, name)
	}
	return false, nil
}

func (m *MockScenarioRepo) ListScenariosByUserUUID(ctx context.Context, userUUID uuid.UUID) ([]*scenarioEntity.Summary, error) {
	if m.ListScenariosByUserUUIDFn != nil {
		return m.ListScenariosByUserUUIDFn(ctx, userUUID)
	}
	return nil, nil
}
```

- [ ] **Step 3: Create mock_campaign_repo.go**

```go
package testutil

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type MockCampaignRepo struct {
	CreateCampaignFn             func(ctx context.Context, campaign *campaign.Campaign) error
	GetCampaignFn                func(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	GetCampaignMasterUUIDFn      func(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCampaignStoryDatesFn      func(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	CountCampaignsByMasterUUIDFn func(ctx context.Context, masterUUID uuid.UUID) (int, error)
	ListCampaignsByMasterUUIDFn  func(ctx context.Context, masterUUID uuid.UUID) ([]*campaign.Summary, error)
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
```

- [ ] **Step 4: Create mock_match_repo.go**

```go
package testutil

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type MockMatchRepo struct {
	CreateMatchFn              func(ctx context.Context, match *match.Match) error
	GetMatchFn                 func(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
	GetMatchCampaignUUIDFn     func(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
	ListMatchesByMasterUUIDFn  func(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
	ListPublicUpcomingMatchesFn func(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error)
}

func (m *MockMatchRepo) CreateMatch(ctx context.Context, mt *match.Match) error {
	if m.CreateMatchFn != nil {
		return m.CreateMatchFn(ctx, mt)
	}
	return nil
}

func (m *MockMatchRepo) GetMatch(ctx context.Context, id uuid.UUID) (*match.Match, error) {
	if m.GetMatchFn != nil {
		return m.GetMatchFn(ctx, id)
	}
	return nil, nil
}

func (m *MockMatchRepo) GetMatchCampaignUUID(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error) {
	if m.GetMatchCampaignUUIDFn != nil {
		return m.GetMatchCampaignUUIDFn(ctx, matchUUID)
	}
	return uuid.Nil, nil
}

func (m *MockMatchRepo) ListMatchesByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error) {
	if m.ListMatchesByMasterUUIDFn != nil {
		return m.ListMatchesByMasterUUIDFn(ctx, masterUUID)
	}
	return nil, nil
}

func (m *MockMatchRepo) ListPublicUpcomingMatches(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error) {
	if m.ListPublicUpcomingMatchesFn != nil {
		return m.ListPublicUpcomingMatchesFn(ctx, after, masterUUID)
	}
	return nil, nil
}
```

- [ ] **Step 5: Create mock_auth_repo.go**

```go
package testutil

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
)

type MockAuthRepo struct {
	CreateUserFn          func(ctx context.Context, user *user.User) error
	GetUserByEmailFn      func(ctx context.Context, email string) (*user.User, error)
	ExistsUserWithNickFn  func(ctx context.Context, nick string) (bool, error)
	ExistsUserWithEmailFn func(ctx context.Context, email string) (bool, error)
}

func (m *MockAuthRepo) CreateUser(ctx context.Context, u *user.User) error {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, u)
	}
	return nil
}

func (m *MockAuthRepo) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *MockAuthRepo) ExistsUserWithNick(ctx context.Context, nick string) (bool, error) {
	if m.ExistsUserWithNickFn != nil {
		return m.ExistsUserWithNickFn(ctx, nick)
	}
	return false, nil
}

func (m *MockAuthRepo) ExistsUserWithEmail(ctx context.Context, email string) (bool, error) {
	if m.ExistsUserWithEmailFn != nil {
		return m.ExistsUserWithEmailFn(ctx, email)
	}
	return false, nil
}
```

- [ ] **Step 6: Create mock_session_repo.go**

```go
package testutil

import (
	"context"

	"github.com/google/uuid"
)

type MockSessionRepo struct {
	CreateSessionFn            func(ctx context.Context, userUUID uuid.UUID, token string) error
	ValidateSessionFn          func(ctx context.Context, userUUID uuid.UUID, token string) (bool, error)
	GetSessionTokenByUserUUIDFn func(ctx context.Context, userUUID uuid.UUID) (string, error)
}

func (m *MockSessionRepo) CreateSession(ctx context.Context, userUUID uuid.UUID, token string) error {
	if m.CreateSessionFn != nil {
		return m.CreateSessionFn(ctx, userUUID, token)
	}
	return nil
}

func (m *MockSessionRepo) ValidateSession(ctx context.Context, userUUID uuid.UUID, token string) (bool, error) {
	if m.ValidateSessionFn != nil {
		return m.ValidateSessionFn(ctx, userUUID, token)
	}
	return true, nil
}

func (m *MockSessionRepo) GetSessionTokenByUserUUID(ctx context.Context, userUUID uuid.UUID) (string, error) {
	if m.GetSessionTokenByUserUUIDFn != nil {
		return m.GetSessionTokenByUserUUIDFn(ctx, userUUID)
	}
	return "", nil
}
```

- [ ] **Step 7: Create mock_submission_repo.go**

```go
package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MockSubmissionRepo struct {
	SubmitCharacterSheetFn                  func(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID, createdAt time.Time) error
	ExistsSubmittedCharacterSheetFn         func(ctx context.Context, uuid uuid.UUID) (bool, error)
	AcceptCharacterSheetSubmissionFn        func(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID) error
	GetSubmissionCampaignUUIDBySheetUUIDFn  func(ctx context.Context, sheetUUID uuid.UUID) (uuid.UUID, error)
	RejectCharacterSheetSubmissionFn        func(ctx context.Context, sheetUUID uuid.UUID) error
}

func (m *MockSubmissionRepo) SubmitCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID, createdAt time.Time) error {
	if m.SubmitCharacterSheetFn != nil {
		return m.SubmitCharacterSheetFn(ctx, sheetUUID, campaignUUID, createdAt)
	}
	return nil
}

func (m *MockSubmissionRepo) ExistsSubmittedCharacterSheet(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.ExistsSubmittedCharacterSheetFn != nil {
		return m.ExistsSubmittedCharacterSheetFn(ctx, id)
	}
	return false, nil
}

func (m *MockSubmissionRepo) AcceptCharacterSheetSubmission(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID) error {
	if m.AcceptCharacterSheetSubmissionFn != nil {
		return m.AcceptCharacterSheetSubmissionFn(ctx, sheetUUID, campaignUUID)
	}
	return nil
}

func (m *MockSubmissionRepo) GetSubmissionCampaignUUIDBySheetUUID(ctx context.Context, sheetUUID uuid.UUID) (uuid.UUID, error) {
	if m.GetSubmissionCampaignUUIDBySheetUUIDFn != nil {
		return m.GetSubmissionCampaignUUIDBySheetUUIDFn(ctx, sheetUUID)
	}
	return uuid.Nil, nil
}

func (m *MockSubmissionRepo) RejectCharacterSheetSubmission(ctx context.Context, sheetUUID uuid.UUID) error {
	if m.RejectCharacterSheetSubmissionFn != nil {
		return m.RejectCharacterSheetSubmissionFn(ctx, sheetUUID)
	}
	return nil
}
```

- [ ] **Step 8: Create mock_enrollment_repo.go**

```go
package testutil

import (
	"context"

	"github.com/google/uuid"
)

type MockEnrollmentRepo struct {
	EnrollCharacterSheetFn         func(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheetFn func(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
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
```

- [ ] **Step 9: Create mock_character_sheet_repo.go**

```go
package testutil

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

type MockCharacterSheetRepo struct {
	CreateCharacterSheetFn                 func(ctx context.Context, sheet *model.CharacterSheet) error
	ExistsCharacterWithNickFn              func(ctx context.Context, nick string) (bool, error)
	CountCharactersByPlayerUUIDFn          func(ctx context.Context, playerUUID uuid.UUID) (int, error)
	GetCharacterSheetPlayerUUIDFn          func(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCharacterSheetByUUIDFn              func(ctx context.Context, uuid string) (*model.CharacterSheet, error)
	ListCharacterSheetsByPlayerUUIDFn      func(ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error)
	UpdateNenHexagonValueFn                func(ctx context.Context, uuid string, val int) error
	GetCharacterSheetRelationshipUUIDsFn   func(ctx context.Context, uuid uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error)
}

func (m *MockCharacterSheetRepo) CreateCharacterSheet(ctx context.Context, sheet *model.CharacterSheet) error {
	if m.CreateCharacterSheetFn != nil {
		return m.CreateCharacterSheetFn(ctx, sheet)
	}
	return nil
}

func (m *MockCharacterSheetRepo) ExistsCharacterWithNick(ctx context.Context, nick string) (bool, error) {
	if m.ExistsCharacterWithNickFn != nil {
		return m.ExistsCharacterWithNickFn(ctx, nick)
	}
	return false, nil
}

func (m *MockCharacterSheetRepo) CountCharactersByPlayerUUID(ctx context.Context, playerUUID uuid.UUID) (int, error) {
	if m.CountCharactersByPlayerUUIDFn != nil {
		return m.CountCharactersByPlayerUUIDFn(ctx, playerUUID)
	}
	return 0, nil
}

func (m *MockCharacterSheetRepo) GetCharacterSheetPlayerUUID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	if m.GetCharacterSheetPlayerUUIDFn != nil {
		return m.GetCharacterSheetPlayerUUIDFn(ctx, id)
	}
	return uuid.Nil, nil
}

func (m *MockCharacterSheetRepo) GetCharacterSheetByUUID(ctx context.Context, id string) (*model.CharacterSheet, error) {
	if m.GetCharacterSheetByUUIDFn != nil {
		return m.GetCharacterSheetByUUIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockCharacterSheetRepo) ListCharacterSheetsByPlayerUUID(ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error) {
	if m.ListCharacterSheetsByPlayerUUIDFn != nil {
		return m.ListCharacterSheetsByPlayerUUIDFn(ctx, playerUUID)
	}
	return nil, nil
}

func (m *MockCharacterSheetRepo) UpdateNenHexagonValue(ctx context.Context, id string, val int) error {
	if m.UpdateNenHexagonValueFn != nil {
		return m.UpdateNenHexagonValueFn(ctx, id, val)
	}
	return nil
}

func (m *MockCharacterSheetRepo) GetCharacterSheetRelationshipUUIDs(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
	if m.GetCharacterSheetRelationshipUUIDsFn != nil {
		return m.GetCharacterSheetRelationshipUUIDsFn(ctx, id)
	}
	return model.CharacterSheetRelationshipUUIDs{}, nil
}
```

- [ ] **Step 10: Verify mocks compile**

Run: `go build ./internal/domain/testutil/...`
Expected: No errors

- [ ] **Step 11: Commit**

```bash
git add internal/domain/testutil/
git commit -m "test(testutil): add mock repository implementations for domain UC tests

Manual mocks for scenario, campaign, match, auth, session,
submission, enrollment, and character_sheet repository interfaces.
Each mock uses configurable function fields."
```

---

### Task 2: Scenario Use Case Tests

**Files:**
- Create: `internal/domain/scenario/scenario_test.go`

- [ ] **Step 1: Write tests**

```go
package scenario_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgScenario "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/scenario"
	"github.com/google/uuid"
)

func TestCreateScenario(t *testing.T) {
	ctx := context.Background()
	userUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &testutil.MockScenarioRepo{}
		uc := scenario.NewCreateScenarioUC(repo)

		result, err := uc.CreateScenario(ctx, &scenario.CreateScenarioInput{
			UserUUID:         userUUID,
			Name:             "Test Scenario",
			BriefDescription: "A brief description",
			Description:      "Full description",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected scenario, got nil")
		}
		if result.Name != "Test Scenario" {
			t.Errorf("expected name 'Test Scenario', got %q", result.Name)
		}
		if result.UserUUID != userUUID {
			t.Errorf("expected userUUID %v, got %v", userUUID, result.UserUUID)
		}
	})

	t.Run("name too short", func(t *testing.T) {
		repo := &testutil.MockScenarioRepo{}
		uc := scenario.NewCreateScenarioUC(repo)

		_, err := uc.CreateScenario(ctx, &scenario.CreateScenarioInput{
			UserUUID: userUUID,
			Name:     "abcd",
		})
		if !errors.Is(err, scenario.ErrMinNameLength) {
			t.Errorf("expected ErrMinNameLength, got %v", err)
		}
	})

	t.Run("name too long", func(t *testing.T) {
		repo := &testutil.MockScenarioRepo{}
		uc := scenario.NewCreateScenarioUC(repo)

		longName := "abcdefghijklmnopqrstuvwxyz1234567"
		_, err := uc.CreateScenario(ctx, &scenario.CreateScenarioInput{
			UserUUID: userUUID,
			Name:     longName,
		})
		if !errors.Is(err, scenario.ErrMaxNameLength) {
			t.Errorf("expected ErrMaxNameLength, got %v", err)
		}
	})

	t.Run("brief description too long", func(t *testing.T) {
		repo := &testutil.MockScenarioRepo{}
		uc := scenario.NewCreateScenarioUC(repo)

		longDesc := make([]byte, 65)
		for i := range longDesc {
			longDesc[i] = 'a'
		}
		_, err := uc.CreateScenario(ctx, &scenario.CreateScenarioInput{
			UserUUID:         userUUID,
			Name:             "Valid Name",
			BriefDescription: string(longDesc),
		})
		if !errors.Is(err, scenario.ErrMaxBriefDescLength) {
			t.Errorf("expected ErrMaxBriefDescLength, got %v", err)
		}
	})

	t.Run("name already exists", func(t *testing.T) {
		repo := &testutil.MockScenarioRepo{
			ExistsScenarioWithNameFn: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
		}
		uc := scenario.NewCreateScenarioUC(repo)

		_, err := uc.CreateScenario(ctx, &scenario.CreateScenarioInput{
			UserUUID: userUUID,
			Name:     "Existing Name",
		})
		if !errors.Is(err, scenario.ErrScenarioNameAlreadyExists) {
			t.Errorf("expected ErrScenarioNameAlreadyExists, got %v", err)
		}
	})

	t.Run("repo error on exists check", func(t *testing.T) {
		dbErr := errors.New("db connection failed")
		repo := &testutil.MockScenarioRepo{
			ExistsScenarioWithNameFn: func(_ context.Context, _ string) (bool, error) {
				return false, dbErr
			},
		}
		uc := scenario.NewCreateScenarioUC(repo)

		_, err := uc.CreateScenario(ctx, &scenario.CreateScenarioInput{
			UserUUID: userUUID,
			Name:     "Valid Name",
		})
		if err != dbErr {
			t.Errorf("expected dbErr, got %v", err)
		}
	})

	t.Run("repo error on create", func(t *testing.T) {
		dbErr := errors.New("insert failed")
		repo := &testutil.MockScenarioRepo{
			CreateScenarioFn: func(_ context.Context, _ *scenarioEntity.Scenario) error {
				return dbErr
			},
		}
		uc := scenario.NewCreateScenarioUC(repo)

		_, err := uc.CreateScenario(ctx, &scenario.CreateScenarioInput{
			UserUUID: userUUID,
			Name:     "Valid Name",
		})
		if err != dbErr {
			t.Errorf("expected dbErr, got %v", err)
		}
	})
}

func TestGetScenario(t *testing.T) {
	ctx := context.Background()
	ownerUUID := uuid.New()
	scenarioUUID := uuid.New()

	t.Run("success as owner", func(t *testing.T) {
		expected := &scenarioEntity.Scenario{
			UUID:     scenarioUUID,
			UserUUID: ownerUUID,
			Name:     "My Scenario",
		}
		repo := &testutil.MockScenarioRepo{
			GetScenarioFn: func(_ context.Context, _ uuid.UUID) (*scenarioEntity.Scenario, error) {
				return expected, nil
			},
		}
		uc := scenario.NewGetScenarioUC(repo)

		result, err := uc.GetScenario(ctx, scenarioUUID, ownerUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Name != "My Scenario" {
			t.Errorf("expected name 'My Scenario', got %q", result.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &testutil.MockScenarioRepo{
			GetScenarioFn: func(_ context.Context, _ uuid.UUID) (*scenarioEntity.Scenario, error) {
				return nil, pgScenario.ErrScenarioNotFound
			},
		}
		uc := scenario.NewGetScenarioUC(repo)

		_, err := uc.GetScenario(ctx, scenarioUUID, ownerUUID)
		if !errors.Is(err, scenario.ErrScenarioNotFound) {
			t.Errorf("expected ErrScenarioNotFound, got %v", err)
		}
	})

	t.Run("insufficient permissions", func(t *testing.T) {
		otherUser := uuid.New()
		expected := &scenarioEntity.Scenario{
			UUID:     scenarioUUID,
			UserUUID: ownerUUID,
		}
		repo := &testutil.MockScenarioRepo{
			GetScenarioFn: func(_ context.Context, _ uuid.UUID) (*scenarioEntity.Scenario, error) {
				return expected, nil
			},
		}
		uc := scenario.NewGetScenarioUC(repo)

		_, err := uc.GetScenario(ctx, scenarioUUID, otherUser)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestListScenarios(t *testing.T) {
	ctx := context.Background()
	userUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		expected := []*scenarioEntity.Summary{
			{UUID: uuid.New(), Name: "Scenario A"},
			{UUID: uuid.New(), Name: "Scenario B"},
		}
		repo := &testutil.MockScenarioRepo{
			ListScenariosByUserUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*scenarioEntity.Summary, error) {
				return expected, nil
			},
		}
		uc := scenario.NewListScenariosUC(repo)

		result, err := uc.ListScenarios(ctx, userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 scenarios, got %d", len(result))
		}
	})

	t.Run("repo error", func(t *testing.T) {
		dbErr := errors.New("db error")
		repo := &testutil.MockScenarioRepo{
			ListScenariosByUserUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*scenarioEntity.Summary, error) {
				return nil, dbErr
			},
		}
		uc := scenario.NewListScenariosUC(repo)

		_, err := uc.ListScenarios(ctx, userUUID)
		if err != dbErr {
			t.Errorf("expected dbErr, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/scenario/ -v`
Expected: All tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/scenario/scenario_test.go
git commit -m "test(scenario): add use case tests for Create, Get, List

Cover validation (name length, brief desc, duplicate name),
permission checks, repo error propagation, and happy paths."
```

---

### Task 3: Campaign Use Case Tests

**Files:**
- Create: `internal/domain/campaign/campaign_test.go`

- [ ] **Step 1: Write tests**

```go
package campaign_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

func TestCreateCampaign(t *testing.T) {
	ctx := context.Background()
	masterUUID := uuid.New()

	validInput := func() *campaign.CreateCampaignInput {
		return &campaign.CreateCampaignInput{
			MasterUUID:              masterUUID,
			Name:                    "My Campaign",
			BriefInitialDescription: "Brief desc",
			Description:             "Full description",
			IsPublic:                true,
			StoryStartAt:            time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}
	}

	t.Run("success", func(t *testing.T) {
		campaignRepo := &testutil.MockCampaignRepo{}
		scenarioRepo := &testutil.MockScenarioRepo{}
		uc := campaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)

		result, err := uc.CreateCampaign(ctx, validInput())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected campaign, got nil")
		}
		if result.Name != "My Campaign" {
			t.Errorf("expected name 'My Campaign', got %q", result.Name)
		}
	})

	t.Run("name too short", func(t *testing.T) {
		campaignRepo := &testutil.MockCampaignRepo{}
		scenarioRepo := &testutil.MockScenarioRepo{}
		uc := campaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)

		input := validInput()
		input.Name = "abcd"
		_, err := uc.CreateCampaign(ctx, input)
		if !errors.Is(err, campaign.ErrMinNameLength) {
			t.Errorf("expected ErrMinNameLength, got %v", err)
		}
	})

	t.Run("name too long", func(t *testing.T) {
		campaignRepo := &testutil.MockCampaignRepo{}
		scenarioRepo := &testutil.MockScenarioRepo{}
		uc := campaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)

		input := validInput()
		input.Name = "abcdefghijklmnopqrstuvwxyz1234567"
		_, err := uc.CreateCampaign(ctx, input)
		if !errors.Is(err, campaign.ErrMaxNameLength) {
			t.Errorf("expected ErrMaxNameLength, got %v", err)
		}
	})

	t.Run("invalid start date", func(t *testing.T) {
		campaignRepo := &testutil.MockCampaignRepo{}
		scenarioRepo := &testutil.MockScenarioRepo{}
		uc := campaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)

		input := validInput()
		input.StoryStartAt = time.Time{}
		_, err := uc.CreateCampaign(ctx, input)
		if !errors.Is(err, campaign.ErrInvalidStartDate) {
			t.Errorf("expected ErrInvalidStartDate, got %v", err)
		}
	})

	t.Run("brief description too long", func(t *testing.T) {
		campaignRepo := &testutil.MockCampaignRepo{}
		scenarioRepo := &testutil.MockScenarioRepo{}
		uc := campaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)

		input := validInput()
		longDesc := make([]byte, 256)
		for i := range longDesc {
			longDesc[i] = 'a'
		}
		input.BriefInitialDescription = string(longDesc)
		_, err := uc.CreateCampaign(ctx, input)
		if !errors.Is(err, campaign.ErrMaxBriefDescLength) {
			t.Errorf("expected ErrMaxBriefDescLength, got %v", err)
		}
	})

	t.Run("max campaigns limit", func(t *testing.T) {
		campaignRepo := &testutil.MockCampaignRepo{
			CountCampaignsByMasterUUIDFn: func(_ context.Context, _ uuid.UUID) (int, error) {
				return 10, nil
			},
		}
		scenarioRepo := &testutil.MockScenarioRepo{}
		uc := campaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)

		_, err := uc.CreateCampaign(ctx, validInput())
		if !errors.Is(err, campaign.ErrMaxCampaignsLimit) {
			t.Errorf("expected ErrMaxCampaignsLimit, got %v", err)
		}
	})

	t.Run("scenario not found", func(t *testing.T) {
		campaignRepo := &testutil.MockCampaignRepo{}
		scenarioRepo := &testutil.MockScenarioRepo{
			ExistsScenarioFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
				return false, nil
			},
		}
		uc := campaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)

		input := validInput()
		scenarioUUID := uuid.New()
		input.ScenarioUUID = &scenarioUUID
		_, err := uc.CreateCampaign(ctx, input)
		if !errors.Is(err, scenario.ErrScenarioNotFound) {
			t.Errorf("expected ErrScenarioNotFound, got %v", err)
		}
	})
}

func TestGetCampaign(t *testing.T) {
	ctx := context.Background()
	masterUUID := uuid.New()
	campaignUUID := uuid.New()

	t.Run("success as owner", func(t *testing.T) {
		expected := &campaignEntity.Campaign{
			UUID:       campaignUUID,
			MasterUUID: masterUUID,
			Name:       "Test Campaign",
		}
		repo := &testutil.MockCampaignRepo{
			GetCampaignFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return expected, nil
			},
		}
		uc := campaign.NewGetCampaignUC(repo)

		result, err := uc.GetCampaign(ctx, campaignUUID, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Name != "Test Campaign" {
			t.Errorf("expected name 'Test Campaign', got %q", result.Name)
		}
	})

	t.Run("success as non-owner with public campaign", func(t *testing.T) {
		otherUser := uuid.New()
		expected := &campaignEntity.Campaign{
			UUID:       campaignUUID,
			MasterUUID: masterUUID,
			IsPublic:   true,
		}
		repo := &testutil.MockCampaignRepo{
			GetCampaignFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return expected, nil
			},
		}
		uc := campaign.NewGetCampaignUC(repo)

		_, err := uc.GetCampaign(ctx, campaignUUID, otherUser)
		if err != nil {
			t.Fatalf("expected no error for public campaign, got %v", err)
		}
	})

	t.Run("insufficient permissions for private campaign", func(t *testing.T) {
		otherUser := uuid.New()
		expected := &campaignEntity.Campaign{
			UUID:       campaignUUID,
			MasterUUID: masterUUID,
			IsPublic:   false,
		}
		repo := &testutil.MockCampaignRepo{
			GetCampaignFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return expected, nil
			},
		}
		uc := campaign.NewGetCampaignUC(repo)

		_, err := uc.GetCampaign(ctx, campaignUUID, otherUser)
		if err == nil {
			t.Fatal("expected error for non-owner on private campaign")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &testutil.MockCampaignRepo{
			GetCampaignFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return nil, pgCampaign.ErrCampaignNotFound
			},
		}
		uc := campaign.NewGetCampaignUC(repo)

		_, err := uc.GetCampaign(ctx, campaignUUID, masterUUID)
		if !errors.Is(err, campaign.ErrCampaignNotFound) {
			t.Errorf("expected ErrCampaignNotFound, got %v", err)
		}
	})
}

func TestListCampaigns(t *testing.T) {
	ctx := context.Background()
	userUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		expected := []*campaignEntity.Summary{
			{UUID: uuid.New(), Name: "Campaign A"},
		}
		repo := &testutil.MockCampaignRepo{
			ListCampaignsByMasterUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*campaignEntity.Summary, error) {
				return expected, nil
			},
		}
		uc := campaign.NewListCampaignsUC(repo)

		result, err := uc.ListCampaigns(ctx, userUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 campaign, got %d", len(result))
		}
	})

	t.Run("repo error", func(t *testing.T) {
		dbErr := errors.New("db error")
		repo := &testutil.MockCampaignRepo{
			ListCampaignsByMasterUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*campaignEntity.Summary, error) {
				return nil, dbErr
			},
		}
		uc := campaign.NewListCampaignsUC(repo)

		_, err := uc.ListCampaigns(ctx, userUUID)
		if err != dbErr {
			t.Errorf("expected dbErr, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/campaign/ -v`
Expected: All tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/campaign/campaign_test.go
git commit -m "test(campaign): add use case tests for Create, Get, List

Cover validation (name, date, brief desc, limit), scenario lookup,
permission checks (owner vs public), error translation, repo errors."
```

---

### Task 4: Match Use Case Tests

**Files:**
- Create: `internal/domain/match/match_uc_test.go`

- [ ] **Step 1: Write tests**

```go
package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	pgMatch "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

func TestCreateMatch(t *testing.T) {
	ctx := context.Background()
	masterUUID := uuid.New()
	campaignUUID := uuid.New()

	campaignStoryStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	validInput := func() *match.CreateMatchInput {
		return &match.CreateMatchInput{
			MasterUUID:              masterUUID,
			CampaignUUID:            campaignUUID,
			Title:                   "Test Match",
			BriefInitialDescription: "Brief",
			Description:             "Full description",
			IsPublic:                true,
			GameStartAt:             time.Now().Add(24 * time.Hour),
			StoryStartAt:            campaignStoryStart.Add(time.Hour),
		}
	}

	defaultCampaignRepo := func() *testutil.MockCampaignRepo {
		return &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return &campaignEntity.Campaign{
					MasterUUID:   masterUUID,
					StoryStartAt: campaignStoryStart,
				}, nil
			},
		}
	}

	t.Run("success", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{}
		campaignRepo := defaultCampaignRepo()
		uc := match.NewCreateMatchUC(matchRepo, campaignRepo)

		result, err := uc.CreateMatch(ctx, validInput())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result == nil {
			t.Fatal("expected match, got nil")
		}
	})

	t.Run("title too short", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{}
		campaignRepo := defaultCampaignRepo()
		uc := match.NewCreateMatchUC(matchRepo, campaignRepo)

		input := validInput()
		input.Title = "abcd"
		_, err := uc.CreateMatch(ctx, input)
		if !errors.Is(err, match.ErrMinTitleLength) {
			t.Errorf("expected ErrMinTitleLength, got %v", err)
		}
	})

	t.Run("title too long", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{}
		campaignRepo := defaultCampaignRepo()
		uc := match.NewCreateMatchUC(matchRepo, campaignRepo)

		input := validInput()
		input.Title = "abcdefghijklmnopqrstuvwxyz1234567"
		_, err := uc.CreateMatch(ctx, input)
		if !errors.Is(err, match.ErrMaxTitleLength) {
			t.Errorf("expected ErrMaxTitleLength, got %v", err)
		}
	})

	t.Run("brief description too long", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{}
		campaignRepo := defaultCampaignRepo()
		uc := match.NewCreateMatchUC(matchRepo, campaignRepo)

		input := validInput()
		longDesc := make([]byte, 256)
		for i := range longDesc {
			longDesc[i] = 'a'
		}
		input.BriefInitialDescription = string(longDesc)
		_, err := uc.CreateMatch(ctx, input)
		if !errors.Is(err, match.ErrMaxBriefDescLength) {
			t.Errorf("expected ErrMaxBriefDescLength, got %v", err)
		}
	})

	t.Run("game start in the past", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{}
		campaignRepo := defaultCampaignRepo()
		uc := match.NewCreateMatchUC(matchRepo, campaignRepo)

		input := validInput()
		input.GameStartAt = time.Now().Add(-1 * time.Hour)
		_, err := uc.CreateMatch(ctx, input)
		if !errors.Is(err, match.ErrMinOfGameStartAt) {
			t.Errorf("expected ErrMinOfGameStartAt, got %v", err)
		}
	})

	t.Run("game start too far in future", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{}
		campaignRepo := defaultCampaignRepo()
		uc := match.NewCreateMatchUC(matchRepo, campaignRepo)

		input := validInput()
		input.GameStartAt = time.Now().AddDate(1, 0, 1)
		_, err := uc.CreateMatch(ctx, input)
		if !errors.Is(err, match.ErrMaxOfGameStartAt) {
			t.Errorf("expected ErrMaxOfGameStartAt, got %v", err)
		}
	})

	t.Run("campaign not found", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return nil, pgCampaign.ErrCampaignNotFound
			},
		}
		uc := match.NewCreateMatchUC(matchRepo, campaignRepo)

		_, err := uc.CreateMatch(ctx, validInput())
		if !errors.Is(err, campaignDomain.ErrCampaignNotFound) {
			t.Errorf("expected ErrCampaignNotFound, got %v", err)
		}
	})

	t.Run("not campaign owner", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{}
		otherMaster := uuid.New()
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return &campaignEntity.Campaign{
					MasterUUID:   otherMaster,
					StoryStartAt: campaignStoryStart,
				}, nil
			},
		}
		uc := match.NewCreateMatchUC(matchRepo, campaignRepo)

		_, err := uc.CreateMatch(ctx, validInput())
		if !errors.Is(err, campaignDomain.ErrNotCampaignOwner) {
			t.Errorf("expected ErrNotCampaignOwner, got %v", err)
		}
	})

	t.Run("story start before campaign start", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{}
		campaignRepo := defaultCampaignRepo()
		uc := match.NewCreateMatchUC(matchRepo, campaignRepo)

		input := validInput()
		input.StoryStartAt = campaignStoryStart.Add(-1 * time.Hour)
		_, err := uc.CreateMatch(ctx, input)
		if !errors.Is(err, match.ErrMinOfStoryStartAt) {
			t.Errorf("expected ErrMinOfStoryStartAt, got %v", err)
		}
	})
}

func TestGetMatch(t *testing.T) {
	ctx := context.Background()
	masterUUID := uuid.New()
	matchUUID := uuid.New()

	t.Run("success as owner", func(t *testing.T) {
		expected := &matchEntity.Match{
			UUID:       matchUUID,
			MasterUUID: masterUUID,
			Title:      "My Match",
		}
		repo := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return expected, nil
			},
		}
		uc := match.NewGetMatchUC(repo)

		result, err := uc.GetMatch(ctx, matchUUID, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.Title != "My Match" {
			t.Errorf("expected title 'My Match', got %q", result.Title)
		}
	})

	t.Run("success for public match as non-owner", func(t *testing.T) {
		otherUser := uuid.New()
		expected := &matchEntity.Match{
			UUID:       matchUUID,
			MasterUUID: masterUUID,
			IsPublic:   true,
		}
		repo := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return expected, nil
			},
		}
		uc := match.NewGetMatchUC(repo)

		_, err := uc.GetMatch(ctx, matchUUID, otherUser)
		if err != nil {
			t.Fatalf("expected no error for public match, got %v", err)
		}
	})

	t.Run("insufficient permissions for private match", func(t *testing.T) {
		otherUser := uuid.New()
		expected := &matchEntity.Match{
			UUID:       matchUUID,
			MasterUUID: masterUUID,
			IsPublic:   false,
		}
		repo := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return expected, nil
			},
		}
		uc := match.NewGetMatchUC(repo)

		_, err := uc.GetMatch(ctx, matchUUID, otherUser)
		if err == nil {
			t.Fatal("expected error for non-owner on private match")
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return nil, pgMatch.ErrMatchNotFound
			},
		}
		uc := match.NewGetMatchUC(repo)

		_, err := uc.GetMatch(ctx, matchUUID, masterUUID)
		if !errors.Is(err, match.ErrMatchNotFound) {
			t.Errorf("expected ErrMatchNotFound, got %v", err)
		}
	})
}

func TestListMatches(t *testing.T) {
	ctx := context.Background()
	masterUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		expected := []*matchEntity.Summary{{Title: "Match 1"}}
		repo := &testutil.MockMatchRepo{
			ListMatchesByMasterUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Summary, error) {
				return expected, nil
			},
		}
		uc := match.NewListMatchesUC(repo)

		result, err := uc.ListMatches(ctx, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 match, got %d", len(result))
		}
	})
}

func TestListPublicUpcomingMatches(t *testing.T) {
	ctx := context.Background()
	masterUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		expected := []*matchEntity.Summary{{Title: "Public Match"}}
		repo := &testutil.MockMatchRepo{
			ListPublicUpcomingMatchesFn: func(_ context.Context, _ time.Time, _ uuid.UUID) ([]*matchEntity.Summary, error) {
				return expected, nil
			},
		}
		uc := match.NewListPublicUpcomingMatchesUC(repo)

		result, err := uc.ListPublicUpcomingMatches(ctx, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 match, got %d", len(result))
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/match/ -v`
Expected: All tests PASS (entity tests + UC tests)

- [ ] **Step 3: Commit**

```bash
git add internal/domain/match/match_uc_test.go
git commit -m "test(match): add use case tests for Create, Get, List, ListPublicUpcoming

Cover title/date validations, campaign ownership, story date range,
permission checks (owner vs public), error translation."
```

---

### Task 5: Auth Use Case Tests (Register)

**Files:**
- Create: `internal/domain/auth/auth_test.go`

- [ ] **Step 1: Write Register tests**

```go
package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
)

func TestRegister(t *testing.T) {
	ctx := context.Background()

	validInput := func() *auth.RegisterInput {
		return &auth.RegisterInput{
			Nick:        "testuser",
			Email:       "test@example.com",
			Password:    "password123",
			ConfirmPass: "password123",
		}
	}

	t.Run("success", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		err := uc.Register(ctx, validInput())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("missing nick", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		input := validInput()
		input.Nick = ""
		err := uc.Register(ctx, input)
		if !errors.Is(err, user.ErrMissingNick) {
			t.Errorf("expected ErrMissingNick, got %v", err)
		}
	})

	t.Run("nick too short", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		input := validInput()
		input.Nick = "ab"
		err := uc.Register(ctx, input)
		if !errors.Is(err, user.ErrInvalidNickLength) {
			t.Errorf("expected ErrInvalidNickLength, got %v", err)
		}
	})

	t.Run("nick too long", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		input := validInput()
		input.Nick = "abcdefghijklmnopqrstu"
		err := uc.Register(ctx, input)
		if !errors.Is(err, user.ErrInvalidNickLength) {
			t.Errorf("expected ErrInvalidNickLength, got %v", err)
		}
	})

	t.Run("missing email", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		input := validInput()
		input.Email = ""
		err := uc.Register(ctx, input)
		if !errors.Is(err, user.ErrMissingEmail) {
			t.Errorf("expected ErrMissingEmail, got %v", err)
		}
	})

	t.Run("email too short", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		input := validInput()
		input.Email = "short@e.com"
		err := uc.Register(ctx, input)
		if !errors.Is(err, user.ErrInvalidEmailLength) {
			t.Errorf("expected ErrInvalidEmailLength, got %v", err)
		}
	})

	t.Run("missing password", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		input := validInput()
		input.Password = ""
		err := uc.Register(ctx, input)
		if !errors.Is(err, user.ErrMissingPassword) {
			t.Errorf("expected ErrMissingPassword, got %v", err)
		}
	})

	t.Run("missing confirm password", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		input := validInput()
		input.ConfirmPass = ""
		err := uc.Register(ctx, input)
		if !errors.Is(err, user.ErrMissingConfirmPass) {
			t.Errorf("expected ErrMissingConfirmPass, got %v", err)
		}
	})

	t.Run("password too short", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		input := validInput()
		input.Password = "short"
		input.ConfirmPass = "short"
		err := uc.Register(ctx, input)
		if !errors.Is(err, user.ErrPasswordMinLenght) {
			t.Errorf("expected ErrPasswordMinLenght, got %v", err)
		}
	})

	t.Run("password mismatch", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		uc := auth.NewRegisterUC(repo)

		input := validInput()
		input.ConfirmPass = "different123"
		err := uc.Register(ctx, input)
		if !errors.Is(err, user.ErrMismatchPassword) {
			t.Errorf("expected ErrMismatchPassword, got %v", err)
		}
	})

	t.Run("nick already exists", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{
			ExistsUserWithNickFn: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
		}
		uc := auth.NewRegisterUC(repo)

		err := uc.Register(ctx, validInput())
		if !errors.Is(err, user.ErrNickAlreadyExists) {
			t.Errorf("expected ErrNickAlreadyExists, got %v", err)
		}
	})

	t.Run("email already exists", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{
			ExistsUserWithEmailFn: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
		}
		uc := auth.NewRegisterUC(repo)

		err := uc.Register(ctx, validInput())
		if !errors.Is(err, user.ErrEmailAlreadyExists) {
			t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
		}
	})

	t.Run("repo error on create", func(t *testing.T) {
		dbErr := errors.New("db error")
		repo := &testutil.MockAuthRepo{
			CreateUserFn: func(_ context.Context, _ *user.User) error {
				return dbErr
			},
		}
		uc := auth.NewRegisterUC(repo)

		err := uc.Register(ctx, validInput())
		if err != dbErr {
			t.Errorf("expected dbErr, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/auth/ -v -run TestRegister`
Expected: All tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/auth/auth_test.go
git commit -m "test(auth): add Register use case tests

Cover all validation errors (nick, email, password length/mismatch),
duplicate checks, and repo error propagation."
```

---

### Task 6: Auth Use Case Tests (Login)

**Files:**
- Modify: `internal/domain/auth/auth_test.go`

- [ ] **Step 1: Add Login tests to auth_test.go**

Append after TestRegister:

```go
func TestLogin(t *testing.T) {
	ctx := context.Background()

	t.Run("missing email", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		sessionRepo := &testutil.MockSessionRepo{}
		sessions := &sync.Map{}
		uc := auth.NewLoginUC(sessions, repo, sessionRepo)

		_, err := uc.Login(ctx, &auth.LoginInput{
			Email:    "",
			Password: "password123",
		})
		if !errors.Is(err, user.ErrMissingEmail) {
			t.Errorf("expected ErrMissingEmail, got %v", err)
		}
	})

	t.Run("email too short", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		sessionRepo := &testutil.MockSessionRepo{}
		sessions := &sync.Map{}
		uc := auth.NewLoginUC(sessions, repo, sessionRepo)

		_, err := uc.Login(ctx, &auth.LoginInput{
			Email:    "short@e.com",
			Password: "password123",
		})
		if !errors.Is(err, user.ErrInvalidEmailLength) {
			t.Errorf("expected ErrInvalidEmailLength, got %v", err)
		}
	})

	t.Run("missing password", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		sessionRepo := &testutil.MockSessionRepo{}
		sessions := &sync.Map{}
		uc := auth.NewLoginUC(sessions, repo, sessionRepo)

		_, err := uc.Login(ctx, &auth.LoginInput{
			Email:    "valid@example.com",
			Password: "",
		})
		if !errors.Is(err, user.ErrMissingPassword) {
			t.Errorf("expected ErrMissingPassword, got %v", err)
		}
	})

	t.Run("password too short", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{}
		sessionRepo := &testutil.MockSessionRepo{}
		sessions := &sync.Map{}
		uc := auth.NewLoginUC(sessions, repo, sessionRepo)

		_, err := uc.Login(ctx, &auth.LoginInput{
			Email:    "valid@example.com",
			Password: "short",
		})
		if !errors.Is(err, user.ErrPasswordMinLenght) {
			t.Errorf("expected ErrPasswordMinLenght, got %v", err)
		}
	})

	t.Run("user not found returns unauthorized", func(t *testing.T) {
		repo := &testutil.MockAuthRepo{
			GetUserByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
				return nil, pgUser.ErrEmailNotFound
			},
		}
		sessionRepo := &testutil.MockSessionRepo{}
		sessions := &sync.Map{}
		uc := auth.NewLoginUC(sessions, repo, sessionRepo)

		_, err := uc.Login(ctx, &auth.LoginInput{
			Email:    "notfound@example.com",
			Password: "password123",
		})
		if !errors.Is(err, auth.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("wrong password returns unauthorized", func(t *testing.T) {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
		repo := &testutil.MockAuthRepo{
			GetUserByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
				return &user.User{
					UUID:     uuid.New(),
					Email:    "user@example.com",
					Password: string(hashedPassword),
				}, nil
			},
		}
		sessionRepo := &testutil.MockSessionRepo{}
		sessions := &sync.Map{}
		uc := auth.NewLoginUC(sessions, repo, sessionRepo)

		_, err := uc.Login(ctx, &auth.LoginInput{
			Email:    "user@example.com",
			Password: "wrongpassword1",
		})
		if !errors.Is(err, auth.ErrUnauthorized) {
			t.Errorf("expected ErrUnauthorized, got %v", err)
		}
	})

	t.Run("successful login", func(t *testing.T) {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		userUUID := uuid.New()
		repo := &testutil.MockAuthRepo{
			GetUserByEmailFn: func(_ context.Context, _ string) (*user.User, error) {
				return &user.User{
					UUID:     userUUID,
					Email:    "user@example.com",
					Password: string(hashedPassword),
				}, nil
			},
		}
		sessionRepo := &testutil.MockSessionRepo{}
		sessions := &sync.Map{}
		uc := auth.NewLoginUC(sessions, repo, sessionRepo)

		output, err := uc.Login(ctx, &auth.LoginInput{
			Email:    "user@example.com",
			Password: "password123",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if output.Token == "" {
			t.Error("expected non-empty token")
		}
		if output.User == nil {
			t.Error("expected user in output")
		}
		if output.User.UUID != userUUID {
			t.Errorf("expected user UUID %v, got %v", userUUID, output.User.UUID)
		}

		// Verify session was stored in sync.Map
		stored, ok := sessions.Load(userUUID)
		if !ok {
			t.Error("expected session stored in sync.Map")
		}
		if stored != output.Token {
			t.Errorf("expected stored token %q, got %q", output.Token, stored)
		}
	})
}
```

Note: Add these imports to the file header:

```go
import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgUser "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/user"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/auth/ -v`
Expected: All tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/auth/auth_test.go
git commit -m "test(auth): add Login use case tests

Cover validation, user-not-found → unauthorized, wrong password,
successful login with token generation and session storage."
```

---

### Task 7: Submission Use Case Tests

**Files:**
- Create: `internal/domain/submission/submission_test.go`

- [ ] **Step 1: Write tests**

```go
package submission_test

import (
	"context"
	"errors"
	"testing"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/submission"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	pgSheet "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	pgSubmission "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/submission"
	"github.com/google/uuid"
)

func TestSubmitCharacterSheet(t *testing.T) {
	ctx := context.Background()
	playerUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()
	masterUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &testutil.MockSubmissionRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return masterUUID, nil
			},
		}
		uc := submission.NewSubmitCharacterSheetUC(repo, sheetRepo, campaignRepo)

		err := uc.Submit(ctx, playerUUID, sheetUUID, campaignUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("sheet not found", func(t *testing.T) {
		repo := &testutil.MockSubmissionRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, pgSheet.ErrCharacterSheetNotFound
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{}
		uc := submission.NewSubmitCharacterSheetUC(repo, sheetRepo, campaignRepo)

		err := uc.Submit(ctx, playerUUID, sheetUUID, campaignUUID)
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
			t.Errorf("expected ErrCharacterSheetNotFound, got %v", err)
		}
	})

	t.Run("not sheet owner", func(t *testing.T) {
		otherPlayer := uuid.New()
		repo := &testutil.MockSubmissionRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return otherPlayer, nil
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{}
		uc := submission.NewSubmitCharacterSheetUC(repo, sheetRepo, campaignRepo)

		err := uc.Submit(ctx, playerUUID, sheetUUID, campaignUUID)
		if !errors.Is(err, charactersheet.ErrNotCharacterSheetOwner) {
			t.Errorf("expected ErrNotCharacterSheetOwner, got %v", err)
		}
	})

	t.Run("already submitted", func(t *testing.T) {
		repo := &testutil.MockSubmissionRepo{
			ExistsSubmittedCharacterSheetFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{}
		uc := submission.NewSubmitCharacterSheetUC(repo, sheetRepo, campaignRepo)

		err := uc.Submit(ctx, playerUUID, sheetUUID, campaignUUID)
		if !errors.Is(err, submission.ErrCharacterAlreadySubmitted) {
			t.Errorf("expected ErrCharacterAlreadySubmitted, got %v", err)
		}
	})

	t.Run("master cannot submit own sheet", func(t *testing.T) {
		repo := &testutil.MockSubmissionRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
		}
		uc := submission.NewSubmitCharacterSheetUC(repo, sheetRepo, campaignRepo)

		err := uc.Submit(ctx, playerUUID, sheetUUID, campaignUUID)
		if !errors.Is(err, submission.ErrMasterCannotSubmitOwnSheet) {
			t.Errorf("expected ErrMasterCannotSubmitOwnSheet, got %v", err)
		}
	})
}

func TestAcceptCharacterSheetSubmission(t *testing.T) {
	ctx := context.Background()
	masterUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &testutil.MockSubmissionRepo{
			GetSubmissionCampaignUUIDBySheetUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return campaignUUID, nil
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return masterUUID, nil
			},
		}
		uc := submission.NewAcceptCharacterSheetSubmissionUC(repo, campaignRepo)

		err := uc.Accept(ctx, sheetUUID, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("submission not found", func(t *testing.T) {
		repo := &testutil.MockSubmissionRepo{
			GetSubmissionCampaignUUIDBySheetUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, pgSubmission.ErrSubmissionNotFound
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{}
		uc := submission.NewAcceptCharacterSheetSubmissionUC(repo, campaignRepo)

		err := uc.Accept(ctx, sheetUUID, masterUUID)
		if !errors.Is(err, submission.ErrSubmissionNotFound) {
			t.Errorf("expected ErrSubmissionNotFound, got %v", err)
		}
	})

	t.Run("campaign not found", func(t *testing.T) {
		repo := &testutil.MockSubmissionRepo{
			GetSubmissionCampaignUUIDBySheetUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return campaignUUID, nil
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, pgCampaign.ErrCampaignNotFound
			},
		}
		uc := submission.NewAcceptCharacterSheetSubmissionUC(repo, campaignRepo)

		err := uc.Accept(ctx, sheetUUID, masterUUID)
		if !errors.Is(err, campaignDomain.ErrCampaignNotFound) {
			t.Errorf("expected ErrCampaignNotFound, got %v", err)
		}
	})

	t.Run("not campaign master", func(t *testing.T) {
		otherMaster := uuid.New()
		repo := &testutil.MockSubmissionRepo{
			GetSubmissionCampaignUUIDBySheetUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return campaignUUID, nil
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return otherMaster, nil
			},
		}
		uc := submission.NewAcceptCharacterSheetSubmissionUC(repo, campaignRepo)

		err := uc.Accept(ctx, sheetUUID, masterUUID)
		if !errors.Is(err, submission.ErrNotCampaignMaster) {
			t.Errorf("expected ErrNotCampaignMaster, got %v", err)
		}
	})
}

func TestRejectCharacterSheetSubmission(t *testing.T) {
	ctx := context.Background()
	masterUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &testutil.MockSubmissionRepo{
			GetSubmissionCampaignUUIDBySheetUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return campaignUUID, nil
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return masterUUID, nil
			},
		}
		uc := submission.NewRejectCharacterSheetSubmissionUC(repo, campaignRepo)

		err := uc.Reject(ctx, sheetUUID, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("submission not found", func(t *testing.T) {
		repo := &testutil.MockSubmissionRepo{
			GetSubmissionCampaignUUIDBySheetUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, pgSubmission.ErrSubmissionNotFound
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{}
		uc := submission.NewRejectCharacterSheetSubmissionUC(repo, campaignRepo)

		err := uc.Reject(ctx, sheetUUID, masterUUID)
		if !errors.Is(err, submission.ErrSubmissionNotFound) {
			t.Errorf("expected ErrSubmissionNotFound, got %v", err)
		}
	})

	t.Run("not campaign master", func(t *testing.T) {
		otherMaster := uuid.New()
		repo := &testutil.MockSubmissionRepo{
			GetSubmissionCampaignUUIDBySheetUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return campaignUUID, nil
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return otherMaster, nil
			},
		}
		uc := submission.NewRejectCharacterSheetSubmissionUC(repo, campaignRepo)

		err := uc.Reject(ctx, sheetUUID, masterUUID)
		if !errors.Is(err, submission.ErrNotCampaignMaster) {
			t.Errorf("expected ErrNotCampaignMaster, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/submission/ -v`
Expected: All tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/submission/submission_test.go
git commit -m "test(submission): add Submit, Accept, Reject use case tests

Cover ownership validation, duplicate submission, master self-submit,
submission lookup, campaign master authorization, error translation."
```

---

### Task 8: Enrollment Use Case Tests

**Files:**
- Create: `internal/domain/enrollment/enrollment_test.go`

- [ ] **Step 1: Write tests**

```go
package enrollment_test

import (
	"context"
	"errors"
	"testing"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgMatch "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	pgSheet "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

func TestEnrollCharacterInMatch(t *testing.T) {
	ctx := context.Background()
	playerUUID := uuid.New()
	sheetUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &testutil.MockEnrollmentRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(_ context.Context, _ uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
				return model.CharacterSheetRelationshipUUIDs{
					PlayerUUID:   &playerUUID,
					CampaignUUID: &campaignUUID,
				}, nil
			},
		}
		matchRepo := &testutil.MockMatchRepo{
			GetMatchCampaignUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return campaignUUID, nil
			},
		}
		uc := enrollment.NewEnrollCharacterInMatchUC(repo, matchRepo, sheetRepo)

		err := uc.Enroll(ctx, matchUUID, sheetUUID, playerUUID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("sheet not found", func(t *testing.T) {
		repo := &testutil.MockEnrollmentRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(_ context.Context, _ uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
				return model.CharacterSheetRelationshipUUIDs{}, pgSheet.ErrCharacterSheetNotFound
			},
		}
		matchRepo := &testutil.MockMatchRepo{}
		uc := enrollment.NewEnrollCharacterInMatchUC(repo, matchRepo, sheetRepo)

		err := uc.Enroll(ctx, matchUUID, sheetUUID, playerUUID)
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
			t.Errorf("expected ErrCharacterSheetNotFound, got %v", err)
		}
	})

	t.Run("not sheet owner", func(t *testing.T) {
		otherPlayer := uuid.New()
		repo := &testutil.MockEnrollmentRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(_ context.Context, _ uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
				return model.CharacterSheetRelationshipUUIDs{
					PlayerUUID:   &otherPlayer,
					CampaignUUID: &campaignUUID,
				}, nil
			},
		}
		matchRepo := &testutil.MockMatchRepo{}
		uc := enrollment.NewEnrollCharacterInMatchUC(repo, matchRepo, sheetRepo)

		err := uc.Enroll(ctx, matchUUID, sheetUUID, playerUUID)
		if !errors.Is(err, charactersheet.ErrNotCharacterSheetOwner) {
			t.Errorf("expected ErrNotCharacterSheetOwner, got %v", err)
		}
	})

	t.Run("nil player UUID", func(t *testing.T) {
		repo := &testutil.MockEnrollmentRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(_ context.Context, _ uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
				return model.CharacterSheetRelationshipUUIDs{
					PlayerUUID:   nil,
					CampaignUUID: &campaignUUID,
				}, nil
			},
		}
		matchRepo := &testutil.MockMatchRepo{}
		uc := enrollment.NewEnrollCharacterInMatchUC(repo, matchRepo, sheetRepo)

		err := uc.Enroll(ctx, matchUUID, sheetUUID, playerUUID)
		if !errors.Is(err, charactersheet.ErrNotCharacterSheetOwner) {
			t.Errorf("expected ErrNotCharacterSheetOwner, got %v", err)
		}
	})

	t.Run("already enrolled", func(t *testing.T) {
		repo := &testutil.MockEnrollmentRepo{
			ExistsEnrolledCharacterSheetFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(_ context.Context, _ uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
				return model.CharacterSheetRelationshipUUIDs{
					PlayerUUID:   &playerUUID,
					CampaignUUID: &campaignUUID,
				}, nil
			},
		}
		matchRepo := &testutil.MockMatchRepo{}
		uc := enrollment.NewEnrollCharacterInMatchUC(repo, matchRepo, sheetRepo)

		err := uc.Enroll(ctx, matchUUID, sheetUUID, playerUUID)
		if !errors.Is(err, enrollment.ErrCharacterAlreadyEnrolled) {
			t.Errorf("expected ErrCharacterAlreadyEnrolled, got %v", err)
		}
	})

	t.Run("match not found", func(t *testing.T) {
		repo := &testutil.MockEnrollmentRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(_ context.Context, _ uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
				return model.CharacterSheetRelationshipUUIDs{
					PlayerUUID:   &playerUUID,
					CampaignUUID: &campaignUUID,
				}, nil
			},
		}
		matchRepo := &testutil.MockMatchRepo{
			GetMatchCampaignUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, pgMatch.ErrMatchNotFound
			},
		}
		uc := enrollment.NewEnrollCharacterInMatchUC(repo, matchRepo, sheetRepo)

		err := uc.Enroll(ctx, matchUUID, sheetUUID, playerUUID)
		if !errors.Is(err, matchDomain.ErrMatchNotFound) {
			t.Errorf("expected ErrMatchNotFound, got %v", err)
		}
	})

	t.Run("character not in campaign", func(t *testing.T) {
		differentCampaign := uuid.New()
		repo := &testutil.MockEnrollmentRepo{}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(_ context.Context, _ uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
				return model.CharacterSheetRelationshipUUIDs{
					PlayerUUID:   &playerUUID,
					CampaignUUID: &differentCampaign,
				}, nil
			},
		}
		matchRepo := &testutil.MockMatchRepo{
			GetMatchCampaignUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return campaignUUID, nil
			},
		}
		uc := enrollment.NewEnrollCharacterInMatchUC(repo, matchRepo, sheetRepo)

		err := uc.Enroll(ctx, matchUUID, sheetUUID, playerUUID)
		if !errors.Is(err, enrollment.ErrCharacterNotInCampaign) {
			t.Errorf("expected ErrCharacterNotInCampaign, got %v", err)
		}
	})

	t.Run("repo error propagation", func(t *testing.T) {
		dbErr := errors.New("db error")
		repo := &testutil.MockEnrollmentRepo{
			EnrollCharacterSheetFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
				return dbErr
			},
		}
		sheetRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(_ context.Context, _ uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
				return model.CharacterSheetRelationshipUUIDs{
					PlayerUUID:   &playerUUID,
					CampaignUUID: &campaignUUID,
				}, nil
			},
		}
		matchRepo := &testutil.MockMatchRepo{
			GetMatchCampaignUUIDFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
				return campaignUUID, nil
			},
		}
		uc := enrollment.NewEnrollCharacterInMatchUC(repo, matchRepo, sheetRepo)

		err := uc.Enroll(ctx, matchUUID, sheetUUID, playerUUID)
		if err != dbErr {
			t.Errorf("expected dbErr, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/domain/enrollment/ -v`
Expected: All tests PASS

- [ ] **Step 3: Commit**

```bash
git add internal/domain/enrollment/enrollment_test.go
git commit -m "test(enrollment): add Enroll use case tests

Cover ownership validation, nil player UUID, duplicate enrollment,
match lookup, campaign membership check, error propagation."
```

---

### Task 9: Final Verification & Documentation

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`
Expected: All packages PASS except `match/turn` (pre-existing failure)

- [ ] **Step 2: Commit plan**

```bash
git add docs/superpowers/plans/2026-04-28-domain-usecases-tests.md
git commit -m "docs(plans): add domain use cases test implementation plan"
```

- [ ] **Step 3: Create game documentation for use cases**

Create `docs/game/plataforma/fluxos.md` (PT-BR) documenting the platform workflows:
- Registration, login, session management
- Campaign creation, listing, access control
- Match creation, public/private visibility
- Character sheet submission workflow
- Enrollment workflow

- [ ] **Step 4: Create design spec (EN + PT-BR)**

Both in same commit per project convention.

- [ ] **Step 5: Final commit with docs**

```bash
git add docs/
git commit -m "docs: add domain use cases design spec and platform workflow docs"
```
