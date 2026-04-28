package submission_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/submission"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	sheetPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	submissionPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/submission"
	"github.com/google/uuid"
)

func TestSubmitCharacterSheet(t *testing.T) {
	playerUUID := uuid.New()
	masterUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name         string
		userUUID     uuid.UUID
		sheetUUID    uuid.UUID
		campaignUUID uuid.UUID
		subMock      *testutil.MockSubmissionRepo
		sheetMock    *testutil.MockCharacterSheetRepo
		campaignMock *testutil.MockCampaignRepo
		wantErr      error
	}{
		{
			name:         "success",
			userUUID:     playerUUID,
			sheetUUID:    sheetUUID,
			campaignUUID: campaignUUID,
			subMock:      &testutil.MockSubmissionRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return playerUUID, nil
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
			name:         "sheet not found",
			userUUID:     playerUUID,
			sheetUUID:    sheetUUID,
			campaignUUID: campaignUUID,
			subMock:      &testutil.MockSubmissionRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, sheetPg.ErrCharacterSheetNotFound
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      charactersheet.ErrCharacterSheetNotFound,
		},
		{
			name:         "not sheet owner",
			userUUID:     uuid.New(),
			sheetUUID:    sheetUUID,
			campaignUUID: campaignUUID,
			subMock:      &testutil.MockSubmissionRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return playerUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      charactersheet.ErrNotCharacterSheetOwner,
		},
		{
			name:         "already submitted",
			userUUID:     playerUUID,
			sheetUUID:    sheetUUID,
			campaignUUID: campaignUUID,
			subMock: &testutil.MockSubmissionRepo{
				ExistsSubmittedCharacterSheetFn: func(ctx context.Context, id uuid.UUID) (bool, error) {
					return true, nil
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return playerUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      submission.ErrCharacterAlreadySubmitted,
		},
		{
			name:         "campaign not found",
			userUUID:     playerUUID,
			sheetUUID:    sheetUUID,
			campaignUUID: campaignUUID,
			subMock:      &testutil.MockSubmissionRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return playerUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, campaignPg.ErrCampaignNotFound
				},
			},
			wantErr: campaign.ErrCampaignNotFound,
		},
		{
			name:         "master cannot submit own sheet",
			userUUID:     masterUUID,
			sheetUUID:    sheetUUID,
			campaignUUID: campaignUUID,
			subMock:      &testutil.MockSubmissionRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: submission.ErrMasterCannotSubmitOwnSheet,
		},
		{
			name:         "repo submit error",
			userUUID:     playerUUID,
			sheetUUID:    sheetUUID,
			campaignUUID: campaignUUID,
			subMock: &testutil.MockSubmissionRepo{
				SubmitCharacterSheetFn: func(ctx context.Context, sUUID uuid.UUID, cUUID uuid.UUID, createdAt time.Time) error {
					return errors.New("submit failed")
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return playerUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: errors.New("submit failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := submission.NewSubmitCharacterSheetUC(tt.subMock, tt.sheetMock, tt.campaignMock)
			err := uc.Submit(context.Background(), tt.userUUID, tt.sheetUUID, tt.campaignUUID)

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

func TestAcceptCharacterSheetSubmission(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name         string
		sheetUUID    uuid.UUID
		masterUUID   uuid.UUID
		subMock      *testutil.MockSubmissionRepo
		campaignMock *testutil.MockCampaignRepo
		wantErr      error
	}{
		{
			name:       "success",
			sheetUUID:  sheetUUID,
			masterUUID: masterUUID,
			subMock: &testutil.MockSubmissionRepo{
				GetSubmissionCampaignUUIDBySheetUUIDFn: func(ctx context.Context, sUUID uuid.UUID) (uuid.UUID, error) {
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
			name:       "submission not found",
			sheetUUID:  sheetUUID,
			masterUUID: masterUUID,
			subMock: &testutil.MockSubmissionRepo{
				GetSubmissionCampaignUUIDBySheetUUIDFn: func(ctx context.Context, sUUID uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, submissionPg.ErrSubmissionNotFound
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      submission.ErrSubmissionNotFound,
		},
		{
			name:       "campaign not found",
			sheetUUID:  sheetUUID,
			masterUUID: masterUUID,
			subMock: &testutil.MockSubmissionRepo{
				GetSubmissionCampaignUUIDBySheetUUIDFn: func(ctx context.Context, sUUID uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, campaignPg.ErrCampaignNotFound
				},
			},
			wantErr: campaign.ErrCampaignNotFound,
		},
		{
			name:       "not campaign master",
			sheetUUID:  sheetUUID,
			masterUUID: otherUUID,
			subMock: &testutil.MockSubmissionRepo{
				GetSubmissionCampaignUUIDBySheetUUIDFn: func(ctx context.Context, sUUID uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: submission.ErrNotCampaignMaster,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := submission.NewAcceptCharacterSheetSubmissionUC(tt.subMock, tt.campaignMock)
			err := uc.Accept(context.Background(), tt.sheetUUID, tt.masterUUID)

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

func TestRejectCharacterSheetSubmission(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name         string
		sheetUUID    uuid.UUID
		masterUUID   uuid.UUID
		subMock      *testutil.MockSubmissionRepo
		campaignMock *testutil.MockCampaignRepo
		wantErr      error
	}{
		{
			name:       "success",
			sheetUUID:  sheetUUID,
			masterUUID: masterUUID,
			subMock: &testutil.MockSubmissionRepo{
				GetSubmissionCampaignUUIDBySheetUUIDFn: func(ctx context.Context, sUUID uuid.UUID) (uuid.UUID, error) {
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
			name:       "submission not found",
			sheetUUID:  sheetUUID,
			masterUUID: masterUUID,
			subMock: &testutil.MockSubmissionRepo{
				GetSubmissionCampaignUUIDBySheetUUIDFn: func(ctx context.Context, sUUID uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, submissionPg.ErrSubmissionNotFound
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      submission.ErrSubmissionNotFound,
		},
		{
			name:       "not campaign master",
			sheetUUID:  sheetUUID,
			masterUUID: otherUUID,
			subMock: &testutil.MockSubmissionRepo{
				GetSubmissionCampaignUUIDBySheetUUIDFn: func(ctx context.Context, sUUID uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: submission.ErrNotCampaignMaster,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := submission.NewRejectCharacterSheetSubmissionUC(tt.subMock, tt.campaignMock)
			err := uc.Reject(context.Background(), tt.sheetUUID, tt.masterUUID)

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
