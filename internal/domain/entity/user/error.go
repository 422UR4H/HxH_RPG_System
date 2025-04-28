package user

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrMissingNick       = domain.NewValidationError(errors.New("nickname is required"))
	ErrInvalidNickLength = domain.NewValidationError(errors.New("nickname must be between 3 and 20 characters"))
	ErrNickAlreadyExists = domain.NewValidationError(errors.New("nickname already exists"))

	ErrInvalidEmailFormat = domain.NewValidationError(errors.New("invalid email format"))
	ErrMissingEmail       = domain.NewValidationError(errors.New("email is required"))
	ErrInvalidEmailLength = domain.NewValidationError(errors.New("email must be between 12 and 64 characters"))
	ErrEmailAlreadyExists = domain.NewValidationError(errors.New("email already exists"))

	ErrMissingPassword   = domain.NewValidationError(errors.New("password is required"))
	ErrMismatchPassword  = domain.NewValidationError(errors.New("passwords mismatch"))
	ErrPasswordMinLenght = domain.NewValidationError(errors.New("password must be longer than 8 characters"))
	ErrPasswordMaxLenght = domain.NewValidationError(errors.New("password is too long"))

	ErrMissingConfirmPass = domain.NewValidationError(errors.New("confirm password is required"))

	ErrUserNotFound = domain.NewValidationError(errors.New("user not found"))
	ErrAccessDenied = domain.NewValidationError(errors.New("access denied"))
)
