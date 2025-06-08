package enrollment

import (
	"context"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
)

type IEnrollCharacterInMatch interface {
	Enroll(
		ctx context.Context,
		matchUUID uuid.UUID,
		characterSheetUUID uuid.UUID,
		playerUUID uuid.UUID,
	) error
}

type EnrollCharacterInMatchUC struct {
	repo      IRepository
	matchRepo matchDomain.IRepository
	sheetRepo charactersheet.IRepository
}

func NewEnrollCharacterInMatchUC(
	repo IRepository,
	matchRepo matchDomain.IRepository,
	sheetRepo charactersheet.IRepository,
) *EnrollCharacterInMatchUC {
	return &EnrollCharacterInMatchUC{
		repo:      repo,
		matchRepo: matchRepo,
		sheetRepo: sheetRepo,
	}
}

func (uc *EnrollCharacterInMatchUC) Enroll(
	ctx context.Context,
	matchUUID uuid.UUID,
	sheetUUID uuid.UUID,
	playerUUID uuid.UUID,
) error {
	sheetRelationship, err := uc.sheetRepo.GetCharacterSheetRelationshipUUIDs(
		ctx, sheetUUID,
	)
	if err == sheet.ErrCharacterSheetNotFound {
		return charactersheet.ErrCharacterSheetNotFound
	}
	if err != nil {
		return err
	}
	// TODO: treat if the request was made by a master too
	if sheetRelationship.PlayerUUID == nil ||
		*sheetRelationship.PlayerUUID != playerUUID {
		return charactersheet.ErrNotCharacterSheetOwner
	}

	alreadyEnrolled, err := uc.repo.ExistsEnrolledCharacterSheet(
		ctx, sheetUUID, matchUUID,
	)
	if err != nil {
		return err
	}
	if alreadyEnrolled {
		return ErrCharacterAlreadyEnrolled
	}

	campaignUUID, err := uc.matchRepo.GetMatchCampaignUUID(ctx, matchUUID)
	if err == matchPg.ErrMatchNotFound {
		return matchDomain.ErrMatchNotFound
	}
	if err != nil {
		return err
	}
	if sheetRelationship.CampaignUUID == nil ||
		*sheetRelationship.CampaignUUID != campaignUUID {
		return ErrCharacterNotInCampaign
	}
	return uc.repo.EnrollCharacterSheet(ctx, matchUUID, sheetUUID)
}
