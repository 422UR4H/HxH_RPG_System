// internal/domain/masteraction/record.go
package masteraction

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Kind identifies what a master action did. Master actions have a table of their own,
// separate from actions: their actor is the master (a USER), not a character sheet, and
// they happen OUTSIDE any turn — see spec §4.8, B14.
type Kind string

const (
	KindMovePiece    Kind = "movePiece"
	KindPlacePiece   Kind = "placePiece"
	KindRemovePiece  Kind = "removePiece"
	KindWallInteract Kind = "wallInteract"
	KindRevealWall   Kind = "revealWall"
	KindTurnNote     Kind = "turnNote"
)

// View is what one player saw of a master action LIVE, at the instant it was applied —
// recorded so the history can show every reader exactly that and no more (spec §4.8).
type View string

const (
	// ViewFull means the player received the master action itself live (a piece_moved, a
	// wall changing, a removal).
	ViewFull View = "full"
	// ViewLeft means the player only saw the piece leave, not where it went (the fog-gated
	// piece_removed pair).
	ViewLeft View = "left"
)

// Record is a master action as persisted: applied at the instant it happened, with or
// without an open turn.
type Record struct {
	UUID       uuid.UUID
	MatchUUID  uuid.UUID
	SceneUUID  uuid.UUID
	RoundUUID  uuid.UUID
	MasterUUID uuid.UUID
	// TurnUUID is nil outside a turn. It has no FK: a turn is only written when it closes,
	// and a restart can lose it after the master action was already recorded — the history
	// then shows it outside any turn.
	TurnUUID *uuid.UUID
	Kind     Kind
	Content  json.RawMessage
	// Views is what each player of the session saw of this action LIVE (connected or not —
	// what counts is their fog at that instant). A player with no entry saw nothing: the
	// entry does not exist for them, not "saw nothing labeled so".
	Views      map[uuid.UUID]View
	HappenedAt time.Time
}

// PieceContent is the Content shape for the three piece kinds (KindMovePiece,
// KindPlacePiece, KindRemovePiece): From nil means the actor had no piece before (a
// "place"); To nil means the piece has no destination (a "remove").
type PieceContent struct {
	CharacterID string  `json:"characterId"`
	PieceID     string  `json:"pieceId"`
	From        *[3]int `json:"from,omitempty"`
	To          *[3]int `json:"to,omitempty"`
}

// ViewFor reports what playerID saw of this action live. false means the entry does not
// exist for them — not "saw nothing labeled so".
func (r Record) ViewFor(playerID uuid.UUID) (View, bool) {
	v, ok := r.Views[playerID]
	return v, ok
}

func isPieceKind(k Kind) bool {
	switch k {
	case KindMovePiece, KindPlacePiece, KindRemovePiece:
		return true
	default:
		return false
	}
}

// ProjectFor is the projection of spec §4.8: every reader sees exactly what they saw live.
// Pure — no I/O, does not mutate r.
//
//   - master: the same record.
//   - full: the same record without Views (who-saw-what is not table data).
//   - left: same as full, and for a piece kind, with the destination erased from Content —
//     a wallInteract has no destination to begin with, so left reads the same as full there.
//     A piece Content that cannot be decoded fails closed: (Record{}, false).
//   - absent from Views: the entry does not exist for this reader — (Record{}, false).
func (r Record) ProjectFor(viewerIsMaster bool, viewer uuid.UUID) (Record, bool) {
	if viewerIsMaster {
		return r, true
	}

	view, ok := r.ViewFor(viewer)
	if !ok {
		return Record{}, false
	}

	proj := r
	proj.Views = nil
	if view == ViewLeft && isPieceKind(r.Kind) {
		cleared, err := clearContentTo(r.Content)
		if err != nil {
			// Fail closed. A Content that cannot be read cannot have its destination erased,
			// and handing it over untouched would show this reader where the piece went —
			// the one thing `left` says they never saw. Losing the entry for them is the
			// lesser harm; the master still has it whole.
			return Record{}, false
		}
		proj.Content = cleared
	}
	return proj, true
}

func clearContentTo(content json.RawMessage) (json.RawMessage, error) {
	var pc PieceContent
	if err := json.Unmarshal(content, &pc); err != nil {
		return nil, err
	}
	pc.To = nil
	return json.Marshal(pc)
}
