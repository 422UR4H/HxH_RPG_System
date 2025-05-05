package scenario

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	CreateScenarioHandler Handler[CreateScenarioRequest, CreateScenarioResponse]
	GetScenarioHandler    Handler[GetScenarioRequest, GetScenarioResponse]
	ListScenariosHandler  Handler[struct{}, ListScenariosResponse]
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/scenarios",
		Description: "Create a new scenario",
		Tags:        []string{"scenarios"},
		Errors: []int{
			http.StatusConflict,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, a.CreateScenarioHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/scenarios/{uuid}",
		Description: "Get a scenario by UUID",
		Tags:        []string{"scenarios"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.GetScenarioHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/scenarios",
		Description: "List all user's scenarios",
		Tags:        []string{"scenarios"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.ListScenariosHandler)
}
