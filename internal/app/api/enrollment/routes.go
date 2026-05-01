package enrollment

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	EnrollCharacterHandler  Handler[EnrollCharacterRequest, EnrollCharacterResponse]
	AcceptEnrollmentHandler Handler[AcceptEnrollmentRequest, AcceptEnrollmentResponse]
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/enrollments/charactersheets/enroll",
		Description: "Enroll a character sheet in a match",
		Tags:        []string{"enrollments"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusConflict,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, a.EnrollCharacterHandler)
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/enrollments/{uuid}/accept",
		Description: "Accept a character sheet enrollment in a match",
		Tags:        []string{"enrollments"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusInternalServerError,
		},
	}, a.AcceptEnrollmentHandler)
}
