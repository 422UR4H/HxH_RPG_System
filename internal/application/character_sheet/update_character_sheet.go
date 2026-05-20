package charactersheet

import (
	"context"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/google/uuid"
)

type IUpdateCharacterSheet interface {
	UpdateCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID, input *CreateCharacterSheetInput) error
}

type UpdateCharacterSheetUC struct {
	characterClasses *sync.Map
	factory          *sheet.CharacterSheetFactory
	repo             IRepository
	checker          IFreeStateChecker
}

func NewUpdateCharacterSheetUC(
	charClasses *sync.Map,
	factory *sheet.CharacterSheetFactory,
	repo IRepository,
	checker IFreeStateChecker,
) *UpdateCharacterSheetUC {
	return &UpdateCharacterSheetUC{
		characterClasses: charClasses,
		factory:          factory,
		repo:             repo,
		checker:          checker,
	}
}

func (uc *UpdateCharacterSheetUC) UpdateCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID, input *CreateCharacterSheetInput,
) error {
	rel, err := uc.repo.GetCharacterSheetRelationshipUUIDs(ctx, sheetUUID)
	if err != nil {
		return err
	}

	isPlayerOwner := rel.PlayerUUID != nil && *rel.PlayerUUID == userUUID
	isMasterNpc := rel.MasterUUID != nil && *rel.MasterUUID == userUUID && rel.PlayerUUID == nil

	if !isPlayerOwner && !isMasterNpc {
		return auth.ErrInsufficientPermissions
	}

	// Player-owned sheets must be free (no campaign, no pending submission).
	// NPC sheets belong to the master and are always editable.
	if isPlayerOwner {
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
	}

	class, exists := uc.characterClasses.Load(input.CharacterClass)
	if !exists {
		return NewCharacterClassNotFoundError(input.CharacterClass.String())
	}
	charClass := class.(cc.CharacterClass)

	if err := charClass.ValidateSkills(input.SkillsExps); err != nil {
		return err
	}
	if err := charClass.ValidateProficiencies(input.ProficienciesExps); err != nil {
		return err
	}
	charClass.ApplySkills(input.SkillsExps)
	charClass.ApplyProficiencies(input.ProficienciesExps)

	charSheet, err := uc.factory.Build(rel.PlayerUUID, rel.MasterUUID, rel.CampaignUUID, input.Profile, nil, nil, &charClass)
	if err != nil {
		return err
	}
	if len(input.AttributePoints) > 0 {
		if err := charSheet.ApplyInitialAttributePoints(input.AttributePoints); err != nil {
			return err
		}
	}
	charSheet.UUID = sheetUUID

	return uc.repo.UpdateCharacterSheet(ctx, charSheet)
}
