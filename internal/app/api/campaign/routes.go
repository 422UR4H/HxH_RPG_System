package campaign

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	CreateCampaignHandler Handler[CreateCampaignRequest, CreateCampaignResponse]
	GetCampaignHandler    Handler[GetCampaignRequest, GetCampaignResponse]
	ListCampaignsHandler  Handler[struct{}, ListCampaignsResponse]
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/campaigns",
		Description: "Create a new campaign from a scenario",
		Tags:        []string{"campaigns"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, a.CreateCampaignHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/campaigns/{uuid}",
		Description: "Get a campaign by UUID",
		Tags:        []string{"campaigns"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusForbidden,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.GetCampaignHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/campaigns",
		Description: "List all user's campaigns",
		Tags:        []string{"campaigns"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.ListCampaignsHandler)
}
