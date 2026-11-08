package match

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
)

const errInvalidStoryStartTime = "storyStartTime must be HH:MM or HH:MM:SS"

// storyStartTimeRe é o formato estrito: dois dígitos por campo, hora 00–23, sem sinal nem espaço.
var storyStartTimeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d(:[0-5]\d)?$`)

// parseStoryStartTime lê "HH:MM" ou "HH:MM:SS" como deslocamento desde a meia-noite.
// Vazio significa "sem horário" (nil).
func parseStoryStartTime(s string) (*time.Duration, error) {
	if s == "" {
		return nil, nil
	}
	if !storyStartTimeRe.MatchString(s) {
		return nil, errors.New(errInvalidStoryStartTime)
	}
	// o regexp garante os dígitos: as conversões não falham
	h, _ := strconv.Atoi(s[0:2])
	m, _ := strconv.Atoi(s[3:5])
	sec := 0
	if len(s) == 8 {
		sec, _ = strconv.Atoi(s[6:8])
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

// formatStoryEndTime devolve o horário de fim na história ("HH:MM:SS") de uma partida encerrada;
// nil (omitido) enquanto ela não terminou.
func formatStoryEndTime(m *matchEntity.Match) *string {
	if !m.IsFinished() {
		return nil
	}
	return formatStoryStartTime(m.StoryEndTime)
}

// formatGameEndAt devolve o instante real do fim em RFC 3339, sempre em UTC (Z); nil sem fim.
func formatGameEndAt(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
