package rules_test

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/422UR4H/HxH_RPG_System/docs"
	"github.com/422UR4H/HxH_RPG_System/internal/application/rules"
)

func TestSplitTitle(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantTitle string
		wantBody  string
		wantErr   error
	}{
		{"h1 and body", "# Atributos\n\nTexto", "Atributos", "Texto", nil},
		{"leading blank lines", "\n\n# Atributos\n\nTexto", "Atributos", "Texto", nil},
		{"h2 only", "## Sub\n", "", "", rules.ErrMissingTitle},
		{"empty", "", "", "", rules.ErrMissingTitle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, body, err := rules.SplitTitle(tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if title != tt.wantTitle || body != tt.wantBody {
				t.Errorf("got (%q, %q), want (%q, %q)", title, body, tt.wantTitle, tt.wantBody)
			}
		})
	}
}

func TestStripDevFooter(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"footer with separator", "corpo\n\n---\n\n> **🔧 Para Desenvolvedores**\n>\n> x\n", "corpo\n"},
		{"no footer", "corpo\n\n\n", "corpo\n"},
		{"marker without separator", "corpo\n> **🔧 Para Desenvolvedores**\n> x\n", "corpo\n"},
		{"separator in the middle kept", "a\n\n---\n\nb\n", "a\n\n---\n\nb\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rules.StripDevFooter(tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadChapters_FromMapFS(t *testing.T) {
	fsys := fstest.MapFS{
		"game/a.md": {Data: []byte("# Alfa\n\ncorpo a\n\n---\n\n> **🔧 Para Desenvolvedores**\n> x\n")},
		"game/b.md": {Data: []byte("# Beta\n\ncorpo b\n")},
	}
	sources := []rules.ChapterSource{
		{Slug: "a", Label: "A", Path: "game/a.md"},
		{Slug: "b", Label: "B", Path: "game/b.md", ComingSoon: true},
	}
	got, err := rules.LoadChapters(fsys, sources)
	if err != nil {
		t.Fatal(err)
	}
	want := []rules.Chapter{
		{Slug: "a", Label: "A", Title: "Alfa", Markdown: "corpo a\n"},
		{Slug: "b", Label: "B", Title: "Beta", Markdown: "corpo b\n", ComingSoon: true},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("chapter %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestLoadChapters_MissingFile(t *testing.T) {
	_, err := rules.LoadChapters(fstest.MapFS{}, []rules.ChapterSource{{Slug: "x", Path: "game/x.md"}})
	if err == nil || !strings.Contains(err.Error(), "game/x.md") {
		t.Fatalf("err = %v, want mention of game/x.md", err)
	}
}

func TestSheetChapters_RealDocs(t *testing.T) {
	chapters, err := rules.LoadChapters(docs.GameFS, rules.SheetChapterSources)
	if err != nil {
		t.Fatal(err)
	}
	slugs := []string{"criacao", "classes", "aptidoes", "atributos", "pericias",
		"proficiencias", "experiencia", "status", "sistema-nen", "glossario"}
	if len(chapters) != len(slugs) {
		t.Fatalf("len = %d, want %d", len(chapters), len(slugs))
	}
	for i, c := range chapters {
		if c.Slug != slugs[i] {
			t.Errorf("chapter %d slug = %s, want %s", i, c.Slug, slugs[i])
		}
		if c.ComingSoon != (i == 8) {
			t.Errorf("chapter %s comingSoon = %v", c.Slug, c.ComingSoon)
		}
		for _, bad := range []string{"internal/", "docs/dev", "Para Desenvolvedores"} {
			if strings.Contains(c.Markdown, bad) {
				t.Errorf("capítulo %s vaza conteúdo de desenvolvedor (%q)", c.Slug, bad)
			}
		}
	}
}
