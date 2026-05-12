package testutil

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type MockMatchRepo struct {
	CreateMatchFn                        func(ctx context.Context, match *match.Match) error
	GetMatchFn                           func(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
	GetMatchCampaignUUIDFn               func(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
	StartMatchFn                  func(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error
	ListParticipantsByMatchUUIDFn func(ctx context.Context, matchUUID uuid.UUID) ([]*match.Participant, error)
	ListMatchesByMasterUUIDFn            func(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
	ListPublicUpcomingMatchesFn          func(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error)
}

func (m *MockMatchRepo) CreateMatch(ctx context.Context, mt *match.Match) error {
	if m.CreateMatchFn != nil {
		return m.CreateMatchFn(ctx, mt)
	}
	return nil
}

func (m *MockMatchRepo) GetMatch(ctx context.Context, id uuid.UUID) (*match.Match, error) {
	if m.GetMatchFn != nil {
		return m.GetMatchFn(ctx, id)
	}
	return nil, nil
}

func (m *MockMatchRepo) GetMatchCampaignUUID(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error) {
	if m.GetMatchCampaignUUIDFn != nil {
		return m.GetMatchCampaignUUIDFn(ctx, matchUUID)
	}
	return uuid.Nil, nil
}

func (m *MockMatchRepo) ListMatchesByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error) {
	if m.ListMatchesByMasterUUIDFn != nil {
		return m.ListMatchesByMasterUUIDFn(ctx, masterUUID)
	}
	return nil, nil
}

func (m *MockMatchRepo) ListPublicUpcomingMatches(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error) {
	if m.ListPublicUpcomingMatchesFn != nil {
		return m.ListPublicUpcomingMatchesFn(ctx, after, masterUUID)
	}
	return nil, nil
}

func (m *MockMatchRepo) StartMatch(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error {
	if m.StartMatchFn != nil {
		return m.StartMatchFn(ctx, matchUUID, gameStartAt)
	}
	return nil
}

func (m *MockMatchRepo) ListParticipantsByMatchUUID(ctx context.Context, matchUUID uuid.UUID) ([]*match.Participant, error) {
	if m.ListParticipantsByMatchUUIDFn != nil {
		return m.ListParticipantsByMatchUUIDFn(ctx, matchUUID)
	}
	return nil, nil
}
