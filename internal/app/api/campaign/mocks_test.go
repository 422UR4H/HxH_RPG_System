package campaign_test

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type mockListPlayerEnrollments struct {
	statuses map[uuid.UUID]string
	err      error
}

func (m *mockListPlayerEnrollments) ListPlayerEnrollmentsForCampaign(
	ctx context.Context,
	playerUUID uuid.UUID,
	campaignUUID uuid.UUID,
) (map[uuid.UUID]string, error) {
	return m.statuses, m.err
}

type mockCreateCampaign struct {
	fn func(ctx context.Context, input *campaign.CreateCampaignInput) (*campaignEntity.Campaign, error)
}

func (m *mockCreateCampaign) CreateCampaign(ctx context.Context, input *campaign.CreateCampaignInput) (*campaignEntity.Campaign, error) {
	return m.fn(ctx, input)
}

type mockGetCampaign struct {
	fn func(ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID) (*campaignEntity.Campaign, error)
}

func (m *mockGetCampaign) GetCampaign(ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID) (*campaignEntity.Campaign, error) {
	return m.fn(ctx, uuid, userUUID)
}

type mockListCampaigns struct {
	fn func(ctx context.Context, userUUID uuid.UUID) ([]*campaignEntity.Summary, error)
}

func (m *mockListCampaigns) ListCampaigns(ctx context.Context, userUUID uuid.UUID) ([]*campaignEntity.Summary, error) {
	return m.fn(ctx, userUUID)
}

type mockListPublicUpcomingCampaigns struct {
	fn func(ctx context.Context, userUUID uuid.UUID) ([]*campaignEntity.PublicSummary, error)
}

func (m *mockListPublicUpcomingCampaigns) ListPublicUpcomingCampaigns(ctx context.Context, userUUID uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
	return m.fn(ctx, userUUID)
}

type mockDeleteCampaign struct {
	fn func(ctx context.Context, input *campaign.DeleteCampaignInput) error
}

func (m *mockDeleteCampaign) Delete(ctx context.Context, input *campaign.DeleteCampaignInput) error {
	return m.fn(ctx, input)
}

type mockUpdateCampaign struct {
	fn func(ctx context.Context, input *campaign.UpdateCampaignInput) (*campaignEntity.Campaign, error)
}

func (m *mockUpdateCampaign) Update(ctx context.Context, input *campaign.UpdateCampaignInput) (*campaignEntity.Campaign, error) {
	return m.fn(ctx, input)
}
