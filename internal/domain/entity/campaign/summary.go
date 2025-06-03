package campaign

import (
	"time"

	"github.com/google/uuid"
)

type Summary struct {
	UUID                    uuid.UUID
	ScenarioUUID            uuid.UUID
	Name                    string
	BriefInitialDescription string
	BriefFinalDescription   *string
	IsPublic                bool
	CallLink                string
	StoryStartAt            time.Time
	StoryCurrentAt          *time.Time
	StoryEndAt              *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
