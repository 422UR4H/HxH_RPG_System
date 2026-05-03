package enrollment

import (
	"time"

	sheetModel "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

type PlayerRef struct {
	UUID uuid.UUID
	Nick string
}

type Enrollment struct {
	UUID      uuid.UUID
	Status    string
	CreatedAt time.Time
	// TODO(architecture): CharacterSheetSummary lives in gateway/pg/model — entity should not
	// import outer layers. Tracked for cleanup: move CharacterSheetSummary to
	// domain/entity/character_sheet/summary.go in a follow-up task and update all call sites
	// (use cases under domain/character_sheet/ already import model.CharacterSheetSummary too,
	// so the cleanup is shared, not specific to enrollment).
	CharacterSheet sheetModel.CharacterSheetSummary
	Player         PlayerRef
}
