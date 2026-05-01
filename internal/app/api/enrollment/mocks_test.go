package enrollment_test

import (
	"context"

	"github.com/google/uuid"
)

type mockEnrollCharacterInMatch struct {
	fn func(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error
}

func (m *mockEnrollCharacterInMatch) Enroll(ctx context.Context, matchUUID, sheetUUID, playerUUID uuid.UUID) error {
	return m.fn(ctx, matchUUID, sheetUUID, playerUUID)
}

type mockAcceptEnrollment struct {
	fn func(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error
}

func (m *mockAcceptEnrollment) Accept(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error {
	return m.fn(ctx, enrollmentUUID, masterUUID)
}

type mockRejectEnrollment struct {
	fn func(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error
}

func (m *mockRejectEnrollment) Reject(ctx context.Context, enrollmentUUID, masterUUID uuid.UUID) error {
	return m.fn(ctx, enrollmentUUID, masterUUID)
}
