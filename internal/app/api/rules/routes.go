package rules

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Api struct {
	GetSheetRulesHandler func(context.Context, *struct{}) (*GetSheetRulesResponse, error)
	GetWeaponsHandler    func(context.Context, *struct{}) (*GetWeaponsResponse, error)
}

func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/rules/character-sheet",
		Description: "Rules chapters for the character sheet panel (public)",
		Tags:        []string{"rules"},
		Errors:      []int{http.StatusInternalServerError},
	}, a.GetSheetRulesHandler)

	huma.Register(api, huma.Operation{
		Method:      http.MethodGet,
		Path:        "/rules/weapons",
		Description: "System weapon catalogue: dice and flat damage (public)",
		Tags:        []string{"rules"},
		Errors:      []int{http.StatusInternalServerError},
	}, a.GetWeaponsHandler)
}
