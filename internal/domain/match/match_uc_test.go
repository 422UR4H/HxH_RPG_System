package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

func validCreateMatchInput() *domainMatch.CreateMatchInput {
	return &domainMatch.CreateMatchInput{
		MasterUUID:              uuid.New(),
		CampaignUUID:            uuid.New(),
		Title:                   "Valid Title",
		BriefInitialDescription: "Brief",
		Description:             "Full description",
		IsPublic:                true,
		GameScheduledAt:         time.Now().Add(24 * time.Hour),
		StoryStartAt:            time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestCreateMatch(t *testing.T) {
	tests := []struct {
		name         string
		input        *domainMatch.CreateMatchInput
		matchMock    *testutil.MockMatchRepo
		campaignMock *testutil.MockCampaignRepo
		wantErr      error
	}{
		{
			name: "success",
			input: func() *domainMatch.CreateMatchInput {
				i := validCreateMatchInput()
				return i
			}(),
			matchMock: &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignStoryDatesFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					i := validCreateMatchInput()
					return &campaignEntity.Campaign{
						MasterUUID:   i.MasterUUID,
						StoryStartAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					}, nil
				},
			},
			wantErr: nil,
		},
		{
			name: "title too short",
			input: func() *domainMatch.CreateMatchInput {
				i := validCreateMatchInput()
				i.Title = "ab"
				return i
			}(),
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      domainMatch.ErrMinTitleLength,
		},
		{
			name: "title too long",
			input: func() *domainMatch.CreateMatchInput {
				i := validCreateMatchInput()
				i.Title = "this title is way too long for the maximum limit"
				return i
			}(),
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      domainMatch.ErrMaxTitleLength,
		},
		{
			name: "brief description too long",
			input: func() *domainMatch.CreateMatchInput {
				i := validCreateMatchInput()
				i.BriefInitialDescription = string(make([]byte, 256))
				return i
			}(),
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      domainMatch.ErrMaxBriefDescLength,
		},
		{
			name: "game scheduled at in the past",
			input: func() *domainMatch.CreateMatchInput {
				i := validCreateMatchInput()
				i.GameScheduledAt = time.Now().Add(-1 * time.Hour)
				return i
			}(),
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      domainMatch.ErrMinOfGameScheduledAt,
		},
		{
			name: "game scheduled at more than 1 year ahead",
			input: func() *domainMatch.CreateMatchInput {
				i := validCreateMatchInput()
				i.GameScheduledAt = time.Now().AddDate(1, 1, 0)
				return i
			}(),
			matchMock:    &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{},
			wantErr:      domainMatch.ErrMaxOfGameScheduledAt,
		},
		{
			name: "campaign not found",
			input: func() *domainMatch.CreateMatchInput {
				i := validCreateMatchInput()
				return i
			}(),
			matchMock: &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignStoryDatesFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					return nil, campaignPg.ErrCampaignNotFound
				},
			},
			wantErr: campaign.ErrCampaignNotFound,
		},
		{
			name: "not campaign owner",
			input: func() *domainMatch.CreateMatchInput {
				i := validCreateMatchInput()
				return i
			}(),
			matchMock: &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignStoryDatesFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					return &campaignEntity.Campaign{
						MasterUUID:   uuid.New(), // different master
						StoryStartAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					}, nil
				},
			},
			wantErr: campaign.ErrNotCampaignOwner,
		},
		{
			name: "story start before campaign start",
			input: func() *domainMatch.CreateMatchInput {
				i := validCreateMatchInput()
				i.StoryStartAt = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				return i
			}(),
			matchMock: &testutil.MockMatchRepo{},
			campaignMock: &testutil.MockCampaignRepo{
				GetCampaignStoryDatesFn: func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					i := validCreateMatchInput()
					return &campaignEntity.Campaign{
						MasterUUID:   i.MasterUUID,
						StoryStartAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					}, nil
				},
			},
			wantErr: domainMatch.ErrMinOfStoryStartAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For success and campaign-related tests, we need matching MasterUUID
			if tt.name == "success" || tt.name == "story start before campaign start" {
				masterUUID := tt.input.MasterUUID
				tt.campaignMock.GetCampaignStoryDatesFn = func(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
					return &campaignEntity.Campaign{
						MasterUUID:   masterUUID,
						StoryStartAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
					}, nil
				}
			}

			uc := domainMatch.NewCreateMatchUC(tt.matchMock, tt.campaignMock)
			result, err := uc.CreateMatch(context.Background(), tt.input)

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
				t.Fatal("expected non-nil match")
			}
			if result.Title != tt.input.Title {
				t.Errorf("expected title %q, got %q", tt.input.Title, result.Title)
			}
		})
	}
}

