package submission

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type IRepository interface {
	SubmitCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID, createdAt time.Time) error
	ExistsSubmittedCharacterSheet(ctx context.Context, uuid uuid.UUID) (bool, error)
	AcceptCharacterSheetSubmission(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID) error
	GetSubmissionCampaignUUIDBySheetUUID(ctx context.Context, sheetUUID uuid.UUID) (uuid.UUID, error)
}
