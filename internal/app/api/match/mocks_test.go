package match_test

import (
	"context"

	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type mockCreateMatch struct {
	fn func(ctx context.Context, input *domainMatch.CreateMatchInput) (*matchEntity.Match, error)
}

func (m *mockCreateMatch) CreateMatch(ctx context.Context, input *domainMatch.CreateMatchInput) (*matchEntity.Match, error) {
	return m.fn(ctx, input)
}

type mockGetMatch struct {
	fn func(ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID) (*matchEntity.Match, error)
}

func (m *mockGetMatch) GetMatch(ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID) (*matchEntity.Match, error) {
	return m.fn(ctx, uuid, userUUID)
}

type mockListMatches struct {
	fn func(ctx context.Context, userUUID uuid.UUID) ([]*matchEntity.Summary, error)
}

func (m *mockListMatches) ListMatches(ctx context.Context, userUUID uuid.UUID) ([]*matchEntity.Summary, error) {
	return m.fn(ctx, userUUID)
}

type mockListPublicUpcomingMatches struct {
	fn func(ctx context.Context, userUUID uuid.UUID) ([]*matchEntity.Summary, error)
}

func (m *mockListPublicUpcomingMatches) ListPublicUpcomingMatches(ctx context.Context, userUUID uuid.UUID) ([]*matchEntity.Summary, error) {
	return m.fn(ctx, userUUID)
}
