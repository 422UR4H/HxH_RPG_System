package matchmapuc

import "errors"

var (
	ErrMatchMapNotFound    = errors.New("match map not found")
	ErrMatchAlreadyStarted = errors.New("cannot change map after match has started")
	ErrNotMatchMaster      = errors.New("only the match master can perform this action")
	ErrMapNotFound         = errors.New("map not found")
	ErrMatchNotFound       = errors.New("match not found")
)
