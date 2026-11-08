package match

import (
	"errors"
	"fmt"
	"time"
)

const errInvalidStoryStartTime = "storyStartTime must be HH:MM or HH:MM:SS"

// parseStoryStartTime lê "HH:MM" ou "HH:MM:SS" como deslocamento desde a meia-noite.
// Vazio significa "sem horário" (nil).
func parseStoryStartTime(s string) (*time.Duration, error) {
	if s == "" {
		return nil, nil
	}
	var h, m, sec int
	switch len(s) {
	case 5:
		if _, err := fmt.Sscanf(s, "%02d:%02d", &h, &m); err != nil {
			return nil, err
		}
	case 8:
		if _, err := fmt.Sscanf(s, "%02d:%02d:%02d", &h, &m, &sec); err != nil {
			return nil, err
		}
	default:
		return nil, errors.New(errInvalidStoryStartTime)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 || sec < 0 || sec > 59 || s[2] != ':' {
		return nil, errors.New(errInvalidStoryStartTime)
	}
	d := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
	return &d, nil
}

// formatStoryStartTime devolve "HH:MM:SS" ou nil (omitido do JSON).
func formatStoryStartTime(d *time.Duration) *string {
	if d == nil {
		return nil
	}
	total := int(d.Seconds())
	s := fmt.Sprintf("%02d:%02d:%02d", total/3600, total%3600/60, total%60)
	return &s
}

func formatGameEndAt(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
