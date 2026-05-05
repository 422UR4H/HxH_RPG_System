package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

func TestGetMatchParticipants(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()

	privateMatch := &matchEntity.Match{
		UUID:         matchUUID,
		MasterUUID:   masterUUID,
		CampaignUUID: campaignUUID,
		IsPublic:     false,
	}
	publicMatch := &matchEntity.Match{
		UUID:         matchUUID,
		MasterUUID:   masterUUID,
		CampaignUUID: campaignUUID,
		IsPublic:     true,
	}
	twoParticipants := []*matchEntity.Participant{
		{UUID: uuid.New(), MatchUUID: matchUUID},
		{UUID: uuid.New(), MatchUUID: matchUUID},
	}

	tests := []struct {
		name             string
		userUUID         uuid.UUID
		matchMock        *testutil.MockMatchRepo
		checker          *mockParticipationChecker
		wantErr          error
		wantLen          int
		wantViewerMaster bool
	}{
		{
			name:     "success as master — private match, returns all with ViewerIsMaster true",
			userUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return privateMatch, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return twoParticipants, nil
				},
			},
			checker:          &mockParticipationChecker{},
			wantLen:          2,
			wantViewerMaster: true,
		},
		{
			name:     "success as player on public match",
			userUUID: otherUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return publicMatch, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return twoParticipants, nil
				},
			},
			checker:          &mockParticipationChecker{},
			wantLen:          2,
			wantViewerMaster: false,
		},
		{
			name:     "private match — non-participant gets forbidden",
			userUUID: otherUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return privateMatch, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					t.Fatal("participant reader should NOT be called")
					return nil, nil
				},
			},
			checker: &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
				return false, nil
			}},
			wantErr: auth.ErrInsufficientPermissions,
		},
		{
			name:     "match not found",
			userUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return nil, matchPg.ErrMatchNotFound
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					t.Fatal("should not be called")
					return nil, nil
				},
			},
			checker: &mockParticipationChecker{},
			wantErr: domainMatch.ErrMatchNotFound,
		},
		{
			name:     "participant repo error propagated",
			userUUID: masterUUID,
			matchMock: &testutil.MockMatchRepo{
				GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
					return privateMatch, nil
				},
				ListParticipantsByMatchUUIDFn: func(_ context.Context, _ uuid.UUID) ([]*matchEntity.Participant, error) {
					return nil, errors.New("db error")
				},
			},
			checker: &mockParticipationChecker{},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := domainMatch.NewGetMatchParticipantsUC(tt.matchMock, tt.checker)
			result, err := uc.Get(context.Background(), matchUUID, tt.userUUID)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected %q, got %q", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result.Participants) != tt.wantLen {
				t.Errorf("len(Participants) = %d, want %d", len(result.Participants), tt.wantLen)
			}
			if result.ViewerIsMaster != tt.wantViewerMaster {
				t.Errorf("ViewerIsMaster = %v, want %v", result.ViewerIsMaster, tt.wantViewerMaster)
			}
		})
	}
}
