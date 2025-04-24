package auth

import (
	"context"
	"net/http"
	"time"

	domainUser "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	pgUser "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/user"
	"github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	repo *pgUser.Repository
}

func NewAuthHandler(repo *pgUser.Repository) *AuthHandler {
	return &AuthHandler{repo: repo}
}

func (h *AuthHandler) Register(
	ctx context.Context, req *RegisterRequest,
) (*RegisterResponse, error) {
	u := &domainUser.User{
		Nick:      req.Nick,
		Email:     req.Email,
		Password:  req.Password,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.repo.CreateUser(ctx, u); err != nil {
		return nil, err
	}
	return &RegisterResponse{Message: "User registered successfully"}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	u, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		return nil, err
	}
	token, err := auth.GenerateToken(u.UUID)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{Token: token}, nil
}

func (h *AuthHandler) RegisterRoutes(r *chi.Mux, api huma.API) {
	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/auth/register",
		Description: "Register a new user",
		Errors: []int{
			http.StatusConflict,
			http.StatusBadRequest,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusCreated,
	}, h.Register)

	huma.Register(api, huma.Operation{
		Method:      http.MethodPost,
		Path:        "/auth/login",
		Description: "Login a user",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
		DefaultStatus: http.StatusOK,
	}, h.Login)
}
