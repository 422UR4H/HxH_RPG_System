package api

import (
	"context"
	"net/http"
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

func ReadinessHandler() func(context.Context, *struct{}) (*HealthResponse, error) {
	return func(ctx context.Context, _ *struct{}) (*HealthResponse, error) {
		return &HealthResponse{
			Status: http.StatusOK,
			Body:   `{"message": "Service is ready"}`,
		}, nil
	}
}
