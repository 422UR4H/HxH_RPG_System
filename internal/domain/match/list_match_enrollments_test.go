package match_test

import (
	"context"
	"errors"
	"testing"

	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
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

type mockParticipationChecker struct {
	fn func(ctx context.Context, playerUUID, campaignUUID uuid.UUID) (bool, error)
}

func (m *mockParticipationChecker) ExistsSheetInCampaign(
	ctx context.Context, playerUUID, campaignUUID uuid.UUID,
) (bool, error) {
	return m.fn(ctx, playerUUID, campaignUUID)
}

func TestListMatchEnrollmentsUC(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	otherUserUUID := uuid.New()
	campaignUUID := uuid.New()

	makeMatch := func(isPublic bool) *matchEntity.Match {
		return &matchEntity.Match{
			UUID:         matchUUID,
			MasterUUID:   masterUUID,
			CampaignUUID: campaignUUID,
			IsPublic:     isPublic,
		}
	}

	checkerNeverCalled := func(t *testing.T) *mockParticipationChecker {
		return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			t.Fatal("participationChecker should NOT be called for this case")
			return false, nil
		}}
	}

	tests := []struct {
		name         string
		userUUID     uuid.UUID
		matchFn      func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error)
		listFn       func(ctx context.Context, matchUUID uuid.UUID) ([]*enrollmentEntity.Enrollment, error)
		checker      func(t *testing.T) *mockParticipationChecker
		wantErr      error
		wantIsMaster bool
		wantLen      int
	}{
		{
			name:     "master on public match — no checker call",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(true), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{{Status: "pending"}}, nil
			},
			checker:      checkerNeverCalled,
			wantIsMaster: true,
			wantLen:      1,
		},
		{
			name:     "master on private match — no checker call",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(false), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{}, nil
			},
			checker:      checkerNeverCalled,
			wantIsMaster: true,
			wantLen:      0,
		},
		{
			name:     "non-master on public match — no checker call",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(true), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{{Status: "accepted"}}, nil
			},
			checker:      checkerNeverCalled,
			wantIsMaster: false,
			wantLen:      1,
		},
		{
			name:     "non-master on private match, participates — allowed",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(false), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return []*enrollmentEntity.Enrollment{{Status: "accepted"}}, nil
			},
			checker: func(_ *testing.T) *mockParticipationChecker {
				return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
					return true, nil
				}}
			},
			wantIsMaster: false,
			wantLen:      1,
		},
		{
			name:     "non-master on private match, does NOT participate — forbidden",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(false), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				t.Fatal("listFn should not be called when forbidden")
				return nil, nil
			},
			checker: func(_ *testing.T) *mockParticipationChecker {
				return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
					return false, nil
				}}
			},
			wantErr: domainAuth.ErrInsufficientPermissions,
		},
		{
			name:     "checker error is propagated",
			userUUID: otherUserUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(false), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				t.Fatal("listFn should not be called when checker errors")
				return nil, nil
			},
			checker: func(_ *testing.T) *mockParticipationChecker {
				return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
					return false, errors.New("db down")
				}}
			},
			wantErr: errors.New("db down"),
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
			checker: checkerNeverCalled,
			wantErr: domainMatch.ErrMatchNotFound,
		},
		{
			name:     "lister error is propagated (master path)",
			userUUID: masterUUID,
			matchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return makeMatch(true), nil
			},
			listFn: func(_ context.Context, _ uuid.UUID) ([]*enrollmentEntity.Enrollment, error) {
				return nil, errors.New("db down")
			},
			checker: checkerNeverCalled,
			wantErr: errors.New("db down"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := domainMatch.NewListMatchEnrollmentsUC(
				&mockMatchRepoForList{getMatchFn: tc.matchFn},
				&mockEnrollmentLister{fn: tc.listFn},
				tc.checker(t),
			)

			got, err := uc.List(context.Background(), matchUUID, tc.userUUID)

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				switch {
				case errors.Is(tc.wantErr, domainMatch.ErrMatchNotFound):
					if !errors.Is(err, domainMatch.ErrMatchNotFound) {
						t.Fatalf("expected ErrMatchNotFound, got %v", err)
					}
				case errors.Is(tc.wantErr, domainAuth.ErrInsufficientPermissions):
					if !errors.Is(err, domainAuth.ErrInsufficientPermissions) {
						t.Fatalf("expected ErrInsufficientPermissions, got %v", err)
					}
				default:
					if err.Error() != tc.wantErr.Error() {
						t.Fatalf("expected error %q, got %q", tc.wantErr.Error(), err.Error())
					}
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
