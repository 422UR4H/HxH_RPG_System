package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
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

func validUpdateMatchInput(matchUUID, masterUUID uuid.UUID) *domainMatch.UpdateMatchInput {
	title := "Updated Title"
	brief := "Updated brief"
	desc := "Updated description"
	isPublic := false
	gameAt := time.Now().Add(48 * time.Hour)
	storyAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	return &domainMatch.UpdateMatchInput{
		MatchUUID:               matchUUID,
		MasterUUID:              masterUUID,
		Title:                   &title,
		BriefInitialDescription: &brief,
		Description:             &desc,
		IsPublic:                &isPublic,
		GameScheduledAt:         &gameAt,
		StoryStartAt:            &storyAt,
	}
}

func loadMatchFn(m *matchEntity.Match) func(context.Context, uuid.UUID) (*matchEntity.Match, error) {
	return func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
		copy := *m
		return &copy, nil
	}
}

func TestUpdateMatch(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()

	baseMatch := &matchEntity.Match{
		UUID:                    matchUUID,
		MasterUUID:              masterUUID,
		CampaignUUID:            uuid.New(),
		Title:                   "Original Title",
		BriefInitialDescription: "Original brief",
		Description:             "Original description",
		IsPublic:                true,
		GameScheduledAt:         time.Now().Add(24 * time.Hour),
		StoryStartAt:            time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:               time.Now().Add(-24 * time.Hour),
		UpdatedAt:               time.Now().Add(-24 * time.Hour),
	}

	validCampaign := func() *campaignEntity.Campaign {
		return &campaignEntity.Campaign{
			MasterUUID:   masterUUID,
			StoryStartAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		}
	}

	t.Run("success full patch", func(t *testing.T) {
		input := validUpdateMatchInput(matchUUID, masterUUID)
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn:    loadMatchFn(baseMatch),
			UpdateMatchFn: func(_ context.Context, m *matchEntity.Match) error { return nil },
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return validCampaign(), nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)

		got, err := uc.Update(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Title != *input.Title {
			t.Errorf("title = %q, want %q", got.Title, *input.Title)
		}
		if got.IsPublic != *input.IsPublic {
			t.Errorf("isPublic = %v, want %v", got.IsPublic, *input.IsPublic)
		}
	})

	t.Run("success partial patch — title only", func(t *testing.T) {
		title := "Only Title Changed"
		input := &domainMatch.UpdateMatchInput{
			MatchUUID: matchUUID, MasterUUID: masterUUID, Title: &title,
		}
		updateCalled := false
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: loadMatchFn(baseMatch),
			UpdateMatchFn: func(_ context.Context, m *matchEntity.Match) error {
				updateCalled = true
				if m.Title != title {
					t.Errorf("persisted title = %q, want %q", m.Title, title)
				}
				if m.BriefInitialDescription != baseMatch.BriefInitialDescription {
					t.Errorf("brief mutated unexpectedly")
				}
				return nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})

		_, err := uc.Update(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !updateCalled {
			t.Fatal("UpdateMatch should have been called")
		}
	})

	t.Run("no-op when all fields nil", func(t *testing.T) {
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID}
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: loadMatchFn(baseMatch),
			UpdateMatchFn: func(_ context.Context, _ *matchEntity.Match) error {
				t.Fatal("UpdateMatch should NOT be called for no-op input")
				return nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})

		got, err := uc.Update(context.Background(), input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Title != baseMatch.Title {
			t.Errorf("title changed unexpectedly")
		}
	})

	t.Run("match not found", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: func(_ context.Context, _ uuid.UUID) (*matchEntity.Match, error) {
				return nil, matchPg.ErrMatchNotFound
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), validUpdateMatchInput(matchUUID, masterUUID))
		if !errors.Is(err, domainMatch.ErrMatchNotFound) {
			t.Fatalf("got %v, want ErrMatchNotFound", err)
		}
	})

	t.Run("not master", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		input := validUpdateMatchInput(matchUUID, uuid.New())
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrNotMatchMaster) {
			t.Fatalf("got %v, want ErrNotMatchMaster", err)
		}
	})

	t.Run("match already started", func(t *testing.T) {
		started := *baseMatch
		now := time.Now()
		started.GameStartAt = &now
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(&started)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), validUpdateMatchInput(matchUUID, masterUUID))
		if !errors.Is(err, domainMatch.ErrMatchAlreadyStarted) {
			t.Fatalf("got %v, want ErrMatchAlreadyStarted", err)
		}
	})

	t.Run("match already finished", func(t *testing.T) {
		finished := *baseMatch
		end := time.Now()
		finished.StoryEndAt = &end
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(&finished)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), validUpdateMatchInput(matchUUID, masterUUID))
		if !errors.Is(err, domainMatch.ErrMatchAlreadyFinished) {
			t.Fatalf("got %v, want ErrMatchAlreadyFinished", err)
		}
	})

	t.Run("title too short", func(t *testing.T) {
		short := "ab"
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, Title: &short}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMinTitleLength) {
			t.Fatalf("got %v, want ErrMinTitleLength", err)
		}
	})

	t.Run("title too long", func(t *testing.T) {
		long := "this title is way too long for the maximum limit"
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, Title: &long}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMaxTitleLength) {
			t.Fatalf("got %v, want ErrMaxTitleLength", err)
		}
	})

	t.Run("brief too long", func(t *testing.T) {
		brief := string(make([]byte, 65))
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, BriefInitialDescription: &brief}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMaxBriefDescLength) {
			t.Fatalf("got %v, want ErrMaxBriefDescLength", err)
		}
	})

	t.Run("game scheduled in past", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, GameScheduledAt: &past}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMinOfGameScheduledAt) {
			t.Fatalf("got %v, want ErrMinOfGameScheduledAt", err)
		}
	})

	t.Run("game scheduled too far", func(t *testing.T) {
		far := time.Now().AddDate(1, 1, 0)
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, GameScheduledAt: &far}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMaxOfGameScheduledAt) {
			t.Fatalf("got %v, want ErrMaxOfGameScheduledAt", err)
		}
	})

	t.Run("story start before campaign start", func(t *testing.T) {
		before := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, StoryStartAt: &before}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return validCampaign(), nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMinOfStoryStartAt) {
			t.Fatalf("got %v, want ErrMinOfStoryStartAt", err)
		}
	})

	t.Run("story start after campaign end", func(t *testing.T) {
		end := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		after := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
		input := &domainMatch.UpdateMatchInput{MatchUUID: matchUUID, MasterUUID: masterUUID, StoryStartAt: &after}
		matchRepo := &testutil.MockMatchRepo{GetMatchFn: loadMatchFn(baseMatch)}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				c := validCampaign()
				c.StoryEndAt = &end
				return c, nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)
		_, err := uc.Update(context.Background(), input)
		if !errors.Is(err, domainMatch.ErrMaxOfStoryStartAt) {
			t.Fatalf("got %v, want ErrMaxOfStoryStartAt", err)
		}
	})

	t.Run("repo update returns not found — race with start", func(t *testing.T) {
		matchRepo := &testutil.MockMatchRepo{
			GetMatchFn: loadMatchFn(baseMatch),
			UpdateMatchFn: func(_ context.Context, _ *matchEntity.Match) error {
				return matchPg.ErrMatchNotFound
			},
		}
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.Campaign, error) {
				return validCampaign(), nil
			},
		}
		uc := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)
		_, err := uc.Update(context.Background(), validUpdateMatchInput(matchUUID, masterUUID))
		if !errors.Is(err, domainMatch.ErrMatchAlreadyStarted) {
			t.Fatalf("got %v, want ErrMatchAlreadyStarted (race mapping)", err)
		}
	})
}