func TestGetMatch(t *testing.T) {
	masterUUID := uuid.New()
	otherUUID := uuid.New()
	matchUUID := uuid.New()
	campaignUUID := uuid.New()

	privateMatch := &matchEntity.Match{
		UUID:         matchUUID,
		MasterUUID:   masterUUID,
		CampaignUUID: campaignUUID,
		Title:        "Private Match",
		IsPublic:     false,
	}

	publicMatch := &matchEntity.Match{
		UUID:         matchUUID,
		MasterUUID:   masterUUID,
		CampaignUUID: campaignUUID,
		Title:        "Public Match",
		IsPublic:     true,
	}

	checkerNeverCalled := func(t *testing.T) *mockParticipationChecker {
		return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			t.Fatal("participationChecker should NOT be called for this case")
			return false, nil
		}}
	}

	tests := []struct {
		name     string
		uuid     uuid.UUID
		userUUID uuid.UUID
		mock     *testutil.MockMatchRepo
		checker  func(t *testing.T) *mockParticipationChecker
		wantErr  error
	}{
		{
			name:     "success as owner",
			uuid:     matchUUID,
			userUUID: masterUUID,
			mock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return privateMatch, nil
				},
			},
			checker: checkerNeverCalled,
			wantErr: nil,
		},
		{
			name:     "success public match other user",
			uuid:     matchUUID,
			userUUID: otherUUID,
			mock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return publicMatch, nil
				},
			},
			checker: checkerNeverCalled,
			wantErr: nil,
		},
		{
			name:     "private match insufficient permissions",
			uuid:     matchUUID,
			userUUID: otherUUID,
			mock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return privateMatch, nil
				},
			},
			checker: func(_ *testing.T) *mockParticipationChecker {
				return &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
					return false, nil
				}}
			},
			wantErr: auth.ErrInsufficientPermissions,
		},
		{
			name:     "match not found",
			uuid:     uuid.New(),
			userUUID: masterUUID,
			mock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, matchPg.ErrMatchNotFound
				},
			},
			checker: checkerNeverCalled,
			wantErr: domainMatch.ErrMatchNotFound,
		},
		{
			name:     "repo error",
			uuid:     matchUUID,
			userUUID: masterUUID,
			mock: &testutil.MockMatchRepo{
				GetMatchFn: func(ctx context.Context, id uuid.UUID) (*matchEntity.Match, error) {
					return nil, errors.New("connection error")
				},
			},
			checker: checkerNeverCalled,
			wantErr: errors.New("connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := domainMatch.NewGetMatchUC(tt.mock, tt.checker(t))
			result, err := uc.GetMatch(context.Background(), tt.uuid, tt.userUUID)

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
				t.Fatal("expected non-nil match")
			}
		})
	}

	t.Run("non-master on private match, participates — allowed", func(t *testing.T) {
		matchUUID := uuid.New()
		masterUUID := uuid.New()
		userUUID := uuid.New()
		campaignUUID := uuid.New()

		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{
					UUID:         matchUUID,
					MasterUUID:   masterUUID,
					CampaignUUID: campaignUUID,
					IsPublic:     false,
				}, nil
			},
		}
		checker := &mockParticipationChecker{fn: func(_ context.Context, p, c uuid.UUID) (bool, error) {
			if p != userUUID || c != campaignUUID {
				t.Errorf("checker called with (%v,%v), want (%v,%v)", p, c, userUUID, campaignUUID)
			}
			return true, nil
		}}
		uc := domainMatch.NewGetMatchUC(matchRepo, checker)

		got, err := uc.GetMatch(context.Background(), matchUUID, userUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.UUID != matchUUID {
			t.Errorf("got match uuid %v, want %v", got.UUID, matchUUID)
		}
	})

	t.Run("non-master on private match, does NOT participate — forbidden", func(t *testing.T) {
		matchUUID := uuid.New()
		masterUUID := uuid.New()
		userUUID := uuid.New()

		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{
					UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: uuid.New(), IsPublic: false,
				}, nil
			},
		}
		checker := &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		}}
		uc := domainMatch.NewGetMatchUC(matchRepo, checker)

		_, err := uc.GetMatch(context.Background(), matchUUID, userUUID)
		if !errors.Is(err, auth.ErrInsufficientPermissions) {
			t.Fatalf("got %v, want ErrInsufficientPermissions", err)
		}
	})

	t.Run("non-master on private match, checker errors — propagated", func(t *testing.T) {
		matchUUID := uuid.New()
		masterUUID := uuid.New()
		userUUID := uuid.New()

		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return &matchEntity.Match{
					UUID: matchUUID, MasterUUID: masterUUID, CampaignUUID: uuid.New(), IsPublic: false,
				}, nil
			},
		}
		wantErr := errors.New("db down")
		checker := &mockParticipationChecker{fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, wantErr
		}}
		uc := domainMatch.NewGetMatchUC(matchRepo, checker)

		_, err := uc.GetMatch(context.Background(), matchUUID, userUUID)
		if err == nil || err.Error() != wantErr.Error() {
			t.Fatalf("got %v, want %v", err, wantErr)
		}
	})
}

