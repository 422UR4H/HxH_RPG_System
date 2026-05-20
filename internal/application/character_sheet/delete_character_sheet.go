package charactersheet

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/google/uuid"
)

// IFreeStateChecker checks whether a sheet has a pending submission.
type IFreeStateChecker interface {
	ExistsSubmittedCharacterSheet(ctx context.Context, sheetUUID uuid.UUID) (bool, error)
}

type IDeleteCharacterSheet interface {
	DeleteCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID) error
}

type DeleteCharacterSheetUC struct {
	repo    IRepository
	checker IFreeStateChecker
}

func NewDeleteCharacterSheetUC(repo IRepository, checker IFreeStateChecker) *DeleteCharacterSheetUC {
	return &DeleteCharacterSheetUC{repo: repo, checker: checker}
}

func (uc *DeleteCharacterSheetUC) DeleteCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID,
) error {
	rel, err := uc.repo.GetCharacterSheetRelationshipUUIDs(ctx, sheetUUID)
	if err != nil {
		return err
	}

	if rel.MasterUUID != nil {
		return uc.deleteNPCSheet(ctx, sheetUUID, *rel.MasterUUID, userUUID)
	}
	return uc.deletePlayerSheet(ctx, sheetUUID, *rel.PlayerUUID, rel.CampaignUUID, userUUID)
}

func (uc *DeleteCharacterSheetUC) deleteNPCSheet(
	ctx context.Context, sheetUUID uuid.UUID, masterUUID uuid.UUID, userUUID uuid.UUID,
) error {
	if masterUUID != userUUID {
		return auth.ErrInsufficientPermissions
	}
	hasParticipated, err := uc.repo.ExistsMatchParticipantForSheet(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if hasParticipated {
		return ErrCharacterSheetNotFreeToManage
	}
	return uc.repo.DeleteNPCCharacterSheet(ctx, sheetUUID, masterUUID)
}

func (uc *DeleteCharacterSheetUC) deletePlayerSheet(
	ctx context.Context, sheetUUID uuid.UUID, playerUUID uuid.UUID, campaignUUID *uuid.UUID, userUUID uuid.UUID,
) error {
	if playerUUID != userUUID {
		return auth.ErrInsufficientPermissions
	}
	if campaignUUID != nil {
		return ErrCharacterSheetNotFreeToManage
	}
	hasSubmission, err := uc.checker.ExistsSubmittedCharacterSheet(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if hasSubmission {
		return ErrCharacterSheetNotFreeToManage
	}
	return uc.repo.DeleteCharacterSheet(ctx, sheetUUID, playerUUID)
}
