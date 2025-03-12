package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var characterClasses sync.Map

// TODO: remove or handle after balancing
var charClassSheets map[enum.CharacterClassName]*sheet.CharacterSheet

type Handler[I, O any] func(context.Context, *I) (*O, error)

type IApi interface {
	RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger)
}

type Api struct {
	LivenessHandler       Handler[struct{}, HealthResponse]
	ReadinessHandler      Handler[struct{}, HealthResponse]
	CharacterSheetHandler IApi
	Logger                *zap.Logger
}

func (a *Api) Routes(r *chi.Mux) huma.API {
	huma.NewError = NewErrorWithType
	// var api huma.API

	// r.Get("/liveness", a.LivenessHandler)
	// r.Get("/readiness", a.ReadinessHandler)

	api := humachi.New(r, newConfig(
		"HxH RPG System", "v0-pre-alpha", "Core Rules API for HxH RPG System (Pre-Alpha Version)",
	))
	a.registerHealthRoutes(api)
	// a.CharacterSheetHandler.RegisterRoutes(r, api, a.Logger)

	charClassSheets = make(map[enum.CharacterClassName]*sheet.CharacterSheet)
	initCharacterClasses()

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

func initCharacterClasses() {
	factory := sheet.NewCharacterSheetFactory()
	ccFactory := cc.NewCharacterClassFactory()
	for name, class := range ccFactory.Build() {
		characterClasses.Store(name, class)
	}

	characterClasses.Range(func(key, value interface{}) bool {
		name := key.(enum.CharacterClassName)
		class := value.(cc.CharacterClass)
		profile := sheet.CharacterProfile{
			NickName:         name.String(),
			Alignment:        class.Profile.Alignment,
			Description:      class.Profile.Description,
			BriefDescription: class.Profile.BriefDescription,
		}
		set, err := sheet.NewTalentByCategorySet(
			map[enum.CategoryName]bool{
				enum.Reinforcement:   true,
				enum.Transmutation:   true,
				enum.Materialization: true,
				enum.Specialization:  true,
				enum.Manipulation:    true,
				enum.Emission:        true,
			},
			nil,
		)
		if err != nil {
			fmt.Println(err)
		}
		newClass, err := factory.Build(profile, set, &class)
		if err != nil {
			fmt.Println(err)
		}
		charClassSheets[name] = newClass
		// uncomment to print all character classes
		fmt.Println(newClass.ToString())
		return true
	})
}

func GetCharacterClass(name enum.CharacterClassName) (cc.CharacterClass, error) {

	class, exists := characterClasses.Load(name)
	if !exists {
		return cc.CharacterClass{}, fmt.Errorf("character class %s not found", name)
	}
	return class.(cc.CharacterClass), nil
}

func GetAllCharacterClasses() map[enum.CharacterClassName]cc.CharacterClass {
	charClasses := make(map[enum.CharacterClassName]cc.CharacterClass)

	characterClasses.Range(func(key, value interface{}) bool {
		name := key.(enum.CharacterClassName)
		class := value.(cc.CharacterClass)
		charClasses[name] = class
		return true
	})
	return charClasses
}
