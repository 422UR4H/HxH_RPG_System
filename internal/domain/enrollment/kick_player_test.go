package enrollment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

func TestKickPlayer(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()
	otherUUID := uuid.New()
	now := time.Now()
	startedAt := now.Add(-time.Hour)

	tests := []struct {
		name       string
		matchUUID  uuid.UUID
		playerUUID uuid.UUID
		masterUUID uuid.UUID
		matchMock  *testutil.MockMatchRepo
		enrollMock *testutil.MockEnrollmentRepo
		wantErr    error
	}{
		{
			name:       "success",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    nil,
		},
		{
			name:       "match not found",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, matchPg.ErrMatchNotFound
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    domainMatch.ErrMatchNotFound,
		},
		{
			name:       "not match master",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: otherUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    enrollment.ErrNotMatchMaster,
		},
		{
			name:       "match already started",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
						GameStartAt:     &startedAt,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    enrollment.ErrMatchAlreadyStarted,
		},
		{
			name:       "cannot kick self (master)",
			matchUUID:  matchUUID,
			playerUUID: masterUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{},
			wantErr:    enrollment.ErrNotMatchMaster,
		},
		{
			name:       "player not enrolled",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{
				RejectEnrollmentByPlayerAndMatchFn: func(ctx context.Context, pUUID uuid.UUID, mUUID uuid.UUID) error {
					return errors.New("enrollment not found in database")
				},
			},
			wantErr: enrollment.ErrPlayerNotEnrolled,
		},
		{
			name:       "repo error on reject",
			matchUUID:  matchUUID,
			playerUUID: playerUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
					}, nil
				},
			},
			enrollMock: &testutil.MockEnrollmentRepo{
				RejectEnrollmentByPlayerAndMatchFn: func(ctx context.Context, pUUID uuid.UUID, mUUID uuid.UUID) error {
					return errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := enrollment.NewKickPlayerUC(tt.matchMock, tt.enrollMock)
			err := uc.Kick(context.Background(), tt.matchUUID, tt.playerUUID, tt.masterUUID)

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
