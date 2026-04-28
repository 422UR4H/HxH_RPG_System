package testutil

import (
	"context"

	"github.com/google/uuid"
)

type MockEnrollmentRepo struct {
	EnrollCharacterSheetFn         func(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheetFn func(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
}

func (m *MockEnrollmentRepo) EnrollCharacterSheet(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error {
	if m.EnrollCharacterSheetFn != nil {
		return m.EnrollCharacterSheetFn(ctx, matchUUID, characterSheetUUID)
	}
	return nil
}

func (m *MockEnrollmentRepo) ExistsEnrolledCharacterSheet(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error) {
	if m.ExistsEnrolledCharacterSheetFn != nil {
		return m.ExistsEnrolledCharacterSheetFn(ctx, characterSheetUUID, matchUUID)
	}
	return false, nil
}
