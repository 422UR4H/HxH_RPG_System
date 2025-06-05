package submission

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrNotCharacterSheetOwner     = domain.NewValidationError(errors.New("not the owner of the character sheet"))
	ErrCharacterAlreadySubmitted  = domain.NewValidationError(errors.New("character sheet is already submitted"))
	ErrMasterCannotSubmitOwnSheet = domain.NewValidationError(errors.New("master cannot submit own character sheet"))
	ErrSubmissionNotFound         = domain.NewValidationError(errors.New("character sheet submission not found"))
	ErrNotCampaignMaster          = domain.NewValidationError(errors.New("user is not the campaign master"))
)
