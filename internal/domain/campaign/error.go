package campaign

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCampaignNotFound   = domain.NewValidationError(errors.New("campaign not found"))
	ErrNotCampaignOwner   = domain.NewValidationError(errors.New("master is not the owner of this campaign"))
	ErrMinNameLength      = domain.NewValidationError(errors.New("name must be at least 5 characters"))
	ErrMaxNameLength      = domain.NewValidationError(errors.New("name cannot exceed 32 characters"))
	ErrInvalidStartDate   = domain.NewValidationError(errors.New("story start date cannot be empty"))
	ErrMaxCampaignsLimit  = domain.NewValidationError(errors.New("user cannot have more than 10 campaigns"))
	ErrMaxBriefDescLength = domain.NewValidationError(errors.New("brief description cannot exceed 64 characters"))
)
