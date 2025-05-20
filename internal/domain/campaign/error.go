package campaign

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCampaignNotFound   = domain.NewDBError(errors.New("campaign not found"))
	ErrMinNameLength      = domain.NewValidationError(errors.New("name must be at least 5 characters"))
	ErrMaxNameLength      = domain.NewValidationError(errors.New("name cannot exceed 32 characters"))
	ErrMaxBriefDescLength = domain.NewValidationError(errors.New("brief description cannot exceed 64 characters"))
)
