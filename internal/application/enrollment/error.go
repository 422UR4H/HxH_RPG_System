package enrollment

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCharacterNotInCampaign   = domain.NewValidationError(errors.New("character sheet does not belong to the match's campaign"))
	ErrCharacterAlreadyEnrolled = domain.NewValidationError(errors.New("character sheet is already enrolled in this match"))
	ErrEnrollmentNotFound       = domain.NewValidationError(errors.New("enrollment not found"))
	ErrNotMatchMaster           = domain.NewValidationError(errors.New("user is not the match's campaign master"))
	ErrMatchAlreadyStarted      = domain.NewValidationError(errors.New("match has already started"))
	ErrMatchAlreadyFinished     = domain.NewValidationError(errors.New("match has already finished"))
	ErrPlayerNotEnrolled        = domain.NewValidationError(errors.New("player is not enrolled in this match"))
)
