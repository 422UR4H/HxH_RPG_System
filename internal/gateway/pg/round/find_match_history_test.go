package round

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

// TestUnmarshalNullablePtr_MoveFrom is B6 (spec §4.3 "B5, B6 e B10"): action.Move.From went
// from a plain [3]int to a *[3]int, and actions.move is whatever json.Marshal(act.Move)
// produced — Go's default field-name encoding, no json tags on action.Move. A row written
// BEFORE this task always carried a "From" key with a real [3]int value (the sentinel
// [0,0,0] included, since the old code used that as "not provided"); this proves
// unmarshalNullablePtr still decodes such a row into a non-nil pointer, unchanged, and that
// a row written AFTER this task with no origin at all (actor had no piece: From omitted or
// explicitly null) decodes into a nil pointer instead of a false [0,0,0].
func TestUnmarshalNullablePtr_MoveFrom(t *testing.T) {
	t.Run("old row: From is a real [0,0,0], decodes into a non-nil pointer", func(t *testing.T) {
		raw := []byte(`{"Category":"Dash","From":[0,0,0],"Position":[4,4,0],"FinalSpeed":9}`)
		move, err := unmarshalNullablePtr[action.Move](raw)
		if err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if move == nil {
			t.Fatal("move itself is nil — the row's own presence must not be lost")
		}
		if move.From == nil {
			t.Fatal("Move.From = nil, want a non-nil pointer to [0 0 0]")
		}
		if *move.From != [3]int{0, 0, 0} {
			t.Fatalf("Move.From = %v, want [0 0 0]", *move.From)
		}
	})

	t.Run("new row: From is null (actor had no piece), decodes into a nil pointer", func(t *testing.T) {
		raw := []byte(`{"Category":"Dash","From":null,"Position":[4,4,0],"FinalSpeed":9}`)
		move, err := unmarshalNullablePtr[action.Move](raw)
		if err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if move == nil {
			t.Fatal("move itself is nil — the row's own presence must not be lost")
		}
		if move.From != nil {
			t.Fatalf("Move.From = %v, want nil", *move.From)
		}
	})

	t.Run("new row: From key absent entirely, decodes into a nil pointer", func(t *testing.T) {
		raw := []byte(`{"Category":"Dash","Position":[4,4,0],"FinalSpeed":9}`)
		move, err := unmarshalNullablePtr[action.Move](raw)
		if err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if move.From != nil {
			t.Fatalf("Move.From = %v, want nil", *move.From)
		}
	})

	t.Run("no row at all: NULL column, decodes into a nil *Move", func(t *testing.T) {
		move, err := unmarshalNullablePtr[action.Move](nil)
		if err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if move != nil {
			t.Fatalf("move = %+v, want nil", move)
		}
	})
}
