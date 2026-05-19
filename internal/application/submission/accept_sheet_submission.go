package submission

import (
	"context"
	"time"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	submissionPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/submission"
	"github.com/google/uuid"
)

type ISheetBirthdayReader interface {
	GetCharacterSheetBirthInfo(ctx context.Context, sheetUUID uuid.UUID) (time.Time, int, error)
	GetCharacterSheetNick(ctx context.Context, sheetUUID uuid.UUID) (string, error)
}

type IAcceptCharacterSheetSubmission interface {
	Accept(ctx context.Context, sheetUUID uuid.UUID, masterUUID uuid.UUID) error
}

type AcceptCharacterSheetSubmissionUC struct {
	repo         IRepository
	campaignRepo campaignDomain.IRepository
	sheetRepo    ISheetBirthdayReader
}

func NewAcceptCharacterSheetSubmissionUC(
	repo IRepository,
	campaignRepo campaignDomain.IRepository,
	sheetRepo ISheetBirthdayReader,
) *AcceptCharacterSheetSubmissionUC {
	return &AcceptCharacterSheetSubmissionUC{
		repo:         repo,
		campaignRepo: campaignRepo,
		sheetRepo:    sheetRepo,
	}
}

func (uc *AcceptCharacterSheetSubmissionUC) Accept(
	ctx context.Context,
	sheetUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	// TODO: optimize that 2 calls to db to only 1
	campaignUUID, err := uc.repo.GetSubmissionCampaignUUIDBySheetUUID(ctx, sheetUUID)
	if err == submissionPg.ErrSubmissionNotFound {
		return ErrSubmissionNotFound
	}
	if err != nil {
		return err
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, campaignUUID)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if campaignMasterUUID != masterUUID {
		return ErrNotCampaignMaster
	}

	nick, err := uc.sheetRepo.GetCharacterSheetNick(ctx, sheetUUID)
	if err != nil {
		return err
	}
	nickTaken, err := uc.repo.ExistsOtherCharacterWithNickInCampaign(ctx, nick, campaignUUID, sheetUUID)
	if err != nil {
		return err
	}
	if nickTaken {
		return ErrNickAlreadyInCampaign
	}

	campaign, err := uc.campaignRepo.GetCampaignStoryDates(ctx, campaignUUID)
	if err != nil {
		return err
	}
	ref := campaign.StoryCurrentAt
	if ref == nil {
		ref = &campaign.StoryStartAt
	}

	birthday, age, err := uc.sheetRepo.GetCharacterSheetBirthInfo(ctx, sheetUUID)
	if err != nil {
		return err
	}
	birthYear := CalcBirthYear(*ref, birthday, age)
	fullBirthday := time.Date(birthYear, birthday.Month(), birthday.Day(), 0, 0, 0, 0, time.UTC)

	if err = uc.repo.AcceptCharacterSheetSubmission(ctx, sheetUUID, campaignUUID, fullBirthday); err != nil {
		if err == submissionPg.ErrNickConflict {
			return ErrNickAlreadyInCampaign
		}
		return err
	}
	return nil
}

func CalcBirthYear(refDate time.Time, birthday time.Time, age int) int {
	year := refDate.Year() - age
	if int(birthday.Month()) > int(refDate.Month()) ||
		(birthday.Month() == refDate.Month() && birthday.Day() > refDate.Day()) {
		year--
	}
	return year
}
