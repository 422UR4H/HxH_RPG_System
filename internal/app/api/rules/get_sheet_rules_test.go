package rules_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	rulesHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/rules"
	"github.com/422UR4H/HxH_RPG_System/internal/application/rules"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func fixtureChapters() []rules.Chapter {
	return []rules.Chapter{
		{Slug: "um", Label: "Um", Title: "Título Um", Markdown: "corpo um\n"},
		{Slug: "dois", Label: "Dois", Title: "Título Dois", Markdown: "corpo dois\n", ComingSoon: true},
	}
}

func TestGetSheetRulesHandler(t *testing.T) {
	_, api := humatest.New(t)
	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/rules/character-sheet",
	}, rulesHandler.GetSheetRulesHandler(fixtureChapters()))

	resp := api.GetCtx(context.Background(), "/rules/character-sheet")

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control = %q", got)
	}
	var body struct {
		Chapters []map[string]any `json:"chapters"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Chapters) != 2 {
		t.Fatalf("chapters = %d, want 2", len(body.Chapters))
	}
	want := []map[string]any{
		{"slug": "um", "label": "Um", "title": "Título Um", "markdown": "corpo um\n", "comingSoon": false},
		{"slug": "dois", "label": "Dois", "title": "Título Dois", "markdown": "corpo dois\n", "comingSoon": true},
	}
	for i, w := range want {
		for k, v := range w {
			if body.Chapters[i][k] != v {
				t.Errorf("chapters[%d].%s = %v, want %v", i, k, body.Chapters[i][k], v)
			}
		}
	}
}
