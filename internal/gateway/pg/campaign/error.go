package campaign

import "errors"

var (
	ErrCampaignNotFound = errors.New("campaign not found in database")
)
