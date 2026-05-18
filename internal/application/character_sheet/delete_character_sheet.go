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
	playerUUID, err := uc.repo.GetCharacterSheetPlayerUUID(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if playerUUID != userUUID {
		return auth.ErrInsufficientPermissions
	}

	rel, err := uc.repo.GetCharacterSheetRelationshipUUIDs(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if rel.CampaignUUID != nil {
		return ErrCharacterSheetNotFreeToManage
	}

	hasSubmission, err := uc.checker.ExistsSubmittedCharacterSheet(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if hasSubmission {
		return ErrCharacterSheetNotFreeToManage
	}

	return uc.repo.DeleteCharacterSheet(ctx, sheetUUID, userUUID)
}
