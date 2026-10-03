package matchmapuc

import "errors"

var (
	ErrMatchMapNotFound    = errors.New("match map not found")
	ErrMatchAlreadyStarted = errors.New("cannot change map after match has started")
	ErrNotMatchMaster      = errors.New("only the match master can perform this action")
	ErrMapNotFound         = errors.New("map not found")
	ErrMatchNotFound       = errors.New("match not found")

	// New for B16 (spec §4.3): a match can inherit the board of another match of the SAME
	// campaign, currently on the SAME map, that already has a board of its own.
	ErrSourceMatchNotInCampaign = errors.New("source match is not in the same campaign")
	ErrSourceMatchOnAnotherMap  = errors.New("source match's board is on a different map")
	ErrSourceMatchHasNoBoard    = errors.New("source match has no board to inherit")
	// ErrSourceMatchIsTheSameMatch guards against inheritBoardFromMatchUuid == the match
	// being attached to. Without this check, Copy(dst, dst) would delete the very row its
	// own INSERT…SELECT reads from and fail with a non-sentinel error (500) instead of a
	// clean 422 — fix round 1 (review finding).
	ErrSourceMatchIsTheSameMatch = errors.New("source match cannot be the same match being attached to")
)
