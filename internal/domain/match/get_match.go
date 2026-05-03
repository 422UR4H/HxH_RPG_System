package match

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IGetMatch interface {
	GetMatch(
		ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID,
	) (*matchEntity.Match, error)
}

type GetMatchUC struct {
	repo                 IRepository
	participationChecker CampaignParticipationChecker
}

func NewGetMatchUC(
	repo IRepository,
	participationChecker CampaignParticipationChecker,
) *GetMatchUC {
	return &GetMatchUC{
		repo:                 repo,
		participationChecker: participationChecker,
	}
}

func (uc *GetMatchUC) GetMatch(
	ctx context.Context, id uuid.UUID, userUUID uuid.UUID,
) (*matchEntity.Match, error) {
	match, err := uc.repo.GetMatch(ctx, id)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	if !match.IsPublic && match.MasterUUID != userUUID {
		ok, err := uc.participationChecker.ExistsSheetInCampaign(
			ctx, userUUID, match.CampaignUUID,
		)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, auth.ErrInsufficientPermissions
		}
	}
	return match, nil
}
