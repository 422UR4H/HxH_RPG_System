// internal/app/api/map/api.go
package mapapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	CreateMapHandler Handler[CreateMapRequest, CreateMapResponse]
	ListMapsHandler  Handler[ListMapsRequest, ListMapsResponse]
	GetMapHandler    Handler[GetMapRequest, GetMapResponse]
	UpdateMapHandler Handler[UpdateMapRequest, UpdateMapResponse]
	DeleteMapHandler Handler[DeleteMapRequest, DeleteMapResponse]
}

func (a *Api) RegisterRoutes(_ *chi.Mux, api huma.API, _ *zap.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "create-map",
		Method:      http.MethodPost,
		Path:        "/campaigns/{campaign_id}/maps",
		Description: "Create a new tactical map for a campaign",
		Tags:        []string{"maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, a.CreateMapHandler)

	huma.Register(api, huma.Operation{
		OperationID: "list-maps",
		Method:      http.MethodGet,
		Path:        "/campaigns/{campaign_id}/maps",
		Description: "List all maps for a campaign (master only)",
		Tags:        []string{"maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusInternalServerError,
		},
	}, a.ListMapsHandler)

	huma.Register(api, huma.Operation{
		OperationID: "get-map",
		Method:      http.MethodGet,
		Path:        "/maps/{map_id}",
		Description: "Get a tactical map by ID",
		Tags:        []string{"maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusNotFound,
			http.StatusInternalServerError,
		},
	}, a.GetMapHandler)

	huma.Register(api, huma.Operation{
		OperationID: "update-map",
		Method:      http.MethodPut,
		Path:        "/maps/{map_id}",
		Description: "Update a tactical map (master only)",
		Tags:        []string{"maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusNoContent,
	}, a.UpdateMapHandler)

	huma.Register(api, huma.Operation{
		OperationID: "delete-map",
		Method:      http.MethodDelete,
		Path:        "/maps/{map_id}",
		Description: "Delete a tactical map (master only)",
		Tags:        []string{"maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusNoContent,
	}, a.DeleteMapHandler)
}
