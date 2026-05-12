package scene

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
)

var (
	ErrSceneIsFinished = domain.NewDomainError(errors.New("scene is finished"))
)
