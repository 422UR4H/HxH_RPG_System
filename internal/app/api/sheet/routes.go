package sheet

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	CreateCharacterSheetHandler  Handler[CreateCharacterSheetRequest, CreateCharacterSheetResponse]
	GetCharacterSheetHandler     Handler[GetCharacterSheetRequest, GetCharacterSheetResponse]
	ListCharacterSheetsHandler   Handler[struct{}, ListCharacterSheetsResponse]
	ListClassesHandler           Handler[struct{}, ListCharacterClassesResponse]
	GetClassHandler              Handler[GetCharacterClassRequest, GetCharacterClassResponse]
	UpdateNenHexagonValueHandler Handler[UpdateNenHexagonValueRequest, UpdateNenHexagonValueResponse]
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/charactersheets",
		Description: "Create a new character sheet",
		Tags:        []string{"character_sheets"},
		Errors: []int{
			http.StatusConflict,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, a.CreateCharacterSheetHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/charactersheets/{uuid}",
		Description: "Get a character sheet by UUID",
		Tags:        []string{"character_sheets"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.GetCharacterSheetHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/charactersheets",
		Description: "Get a list of player character sheets",
		Tags:        []string{"character_sheets"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.ListCharacterSheetsHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/classes",
		Description: "List all available character classes",
		Tags:        []string{"character_classes"},
		Errors: []int{
			http.StatusUnauthorized,
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
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.GetClassHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/charactersheets/{character_sheet_uuid}/nen-hexagon/{method}",
		Description: "Update the nen hexagon value of a character sheet",
		Tags:        []string{"character_sheets"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusOK,
	}, a.UpdateNenHexagonValueHandler)
}
