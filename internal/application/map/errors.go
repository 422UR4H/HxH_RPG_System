// internal/application/map/errors.go
package mapuc

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
)

var (
	ErrMapNotFound  = errtext.Sentence("Mapa não encontrado.")
	ErrNotMapMaster = errtext.Sentence("Só o mestre da campanha pode fazer isso.")
)
