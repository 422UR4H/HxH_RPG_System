package masteraction_test

import (
	"encoding/json"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/google/uuid"
)

func TestRecord_ProjectFor(t *testing.T) {
	master := uuid.New()
	player := uuid.New()
	other := uuid.New()

	moveContent := func() json.RawMessage {
		b, _ := json.Marshal(masteraction.PieceContent{
			CharacterID: "char-1",
			PieceID:     "piece-1",
			From:        &[3]int{1, 2, 0},
			To:          &[3]int{3, 4, 0},
		})
		return b
	}()

	wallContent := func() json.RawMessage {
		b, _ := json.Marshal(map[string]any{"wallId": "wall-1", "interact": "open"})
		return b
	}()

	baseMove := masteraction.Record{
		UUID:       uuid.New(),
		MatchUUID:  uuid.New(),
		SceneUUID:  uuid.New(),
		RoundUUID:  uuid.New(),
		MasterUUID: master,
		Kind:       masteraction.KindMovePiece,
		Content:    moveContent,
		Views: map[uuid.UUID]masteraction.View{
			player: masteraction.ViewFull,
		},
	}

	baseWall := masteraction.Record{
		UUID:       uuid.New(),
		MatchUUID:  uuid.New(),
		SceneUUID:  uuid.New(),
		RoundUUID:  uuid.New(),
		MasterUUID: master,
		Kind:       masteraction.KindWallInteract,
		Content:    wallContent,
		Views: map[uuid.UUID]masteraction.View{
			player: masteraction.ViewLeft,
		},
	}

	baseTurnNote := masteraction.Record{
		UUID:       uuid.New(),
		MatchUUID:  uuid.New(),
		SceneUUID:  uuid.New(),
		RoundUUID:  uuid.New(),
		MasterUUID: master,
		Kind:       masteraction.KindTurnNote,
		Content:    json.RawMessage(`{"note":"secret"}`),
		Views:      map[uuid.UUID]masteraction.View{},
	}

	t.Run("master sees the same record", func(t *testing.T) {
		proj, ok := baseMove.ProjectFor(true, master)
		if !ok {
			t.Fatalf("expected ok=true for master")
		}
		if proj.Views == nil {
			t.Fatalf("master projection must keep Views")
		}
		if string(proj.Content) != string(baseMove.Content) {
			t.Fatalf("master content changed: got %s want %s", proj.Content, baseMove.Content)
		}
	})

	t.Run("full view keeps content but drops Views", func(t *testing.T) {
		r := baseMove
		r.Views = map[uuid.UUID]masteraction.View{player: masteraction.ViewFull}

		proj, ok := r.ProjectFor(false, player)
		if !ok {
			t.Fatalf("expected ok=true for a player with a full view")
		}
		if proj.Views != nil {
			t.Fatalf("full projection must not carry Views, got %v", proj.Views)
		}

		var pc masteraction.PieceContent
		if err := json.Unmarshal(proj.Content, &pc); err != nil {
			t.Fatalf("unmarshal projected content: %v", err)
		}
		if pc.To == nil || *pc.To != [3]int{3, 4, 0} {
			t.Fatalf("full projection must keep To, got %+v", pc.To)
		}
		if pc.From == nil || *pc.From != [3]int{1, 2, 0} {
			t.Fatalf("full projection must keep From, got %+v", pc.From)
		}
	})

	t.Run("left view on movePiece erases To but keeps From", func(t *testing.T) {
		r := baseMove
		r.Views = map[uuid.UUID]masteraction.View{player: masteraction.ViewLeft}

		proj, ok := r.ProjectFor(false, player)
		if !ok {
			t.Fatalf("expected ok=true for a player with a left view")
		}
		if proj.Views != nil {
			t.Fatalf("left projection must not carry Views, got %v", proj.Views)
		}

		var pc masteraction.PieceContent
		if err := json.Unmarshal(proj.Content, &pc); err != nil {
			t.Fatalf("unmarshal projected content: %v", err)
		}
		if pc.To != nil {
			t.Fatalf("left projection must erase To, got %+v", pc.To)
		}
		if pc.From == nil || *pc.From != [3]int{1, 2, 0} {
			t.Fatalf("left projection must keep From, got %+v", pc.From)
		}
	})

	t.Run("left view on wallInteract reads the same as full (no destination)", func(t *testing.T) {
		proj, ok := baseWall.ProjectFor(false, player)
		if !ok {
			t.Fatalf("expected ok=true for a player with a left view")
		}
		if proj.Views != nil {
			t.Fatalf("projection must not carry Views, got %v", proj.Views)
		}
		if string(proj.Content) != string(baseWall.Content) {
			t.Fatalf("wallInteract content must be untouched by left, got %s want %s", proj.Content, baseWall.Content)
		}
	})

	t.Run("absent from Views: the entry does not exist for this reader", func(t *testing.T) {
		_, ok := baseMove.ProjectFor(false, other)
		if ok {
			t.Fatalf("expected ok=false for a viewer absent from Views")
		}
	})

	t.Run("turnNote with empty Views is master-only", func(t *testing.T) {
		_, ok := baseTurnNote.ProjectFor(false, player)
		if ok {
			t.Fatalf("expected ok=false for a player on a turnNote (master-only)")
		}

		proj, ok := baseTurnNote.ProjectFor(true, master)
		if !ok {
			t.Fatalf("expected ok=true for the master")
		}
		if string(proj.Content) != string(baseTurnNote.Content) {
			t.Fatalf("master content changed: got %s want %s", proj.Content, baseTurnNote.Content)
		}
	})
}

func TestRecord_ViewFor(t *testing.T) {
	player := uuid.New()
	absent := uuid.New()

	r := masteraction.Record{
		Views: map[uuid.UUID]masteraction.View{
			player: masteraction.ViewFull,
		},
	}

	t.Run("present player", func(t *testing.T) {
		v, ok := r.ViewFor(player)
		if !ok || v != masteraction.ViewFull {
			t.Fatalf("got (%q, %v), want (%q, true)", v, ok, masteraction.ViewFull)
		}
	})

	t.Run("absent player", func(t *testing.T) {
		v, ok := r.ViewFor(absent)
		if ok || v != "" {
			t.Fatalf("got (%q, %v), want (\"\", false)", v, ok)
		}
	})
}
