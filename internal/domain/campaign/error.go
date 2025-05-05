package campaign

import "errors"

var (
	ErrScenarioNotFound   = errors.New("scenario not found")
	ErrCampaignNotFound   = errors.New("campaign not found")
	ErrMinNameLength      = errors.New("name must be at least 5 characters")
	ErrMaxNameLength      = errors.New("name cannot exceed 32 characters")
	ErrMaxBriefDescLength = errors.New("brief description cannot exceed 64 characters")
)
