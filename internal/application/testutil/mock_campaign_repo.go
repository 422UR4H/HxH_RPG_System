package testutil

import (
	"context"
	"errors"
	"time"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type MockCampaignRepo struct {
	CreateCampaignFn              func(ctx context.Context, campaign *campaignEntity.Campaign) error
	GetCampaignFn                 func(ctx context.Context, uuid uuid.UUID) (*campaignEntity.Campaign, error)
	GetCampaignMasterUUIDFn       func(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCampaignStoryDatesFn       func(ctx context.Context, uuid uuid.UUID) (*campaignEntity.Campaign, error)
	CountCampaignsByMasterUUIDFn  func(ctx context.Context, masterUUID uuid.UUID) (int, error)
	ListCampaignsByMasterUUIDFn   func(ctx context.Context, masterUUID uuid.UUID) ([]*campaignEntity.Summary, error)
	ListPublicUpcomingCampaignsFn func(ctx context.Context, after time.Time, userUUID uuid.UUID) ([]*campaignEntity.PublicSummary, error)
	DeleteCampaignFn              func(ctx context.Context, uuid uuid.UUID) error
	GetCampaignForUpdateFn        func(ctx context.Context, uuid uuid.UUID) (*campaignEntity.CampaignUpdateContext, error)
	UpdateCampaignFn              func(ctx context.Context, c *campaignEntity.Campaign) error
}

func (m *MockCampaignRepo) CreateCampaign(ctx context.Context, c *campaignEntity.Campaign) error {
	if m.CreateCampaignFn != nil {
		return m.CreateCampaignFn(ctx, c)
	}
	return nil
}

func (m *MockCampaignRepo) GetCampaign(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
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

func (m *MockCampaignRepo) GetCampaignStoryDates(ctx context.Context, id uuid.UUID) (*campaignEntity.Campaign, error) {
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

func (m *MockCampaignRepo) ListCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*campaignEntity.Summary, error) {
	if m.ListCampaignsByMasterUUIDFn != nil {
		return m.ListCampaignsByMasterUUIDFn(ctx, masterUUID)
	}
	return nil, nil
}

func (m *MockCampaignRepo) ListPublicUpcomingCampaigns(ctx context.Context, after time.Time, userUUID uuid.UUID) ([]*campaignEntity.PublicSummary, error) {
	if m.ListPublicUpcomingCampaignsFn != nil {
		return m.ListPublicUpcomingCampaignsFn(ctx, after, userUUID)
	}
	return nil, nil
}

func (m *MockCampaignRepo) DeleteCampaign(ctx context.Context, id uuid.UUID) error {
	if m.DeleteCampaignFn != nil {
		return m.DeleteCampaignFn(ctx, id)
	}
	return nil
}

func (m *MockCampaignRepo) GetCampaignForUpdate(ctx context.Context, id uuid.UUID) (*campaignEntity.CampaignUpdateContext, error) {
	if m.GetCampaignForUpdateFn != nil {
		return m.GetCampaignForUpdateFn(ctx, id)
	}
	return nil, errors.New("campaign not found")
}

func (m *MockCampaignRepo) UpdateCampaign(ctx context.Context, c *campaignEntity.Campaign) error {
	if m.UpdateCampaignFn != nil {
		return m.UpdateCampaignFn(ctx, c)
	}
	return nil
}
