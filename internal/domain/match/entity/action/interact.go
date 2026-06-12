package action

type InteractKind string

const (
	InteractOpen     InteractKind = "open"
	InteractClose    InteractKind = "close"
	InteractToggle   InteractKind = "toggle"
	InteractLockpick InteractKind = "lockpick"
	InteractExamine  InteractKind = "examine"
)

type Interact struct {
	Kind InteractKind
}
