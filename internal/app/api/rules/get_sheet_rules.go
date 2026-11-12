package rules

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/application/rules"
)

type RuleChapterResponse struct {
	Slug       string `json:"slug"`
	Label      string `json:"label"`
	Title      string `json:"title"`
	Markdown   string `json:"markdown"`
	ComingSoon bool   `json:"comingSoon"`
}

type GetSheetRulesBody struct {
	Chapters []RuleChapterResponse `json:"chapters"`
}

type GetSheetRulesResponse struct {
	CacheControl string `header:"Cache-Control"`
	Body         GetSheetRulesBody
}

// GetSheetRulesHandler serve capítulos já carregados no boot; o conteúdo é
// imutável durante a vida do processo, então a resposta é montada uma vez.
func GetSheetRulesHandler(
	chapters []rules.Chapter,
) func(context.Context, *struct{}) (*GetSheetRulesResponse, error) {
	out := make([]RuleChapterResponse, len(chapters))
	for i, c := range chapters {
		out[i] = RuleChapterResponse{
			Slug: c.Slug, Label: c.Label, Title: c.Title,
			Markdown: c.Markdown, ComingSoon: c.ComingSoon,
		}
	}
	return func(_ context.Context, _ *struct{}) (*GetSheetRulesResponse, error) {
		return &GetSheetRulesResponse{
			CacheControl: "public, max-age=300",
			Body:         GetSheetRulesBody{Chapters: out},
		}, nil
	}
}
