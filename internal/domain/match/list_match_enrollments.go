package match

import (
	"context"
	"errors"

	enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

// EnrollmentLister is a local interface (defined at the consumer to avoid a
// cycle: domain/enrollment already imports domain/match). The pg enrollment
// repository satisfies it via structural typing.
type EnrollmentLister interface {
	ListByMatchUUID(
		ctx context.Context, matchUUID uuid.UUID,
	) ([]*enrollmentEntity.Enrollment, error)
}

type ListMatchEnrollmentsResult struct {
	Enrollments    []*enrollmentEntity.Enrollment
	ViewerIsMaster bool
}

type IListMatchEnrollments interface {
	List(
		ctx context.Context, matchUUID uuid.UUID, userUUID uuid.UUID,
	) (*ListMatchEnrollmentsResult, error)
}

type ListMatchEnrollmentsUC struct {
	matchRepo        IRepository
	enrollmentLister EnrollmentLister
}

func NewListMatchEnrollmentsUC(
	matchRepo IRepository,
	enrollmentLister EnrollmentLister,
) *ListMatchEnrollmentsUC {
	return &ListMatchEnrollmentsUC{
		matchRepo:        matchRepo,
		enrollmentLister: enrollmentLister,
	}
}

func (uc *ListMatchEnrollmentsUC) List(
	ctx context.Context, matchUUID uuid.UUID, userUUID uuid.UUID,
) (*ListMatchEnrollmentsResult, error) {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	enrollments, err := uc.enrollmentLister.ListByMatchUUID(ctx, matchUUID)
	if err != nil {
		return nil, err
	}

	return &ListMatchEnrollmentsResult{
		Enrollments:    enrollments,
		ViewerIsMaster: match.MasterUUID == userUUID,
	}, nil
}
