package upload

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Handler is a generic huma handler function type, matching the pattern used
// across other api handler packages (e.g. internal/app/api/sheet).
type Handler[I, O any] func(context.Context, *I) (*O, error)

// Api groups all upload-related huma handlers.
type Api struct {
	PresignedURLHandler Handler[PresignedURLRequest, PresignedURLResponse]
}

// RegisterRoutes registers all upload routes on the given huma.API instance.
func (a *Api) RegisterRoutes(_ *chi.Mux, api huma.API, _ *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/upload/presigned-url",
		Description: "Generate a presigned PUT URL for direct R2 upload",
		Tags:        []string{"upload"},
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusOK,
	}, a.PresignedURLHandler)
}
