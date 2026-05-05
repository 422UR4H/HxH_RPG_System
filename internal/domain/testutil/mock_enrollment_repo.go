package testutil

import (
	"context"

	"github.com/google/uuid"
)

type MockEnrollmentRepo struct {
	EnrollCharacterSheetFn             func(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheetFn     func(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
	GetEnrollmentByUUIDFn              func(ctx context.Context, enrollmentUUID uuid.UUID) (string, uuid.UUID, error)
	AcceptEnrollmentFn                 func(ctx context.Context, enrollmentUUID uuid.UUID) error
	RejectEnrollmentFn                 func(ctx context.Context, enrollmentUUID uuid.UUID) error
	RejectEnrollmentByPlayerAndMatchFn func(ctx context.Context, playerUUID uuid.UUID, matchUUID uuid.UUID) error
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

func (m *MockEnrollmentRepo) GetEnrollmentByUUID(ctx context.Context, enrollmentUUID uuid.UUID) (string, uuid.UUID, error) {
	if m.GetEnrollmentByUUIDFn != nil {
		return m.GetEnrollmentByUUIDFn(ctx, enrollmentUUID)
	}
	return "", uuid.Nil, nil
}

func (m *MockEnrollmentRepo) AcceptEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error {
	if m.AcceptEnrollmentFn != nil {
		return m.AcceptEnrollmentFn(ctx, enrollmentUUID)
	}
	return nil
}

func (m *MockEnrollmentRepo) RejectEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error {
	if m.RejectEnrollmentFn != nil {
		return m.RejectEnrollmentFn(ctx, enrollmentUUID)
	}
	return nil
}

func (m *MockEnrollmentRepo) RejectEnrollmentByPlayerAndMatch(ctx context.Context, playerUUID uuid.UUID, matchUUID uuid.UUID) error {
	if m.RejectEnrollmentByPlayerAndMatchFn != nil {
		return m.RejectEnrollmentByPlayerAndMatchFn(ctx, playerUUID, matchUUID)
	}
	return nil
}
