// Package rules monta os capítulos de regras da ficha a partir dos docs de jogo.
package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

// ChapterSource descreve um capítulo. Path é relativo ao FS dos docs,
// ex.: "game/classes.md".
type ChapterSource struct {
	Slug, Label, Path string
	ComingSoon        bool
}

// Chapter é um capítulo pronto para servir: título separado e sem rodapé de dev.
type Chapter struct {
	Slug, Label, Title, Markdown string
	ComingSoon                   bool
}

// ErrMissingTitle indica um doc sem H1 na primeira linha não vazia.
var ErrMissingTitle = errors.New("rules: chapter has no H1 title")

// SheetChapterSources lista os capítulos da ficha, na ordem de exibição.
var SheetChapterSources = []ChapterSource{
	{Slug: "criacao", Label: "Criação", Path: "game/ficha-de-personagem/criacao.md"},
	{Slug: "classes", Label: "Classes", Path: "game/classes.md"},
	{Slug: "aptidoes", Label: "Aptidões", Path: "game/ficha-de-personagem/aptidoes.md"},
	{Slug: "atributos", Label: "Atributos", Path: "game/ficha-de-personagem/atributos.md"},
	{Slug: "pericias", Label: "Perícias", Path: "game/ficha-de-personagem/pericias.md"},
	{Slug: "proficiencias", Label: "Proficiências", Path: "game/ficha-de-personagem/proficiencias.md"},
	{Slug: "experiencia", Label: "Experiência", Path: "game/ficha-de-personagem/experiencia.md"},
	{Slug: "status", Label: "Vida e estamina", Path: "game/ficha-de-personagem/status.md"},
	{Slug: "sistema-nen", Label: "Nen", Path: "game/ficha-de-personagem/sistema-nen.md", ComingSoon: true},
	{Slug: "glossario", Label: "Glossário", Path: "game/glossario.md"},
}

const devFooterMarker = "> **🔧 Para Desenvolvedores**"

// SplitTitle separa o H1 (primeira linha não vazia) do resto do texto.
func SplitTitle(md string) (title, body string, err error) {
	rest := strings.TrimLeft(md, "\r\n \t")
	line, after, _ := strings.Cut(rest, "\n")
	if !strings.HasPrefix(line, "# ") {
		return "", "", ErrMissingTitle
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "# ")), strings.TrimLeft(after, "\r\n"), nil
}

// StripDevFooter remove o rodapé "Para Desenvolvedores" (e o "---" que o
// antecede) e normaliza o fim do texto para um único "\n".
func StripDevFooter(md string) string {
	lines := strings.Split(md, "\n")
	cut := len(lines)
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], devFooterMarker) {
			cut = i
			break
		}
	}
	if cut < len(lines) {
		lines = lines[:cut]
		for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
			lines = lines[:len(lines)-1]
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "---" {
			lines = lines[:len(lines)-1]
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), " \t\r\n") + "\n"
}

// LoadChapters lê e processa cada fonte, preservando a ordem.
func LoadChapters(fsys fs.FS, sources []ChapterSource) ([]Chapter, error) {
	chapters := make([]Chapter, 0, len(sources))
	for _, s := range sources {
		raw, err := fs.ReadFile(fsys, s.Path)
		if err != nil {
			return nil, fmt.Errorf("rules: reading %s: %w", s.Path, err)
		}
		title, body, err := SplitTitle(string(raw))
		if err != nil {
			return nil, fmt.Errorf("rules: %s: %w", s.Path, err)
		}
		chapters = append(chapters, Chapter{
			Slug: s.Slug, Label: s.Label, Title: title,
			Markdown: StripDevFooter(body), ComingSoon: s.ComingSoon,
		})
	}
	return chapters, nil
}
