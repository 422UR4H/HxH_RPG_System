package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

func TestStartMatch(t *testing.T) {
	masterUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()
	otherUUID := uuid.New()
	now := time.Now()
	finishedAt := now.Add(-time.Hour)

	tests := []struct {
		name            string
		matchUUID       uuid.UUID
		masterUUID      uuid.UUID
		matchMock       *testutil.MockMatchRepo
		enrollMock      *testutil.MockEnrollmentRepo
		participantMock *testutil.MockMatchParticipantWriter
		wantErr         error
	}{
		{
			name:       "success",
			matchUUID:  matchUUID,
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
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         nil,
		},
		{
			name:       "match not found",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, matchPg.ErrMatchNotFound
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         domainMatch.ErrMatchNotFound,
		},
		{
			name:       "not match master",
			matchUUID:  matchUUID,
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
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         domainMatch.ErrNotMatchMaster,
		},
		{
			name:       "match already started",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
						GameStartAt:     &now,
					}, nil
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         domainMatch.ErrMatchAlreadyStarted,
		},
		{
			name:       "match already finished",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return &matchEntity.Match{
						UUID:            matchUUID,
						MasterUUID:      masterUUID,
						CampaignUUID:    campaignUUID,
						GameScheduledAt: now,
						StoryEndAt:      &finishedAt,
					}, nil
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         domainMatch.ErrMatchAlreadyFinished,
		},
		{
			name:       "repo error on GetMatch",
			matchUUID:  matchUUID,
			masterUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, errors.New("db error")
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         errors.New("db error"),
		},
		{
			name:       "repo error on StartMatch",
			matchUUID:  matchUUID,
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
				StartMatchFn: func(ctx context.Context, id uuid.UUID, gameStartAt time.Time) error {
					return errors.New("db error")
				},
			},
			enrollMock:      &testutil.MockEnrollmentRepo{},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         errors.New("db error"),
		},
		{
			name:       "repo error on RejectPendingEnrollments",
			matchUUID:  matchUUID,
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
				RejectPendingEnrollmentsFn: func(ctx context.Context, id uuid.UUID) error {
					return errors.New("db error")
				},
			},
			participantMock: &testutil.MockMatchParticipantWriter{},
			wantErr:         errors.New("db error"),
		},
		{
			name:       "repo error on RegisterFromAcceptedEnrollments",
			matchUUID:  matchUUID,
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
			participantMock: &testutil.MockMatchParticipantWriter{
				RegisterFromAcceptedEnrollmentsFn: func(ctx context.Context, mu uuid.UUID, gameStartAt time.Time) error {
					return errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := domainMatch.NewStartMatchUC(tt.matchMock, tt.enrollMock, tt.participantMock)
			err := uc.Start(context.Background(), tt.matchUUID, tt.masterUUID)

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
