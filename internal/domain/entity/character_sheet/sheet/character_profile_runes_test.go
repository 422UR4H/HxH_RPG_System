package sheet_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

// Os limites contam caracteres (como o VARCHAR do Postgres), não bytes.
func TestCharacterProfile_LimitsCountCharacters(t *testing.T) {
	profile := func() sheet.CharacterProfile {
		return sheet.CharacterProfile{
			NickName:  "Gon",
			FullName:  "Gon Freecss",
			Alignment: "Chaotic-Good",
			Age:       12,
			Birthday:  time.Date(0, 5, 15, 0, 0, 0, 0, time.UTC),
		}
	}
	tests := []struct {
		name    string
		mutate  func(p *sheet.CharacterProfile)
		wantErr bool
	}{
		{"nickname 10 accented", func(p *sheet.CharacterProfile) { p.NickName = strings.Repeat("ã", 10) }, false},
		{"nickname 11 accented", func(p *sheet.CharacterProfile) { p.NickName = strings.Repeat("ã", 11) }, true},
		{"nickname 3 accented", func(p *sheet.CharacterProfile) { p.NickName = "ããã" }, false},
		{"fullname 32 accented", func(p *sheet.CharacterProfile) { p.FullName = strings.Repeat("ã", 32) }, false},
		{"fullname 33 accented", func(p *sheet.CharacterProfile) { p.FullName = strings.Repeat("ã", 33) }, true},
		{"fullname 6 accented", func(p *sheet.CharacterProfile) { p.FullName = strings.Repeat("ã", 6) }, false},
		{"brief 255 accented", func(p *sheet.CharacterProfile) { p.BriefDescription = strings.Repeat("ã", 255) }, false},
		{"brief 256 accented", func(p *sheet.CharacterProfile) { p.BriefDescription = strings.Repeat("ã", 256) }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := profile()
			tt.mutate(&p)
			if err := p.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("wantErr=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCharacterProfile_RejectsEmoji(t *testing.T) {
	base := func() sheet.CharacterProfile {
		return sheet.CharacterProfile{
			NickName:  "Gon",
			FullName:  "Gon Freecss",
			Alignment: "Chaotic-Good",
			Age:       12,
			Birthday:  time.Date(0, 5, 15, 0, 0, 0, 0, time.UTC),
		}
	}
	p := base()
	p.NickName = "Gon😀"
	if err := p.Validate(); !errors.Is(err, sheet.ErrNicknameHasEmoji) {
		t.Errorf("nickname with emoji: got %v", err)
	}
	p = base()
	p.FullName = "Gon Freecss ❤"
	if err := p.Validate(); !errors.Is(err, sheet.ErrFullNameHasEmoji) {
		t.Errorf("fullname with emoji: got %v", err)
	}
	p = base()
	p.BriefDescription = "Gosta de 😀"
	if err := p.Validate(); err != nil {
		t.Errorf("emoji in brief description must pass: %v", err)
	}
	if err := (func() *sheet.CharacterProfile { q := base(); return &q })().Validate(); err != nil {
		t.Errorf("plain profile must pass: %v", err)
	}
}
