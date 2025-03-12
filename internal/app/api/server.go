package api

import (
	"github.com/go-chi/chi/v5"
)

func NewServer( /*logger *zap.Logger*/ ) *chi.Mux {
	router := chi.NewRouter()

	// TODO: middleware initialization
	router.Use(
	// add middlewares
	)
	return router
}
