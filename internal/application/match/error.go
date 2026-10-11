package match

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrMatchNotFound        = domain.NewValidationError(errors.New("match not found"))
	ErrMinTitleLength       = domain.NewValidationError(errors.New("title must be at least 5 characters"))
	ErrMaxTitleLength       = domain.NewValidationError(errors.New("title cannot exceed 32 characters"))
	ErrMinOfStoryStartAt    = domain.NewValidationError(errors.New("story start date must be after campaign start date"))
	ErrMaxOfStoryStartAt    = domain.NewValidationError(errors.New("story start date must be before campaign end date"))
	ErrMinOfGameScheduledAt = domain.NewValidationError(errors.New("game scheduled at cannot be in the past"))
	ErrMaxOfGameScheduledAt = domain.NewValidationError(errors.New("game scheduled at cannot be greater than one year from now"))
	ErrMaxBriefDescLength   = domain.NewValidationError(errors.New("brief description cannot exceed 255 characters"))
	ErrMatchAlreadyStarted  = domain.NewValidationError(errors.New("match has already started"))
	ErrMatchAlreadyFinished = domain.NewValidationError(errors.New("match has already finished"))
	ErrNotMatchMaster       = domain.NewValidationError(errors.New("user is not the match master"))
	// ErrRoundHasNoTurns: ending a round in which no turn ever opened. A round closes on what
	// happened in it; ending empty rounds one after another would only spin the clock.
	ErrRoundHasNoTurns = domain.NewValidationError(errors.New("no turn opened in this round yet"))
	ErrInvalidRoundMode     = domain.NewValidationError(errors.New("round mode must be Free or Race"))

	ErrSheetNotNPC            = domain.NewValidationError(errors.New("character sheet is not an npc"))
	ErrSheetNotOwnedByMaster  = domain.NewValidationError(errors.New("character sheet does not belong to the master or the campaign"))
	ErrNPCAlreadyInMatch      = domain.NewValidationError(errors.New("npc is already in this match"))
	ErrNPCNotInMatch          = domain.NewValidationError(errors.New("npc is not in this match"))
	ErrCharacterSheetNotFound = domain.NewValidationError(errors.New("character sheet not found"))
)
