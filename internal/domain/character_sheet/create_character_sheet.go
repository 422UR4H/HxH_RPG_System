// internal/domain/character_sheet/create_character_sheet.go
package charactersheet

import (
	"context"
	"sync"

	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type ICreateCharacterSheet interface {
	CreateCharacterSheet(
		ctx context.Context, input *CreateCharacterSheetInput,
	) (*sheet.CharacterSheet, error)
}

type CreateCharacterSheetUC struct {
	characterClasses *sync.Map
	characterSheets  *sync.Map
	factory          *sheet.CharacterSheetFactory
	repo             IRepository
	campaignRepo     domainCampaign.IRepository
}

func NewCreateCharacterSheetUC(
	charClasses *sync.Map,
	charSheets *sync.Map,
	factory *sheet.CharacterSheetFactory,
	repo IRepository,
	campaignRepo domainCampaign.IRepository,
) *CreateCharacterSheetUC {
	return &CreateCharacterSheetUC{
		characterClasses: charClasses,
		characterSheets:  charSheets,
		factory:          factory,
		repo:             repo,
		campaignRepo:     campaignRepo,
	}
}

type DistributionInput struct {
}

type CreateCharacterSheetInput struct {
	PlayerUUID        *uuid.UUID
	MasterUUID        *uuid.UUID
	CampaignUUID      *uuid.UUID
	Profile           sheet.CharacterProfile
	CharacterClass    enum.CharacterClassName
	CategorySet       sheet.TalentByCategorySet
	SkillsExps        map[enum.SkillName]int
	ProficienciesExps map[enum.WeaponName]int
}

func (uc *CreateCharacterSheetUC) CreateCharacterSheet(
	ctx context.Context, input *CreateCharacterSheetInput,
) (*sheet.CharacterSheet, error) {

	class, exists := uc.characterClasses.Load(input.CharacterClass)
	if !exists {
		return nil, NewCharacterClassNotFoundError(input.CharacterClass.String())
	}
	charClass := class.(cc.CharacterClass)

	skillsExps := input.SkillsExps
	if err := charClass.ValidateSkills(skillsExps); err != nil {
		return nil, err
	}
	profExps := input.ProficienciesExps
	if err := charClass.ValidateProficiencies(profExps); err != nil {
		return nil, err
	}
	charClass.ApplySkills(skillsExps)
	charClass.ApplyProficiencies(profExps)

	if err := uc.validateNickName(input.Profile.NickName); err != nil {
		return nil, err
	}

	if input.PlayerUUID != nil {
		characterSheetsCount, err := uc.repo.CountCharactersByPlayerUUID(
			ctx, *input.PlayerUUID,
		)
		if err != nil {
			return nil, err
		}
		if characterSheetsCount >= 20 {
			return nil, ErrMaxCharacterSheetsLimit
		}
	}
	if input.CampaignUUID != nil {
		masterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, *input.CampaignUUID)
		if err == pgCampaign.ErrCampaignNotFound {
			return nil, domainCampaign.ErrCampaignNotFound
		}
		if err != nil {
			return nil, err
		}
		if *input.MasterUUID != masterUUID {
			return nil, domainCampaign.ErrNotCampaignOwner
		}
	}

	set := input.CategorySet
	characterSheet, err := uc.factory.Build(
		input.PlayerUUID,
		input.MasterUUID,
		input.CampaignUUID,
		input.Profile,
		set.GetInitialHexValue(),
		nil,
		&charClass,
	)
	if err != nil {
		return nil, err
	}
	talentLvl := set.GetTalentLvl()
	characterSheet.InitTalentWithLvl(talentLvl)

	characterSheet.UUID = uuid.New()
	uc.characterSheets.Store(characterSheet.UUID, characterSheet)

	if err = uc.repo.CreateCharacterSheet(ctx, characterSheet); err != nil {
		return nil, err
	}
	return characterSheet, nil
}

func (uc *CreateCharacterSheetUC) validateNickName(nick string) error {
	var allowedNickName = true
	uc.characterClasses.Range(func(_, value any) bool {
		charClass := value.(cc.CharacterClass)
		if charClass.GetNameString() == nick {
			allowedNickName = false
			return false
		}
		return true
	})
	if !allowedNickName {
		return NewNicknameNotAllowedError(nick)
	}
	return nil
}
