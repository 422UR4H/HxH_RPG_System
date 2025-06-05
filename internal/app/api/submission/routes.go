package submission

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Handler[I, O any] func(context.Context, *I) (*O, error)

type Api struct {
	SubmitCharacterSheetHandler  Handler[SubmitCharacterRequest, SubmitCharacterSheetResponse]
	AcceptSheetSubmissionHandler Handler[AcceptSheetSubmissionRequest, AcceptSheetSubmissionResponse]
	RejectSheetSubmissionHandler Handler[RejectSheetSubmissionRequest, RejectSheetSubmissionResponse]
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/submissions/charactersheets/submit",
		Description: "Submit a character sheet to a campaign",
		Tags:        []string{"character_sheets"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusConflict,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, a.SubmitCharacterSheetHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/submissions/{sheet_uuid}/accept",
		Description: "Accept a character sheet submission for a campaign",
		Tags:        []string{"campaigns", "character_sheets"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusInternalServerError,
		},
	}, a.AcceptSheetSubmissionHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/submissions/{sheet_uuid}/reject",
		Description: "Rejeitar uma submissão de ficha de personagem para campanha",
		Tags:        []string{"campaigns", "character_sheets"},
		Errors: []int{
			http.StatusNotFound,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusInternalServerError,
		},
	}, a.RejectSheetSubmissionHandler)
}
