package domain_test

import (
	"testing"
	"unicode/utf8"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

func TestContainsEmoji(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"Gon😀", true}, {"❤", true}, {"✅ ok", true}, {"🇧🇷", true}, {"👍🏽", true},
		{"1️⃣", true}, {"Killua™", true}, {"a©b", true}, {"⭐", true}, {"👨\u200d👩\u200d👧", true},
		{"", false}, {"Gon Freecss", false}, {"Killua_99", false}, {"José Ção", false},
		{"ゴン", false}, {"Леорио", false}, {"a.b-c@d.com", false}, {"क्\u200dष", false},
	}
	for _, tt := range tests {
		if got := domain.ContainsEmoji(tt.in); got != tt.want {
			t.Errorf("ContainsEmoji(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestRuneCountOfEmoji(t *testing.T) {
	tests := map[string]int{"😀": 1, "Gon😀": 4, "🇧🇷": 2, "👨\u200d👩\u200d👧": 5}
	for in, want := range tests {
		if got := utf8.RuneCountInString(in); got != want {
			t.Errorf("rune count of %q = %d, want %d", in, got, want)
		}
	}
}