func TestListMatches(t *testing.T) {
	userUUID := uuid.New()

	tests := []struct {
		name    string
		mock    *testutil.MockMatchRepo
		wantErr error
		wantLen int
	}{
		{
			name: "success with results",
			mock: &testutil.MockMatchRepo{
				ListMatchesByMasterUUIDFn: func(ctx context.Context, id uuid.UUID) ([]*matchEntity.Summary, error) {
					return []*matchEntity.Summary{{Title: "M1"}, {Title: "M2"}}, nil
				},
			},
			wantLen: 2,
		},
		{
			name: "success empty",
			mock: &testutil.MockMatchRepo{
				ListMatchesByMasterUUIDFn: func(ctx context.Context, id uuid.UUID) ([]*matchEntity.Summary, error) {
					return []*matchEntity.Summary{}, nil
				},
			},
			wantLen: 0,
		},
		{
			name: "repo error",
			mock: &testutil.MockMatchRepo{
				ListMatchesByMasterUUIDFn: func(ctx context.Context, id uuid.UUID) ([]*matchEntity.Summary, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := domainMatch.NewListMatchesUC(tt.mock)
			result, err := uc.ListMatches(context.Background(), userUUID)

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

func TestListPublicUpcomingMatches(t *testing.T) {
	userUUID := uuid.New()

	tests := []struct {
		name    string
		mock    *testutil.MockMatchRepo
		wantErr error
		wantLen int
	}{
		{
			name: "success with results",
			mock: &testutil.MockMatchRepo{
				ListPublicUpcomingMatchesFn: func(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*matchEntity.Summary, error) {
					return []*matchEntity.Summary{{Title: "M1"}}, nil
				},
			},
			wantLen: 1,
		},
		{
			name: "success empty",
			mock: &testutil.MockMatchRepo{
				ListPublicUpcomingMatchesFn: func(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*matchEntity.Summary, error) {
					return []*matchEntity.Summary{}, nil
				},
			},
			wantLen: 0,
		},
		{
			name: "repo error",
			mock: &testutil.MockMatchRepo{
				ListPublicUpcomingMatchesFn: func(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*matchEntity.Summary, error) {
					return nil, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := domainMatch.NewListPublicUpcomingMatchesUC(tt.mock)
			result, err := uc.ListPublicUpcomingMatches(context.Background(), userUUID)

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
