package enrollment

import (
	"context"

	"github.com/google/uuid"
)

type IRepository interface {
	EnrollCharacterSheet(ctx context.Context, matchUUID uuid.UUID, characterSheetUUID uuid.UUID) error
	ExistsEnrolledCharacterSheet(ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID) (bool, error)
	GetEnrollmentByUUID(ctx context.Context, enrollmentUUID uuid.UUID) (status string, matchUUID uuid.UUID, err error)
	AcceptEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error
	RejectEnrollment(ctx context.Context, enrollmentUUID uuid.UUID) error
}
