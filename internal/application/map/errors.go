// internal/application/map/errors.go
package mapuc

import "errors"

var (
	ErrMapNotFound  = errors.New("map not found")
	ErrNotMapMaster = errors.New("only the campaign master can perform this action")
)
