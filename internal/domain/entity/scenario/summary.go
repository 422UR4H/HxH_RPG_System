package scenario

import (
	"time"

	"github.com/google/uuid"
)

type Summary struct {
	UUID             uuid.UUID
	Name             string
	BriefDescription string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
