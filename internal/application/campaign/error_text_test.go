package campaign_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := campaign.ErrCampaignNotFound.Error(), "Campanha não encontrada."; got != want {
		t.Errorf("ErrCampaignNotFound = %q, want %q", got, want)
	}
}
