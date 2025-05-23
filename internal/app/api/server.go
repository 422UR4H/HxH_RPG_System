package api

import (
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func NewServer( /*logger *zap.Logger*/ ) *chi.Mux {
	corsConfig := config.LoadCORS()
	corsMiddleware := CreateCORSHandler(corsConfig)

	router := chi.NewRouter()

	router.Use(
		corsMiddleware,
	)
	return router
}

func CreateCORSHandler(
	corsConfig config.CORSConfig) func(http.Handler) http.Handler {

	return cors.Handler(cors.Options{
		AllowedOrigins:   corsConfig.AllowedOrigins,
		AllowedMethods:   corsConfig.AllowedMethods,
		AllowedHeaders:   corsConfig.AllowedHeaders,
		ExposedHeaders:   corsConfig.ExposedHeaders,
		AllowCredentials: corsConfig.AllowCredentials,
		MaxAge:           corsConfig.MaxAge,
	})
}
