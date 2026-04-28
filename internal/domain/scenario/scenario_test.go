package scenario_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	scenarioPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/scenario"
	"github.com/google/uuid"
)

func TestCreateScenario(t *testing.T) {
	tests := []struct {
		name    string
		input   *scenario.CreateScenarioInput
		mock    *testutil.MockScenarioRepo
		wantErr error
	}{
		{
			name: "success",
			input: &scenario.CreateScenarioInput{
				UserUUID:         uuid.New(),
				Name:             "Valid Name",
				BriefDescription: "A brief desc",
				Description:      "Full description",
			},
			mock:    &testutil.MockScenarioRepo{},
			wantErr: nil,
		},
		{
			name: "name too short",
			input: &scenario.CreateScenarioInput{
				UserUUID: uuid.New(),
				Name:     "ab",
			},
			mock:    &testutil.MockScenarioRepo{},
			wantErr: scenario.ErrMinNameLength,
		},
		{
			name: "name too long",
			input: &scenario.CreateScenarioInput{
				UserUUID: uuid.New(),
				Name:     "this name is way too long for the limit of 32 characters",
			},
			mock:    &testutil.MockScenarioRepo{},
			wantErr: scenario.ErrMaxNameLength,
		},
		{
			name: "brief description too long",
			input: &scenario.CreateScenarioInput{
				UserUUID:         uuid.New(),
				Name:             "Valid Name",
				BriefDescription: string(make([]byte, 65)),
			},
			mock:    &testutil.MockScenarioRepo{},
			wantErr: scenario.ErrMaxBriefDescLength,
		},
		{
			name: "name already exists",
			input: &scenario.CreateScenarioInput{
				UserUUID:         uuid.New(),
				Name:             "Existing Name",
				BriefDescription: "desc",
				Description:      "full",
			},
			mock: &testutil.MockScenarioRepo{
				ExistsScenarioWithNameFn: func(ctx context.Context, name string) (bool, error) {
					return true, nil
				},
			},
			wantErr: scenario.ErrScenarioNameAlreadyExists,
		},
		{
			name: "repo ExistsScenarioWithName error",
			input: &scenario.CreateScenarioInput{
				UserUUID:         uuid.New(),
				Name:             "Valid Name",
				BriefDescription: "desc",
			},
			mock: &testutil.MockScenarioRepo{
				ExistsScenarioWithNameFn: func(ctx context.Context, name string) (bool, error) {
					return false, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name: "repo CreateScenario error",
			input: &scenario.CreateScenarioInput{
				UserUUID:         uuid.New(),
				Name:             "Valid Name",
				BriefDescription: "desc",
				Description:      "full",
			},
			mock: &testutil.MockScenarioRepo{
				CreateScenarioFn: func(ctx context.Context, s *scenarioEntity.Scenario) error {
					return errors.New("create failed")
				},
			},
			wantErr: errors.New("create failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := scenario.NewCreateScenarioUC(tt.mock)
			result, err := uc.CreateScenario(context.Background(), tt.input)

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
				t.Fatal("expected non-nil scenario")
			}
			if result.Name != tt.input.Name {
				t.Errorf("expected name %q, got %q", tt.input.Name, result.Name)
			}
			if result.UserUUID != tt.input.UserUUID {
				t.Errorf("expected userUUID %v, got %v", tt.input.UserUUID, result.UserUUID)
			}
		})
	}
}

func TestGetScenario(t *testing.T) {
	ownerUUID := uuid.New()
	otherUUID := uuid.New()
	scenarioUUID := uuid.New()

	existingScenario := &scenarioEntity.Scenario{
		UUID:     scenarioUUID,
		UserUUID: ownerUUID,
		Name:     "Test Scenario",
	}

	tests := []struct {
		name     string
		uuid     uuid.UUID
		userUUID uuid.UUID
		mock     *testutil.MockScenarioRepo
		wantErr  error
	}{
		{
			name:     "success as owner",
			uuid:     scenarioUUID,
			userUUID: ownerUUID,
			mock: &testutil.MockScenarioRepo{
				GetScenarioFn: func(ctx context.Context, id uuid.UUID) (*scenarioEntity.Scenario, error) {
					return existingScenario, nil
				},
			},
			wantErr: nil,
		},
		{
			name:     "not found",
			uuid:     uuid.New(),
			userUUID: ownerUUID,
			mock: &testutil.MockScenarioRepo{
				GetScenarioFn: func(ctx context.Context, id uuid.UUID) (*scenarioEntity.Scenario, error) {
					return nil, scenarioPg.ErrScenarioNotFound
				},
			},
			wantErr: scenario.ErrScenarioNotFound,
		},
		{
			name:     "insufficient permissions",
			uuid:     scenarioUUID,
			userUUID: otherUUID,
			mock: &testutil.MockScenarioRepo{
				GetScenarioFn: func(ctx context.Context, id uuid.UUID) (*scenarioEntity.Scenario, error) {
					return existingScenario, nil
				},
			},
			wantErr: auth.ErrInsufficientPermissions,
		},
		{
			name:     "repo error",
			uuid:     scenarioUUID,
			userUUID: ownerUUID,
			mock: &testutil.MockScenarioRepo{
				GetScenarioFn: func(ctx context.Context, id uuid.UUID) (*scenarioEntity.Scenario, error) {
					return nil, errors.New("connection lost")
				},
			},
			wantErr: errors.New("connection lost"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := scenario.NewGetScenarioUC(tt.mock)
			result, err := uc.GetScenario(context.Background(), tt.uuid, tt.userUUID)

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
				t.Fatal("expected non-nil scenario")
			}
		})
	}
}

func TestListScenarios(t *testing.T) {
	userUUID := uuid.New()

	tests := []struct {
		name    string
		mock    *testutil.MockScenarioRepo
		wantErr error
		wantLen int
	}{
		{
			name: "success with results",
			mock: &testutil.MockScenarioRepo{
				ListScenariosByUserUUIDFn: func(ctx context.Context, id uuid.UUID) ([]*scenarioEntity.Summary, error) {
					return []*scenarioEntity.Summary{{Name: "S1"}, {Name: "S2"}}, nil
				},
			},
			wantLen: 2,
		},
		{
			name: "success empty",
			mock: &testutil.MockScenarioRepo{
				ListScenariosByUserUUIDFn: func(ctx context.Context, id uuid.UUID) ([]*scenarioEntity.Summary, error) {
					return []*scenarioEntity.Summary{}, nil
				},
			},
			wantLen: 0,
		},
		{
			name: "repo error",
			mock: &testutil.MockScenarioRepo{
				ListScenariosByUserUUIDFn: func(ctx context.Context, id uuid.UUID) ([]*scenarioEntity.Summary, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := scenario.NewListScenariosUC(tt.mock)
			result, err := uc.ListScenarios(context.Background(), userUUID)

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
