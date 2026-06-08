// internal/app/api/matchmap/api.go
package matchmapapi

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	AttachMatchMapHandler Handler[AttachMatchMapRequest, AttachMatchMapResponse]
	GetMatchMapHandler    Handler[GetMatchMapRequest, GetMatchMapResponse]
	DetachMatchMapHandler Handler[DetachMatchMapRequest, DetachMatchMapResponse]
}

func (a *Api) RegisterRoutes(_ *chi.Mux, api huma.API, _ *zap.Logger) {
	huma.Register(api, huma.Operation{
		OperationID: "attach-match-map",
		Method:      http.MethodPost,
		Path:        "/matches/{match_uuid}/map",
		Description: "Attach a tactical map to a match (master only, before game starts)",
		Tags:        []string{"match-maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, a.AttachMatchMapHandler)

	huma.Register(api, huma.Operation{
		OperationID: "get-match-map",
		Method:      http.MethodGet,
		Path:        "/matches/{match_uuid}/map",
		Description: "Get the map attached to a match (204 if none)",
		Tags:        []string{"match-maps"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusInternalServerError,
		},
	}, a.GetMatchMapHandler)

	huma.Register(api, huma.Operation{
		OperationID:   "detach-match-map",
		Method:        http.MethodDelete,
		Path:          "/matches/{match_uuid}/map",
		Description:   "Detach the map from a match (master only, before game starts)",
		Tags:          []string{"match-maps"},
		DefaultStatus: http.StatusNoContent,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, a.DetachMatchMapHandler)
}
