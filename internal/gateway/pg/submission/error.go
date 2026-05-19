package submission

import "errors"

var (
	ErrSubmissionNotFound = errors.New("submission not found in database")
	ErrNickConflict       = errors.New("nick conflict in campaign")
)
