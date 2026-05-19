package submission

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IRepository interface {
	SubmitCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID, createdAt time.Time) error
	ExistsSubmittedCharacterSheet(ctx context.Context, uuid uuid.UUID) (bool, error)
	AcceptCharacterSheetSubmission(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID, birthday time.Time) error
	GetSubmissionCampaignUUIDBySheetUUID(ctx context.Context, sheetUUID uuid.UUID) (uuid.UUID, error)
	RejectCharacterSheetSubmission(ctx context.Context, sheetUUID uuid.UUID) error
	ExistsOtherCharacterWithNickInCampaign(ctx context.Context, nick string, campaignUUID uuid.UUID, excludedSheetUUID uuid.UUID) (bool, error)
}
