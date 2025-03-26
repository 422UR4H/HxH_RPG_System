package domain

import (
	"errors"
	"fmt"
)

var (
	ErrValidation = errors.New("validation error")
	ErrDomain     = errors.New("domain base error")
	ErrDB         = errors.New("database error")
)

func NewValidationError(err error) error {
	return fmt.Errorf("%w: %v", ErrValidation, err)
}

func NewDomainError(err error) error {
	return fmt.Errorf("%w: %v", ErrDomain, err)
}

func NewDBError(err error) error {
	return fmt.Errorf("%w: %v", ErrDB, err)
}
