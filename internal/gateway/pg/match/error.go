package match

import "errors"

var (
	ErrMatchNotFound = errors.New("match not found in database")

	ErrNPCAlreadyInMatch   = errors.New("npc already in match")
	ErrNPCNotInMatch       = errors.New("npc not found in match")
	ErrSheetNotEligibleNPC = errors.New("character sheet is not an npc")
)
