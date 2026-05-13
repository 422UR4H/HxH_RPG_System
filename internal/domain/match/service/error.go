package service

import "errors"

var (
	ErrQueueEmpty            = errors.New("action queue is empty")
	ErrActionNotFound        = errors.New("action not found in queue")
	ErrNoCurrentTurn         = errors.New("no current turn in round")
	ErrReactionNotCompatible = errors.New("reaction does not target the current action")
)
