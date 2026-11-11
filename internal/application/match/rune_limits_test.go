package match_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
)

// Os limites contam caracteres (como o VARCHAR do Postgres), não bytes.
func TestCreateMatch_LimitsCountCharacters(t *testing.T) {
	run := func(mutate func(*match.CreateMatchInput)) error {
		in := validCreateMatchInput()
		mutate(in)
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(context.Context, uuid.UUID) (*campaignEntity.Campaign, error) {
				return &campaignEntity.Campaign{MasterUUID: in.MasterUUID, StoryStartAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
			},
		}
		uc := match.NewCreateMatchUC(&testutil.MockMatchRepo{}, campaignRepo)
		_, err := uc.CreateMatch(context.Background(), in)
		return err
	}
	t.Run("title at limit with accent passes", func(t *testing.T) {
		if err := run(func(i *match.CreateMatchInput) { i.Title = strings.Repeat("ã", 32) }); errors.Is(err, match.ErrMaxTitleLength) {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("title over limit fails", func(t *testing.T) {
		if err := run(func(i *match.CreateMatchInput) { i.Title = strings.Repeat("ã", 33) }); !errors.Is(err, match.ErrMaxTitleLength) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("title with 4 accented characters is too short", func(t *testing.T) {
		if err := run(func(i *match.CreateMatchInput) { i.Title = "ãããã" }); !errors.Is(err, match.ErrMinTitleLength) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("brief at limit with accent passes", func(t *testing.T) {
		if err := run(func(i *match.CreateMatchInput) { i.BriefInitialDescription = strings.Repeat("ã", 255) }); errors.Is(err, match.ErrMaxBriefDescLength) {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("brief over limit fails", func(t *testing.T) {
		if err := run(func(i *match.CreateMatchInput) { i.BriefInitialDescription = strings.Repeat("ã", 256) }); !errors.Is(err, match.ErrMaxBriefDescLength) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestUpdateMatch_LimitsCountCharacters(t *testing.T) {
	run := func(in *match.UpdateMatchInput) error {
		base := *validBaseMatchForRunes()
		uc := match.NewUpdateMatchUC(&testutil.MockMatchRepo{GetMatchFn: loadMatchFn(&base)}, &testutil.MockCampaignRepo{})
		_, err := uc.Update(context.Background(), in)
		return err
	}
	mk := func() *match.UpdateMatchInput {
		return &match.UpdateMatchInput{MatchUUID: uuid.New(), MasterUUID: runesMaster}
	}
	t.Run("title at limit with accent passes", func(t *testing.T) {
		s := strings.Repeat("ã", 32)
		in := mk()
		in.Title = &s
		if err := run(in); errors.Is(err, match.ErrMaxTitleLength) {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("title over limit fails", func(t *testing.T) {
		s := strings.Repeat("ã", 33)
		in := mk()
		in.Title = &s
		if err := run(in); !errors.Is(err, match.ErrMaxTitleLength) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("brief at limit with accent passes", func(t *testing.T) {
		s := strings.Repeat("ã", 255)
		in := mk()
		in.BriefInitialDescription = &s
		if err := run(in); errors.Is(err, match.ErrMaxBriefDescLength) {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("brief over limit fails", func(t *testing.T) {
		s := strings.Repeat("ã", 256)
		in := mk()
		in.BriefInitialDescription = &s
		if err := run(in); !errors.Is(err, match.ErrMaxBriefDescLength) {
			t.Fatalf("got %v", err)
		}
	})
}

var runesMaster = uuid.New()

func validBaseMatchForRunes() *matchEntity.Match {
	return &matchEntity.Match{
		UUID:            uuid.New(),
		MasterUUID:      runesMaster,
		CampaignUUID:    uuid.New(),
		Title:           "Original Title",
		GameScheduledAt: time.Now().Add(24 * time.Hour),
		StoryStartAt:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
}
