package clock

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrNotSet              = domain.NewDomainError(errors.New("story clock is not set"))
	ErrAlreadySet          = domain.NewDomainError(errors.New("story clock is already set"))
	ErrBelowLimit          = domain.NewDomainError(errors.New("story time cannot go below the clock limit"))
	ErrLongSkipUnconfirmed = domain.NewDomainError(errors.New("long forward skip requires confirmation"))
	ErrZeroDelta           = domain.NewDomainError(errors.New("story time adjustment has no effect"))
	ErrAlreadyRunning      = domain.NewDomainError(errors.New("story clock is already running"))
)
