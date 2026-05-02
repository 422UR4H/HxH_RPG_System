package enrollment

import (
	"context"

	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IKickPlayer interface {
	Kick(ctx context.Context, matchUUID uuid.UUID, playerUUID uuid.UUID, masterUUID uuid.UUID) error
}

type KickPlayerUC struct {
	matchRepo matchDomain.IRepository
	repo      IRepository
}

func NewKickPlayerUC(
	matchRepo matchDomain.IRepository,
	repo IRepository,
) *KickPlayerUC {
	return &KickPlayerUC{
		matchRepo: matchRepo,
		repo:      repo,
	}
}

func (uc *KickPlayerUC) Kick(
	ctx context.Context,
	matchUUID uuid.UUID,
	playerUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if err == matchPg.ErrMatchNotFound {
			return matchDomain.ErrMatchNotFound
		}
		return err
	}

	if match.MasterUUID != masterUUID || playerUUID == masterUUID {
		return ErrNotMatchMaster
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}

	err = uc.repo.RejectEnrollmentByPlayerAndMatch(ctx, playerUUID, matchUUID)
	if err != nil {
		if err.Error() == enrollmentPg.ErrEnrollmentNotFound.Error() {
			return ErrPlayerNotEnrolled
		}
		return err
	}
	return nil
}
