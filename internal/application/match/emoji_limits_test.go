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
	"github.com/google/uuid"
)

// Emoji simples conta 1 caractere (não 2 em UTF-16 nem 4 em bytes).
func TestCreateMatch_EmojiCountsAsOneCharacter(t *testing.T) {
	run := func(mutate func(*match.CreateMatchInput)) error {
		in := validCreateMatchInput()
		mutate(in)
		campaignRepo := &testutil.MockCampaignRepo{
			GetCampaignStoryDatesFn: func(context.Context, uuid.UUID) (*campaignEntity.Campaign, error) {
				return &campaignEntity.Campaign{MasterUUID: in.MasterUUID, StoryStartAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
			},
		}
		_, err := match.NewCreateMatchUC(&testutil.MockMatchRepo{}, campaignRepo).CreateMatch(context.Background(), in)
		return err
	}
	title := func(n int) func(*match.CreateMatchInput) {
		return func(i *match.CreateMatchInput) { i.Title = strings.Repeat("😀", n) }
	}
	brief := func(n int) func(*match.CreateMatchInput) {
		return func(i *match.CreateMatchInput) { i.BriefInitialDescription = strings.Repeat("😀", n) }
	}
	if err := run(title(32)); err != nil {
		t.Errorf("32 emoji title must be accepted: %v", err)
	}
	if err := run(title(33)); !errors.Is(err, match.ErrMaxTitleLength) {
		t.Errorf("33 emoji title: got %v", err)
	}
	if err := run(brief(255)); err != nil {
		t.Errorf("255 emoji brief must be accepted: %v", err)
	}
	if err := run(brief(256)); !errors.Is(err, match.ErrMaxBriefDescLength) {
		t.Errorf("256 emoji brief: got %v", err)
	}
}
