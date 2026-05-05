package match

import (
	"context"
	"time"

	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IStartMatch interface {
	Start(ctx context.Context, matchUUID uuid.UUID, masterUUID uuid.UUID) error
}

// IEnrollmentRepository is a narrow interface to avoid import cycles with the enrollment package.
type IEnrollmentRepository interface {
	RejectPendingEnrollments(ctx context.Context, matchUUID uuid.UUID) error
}

type StartMatchUC struct {
	matchRepo      IRepository
	enrollmentRepo IEnrollmentRepository
}

func NewStartMatchUC(
	matchRepo IRepository,
	enrollmentRepo IEnrollmentRepository,
) *StartMatchUC {
	return &StartMatchUC{
		matchRepo:      matchRepo,
		enrollmentRepo: enrollmentRepo,
	}
}

func (uc *StartMatchUC) Start(
	ctx context.Context,
	matchUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if err == matchPg.ErrMatchNotFound {
			return ErrMatchNotFound
		}
		return err
	}

	if match.MasterUUID != masterUUID {
		return ErrNotMatchMaster
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}
	if match.StoryEndAt != nil {
		return ErrMatchAlreadyFinished
	}

	gameStartAt := time.Now()
	if err := uc.matchRepo.StartMatch(ctx, matchUUID, gameStartAt); err != nil {
		return err
	}

	return uc.enrollmentRepo.RejectPendingEnrollments(ctx, matchUUID)
}
