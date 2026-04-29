package domain_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

func TestNewValidationError(t *testing.T) {
	inner := errors.New("field is required")
	err := domain.NewValidationError(inner)

	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(err, domain.ErrValidation) {
		t.Error("expected error to wrap ErrValidation")
	}
}

func TestNewDomainError(t *testing.T) {
	inner := errors.New("business rule violation")
	err := domain.NewDomainError(inner)

	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(err, domain.ErrDomain) {
		t.Error("expected error to wrap ErrDomain")
	}
}

func TestNewDBError(t *testing.T) {
	inner := errors.New("connection refused")
	err := domain.NewDBError(inner)

	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(err, domain.ErrDB) {
		t.Error("expected error to wrap ErrDB")
	}
}

func TestErrorWrapping_ContainsInnerMessage(t *testing.T) {
	inner := errors.New("specific detail")
	err := domain.NewValidationError(inner)

	if got := err.Error(); got == "" {
		t.Error("expected non-empty error message")
	}
}
