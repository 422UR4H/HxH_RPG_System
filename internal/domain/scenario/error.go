package scenario

import "errors"

var (
	ErrScenarioNameAlreadyExists = errors.New("scenario name already exists")
	ErrScenarioNotFound          = errors.New("scenario not found")
)
