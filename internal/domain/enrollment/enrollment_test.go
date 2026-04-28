package enrollment_test

import (
	"context"
	"errors"
	"testing"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
)

func TestEnrollCharacterSheet(t *testing.T) {
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	sheetUUID := uuid.New()
	campaignUUID := uuid.New()

	tests := []struct {
		name           string
		matchUUID      uuid.UUID
		sheetUUID      uuid.UUID
		playerUUID     uuid.UUID
		enrollMock     *testutil.MockEnrollmentRepo
		matchMock      *testutil.MockMatchRepo
		sheetMock      *testutil.MockCharacterSheetRepo
		wantErr        error
	}{
		{
			name:       "success",
			matchUUID:  matchUUID,
			sheetUUID:  sheetUUID,
			playerUUID: playerUUID,
			enrollMock: &testutil.MockEnrollmentRepo{},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
					return model.CharacterSheetRelationshipUUIDs{
						PlayerUUID:   &playerUUID,
						CampaignUUID: &campaignUUID,
					}, nil
				},
			},
			wantErr: nil,
		},
		{
			name:       "sheet not found",
			matchUUID:  matchUUID,
			sheetUUID:  sheetUUID,
			playerUUID: playerUUID,
			enrollMock: &testutil.MockEnrollmentRepo{},
			matchMock:  &testutil.MockMatchRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
					return model.CharacterSheetRelationshipUUIDs{}, sheet.ErrCharacterSheetNotFound
				},
			},
			wantErr: charactersheet.ErrCharacterSheetNotFound,
		},
		{
			name:       "not sheet owner",
			matchUUID:  matchUUID,
			sheetUUID:  sheetUUID,
			playerUUID: uuid.New(), // different player
			enrollMock: &testutil.MockEnrollmentRepo{},
			matchMock:  &testutil.MockMatchRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
					return model.CharacterSheetRelationshipUUIDs{
						PlayerUUID:   &playerUUID,
						CampaignUUID: &campaignUUID,
					}, nil
				},
			},
			wantErr: charactersheet.ErrNotCharacterSheetOwner,
		},
		{
			name:       "nil player UUID in relationship",
			matchUUID:  matchUUID,
			sheetUUID:  sheetUUID,
			playerUUID: playerUUID,
			enrollMock: &testutil.MockEnrollmentRepo{},
			matchMock:  &testutil.MockMatchRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
					return model.CharacterSheetRelationshipUUIDs{
						PlayerUUID:   nil,
						CampaignUUID: &campaignUUID,
					}, nil
				},
			},
			wantErr: charactersheet.ErrNotCharacterSheetOwner,
		},
		{
			name:       "already enrolled",
			matchUUID:  matchUUID,
			sheetUUID:  sheetUUID,
			playerUUID: playerUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				ExistsEnrolledCharacterSheetFn: func(ctx context.Context, sUUID uuid.UUID, mUUID uuid.UUID) (bool, error) {
					return true, nil
				},
			},
			matchMock: &testutil.MockMatchRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
					return model.CharacterSheetRelationshipUUIDs{
						PlayerUUID:   &playerUUID,
						CampaignUUID: &campaignUUID,
					}, nil
				},
			},
			wantErr: enrollment.ErrCharacterAlreadyEnrolled,
		},
		{
			name:       "match not found",
			matchUUID:  matchUUID,
			sheetUUID:  sheetUUID,
			playerUUID: playerUUID,
			enrollMock: &testutil.MockEnrollmentRepo{},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return uuid.Nil, matchPg.ErrMatchNotFound
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
					return model.CharacterSheetRelationshipUUIDs{
						PlayerUUID:   &playerUUID,
						CampaignUUID: &campaignUUID,
					}, nil
				},
			},
			wantErr: domainMatch.ErrMatchNotFound,
		},
		{
			name:       "character not in campaign",
			matchUUID:  matchUUID,
			sheetUUID:  sheetUUID,
			playerUUID: playerUUID,
			enrollMock: &testutil.MockEnrollmentRepo{},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return uuid.New(), nil // different campaign
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
					return model.CharacterSheetRelationshipUUIDs{
						PlayerUUID:   &playerUUID,
						CampaignUUID: &campaignUUID,
					}, nil
				},
			},
			wantErr: enrollment.ErrCharacterNotInCampaign,
		},
		{
			name:       "nil campaign UUID in relationship",
			matchUUID:  matchUUID,
			sheetUUID:  sheetUUID,
			playerUUID: playerUUID,
			enrollMock: &testutil.MockEnrollmentRepo{},
			matchMock: &testutil.MockMatchRepo{
				GetMatchCampaignUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return campaignUUID, nil
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
					return model.CharacterSheetRelationshipUUIDs{
						PlayerUUID:   &playerUUID,
						CampaignUUID: nil,
					}, nil
				},
			},
			wantErr: enrollment.ErrCharacterNotInCampaign,
		},
		{
			name:       "repo error on relationship fetch",
			matchUUID:  matchUUID,
			sheetUUID:  sheetUUID,
			playerUUID: playerUUID,
			enrollMock: &testutil.MockEnrollmentRepo{},
			matchMock:  &testutil.MockMatchRepo{},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (model.CharacterSheetRelationshipUUIDs, error) {
					return model.CharacterSheetRelationshipUUIDs{}, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := enrollment.NewEnrollCharacterInMatchUC(tt.enrollMock, tt.matchMock, tt.sheetMock)
			err := uc.Enroll(context.Background(), tt.matchUUID, tt.sheetUUID, tt.playerUUID)

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
