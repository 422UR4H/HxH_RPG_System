package testutil

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type MockCampaignRepo struct {
	CreateCampaignFn             func(ctx context.Context, campaign *campaign.Campaign) error
	GetCampaignFn                func(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	GetCampaignMasterUUIDFn      func(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCampaignStoryDatesFn      func(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	CountCampaignsByMasterUUIDFn func(ctx context.Context, masterUUID uuid.UUID) (int, error)
	ListCampaignsByMasterUUIDFn  func(ctx context.Context, masterUUID uuid.UUID) ([]*campaign.Summary, error)
}

func (m *MockCampaignRepo) CreateCampaign(ctx context.Context, c *campaign.Campaign) error {
	if m.CreateCampaignFn != nil {
		return m.CreateCampaignFn(ctx, c)
	}
	return nil
}

func (m *MockCampaignRepo) GetCampaign(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	if m.GetCampaignFn != nil {
		return m.GetCampaignFn(ctx, id)
	}
	return nil, nil
}

func (m *MockCampaignRepo) GetCampaignMasterUUID(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
	if m.GetCampaignMasterUUIDFn != nil {
		return m.GetCampaignMasterUUIDFn(ctx, id)
	}
	return uuid.Nil, nil
}

func (m *MockCampaignRepo) GetCampaignStoryDates(ctx context.Context, id uuid.UUID) (*campaign.Campaign, error) {
	if m.GetCampaignStoryDatesFn != nil {
		return m.GetCampaignStoryDatesFn(ctx, id)
	}
	return nil, nil
}

func (m *MockCampaignRepo) CountCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) (int, error) {
	if m.CountCampaignsByMasterUUIDFn != nil {
		return m.CountCampaignsByMasterUUIDFn(ctx, masterUUID)
	}
	return 0, nil
}

func (m *MockCampaignRepo) ListCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*campaign.Summary, error) {
	if m.ListCampaignsByMasterUUIDFn != nil {
		return m.ListCampaignsByMasterUUIDFn(ctx, masterUUID)
	}
	return nil, nil
}
