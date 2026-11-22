package campaign_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/campaign"
	campaignUC "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/google/uuid"
)

func TestDeleteCampaignHandler_ErrorsReadInPortuguese(t *testing.T) {
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())

	t.Run("invalid uuid", func(t *testing.T) {
		_, err := campaign.DeleteCampaignHandler(nil)(ctx, &campaign.DeleteCampaignRequest{UUID: "x"})
		if err == nil || err.Error() != "Campanha inválida." {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("missing user", func(t *testing.T) {
		_, err := campaign.DeleteCampaignHandler(nil)(context.Background(), &campaign.DeleteCampaignRequest{UUID: "x"})
		if err == nil || err.Error() != "Não foi possível identificar sua conta. Entre de novo." {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("internal error is hidden", func(t *testing.T) {
		mock := &mockDeleteCampaign{fn: func(context.Context, *campaignUC.DeleteCampaignInput) error { return errors.New("pq: boom") }}
		_, err := campaign.DeleteCampaignHandler(mock)(ctx, &campaign.DeleteCampaignRequest{UUID: uuid.NewString()})
		if err == nil || err.Error() != "Algo deu errado. Tente de novo." {
			t.Errorf("err = %v", err)
		}
	})
}
