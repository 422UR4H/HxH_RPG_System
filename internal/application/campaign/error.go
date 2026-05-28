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
	ErrMaxBriefDescLength      = domain.NewValidationError(errors.New("brief description cannot exceed 255 characters"))
	ErrMaxCallLinkLength       = domain.NewValidationError(errors.New("call link cannot exceed 255 characters"))
	ErrCampaignHasStartedMatch = domain.NewValidationError(errors.New("campaign has a match that has already started"))
	ErrCampaignAlreadyEnded        = domain.NewValidationError(errors.New("campaign has already ended"))
	ErrLockedAfterMatchStart       = domain.NewValidationError(errors.New("name and story_start_at cannot be changed after a match has started"))
	ErrCannotRegressStoryCurrentAt = domain.NewValidationError(errors.New("story_current_at cannot be set to a date earlier than the current value"))
)
