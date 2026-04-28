package sheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

func TestCharacterProfile_Validate(t *testing.T) {
	validProfile := func() sheet.CharacterProfile {
		return sheet.CharacterProfile{
			NickName:         "Gon",
			FullName:         "Gon Freecss",
			Alignment:        "Chaotic-Good",
			BriefDescription: "A young hunter",
			Age:              12,
		}
	}

	t.Run("valid profile", func(t *testing.T) {
		p := validProfile()
		if err := p.Validate(); err != nil {
			t.Errorf("valid profile returned error: %v", err)
		}
	})

	t.Run("nickname too short", func(t *testing.T) {
		p := validProfile()
		p.NickName = "Go"
		if err := p.Validate(); err == nil {
			t.Error("should reject nickname < 3 chars")
		}
	})

	t.Run("nickname too long", func(t *testing.T) {
		p := validProfile()
		p.NickName = "GonFreecss!"
		if err := p.Validate(); err == nil {
			t.Error("should reject nickname > 10 chars")
		}
	})

	t.Run("fullname too short", func(t *testing.T) {
		p := validProfile()
		p.FullName = "Gon"
		if err := p.Validate(); err == nil {
			t.Error("should reject fullname < 6 chars")
		}
	})

	t.Run("fullname too long", func(t *testing.T) {
		p := validProfile()
		p.FullName = "Gon Freecss The Great Hunter Of All Time!!"
		if err := p.Validate(); err == nil {
			t.Error("should reject fullname > 32 chars")
		}
	})

	t.Run("brief description too long", func(t *testing.T) {
		p := validProfile()
		longDesc := ""
		for i := 0; i < 256; i++ {
			longDesc += "x"
		}
		p.BriefDescription = longDesc
		if err := p.Validate(); err == nil {
			t.Error("should reject brief description > 255 chars")
		}
	})

	t.Run("negative age", func(t *testing.T) {
		p := validProfile()
		p.Age = -1
		if err := p.Validate(); err == nil {
			t.Error("should reject negative age")
		}
	})

	t.Run("zero age valid", func(t *testing.T) {
		p := validProfile()
		p.Age = 0
		if err := p.Validate(); err != nil {
			t.Errorf("age 0 should be valid: %v", err)
		}
	})

	t.Run("empty alignment valid", func(t *testing.T) {
		p := validProfile()
		p.Alignment = ""
		if err := p.Validate(); err != nil {
			t.Errorf("empty alignment should be valid: %v", err)
		}
	})
}

func TestCharacterProfile_ValidateAlignment(t *testing.T) {
	tests := []struct {
		name      string
		alignment string
		wantErr   bool
	}{
		{"Lawful-Good", "Lawful-Good", false},
		{"Neutral-Neutral", "Neutral-Neutral", false},
		{"Chaotic-Evil", "Chaotic-Evil", false},
		{"Lawful-Neutral", "Lawful-Neutral", false},
		{"empty", "", false},
		{"invalid format no dash", "LawfulGood", true},
		{"invalid first", "Random-Good", true},
		{"invalid second", "Lawful-Random", true},
		{"too many parts", "Lawful-Good-Extra", true},
		{"lowercase", "lawful-good", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := sheet.CharacterProfile{Alignment: tt.alignment}
			err := p.ValidateAlignment()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAlignment(%q) error = %v, wantErr %v", tt.alignment, err, tt.wantErr)
			}
		})
	}
}
