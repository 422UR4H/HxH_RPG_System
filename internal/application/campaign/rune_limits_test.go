package campaign_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

// Os limites contam caracteres (como o VARCHAR do Postgres), não bytes.
func TestCreateCampaign_LimitsCountCharacters(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(i *campaign.CreateCampaignInput, s string)
		limit   int
		wantErr error
	}{
		{"name", func(i *campaign.CreateCampaignInput, s string) { i.Name = s }, 32, campaign.ErrMaxNameLength},
		{"brief", func(i *campaign.CreateCampaignInput, s string) { i.BriefInitialDescription = s }, 255, campaign.ErrMaxBriefDescLength},
		{"call link", func(i *campaign.CreateCampaignInput, s string) { i.CallLink = s }, 255, campaign.ErrMaxCallLinkLength},
	}
	for _, c := range cases {
		t.Run(c.name+" at limit with accent passes", func(t *testing.T) {
			in := validCreateCampaignInput()
			c.mutate(in, strings.Repeat("ã", c.limit))
			_, err := campaign.NewCreateCampaignUC(&testutil.MockCampaignRepo{}, &testutil.MockScenarioRepo{}).
				CreateCampaign(context.Background(), in)
			if errors.Is(err, c.wantErr) {
				t.Fatalf("unexpected length error: %v", err)
			}
		})
		t.Run(c.name+" over limit fails", func(t *testing.T) {
			in := validCreateCampaignInput()
			c.mutate(in, strings.Repeat("ã", c.limit+1))
			_, err := campaign.NewCreateCampaignUC(&testutil.MockCampaignRepo{}, &testutil.MockScenarioRepo{}).
				CreateCampaign(context.Background(), in)
			if err == nil || err.Error() != c.wantErr.Error() {
				t.Fatalf("got %v, want %v", err, c.wantErr)
			}
		})
	}
	t.Run("name with 5 accented characters is not too short", func(t *testing.T) {
		in := validCreateCampaignInput()
		in.Name = "ãããã"
		_, err := campaign.NewCreateCampaignUC(&testutil.MockCampaignRepo{}, &testutil.MockScenarioRepo{}).
			CreateCampaign(context.Background(), in)
		if err == nil || err.Error() != campaign.ErrMinNameLength.Error() {
			t.Fatalf("4 characters must be too short, got %v", err)
		}
		in.Name = "ããããã"
		_, err = campaign.NewCreateCampaignUC(&testutil.MockCampaignRepo{}, &testutil.MockScenarioRepo{}).
			CreateCampaign(context.Background(), in)
		if err != nil && err.Error() == campaign.ErrMinNameLength.Error() {
			t.Fatalf("5 characters must not be too short")
		}
	})
}

func TestUpdateCampaign_LimitsCountCharacters(t *testing.T) {
	repo := func() *testutil.MockCampaignRepo {
		return &testutil.MockCampaignRepo{
			GetCampaignForUpdateFn: func(_ context.Context, _ uuid.UUID) (*campaignEntity.CampaignUpdateContext, error) {
				return &campaignEntity.CampaignUpdateContext{MasterUUID: master, Name: "Valid Name"}, nil
			},
		}
	}
	cases := []struct {
		name    string
		input   func(s string) *campaign.UpdateCampaignInput
		limit   int
		wantErr error
	}{
		{"name", func(s string) *campaign.UpdateCampaignInput {
			return &campaign.UpdateCampaignInput{MasterUUID: master, Name: &s}
		}, 32, campaign.ErrMaxNameLength},
		{"brief", func(s string) *campaign.UpdateCampaignInput {
			return &campaign.UpdateCampaignInput{MasterUUID: master, BriefInitialDescription: &s}
		}, 255, campaign.ErrMaxBriefDescLength},
		{"call link", func(s string) *campaign.UpdateCampaignInput {
			return &campaign.UpdateCampaignInput{MasterUUID: master, CallLink: &s}
		}, 255, campaign.ErrMaxCallLinkLength},
	}
	for _, c := range cases {
		t.Run(c.name+" at limit with accent passes", func(t *testing.T) {
			_, err := campaign.NewUpdateCampaignUC(repo()).Update(context.Background(), c.input(strings.Repeat("ã", c.limit)))
			if err != nil && err.Error() == c.wantErr.Error() {
				t.Fatalf("unexpected length error: %v", err)
			}
		})
		t.Run(c.name+" over limit fails", func(t *testing.T) {
			_, err := campaign.NewUpdateCampaignUC(repo()).Update(context.Background(), c.input(strings.Repeat("ã", c.limit+1)))
			if err == nil || err.Error() != c.wantErr.Error() {
				t.Fatalf("got %v, want %v", err, c.wantErr)
			}
		})
	}
}

var master = uuid.New()
