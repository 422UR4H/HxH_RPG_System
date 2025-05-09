package match

import "errors"

var (
	ErrCampaignNotFound   = errors.New("campaign not found")
	ErrMinTitleLength     = errors.New("title must be at least 5 characters")
	ErrMaxTitleLength     = errors.New("title cannot exceed 32 characters")
	ErrMinOfStartDate     = errors.New("story start date must be after campaign start date")
	ErrMaxOfStartDate     = errors.New("story start date must be before campaign end date")
	ErrMaxBriefDescLength = errors.New("brief description cannot exceed 64 characters")
)
