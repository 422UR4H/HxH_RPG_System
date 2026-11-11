package domain_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
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

// Impressão digital da tabela: varre todos os code points (sem surrogates) e confere
// quantidade e SHA-256 da lista. Se a tabela mudar, a mudança vale para os dois
// repositórios: recalcule os dois valores e atualize o teste do front
// (System_X_System_React) no mesmo par de PRs.
func TestContainsEmoji_TableFingerprint(t *testing.T) {
	var parts []string
	for r := rune(0); r <= 0x10FFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue
		}
		if domain.ContainsEmoji(string(r)) {
			parts = append(parts, strings.ToUpper(strconv.FormatInt(int64(r), 16)))
		}
	}
	const wantCount = 3416
	const wantHash = "63362d08506626d36ea1fb3cfe4fcc66b63dfba9c9d38bab50f5bb198980e780"
	if len(parts) != wantCount {
		t.Errorf("emoji code point count = %d, want %d", len(parts), wantCount)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, ",")))
	if got := hex.EncodeToString(sum[:]); got != wantHash {
		t.Errorf("fingerprint = %s, want %s", got, wantHash)
	}
}

// Bordas de cada faixa; a lista é escrita aqui de propósito, sem importar a de produção.
func TestContainsEmoji_RangeEdges(t *testing.T) {
	ranges := [][2]rune{
		{0x00A9, 0x00A9}, {0x00AE, 0x00AE}, {0x203C, 0x203C}, {0x2049, 0x2049},
		{0x2122, 0x2122}, {0x2139, 0x2139}, {0x2194, 0x2199}, {0x21A9, 0x21AA},
		{0x231A, 0x231B}, {0x2328, 0x2328}, {0x23CF, 0x23CF}, {0x23E9, 0x23F3},
		{0x23F8, 0x23FA}, {0x24C2, 0x24C2}, {0x25AA, 0x25AB}, {0x25B6, 0x25B6},
		{0x25C0, 0x25C0}, {0x25FB, 0x25FE}, {0x2600, 0x27BF}, {0x2934, 0x2935},
		{0x2B05, 0x2B07}, {0x2B1B, 0x2B1C}, {0x2B50, 0x2B50}, {0x2B55, 0x2B55},
		{0x3030, 0x3030}, {0x303D, 0x303D}, {0x3297, 0x3297}, {0x3299, 0x3299},
		{0xFE0F, 0xFE0F}, {0x20E3, 0x20E3}, {0x1F000, 0x1FAFF}, {0xE0020, 0xE007F},
	}
	inRanges := func(r rune) bool {
		for _, rg := range ranges {
			if r >= rg[0] && r <= rg[1] {
				return true
			}
		}
		return false
	}
	for _, rg := range ranges {
		for _, r := range []rune{rg[0], rg[1]} {
			if !domain.ContainsEmoji(string(r)) {
				t.Errorf("U+%04X should be emoji", r)
			}
		}
		for _, r := range []rune{rg[0] - 1, rg[1] + 1} {
			if inRanges(r) || (r >= 0xD800 && r <= 0xDFFF) {
				continue
			}
			if domain.ContainsEmoji(string(r)) {
				t.Errorf("U+%04X should not be emoji", r)
			}
		}
	}
}

func TestContainsEmoji_PositionAndLongText(t *testing.T) {
	for _, in := range []string{"Go😀n", "Gon 😀", "😀Gon"} {
		if !domain.ContainsEmoji(in) {
			t.Errorf("ContainsEmoji(%q) = false, want true", in)
		}
	}
	// 1000 caracteres de várias escritas, sem emoji.
	unit := "Ação José ゴンさん Леорио العربية क्\u200dष "
	long := strings.Repeat(unit, 1000/utf8.RuneCountInString(unit)+1)
	long = string([]rune(long)[:1000])
	if domain.ContainsEmoji(long) {
		t.Error("long multi-script text must not contain emoji")
	}
	if n := utf8.RuneCountInString(strings.Repeat("😀", 32)); n != 32 {
		t.Errorf("32 emojis count = %d, want 32", n)
	}
}
