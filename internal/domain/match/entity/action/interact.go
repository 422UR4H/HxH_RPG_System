package action

type InteractKind string

const (
	InteractOpen     InteractKind = "open"
	InteractClose    InteractKind = "close"
	InteractToggle   InteractKind = "toggle"
	InteractLockpick InteractKind = "lockpick"
	InteractExamine  InteractKind = "examine"
	// InteractReveal is a master-only action that reveals a secret door to all players.
	InteractReveal InteractKind = "reveal"
)

type Interact struct {
	Kind InteractKind
}
