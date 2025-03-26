package domain

import (
	"errors"
	"fmt"
)

var (
	ErrDomain = errors.New("domain base error")
	ErrDB     = errors.New("database error")
)

func NewDomainError(err error) error {
	return fmt.Errorf("%w: %v", ErrDomain, err)
}

func NewDBError(err error) error {
	return fmt.Errorf("%w: %v", ErrDB, err)
}
