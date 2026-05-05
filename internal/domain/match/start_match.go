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

type IMatchParticipantWriter interface {
	RegisterFromAcceptedEnrollments(
		ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
	) error
}

type StartMatchUC struct {
	matchRepo       IRepository
	enrollmentRepo  IEnrollmentRepository
	participantRepo IMatchParticipantWriter
}

func NewStartMatchUC(
	matchRepo IRepository,
	enrollmentRepo IEnrollmentRepository,
	participantRepo IMatchParticipantWriter,
) *StartMatchUC {
	return &StartMatchUC{
		matchRepo:       matchRepo,
		enrollmentRepo:  enrollmentRepo,
		participantRepo: participantRepo,
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
	if err := uc.enrollmentRepo.RejectPendingEnrollments(ctx, matchUUID); err != nil {
		return err
	}
	return uc.participantRepo.RegisterFromAcceptedEnrollments(ctx, matchUUID, gameStartAt)
}
