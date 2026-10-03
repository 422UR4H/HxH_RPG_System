package game

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/google/uuid"
)

// gatePieceMoveLocked's edges that no WebSocket path reaches today: actionwire.From always
// fills move.position, so a nil destination never arrives from turn_opened — but the gate takes
// a pointer, and nil there has to mean "no destination", never cell (0,0). fogTestRoom's player
// stands on (0,0) and sees the whole open board, so reading nil as (0,0) would turn every one of
// these into a full view.
func TestGatePieceMoveLocked_Edges(t *testing.T) {
	room, playerUUID, _ := fogTestRoom(t)
	actor := uuid.New()
	hiddenActor := uuid.New()
	hidden := false
	room.pieces["actor"] = PieceMovedPayload{PieceID: "actor", CharacterID: actor.String(), Slot: squareSlot(1, 0)}
	room.pieces["hidden"] = PieceMovedPayload{
		PieceID: "hidden", CharacterID: hiddenActor.String(), Slot: squareSlot(1, 1), Visible: &hidden,
	}
	seen := [3]int{1, 0, 0}

	tests := []struct {
		name     string
		char     uuid.UUID
		from, to *[3]int
		want     masteraction.View
		wantOK   bool
	}{
		{name: "no destination, origin in sight: left", char: actor, from: &seen, to: nil, want: masteraction.ViewLeft, wantOK: true},
		{name: "no destination, no origin: nothing", char: actor, from: nil, to: nil, wantOK: false},
		{name: "destination in sight: full", char: actor, from: nil, to: &seen, want: masteraction.ViewFull, wantOK: true},
		{name: "a character with no piece: nothing, even in sight", char: uuid.New(), from: &seen, to: &seen, wantOK: false},
		{name: "a visible:false piece: nothing, even in sight", char: hiddenActor, from: &seen, to: &seen, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room.mu.RLock()
			got, ok := room.gatePieceMoveLocked(playerUUID, tt.char, tt.from, tt.to)
			room.mu.RUnlock()
			if ok != tt.wantOK || (ok && got != tt.want) {
				t.Fatalf("gate = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
