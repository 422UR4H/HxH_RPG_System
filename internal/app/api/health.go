package api

import (
	"context"
	"net/http"
	"time"
)

type HealthResponse struct {
	Status int
	Body   string
}

func LivenessHandler() func(context.Context, *struct{}) (*HealthResponse, error) {
	return func(ctx context.Context, _ *struct{}) (*HealthResponse, error) {
		return &HealthResponse{
			Status: http.StatusOK,
			Body:   `{"message": "Service is healthy"}`,
		}, nil
	}
}

// DBPinger is satisfied by *pgxpool.Pool.
type DBPinger interface {
	Ping(ctx context.Context) error
}

const readinessTimeout = 2 * time.Second

// ReadinessHandler reports whether the service can serve requests, which
// requires the database to answer a ping within readinessTimeout.
func ReadinessHandler(db DBPinger) func(context.Context, *struct{}) (*HealthResponse, error) {
	return func(ctx context.Context, _ *struct{}) (*HealthResponse, error) {
		pingCtx, cancel := context.WithTimeout(ctx, readinessTimeout)
		defer cancel()

		if err := db.Ping(pingCtx); err != nil {
			return &HealthResponse{
				Status: http.StatusServiceUnavailable,
				Body:   `{"message": "Database unavailable"}`,
			}, nil
		}
		return &HealthResponse{
			Status: http.StatusOK,
			Body:   `{"message": "Service is ready"}`,
		}, nil
	}
}
