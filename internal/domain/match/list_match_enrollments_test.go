package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type mockMatchRepoForList struct {
	getMatchFn func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error)
	domainMatch.IRepository
}

func (m *mockMatchRepoForList) GetMatch(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
	return m.getMatchFn(ctx, id)
}

type mockEnrollmentLister struct {
	fn func(ctx context.Context, matchUUID uuid.UUID) ([]*enrollmentEntity.Enrollment, error)
}

func (m *mockEnrollmentLister) ListByMatchUUID(ctx context.Context, matchUUID uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
	return m.fn(ctx, matchUUID)
}

func TestListMatchEnrollmentsUC(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	otherUserUUID := uuid.New()

	makeMatch := func() *matchEntity.Match {
		return &matchEntity.Match{
			UUID:         matchUUID,
			MasterUUID:   masterUUID,
			CampaignUUID: uuid.New(),
			IsPublic:     true,
		}
	}

	tests := []struct {
		name         string
		userUUID     uuid.UUID
		matchFn      func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error)
		listFn       func(ctx context.Context, matchUUID uuid.UUID) ([]*enrollmentEntity.Enrollment, error)
		wantErr      error
		wantIsMaster bool
		wantLen      int
	}{
		{
			name:     "master sees ViewerIsMaster=true",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{
					{UUID: uuid.New(), Status: "pending", CreatedAt: time.Now()},
					{UUID: uuid.New(), Status: "accepted", CreatedAt: time.Now()},
				}, nil
			},
			wantIsMaster: true,
			wantLen:      2,
		},
		{
			name:     "non-master on public match sees ViewerIsMaster=false",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{{UUID: uuid.New(), Status: "pending"}}, nil
			},
			wantIsMaster: false,
			wantLen:      1,
		},
		{
			name:     "match not found maps to ErrMatchNotFound",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return nil, matchPg.ErrMatchNotFound
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				t.Fatal("listFn should not be called when match is missing")
				return nil, nil
			},
			wantErr: domainMatch.ErrMatchNotFound,
		},
		{
			name:     "lister returns empty slice and no error",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{}, nil
			},
			wantIsMaster: true,
			wantLen:      0,
		},
		{
			name:     "lister error is propagated",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return nil, errors.New("db down")
			},
			wantErr: errors.New("db down"),
		},
		{
			name:     "match repo error (other than not found) is propagated",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return nil, errors.New("conn refused")
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				t.Fatal("listFn should not be called when match repo errors")
				return nil, nil
			},
			wantErr: errors.New("conn refused"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := domainMatch.NewListMatchEnrollmentsUC(
				&mockMatchRepoForList{getMatchFn: tc.matchFn},
				&mockEnrollmentLister{fn: tc.listFn},
			)

			got, err := uc.List(context.Background(), matchUUID, tc.userUUID)

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				if errors.Is(tc.wantErr, domainMatch.ErrMatchNotFound) {
					if !errors.Is(err, domainMatch.ErrMatchNotFound) {
						t.Fatalf("expected ErrMatchNotFound, got %v", err)
					}
					return
				}
				if err.Error() != tc.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tc.wantErr.Error(), err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ViewerIsMaster != tc.wantIsMaster {
				t.Errorf("ViewerIsMaster = %v, want %v", got.ViewerIsMaster, tc.wantIsMaster)
			}
			if len(got.Enrollments) != tc.wantLen {
				t.Errorf("Enrollments len = %d, want %d", len(got.Enrollments), tc.wantLen)
			}
		})
	}
}
