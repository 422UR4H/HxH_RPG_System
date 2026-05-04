package charactersheet

import "github.com/google/uuid"

type RelationshipUUIDs struct {
	CampaignUUID *uuid.UUID
	PlayerUUID   *uuid.UUID
	MasterUUID   *uuid.UUID
}
