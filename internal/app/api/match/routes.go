package match

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	CreateMatchHandler               Handler[CreateMatchRequest, CreateMatchResponse]
	GetMatchHandler                  Handler[GetMatchRequest, GetMatchResponse]
	ListMatchesHandler               Handler[struct{}, ListMatchesResponse]
	ListPublicUpcomingMatchesHandler Handler[struct{}, ListMatchesResponse]
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/matches",
		Description: "Create a new match for a campaign",
		Tags:        []string{"matches"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusForbidden,
			http.StatusUnauthorized,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, a.CreateMatchHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/matches/{uuid}",
		Description: "Get a match by UUID",
		Tags:        []string{"matches"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusForbidden,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.GetMatchHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/matches",
		Description: "List all master's matches summaries sorted by story_start_at",
		Tags:        []string{"matches"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.ListMatchesHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/public/matches",
		Description: "List all upcoming public matches sorted by game_start_at",
		Tags:        []string{"matches"},
		Errors: []int{
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.ListPublicUpcomingMatchesHandler)
}
