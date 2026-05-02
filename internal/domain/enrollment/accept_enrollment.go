package enrollment

import (
	"context"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IAcceptEnrollment interface {
	Accept(ctx context.Context, enrollmentUUID uuid.UUID, masterUUID uuid.UUID) error
}

type AcceptEnrollmentUC struct {
	repo         IRepository
	matchRepo    matchDomain.IRepository
	campaignRepo campaignDomain.IRepository
}

func NewAcceptEnrollmentUC(
	repo IRepository,
	matchRepo matchDomain.IRepository,
	campaignRepo campaignDomain.IRepository,
) *AcceptEnrollmentUC {
	return &AcceptEnrollmentUC{
		repo:         repo,
		matchRepo:    matchRepo,
		campaignRepo: campaignRepo,
	}
}

func (uc *AcceptEnrollmentUC) Accept(
	ctx context.Context,
	enrollmentUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	status, matchUUID, err := uc.repo.GetEnrollmentByUUID(ctx, enrollmentUUID)
	if err == enrollmentPg.ErrEnrollmentNotFound {
		return ErrEnrollmentNotFound
	}
	if err != nil {
		return err
	}
	if status == "accepted" {
		return nil
	}

	// TODO: check if match has already started (temporal guard)
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err == matchPg.ErrMatchNotFound {
		return matchDomain.ErrMatchNotFound
	}
	if err != nil {
		return err
	}
	if match.GameStartAt != nil {
		return ErrMatchAlreadyStarted
	}
	if match.StoryEndAt != nil {
		return ErrMatchAlreadyFinished
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, match.CampaignUUID)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if campaignMasterUUID != masterUUID {
		return ErrNotMatchMaster
	}

	return uc.repo.AcceptEnrollment(ctx, enrollmentUUID)
}
