package fog

import (
	"time"

	"github.com/google/uuid"
)

// FeatureKind identifies which kind of static map object a memory entry refers to.
// "Static" means it does not move on its own: walls today, decorations in phase 11.
//
// Character pieces are deliberately NOT memorable. Characters move freely, so the last
// position a player saw tells them nothing reliable about where the piece is now —
// remembering it would misinform rather than inform.
type FeatureKind string

const (
	FeatureWall FeatureKind = "wall"
)

// FeatureRef is a stable reference to one static map object.
type FeatureRef struct {
	Kind FeatureKind
	ID   string
}

// PlayerMemory is everything one player has ever observed on one (match, map).
//
// It replaces the older cell-based explored set. A cell was a lossy proxy for "did I
// see this wall", and it failed in both directions: a wall in plain view whose cell
// centre was never lit got forgotten the moment the player stepped away, and an
// occluded wall stretch inside a cell whose centre WAS lit got remembered although the
// player never saw it. Recording the observed object itself removes both by
// construction.
type PlayerMemory struct {
	PlayerID  uuid.UUID
	MatchID   uuid.UUID
	MapID     uuid.UUID
	Seen      map[FeatureRef]struct{}
	UpdatedAt time.Time
}

func NewPlayerMemory(playerID, matchID, mapID uuid.UUID) *PlayerMemory {
	return &PlayerMemory{
		PlayerID:  playerID,
		MatchID:   matchID,
		MapID:     mapID,
		Seen:      make(map[FeatureRef]struct{}),
		UpdatedAt: time.Now(),
	}
}

// Remember unions refs into the seen set and reports how many were new.
func (m *PlayerMemory) Remember(refs []FeatureRef) int {
	added := 0
	for _, r := range refs {
		if _, ok := m.Seen[r]; ok {
			continue
		}
		m.Seen[r] = struct{}{}
		added++
	}
	if added > 0 {
		m.UpdatedAt = time.Now()
	}
	return added
}

// Has reports whether the player has ever observed this feature.
//
// The nil guard is load-bearing: callers legitimately hold a nil memory (a player with
// no memory yet, and every master/lobby code path), and FilterMapState calls this on
// every wall. Without the guard the game server panics.
func (m *PlayerMemory) Has(kind FeatureKind, id string) bool {
	if m == nil || m.Seen == nil {
		return false
	}
	_, ok := m.Seen[FeatureRef{Kind: kind, ID: id}]
	return ok
}
