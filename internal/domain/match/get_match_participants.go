package match

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type GetMatchParticipantsResult struct {
	Participants   []*matchEntity.Participant
	ViewerIsMaster bool
}

type IGetMatchParticipants interface {
	Get(ctx context.Context, matchUUID, userUUID uuid.UUID) (*GetMatchParticipantsResult, error)
}

type GetMatchParticipantsUC struct {
	matchRepo            IRepository
	participationChecker CampaignParticipationChecker
}

func NewGetMatchParticipantsUC(
	matchRepo IRepository,
	participationChecker CampaignParticipationChecker,
) *GetMatchParticipantsUC {
	return &GetMatchParticipantsUC{
		matchRepo:            matchRepo,
		participationChecker: participationChecker,
	}
}

func (uc *GetMatchParticipantsUC) Get(
	ctx context.Context, matchUUID, userUUID uuid.UUID,
) (*GetMatchParticipantsResult, error) {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	viewerIsMaster := match.MasterUUID == userUUID
	if !match.IsPublic && !viewerIsMaster {
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

	participants, err := uc.matchRepo.ListParticipantsByMatchUUID(ctx, matchUUID)
	if err != nil {
		return nil, err
	}

	return &GetMatchParticipantsResult{
		Participants:   participants,
		ViewerIsMaster: viewerIsMaster,
	}, nil
}
