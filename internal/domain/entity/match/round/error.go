package round

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrCloseRoundTriggeredCantBeNil = domain.NewDomainError(errors.New("closeRoundTriggered flag cannot be nil"))
)
