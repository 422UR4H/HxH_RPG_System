package testutil

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MockMatchParticipantWriter struct {
	RegisterFromAcceptedEnrollmentsFn func(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error
}

func (m *MockMatchParticipantWriter) RegisterFromAcceptedEnrollments(
	ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
) error {
	if m.RegisterFromAcceptedEnrollmentsFn != nil {
		return m.RegisterFromAcceptedEnrollmentsFn(ctx, matchUUID, gameStartAt)
	}
	return nil
}
