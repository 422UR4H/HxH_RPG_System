package charactersheet_test

import (
	"fmt"
	"sync"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
)

func newTestFactory() *sheet.CharacterSheetFactory {
	return sheet.NewCharacterSheetFactory()
}

func newTestClassMap() *sync.Map {
	classMap := &sync.Map{}
	swordsman := cc.BuildSwordsman()
	classMap.Store(enum.Swordsman, swordsman)
	ninja := cc.BuildNinja()
	classMap.Store(enum.Ninja, ninja)
	return classMap
}

func newTestSheetMap() *sync.Map {
	return &sync.Map{}
}

func newValidProfile() sheet.CharacterProfile {
	return sheet.CharacterProfile{
		NickName:  "Gon",
		FullName:  "Gon Freecss",
		Alignment: "Chaotic-Good",
	}
}

func newValidCreateInput() *charactersheet.CreateCharacterSheetInput {
	playerUUID := uuid.New()
	return &charactersheet.CreateCharacterSheetInput{
		PlayerUUID:        &playerUUID,
		MasterUUID:        nil,
		CampaignUUID:      nil,
		Profile:           newValidProfile(),
		CharacterClass:    enum.Swordsman,
		CategorySet:       *newValidCategorySet(),
		SkillsExps:        map[enum.SkillName]int{},
		ProficienciesExps: map[enum.WeaponName]int{},
	}
}

func newValidCategorySet() *sheet.TalentByCategorySet {
	categories := map[enum.CategoryName]bool{
		enum.Reinforcement: true,
	}
	set, _ := sheet.NewTalentByCategorySet(categories, nil)
	return set
}

func newValidDomainSheet(playerUUID, masterUUID, campaignUUID *uuid.UUID) *sheet.CharacterSheet {
	factory := sheet.NewCharacterSheetFactory()
	profile := sheet.CharacterProfile{
		NickName:  "Gon",
		FullName:  "Gon Freecss",
		Alignment: "Chaotic-Good",
	}
	s, err := factory.Build(playerUUID, masterUUID, campaignUUID, profile, nil, nil, nil)
	if err != nil {
		panic(fmt.Sprintf("newValidDomainSheet: %v", err))
	}
	s.UUID = uuid.New()
	return s
}
