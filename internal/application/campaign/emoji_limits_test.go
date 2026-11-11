package campaign_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
)

// Emoji simples conta 1 caractere (não 2 em UTF-16 nem 4 em bytes).
func TestCreateCampaign_EmojiCountsAsOneCharacter(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(i *campaign.CreateCampaignInput, s string)
		limit  int
		maxErr error
	}{
		{"name", func(i *campaign.CreateCampaignInput, s string) { i.Name = s }, 32, campaign.ErrMaxNameLength},
		{"brief", func(i *campaign.CreateCampaignInput, s string) { i.BriefInitialDescription = s }, 255, campaign.ErrMaxBriefDescLength},
	}
	for _, c := range cases {
		run := func(n int) error {
			in := validCreateCampaignInput()
			c.mutate(in, strings.Repeat("😀", n))
			_, err := campaign.NewCreateCampaignUC(&testutil.MockCampaignRepo{}, &testutil.MockScenarioRepo{}).
				CreateCampaign(context.Background(), in)
			return err
		}
		t.Run(c.name+" max emojis is accepted", func(t *testing.T) {
			if err := run(c.limit); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
		t.Run(c.name+" max+1 emojis fails", func(t *testing.T) {
			if err := run(c.limit + 1); !errors.Is(err, c.maxErr) {
				t.Fatalf("got %v, want %v", err, c.maxErr)
			}
		})
	}
}
