package submission_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/submission"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
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
				GetCharacterSheetNickFn: func(ctx context.Context, id uuid.UUID) (string, error) {
					return "Gon", nil
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
			name:         "nick already in campaign",
			userUUID:     playerUUID,
			sheetUUID:    sheetUUID,
			campaignUUID: campaignUUID,
			subMock: &testutil.MockSubmissionRepo{
				ExistsOtherCharacterWithNickInCampaignFn: func(ctx context.Context, nick string, cUUID uuid.UUID, excludedUUID uuid.UUID) (bool, error) {
					return true, nil
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return playerUUID, nil
				},
				GetCharacterSheetNickFn: func(ctx context.Context, id uuid.UUID) (string, error) {
					return "Gon", nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      submission.ErrNickAlreadyInCampaign,
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
				GetCharacterSheetNickFn: func(ctx context.Context, id uuid.UUID) (string, error) {
					return "Gon", nil
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
				GetCharacterSheetNickFn: func(ctx context.Context, id uuid.UUID) (string, error) {
					return "Gon", nil
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
				GetCharacterSheetNickFn: func(ctx context.Context, id uuid.UUID) (string, error) {
					return "Gon", nil
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

func Test_calcBirthYear(t *testing.T) {
	tests := []struct {
		name     string
		refDate  time.Time
		birthday time.Time
		age      int
		want     int
	}{
		{
			name:     "birthday already passed this year",
			refDate:  time.Date(2045, 10, 15, 0, 0, 0, 0, time.UTC),
			birthday: time.Date(0, 3, 1, 0, 0, 0, 0, time.UTC),
			age:      25,
			want:     2020,
		},
		{
			name:     "birthday not yet this year",
			refDate:  time.Date(2045, 3, 1, 0, 0, 0, 0, time.UTC),
			birthday: time.Date(0, 12, 25, 0, 0, 0, 0, time.UTC),
			age:      25,
			want:     2019,
		},
		{
			name:     "birthday is today",
			refDate:  time.Date(2045, 7, 4, 0, 0, 0, 0, time.UTC),
			birthday: time.Date(0, 7, 4, 0, 0, 0, 0, time.UTC),
			age:      30,
			want:     2015,
		},
		{
			name:     "uses story_current_at",
			refDate:  time.Date(2060, 1, 1, 0, 0, 0, 0, time.UTC),
			birthday: time.Date(0, 6, 15, 0, 0, 0, 0, time.UTC),
			age:      40,
			want:     2019,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := submission.CalcBirthYear(tt.refDate, tt.birthday, tt.age)
			if got != tt.want {
				t.Errorf("CalcBirthYear() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAcceptCharacterSheetSubmission(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()

	storyStart := time.Date(2045, 3, 1, 0, 0, 0, 0, time.UTC)
	birthdayMonthDay := time.Date(0, 5, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		sheetUUID    uuid.UUID
		masterUUID   uuid.UUID
		subMock      *testutil.MockSubmissionRepo
		campaignMock *testutil.MockCampaignRepo
		sheetMock    *testutil.MockSheetBirthdayReader
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
				GetCampaignStoryDatesFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					return &campaignEntity.Campaign{
						StoryStartAt:   storyStart,
						StoryCurrentAt: nil,
					}, nil
				},
			},
			sheetMock: &testutil.MockSheetBirthdayReader{
				GetCharacterSheetNickFn: func(ctx context.Context, sUUID uuid.UUID) (string, error) {
					return "Gon", nil
				},
				GetCharacterSheetBirthInfoFn: func(ctx context.Context, sUUID uuid.UUID) (time.Time, int, error) {
					return birthdayMonthDay, 25, nil
				},
			},
			wantErr: nil,
		},
		{
			name:      "nick already in campaign",
			sheetUUID: sheetUUID,
			masterUUID: masterUUID,
			subMock: &testutil.MockSubmissionRepo{
				GetSubmissionCampaignUUIDBySheetUUIDFn: func(ctx context.Context, sUUID uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
				ExistsOtherCharacterWithNickInCampaignFn: func(ctx context.Context, nick string, cUUID uuid.UUID, excludedUUID uuid.UUID) (bool, error) {
					return true, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			sheetMock: &testutil.MockSheetBirthdayReader{
				GetCharacterSheetNickFn: func(ctx context.Context, sUUID uuid.UUID) (string, error) {
					return "Gon", nil
				},
			},
			wantErr: submission.ErrNickAlreadyInCampaign,
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
			sheetMock:    &testutil.MockSheetBirthdayReader{},
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
			sheetMock: &testutil.MockSheetBirthdayReader{},
			wantErr:   campaign.ErrCampaignNotFound,
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
			sheetMock: &testutil.MockSheetBirthdayReader{},
			wantErr:   submission.ErrNotCampaignMaster,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := submission.NewAcceptCharacterSheetSubmissionUC(tt.subMock, tt.campaignMock, tt.sheetMock)
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
