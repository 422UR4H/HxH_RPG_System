package match_test

import (
	"context"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/google/uuid"
)

type mockListMatchEnrollments struct {
	fn func(ctx context.Context, matchUUID, userUUID uuid.UUID) (*match.ListMatchEnrollmentsResult, error)
}

func (m *mockListMatchEnrollments) List(
	ctx context.Context, matchUUID, userUUID uuid.UUID,
) (*match.ListMatchEnrollmentsResult, error) {
	return m.fn(ctx, matchUUID, userUUID)
}

type mockCreateMatch struct {
	fn func(ctx context.Context, input *match.CreateMatchInput) (*matchEntity.Match, error)
}

func (m *mockCreateMatch) CreateMatch(ctx context.Context, input *match.CreateMatchInput) (*matchEntity.Match, error) {
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

type mockGetMatchParticipants struct {
	fn func(ctx context.Context, matchUUID, userUUID uuid.UUID) (*match.GetMatchParticipantsResult, error)
}

func (m *mockGetMatchParticipants) Get(
	ctx context.Context, matchUUID, userUUID uuid.UUID,
) (*match.GetMatchParticipantsResult, error) {
	return m.fn(ctx, matchUUID, userUUID)
}

type mockUpdateMatch struct {
	fn func(ctx context.Context, input *match.UpdateMatchInput) (*matchEntity.Match, error)
}

func (m *mockUpdateMatch) Update(
	ctx context.Context, input *match.UpdateMatchInput,
) (*matchEntity.Match, error) {
	return m.fn(ctx, input)
}

type mockDeleteMatch struct {
	fn func(ctx context.Context, input *match.DeleteMatchInput) error
}

func (m *mockDeleteMatch) Delete(ctx context.Context, input *match.DeleteMatchInput) error {
	return m.fn(ctx, input)
}
