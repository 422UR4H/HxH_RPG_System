package campaign

import "time"

type PublicSummary struct {
	Summary
	NextGameScheduledAt *time.Time
}
