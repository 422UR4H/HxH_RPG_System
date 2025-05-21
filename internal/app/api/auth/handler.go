package auth

import (
	"context"
	"net/http"

	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	du "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
)

type AuthHandler struct {
	registerUC domainAuth.IRegister
	loginUC    domainAuth.ILogin
}

func NewAuthHandler(registerUC domainAuth.IRegister, loginUC domainAuth.ILogin) *AuthHandler {
	return &AuthHandler{registerUC: registerUC, loginUC: loginUC}
}

func (h *AuthHandler) Register(
	ctx context.Context, req *RegisterRequest,
) (*RegisterResponse, error) {

	input := &domainAuth.RegisterInput{
		Nick:        req.Body.Nick,
		Email:       req.Body.Email,
		Password:    req.Body.Password,
		ConfirmPass: req.Body.ConfirmPass,
	}
	err := h.registerUC.Register(input)
	if err != nil {
		switch err {
		case du.ErrMissingNick,
			du.ErrMissingEmail,
			du.ErrMissingPassword,
			du.ErrMissingConfirmPass:
			return nil, huma.Error400BadRequest(err.Error())

		case du.ErrInvalidNickLength,
			du.ErrInvalidEmailLength,
			du.ErrPasswordMinLenght,
			du.ErrPasswordMaxLenght,
			du.ErrMismatchPassword:
			return nil, huma.Error422UnprocessableEntity(err.Error())

		case du.ErrNickAlreadyExists,
			du.ErrEmailAlreadyExists:
			return nil, huma.Error409Conflict(err.Error())

		default:
			return nil, huma.Error500InternalServerError(err.Error())
		}
	}
	return &RegisterResponse{Status: http.StatusCreated}, nil
}

func (h *AuthHandler) Login(
	ctx context.Context, req *LoginRequest,
) (*LoginResponse, error) {

	input := &domainAuth.LoginInput{
		Email:    req.Body.Email,
		Password: req.Body.Password,
	}
	output, err := h.loginUC.Login(input)
	if err != nil {
		switch err {
		case du.ErrMissingEmail,
			du.ErrMissingPassword:
			return nil, huma.Error400BadRequest(err.Error())
		case du.ErrInvalidEmailLength,
			du.ErrPasswordMinLenght,
			du.ErrPasswordMaxLenght:
			return nil, huma.Error422UnprocessableEntity(err.Error())
		case domainAuth.ErrUnauthorized:
			return nil, huma.Error401Unauthorized(err.Error())
		default:
			return nil, huma.Error500InternalServerError(err.Error())
		}
	}
	return &LoginResponse{
		Body:   LoginResponseBody{Token: output.Token, User: *output.User},
		Status: http.StatusOK,
	}, nil
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
