package enum

// GameEventCategory all types of information that should appear in the match history
type GameEventCategory string

const (
	Death       GameEventCategory = "death"
	Acquisition GameEventCategory = "acquisition"
	Achievement GameEventCategory = "achievement"
	DateChange  GameEventCategory = "date_change"
	Politics    GameEventCategory = "politics"
	Other       GameEventCategory = "other"
)
