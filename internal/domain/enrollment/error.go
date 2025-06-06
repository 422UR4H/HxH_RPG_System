package enrollment

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCharacterNotInCampaign     = domain.NewValidationError(errors.New("character sheet does not belong to the match's campaign"))
	ErrCharacterAlreadyEnrolled   = domain.NewValidationError(errors.New("character sheet is already enrolled in a match"))
	ErrMasterCannotEnrollOwnSheet = domain.NewValidationError(errors.New("the master cannot enroll his character sheet in the match itself"))
)
