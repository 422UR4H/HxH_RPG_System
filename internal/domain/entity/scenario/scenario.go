package scenario

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type Scenario struct {
	UUID             uuid.UUID
	UserUUID         uuid.UUID
	Name             string
	BriefDescription string
	Description      string
	Campaigns        []*campaign.Summary
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewScenario(
	userUUID uuid.UUID,
	name string,
	briefDescription string,
	description string,
) (*Scenario, error) {
	now := time.Now()
	return &Scenario{
		UUID:             uuid.New(),
		UserUUID:         userUUID,
		Name:             name,
		BriefDescription: briefDescription,
		Description:      description,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}
