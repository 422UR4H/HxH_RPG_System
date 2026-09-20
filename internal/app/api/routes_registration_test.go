package api_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	campaignHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/campaign"
	enrollmentHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/enrollment"
	mapHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/map"
	matchHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	matchmapHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
	scenarioHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/scenario"
	sheetHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	submissionHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/submission"
	uploadHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/upload"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

// TestRoutes_AllHandlersRegisterTogether builds the exact Api graph cmd/api/main.go
// assembles - every handler package registered on ONE huma.API, the same as a real
// server boot - and asserts it does not panic.
//
// This is the guard that a dead API slipped past: huma's schema registry keys
// components by bare type name across packages, so two DTOs with the same name in
// different packages (e.g. match.SkillResponse vs sheet.SkillResponse) panic the
// moment both are registered together. Every handler test uses humatest, which
// builds a fresh API per test with only that test's own routes - so a cross-package
// collision like that never surfaces there. Only real registration, the one
// cmd/api/main.go performs at startup, exercises the whole schema at once.
//
// Handlers are left as zero-value stubs (nil funcs / nil use-case interfaces):
// huma.Register only needs the request/response TYPES to build a schema, it never
// invokes the handler during registration, so nil handlers are enough to prove
// registration succeeds without wiring any real business logic.
func TestRoutes_AllHandlersRegisterTogether(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("registering all routes together panicked: %v", r)
		}
	}()

	a := api.Api{
		LivenessHandler:       api.LivenessHandler(),
		ReadinessHandler:      api.ReadinessHandler(),
		CharacterSheetHandler: &sheetHandler.Api{},
		ScenarioHandler:       &scenarioHandler.Api{},
		CampaignHandler:       &campaignHandler.Api{},
		MatchHandler:          &matchHandler.Api{},
		SubmissionHandler:     &submissionHandler.Api{},
		EnrollmentHandler:     &enrollmentHandler.Api{},
		MapHandler:            &mapHandler.Api{},
		MatchMapHandler:       &matchmapHandler.Api{},
		UploadHandler:         &uploadHandler.Api{},
		AuthHandler:           auth.NewAuthHandler(nil, nil),
	}

	noopMiddleware := func(ctx huma.Context, next func(huma.Context)) { next(ctx) }
	a.Routes(chi.NewRouter(), noopMiddleware)
}
