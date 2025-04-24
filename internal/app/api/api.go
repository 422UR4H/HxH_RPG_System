package api

import (
	"context"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type IApi interface {
	RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger)
}

type Api struct {
	LivenessHandler       Handler[struct{}, HealthResponse]
	ReadinessHandler      Handler[struct{}, HealthResponse]
	CharacterSheetHandler IApi
	AuthHandler           *auth.AuthHandler
	Logger                *zap.Logger
}

func (a *Api) Routes(r *chi.Mux, authMiddleware func(ctx huma.Context, next func(huma.Context))) huma.API {
	huma.NewError = NewErrorWithType

	api := humachi.New(r, newConfig(
		"HxH RPG System", "v0-pre-alpha", "Core Rules API for HxH RPG System (Pre-Alpha Version)",
	))
	a.registerHealthRoutes(api)
	a.AuthHandler.RegisterRoutes(r, api)

	api.UseMiddleware(authMiddleware)
	a.CharacterSheetHandler.RegisterRoutes(r, api, a.Logger)

	return api
}

func (a *Api) registerHealthRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/liveness",
	}, a.LivenessHandler)

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/readiness",
	}, a.ReadinessHandler)
}

func newConfig(title, version, description string) huma.Config {
	schemaPrefix := "#/components/schemas/"
	schemasPath := "/schemas"
	registry := huma.NewMapRegistry(schemaPrefix, huma.DefaultSchemaNamer)
	linkTransformer := huma.NewSchemaLinkTransformer(schemaPrefix, schemasPath)

	return huma.Config{
		OpenAPI: &huma.OpenAPI{
			OpenAPI:        "3.1.0",
			Components:     &huma.Components{Schemas: registry},
			OnAddOperation: []huma.AddOpFunc{linkTransformer.OnAddOperation},

			Info: &huma.Info{
				Title:       title,
				Version:     version,
				Description: description,
			},

			Servers: []*huma.Server{
				{URL: "http://localhost:5000"},
			},
		},
		OpenAPIPath:  "/openapi",
		DocsPath:     "/docs",
		SchemasPath:  schemasPath,
		Transformers: []huma.Transformer{},

		Formats: map[string]huma.Format{
			"application/json": huma.DefaultJSONFormat,
			"json":             huma.DefaultJSONFormat,
		},
		DefaultFormat: "application/json",
	}
}
