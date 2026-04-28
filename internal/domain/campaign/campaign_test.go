package campaign_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

func validCreateCampaignInput() *campaign.CreateCampaignInput {
	return &campaign.CreateCampaignInput{
		MasterUUID:              uuid.New(),
		Name:                    "Valid Campaign",
		BriefInitialDescription: "Brief desc",
		Description:             "Full description",
		IsPublic:                true,
		CallLink:                "https://meet.example.com",
		StoryStartAt:            time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestCreateCampaign(t *testing.T) {
	tests := []struct {
		name         string
		input        *campaign.CreateCampaignInput
		campaignMock *testutil.MockCampaignRepo
		scenarioMock *testutil.MockScenarioRepo
		wantErr      error
	}{
		{
			name:         "success without scenario",
			input:        validCreateCampaignInput(),
			campaignMock: &testutil.MockCampaignRepo{},
			scenarioMock: &testutil.MockScenarioRepo{},
			wantErr:      nil,
		},
		{
			name: "success with valid scenario",
			input: func() *campaign.CreateCampaignInput {
				i := validCreateCampaignInput()
				scenarioID := uuid.New()
				i.ScenarioUUID = &scenarioID
				return i
			}(),
			campaignMock: &testutil.MockCampaignRepo{},
			scenarioMock: &testutil.MockScenarioRepo{
				ExistsScenarioFn: func(ctx context.Context, id uuid.UUID) (bool, error) {
					return true, nil
				},
			},
			wantErr: nil,
		},
		{
			name: "name too short",
			input: func() *campaign.CreateCampaignInput {
				i := validCreateCampaignInput()
				i.Name = "ab"
				return i
			}(),
			campaignMock: &testutil.MockCampaignRepo{},
			scenarioMock: &testutil.MockScenarioRepo{},
			wantErr:      campaign.ErrMinNameLength,
		},
		{
			name: "name too long",
			input: func() *campaign.CreateCampaignInput {
				i := validCreateCampaignInput()
				i.Name = "this name is way too long for the maximum limit"
				return i
			}(),
			campaignMock: &testutil.MockCampaignRepo{},
			scenarioMock: &testutil.MockScenarioRepo{},
			wantErr:      campaign.ErrMaxNameLength,
		},
		{
			name: "story start date is zero",
			input: func() *campaign.CreateCampaignInput {
				i := validCreateCampaignInput()
				i.StoryStartAt = time.Time{}
				return i
			}(),
			campaignMock: &testutil.MockCampaignRepo{},
			scenarioMock: &testutil.MockScenarioRepo{},
			wantErr:      campaign.ErrInvalidStartDate,
		},
		{
			name: "brief description too long",
			input: func() *campaign.CreateCampaignInput {
				i := validCreateCampaignInput()
				i.BriefInitialDescription = string(make([]byte, 256))
				return i
			}(),
			campaignMock: &testutil.MockCampaignRepo{},
			scenarioMock: &testutil.MockScenarioRepo{},
			wantErr:      campaign.ErrMaxBriefDescLength,
		},
		{
			name:  "max campaigns limit reached",
			input: validCreateCampaignInput(),
			campaignMock: &testutil.MockCampaignRepo{
				CountCampaignsByMasterUUIDFn: func(ctx context.Context, masterUUID uuid.UUID) (int, error) {
					return 10, nil
				},
			},
			scenarioMock: &testutil.MockScenarioRepo{},
			wantErr:      campaign.ErrMaxCampaignsLimit,
		},
		{
			name: "scenario not found",
			input: func() *campaign.CreateCampaignInput {
				i := validCreateCampaignInput()
				scenarioID := uuid.New()
				i.ScenarioUUID = &scenarioID
				return i
			}(),
			campaignMock: &testutil.MockCampaignRepo{},
			scenarioMock: &testutil.MockScenarioRepo{
				ExistsScenarioFn: func(ctx context.Context, id uuid.UUID) (bool, error) {
					return false, nil
				},
			},
			wantErr: scenario.ErrScenarioNotFound,
		},
		{
			name:  "repo count error",
			input: validCreateCampaignInput(),
			campaignMock: &testutil.MockCampaignRepo{
				CountCampaignsByMasterUUIDFn: func(ctx context.Context, masterUUID uuid.UUID) (int, error) {
					return 0, errors.New("db error")
				},
			},
			scenarioMock: &testutil.MockScenarioRepo{},
			wantErr:      errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := campaign.NewCreateCampaignUC(tt.campaignMock, tt.scenarioMock)
			result, err := uc.CreateCampaign(context.Background(), tt.input)

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
			if result.Name != tt.input.Name {
				t.Errorf("expected name %q, got %q", tt.input.Name, result.Name)
			}
		})
	}
}

func TestGetCampaign(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	campaignUUID := uuid.New()

	privateCampaign := &campaignEntity.Campaign{
		UUID:       campaignUUID,
		MasterUUID: masterUUID,
		Name:       "Private",
		IsPublic:   false,
	}

	publicCampaign := &campaignEntity.Campaign{
		UUID:       campaignUUID,
		MasterUUID: masterUUID,
		Name:       "Public",
		IsPublic:   true,
	}

	tests := []struct {
		name     string
		uuid     uuid.UUID
		userUUID uuid.UUID
		mock     *testutil.MockCampaignRepo
		wantErr  error
	}{
		{
			name:     "success as owner",
			uuid:     campaignUUID,
			userUUID: masterUUID,
			mock: &testutil.MockCampaignRepo{
				GetCampaignFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					return privateCampaign, nil
				},
			},
			wantErr: nil,
		},
		{
			name:     "success public campaign other user",
			uuid:     campaignUUID,
			userUUID: otherUUID,
			mock: &testutil.MockCampaignRepo{
				GetCampaignFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					return publicCampaign, nil
				},
			},
			wantErr: nil,
		},
		{
			name:     "private campaign insufficient permissions",
			uuid:     campaignUUID,
			userUUID: otherUUID,
			mock: &testutil.MockCampaignRepo{
				GetCampaignFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					return privateCampaign, nil
				},
			},
			wantErr: auth.ErrInsufficientPermissions,
		},
		{
			name:     "campaign not found",
			uuid:     uuid.New(),
			userUUID: masterUUID,
			mock: &testutil.MockCampaignRepo{
				GetCampaignFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					return nil, campaignPg.ErrCampaignNotFound
				},
			},
			wantErr: campaign.ErrCampaignNotFound,
		},
		{
			name:     "repo error",
			uuid:     campaignUUID,
			userUUID: masterUUID,
			mock: &testutil.MockCampaignRepo{
				GetCampaignFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					return nil, errors.New("connection error")
				},
			},
			wantErr: errors.New("connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := campaign.NewGetCampaignUC(tt.mock)
			result, err := uc.GetCampaign(context.Background(), tt.uuid, tt.userUUID)

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
		})
	}
}

func TestListCampaigns(t *testing.T) {
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
				ListCampaignsByMasterUUIDFn: func(ctx context.Context, id uuid.UUID) ([]*campaignEntity.Summary, error) {
					return []*campaignEntity.Summary{{Name: "C1"}, {Name: "C2"}}, nil
				},
			},
			wantLen: 2,
		},
		{
			name: "success empty",
			mock: &testutil.MockCampaignRepo{
				ListCampaignsByMasterUUIDFn: func(ctx context.Context, id uuid.UUID) ([]*campaignEntity.Summary, error) {
					return []*campaignEntity.Summary{}, nil
				},
			},
			wantLen: 0,
		},
		{
			name: "repo error",
			mock: &testutil.MockCampaignRepo{
				ListCampaignsByMasterUUIDFn: func(ctx context.Context, id uuid.UUID) ([]*campaignEntity.Summary, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := campaign.NewListCampaignsUC(tt.mock)
			result, err := uc.ListCampaigns(context.Background(), userUUID)

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


