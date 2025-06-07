package spiritual

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrNenHexAlreadyInitialized = domain.NewDomainError(errors.New("nen hexagon already initialized"))
	ErrNenHexNotInitialized     = domain.NewDomainError(errors.New("nen hexagon not initialized"))
	ErrInvalidCategoryPercents  = domain.NewDomainError(errors.New("category percents must have 6 elements"))
	ErrCategoryNotFound         = domain.NewDomainError(errors.New("category not found"))
	ErrPrincipleNotFound        = domain.NewDomainError(errors.New("principle not found"))
)
