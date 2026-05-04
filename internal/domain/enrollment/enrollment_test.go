package enrollment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						CampaignUUID:    campaignUUID,
						GameScheduledAt: time.Now(),
					}, nil
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
					return csEntity.RelationshipUUIDs{
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
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
					return csEntity.RelationshipUUIDs{}, charactersheet.ErrCharacterSheetNotFound
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
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
					return csEntity.RelationshipUUIDs{
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
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
					return csEntity.RelationshipUUIDs{
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
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
					return csEntity.RelationshipUUIDs{
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, matchPg.ErrMatchNotFound
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
					return csEntity.RelationshipUUIDs{
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						CampaignUUID:    uuid.New(), // different campaign
						GameScheduledAt: time.Now(),
					}, nil
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
					return csEntity.RelationshipUUIDs{
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						CampaignUUID:    campaignUUID,
						GameScheduledAt: time.Now(),
					}, nil
				},
			},
			sheetMock: &testutil.MockCharacterSheetRepo{
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
					return csEntity.RelationshipUUIDs{
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
				GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
					return csEntity.RelationshipUUIDs{}, errors.New("db error")
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						CampaignUUID:    campaignUUID,
						GameScheduledAt: time.Now(),
					}, nil
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						CampaignUUID:    campaignUUID,
						GameScheduledAt: time.Now(),
					}, nil
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, matchPg.ErrMatchNotFound
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						CampaignUUID:    campaignUUID,
						GameScheduledAt: time.Now(),
					}, nil
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						CampaignUUID:    campaignUUID,
						GameScheduledAt: time.Now(),
					}, nil
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
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						CampaignUUID:    campaignUUID,
						GameScheduledAt: time.Now(),
					}, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
					return masterUUID, nil
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name:       "match already started",
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					gameStartAt := time.Now()
					return &matchEntity.Match{
						CampaignUUID:    campaignUUID,
						GameScheduledAt: time.Now(),
						GameStartAt:     &gameStartAt,
					}, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      enrollment.ErrMatchAlreadyStarted,
		},
		{
			name:       "match already finished",
			enrollUUID: enrollmentUUID,
			masterUUID: masterUUID,
			enrollMock: &testutil.MockEnrollmentRepo{
				GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
					return "pending", matchUUID, nil
				},
			},
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					storyEndAt := time.Now()
					return &matchEntity.Match{
						CampaignUUID:    campaignUUID,
						GameScheduledAt: time.Now(),
						StoryEndAt:      &storyEndAt,
					}, nil
				},
			},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      enrollment.ErrMatchAlreadyFinished,
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
GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
return &matchEntity.Match{
CampaignUUID:    campaignUUID,
GameScheduledAt: time.Now(),
}, nil
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
GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
return &matchEntity.Match{
CampaignUUID:    campaignUUID,
GameScheduledAt: time.Now(),
}, nil
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
GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
return nil, matchPg.ErrMatchNotFound
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
GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
return &matchEntity.Match{
CampaignUUID:    campaignUUID,
GameScheduledAt: time.Now(),
}, nil
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
GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
return &matchEntity.Match{
CampaignUUID:    campaignUUID,
GameScheduledAt: time.Now(),
}, nil
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
GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
return &matchEntity.Match{
CampaignUUID:    campaignUUID,
GameScheduledAt: time.Now(),
}, nil
},
},
campaignMock: &testutil.MockCampaignRepo{
GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
return masterUUID, nil
},
},
wantErr: errors.New("db error"),
},
{
name:       "match already started",
enrollUUID: enrollmentUUID,
masterUUID: masterUUID,
enrollMock: &testutil.MockEnrollmentRepo{
GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
return "pending", matchUUID, nil
},
},
matchMock: &testutil.MockMatchRepo{
GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
gameStartAt := time.Now()
return &matchEntity.Match{
CampaignUUID:    campaignUUID,
GameScheduledAt: time.Now(),
GameStartAt:     &gameStartAt,
}, nil
},
},
campaignMock: &testutil.MockCampaignRepo{},
wantErr:      enrollment.ErrMatchAlreadyStarted,
},
{
name:       "match already finished",
enrollUUID: enrollmentUUID,
masterUUID: masterUUID,
enrollMock: &testutil.MockEnrollmentRepo{
GetEnrollmentByUUIDFn: func(ctx context.Context, id uuid.UUID) (string, uuid.UUID, error) {
return "pending", matchUUID, nil
},
},
matchMock: &testutil.MockMatchRepo{
GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
storyEndAt := time.Now()
return &matchEntity.Match{
CampaignUUID:    campaignUUID,
GameScheduledAt: time.Now(),
StoryEndAt:      &storyEndAt,
}, nil
},
},
campaignMock: &testutil.MockCampaignRepo{},
wantErr:      enrollment.ErrMatchAlreadyFinished,
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
