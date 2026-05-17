package charactersheet_test

import (
	"fmt"
	"sync"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
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
	// Swordsman physLvl=3; must distribute exactly 3 primary physical points.
	return &charactersheet.CreateCharacterSheetInput{
		PlayerUUID:     &playerUUID,
		MasterUUID:     nil,
		CampaignUUID:   nil,
		Profile:        newValidProfile(),
		CharacterClass: enum.Swordsman,
		SkillsExps:     map[enum.SkillName]int{},
		ProficienciesExps: map[enum.WeaponName]int{},
		AttributePoints: map[enum.AttributeName]int{
			enum.Resistance:  1,
			enum.Agility:     1,
			enum.Flexibility: 1,
		},
	}
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
