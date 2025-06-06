package model

import "github.com/google/uuid"

type CharacterSheetRelationshipUUIDs struct {
	CampaignUUID *uuid.UUID
	PlayerUUID   *uuid.UUID
	MasterUUID   *uuid.UUID
}
