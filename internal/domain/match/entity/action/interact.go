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
	// InteractLock and InteractUnlock are master-only: they set a door's or a window's lock.
	// A character gets past a lock with InteractLockpick, never by unlocking it.
	InteractLock   InteractKind = "lock"
	InteractUnlock InteractKind = "unlock"
)

// IsCharacterInteract reports whether a character may declare this kind in a turn action
// (enqueue_action). Reveal, lock and unlock are the master's own hand on the board.
func (k InteractKind) IsCharacterInteract() bool {
	switch k {
	case InteractOpen, InteractClose, InteractToggle, InteractLockpick, InteractExamine:
		return true
	}
	return false
}

type Interact struct {
	Kind InteractKind
}
