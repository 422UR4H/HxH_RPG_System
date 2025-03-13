package sheet

import (
	"context"
	"net/http"

	// cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	ListClassesHandler Handler[struct{}, ListCharacterClassesResponse]
	GetClassHandler    Handler[GetCharacterClassRequest, GetCharacterClassResponse]
	// CreateCharacterSheetHandler Handler[CreateCharacterSheetRequest, CreateCharacterSheetResponse]
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/classes",
		Description: "List all available character classes",
		Tags:        []string{"character_classes"},
		Errors: []int{
			http.StatusInternalServerError,
		},
	}, a.ListClassesHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/classes/{name}",
		Description: "Get character classe by name",
		Tags:        []string{"character_classes"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusInternalServerError,
		},
	}, a.GetClassHandler)
}
