package actioncodec_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/actioncodec"
	"github.com/google/uuid"
)

// fullAction carries every piece the actions table has a column for, so a round trip that
// drops one shows up as a diff.
func fullAction() *action.Action {
	weapon := enum.WeaponName("Sword")
	dc := 12
	from := [3]int{1, 2, 0}
	a := action.NewAction(
		uuid.New(), []uuid.UUID{uuid.New(), uuid.New()}, uuid.Nil,
		// Skill.SkillName shadows the embedded RollCheck.SkillName in JSON — the actions.skills
		// column has always stored only the outer one — so the inner is left empty here.
		[]action.Skill{{SkillName: "Lockpick", Difficulty: &dc, RollCheck: action.RollCheck{Result: 9}}},
		action.ActionSpeed{Bar: 2, RollCheck: action.RollCheck{SkillName: "Legerity", SkillValue: 4, Result: 11}},
		&action.RollCheck{SkillName: "Feint", Result: 7},
		&action.Move{Category: enum.MoveCategory("Dash"), From: &from, Position: [3]int{4, 4, 0}, FinalSpeed: 9},
		&action.Attack{Weapon: &weapon, Hit: action.RollCheck{SkillName: "Accuracy", Result: 13}, Damage: action.RollCheck{SkillName: "Push"}},
		&action.Defense{Weapon: &weapon, RollCheck: action.RollCheck{SkillName: "Defense"}},
		&action.Dodge{RollCheck: action.RollCheck{SkillName: "Dodge"}},
		&action.Trigger{},
		&action.Interact{Kind: action.InteractLockpick},
	)
	a.Repel = &action.Repel{Weapon: &weapon, RollCheck: action.RollCheck{SkillName: "Repel"}}
	a.DefaultDefense = &action.RollCheck{SkillName: "Defense"}
	a.SystemBias = -1
	return a
}

func TestMarshalUnmarshalRoundTripsTheWholeAction(t *testing.T) {
	want := fullAction()
	raw, err := actioncodec.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := actioncodec.Unmarshal(want.GetID(), want.GetActorID(), raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip differs:\n got  %+v\n want %+v", got, want)
	}
}

func TestMarshalUnmarshalRoundTripsAReaction(t *testing.T) {
	want := action.NewAction(uuid.New(), []uuid.UUID{uuid.New()}, uuid.New(), nil, action.ActionSpeed{},
		nil, nil, nil, nil, &action.Dodge{}, nil, nil)
	want.ReactionKind = action.ReactionKind("dodge")
	want.ConsumedActionIDs = []uuid.UUID{uuid.New()}
	raw, err := actioncodec.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got, err := actioncodec.Unmarshal(want.GetID(), want.GetActorID(), raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip differs:\n got  %+v\n want %+v", got, want)
	}
}

// The document's keys are the actions table's column names — the stable storage keys the
// migration documents. An absent piece is an absent key, never a JSON null.
func TestMarshalUsesTheColumnNamesAndOmitsAbsentPieces(t *testing.T) {
	a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{Bar: 1},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
	raw, err := actioncodec.Marshal(a)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("unmarshal document: %v", err)
	}
	for _, k := range []string{"speed", "attack", "target_ids", "system_bias"} {
		if _, ok := doc[k]; !ok {
			t.Errorf("key %q missing from %s", k, raw)
		}
	}
	for _, k := range []string{"skills", "move", "defense", "dodge", "repel", "feint", "trigger", "interact",
		"default_defense", "react_to_uuid", "reaction_kind", "consumed_action_ids"} {
		if _, ok := doc[k]; ok {
			t.Errorf("key %q present for an absent piece in %s", k, raw)
		}
	}
	if string(doc["target_ids"]) != "[]" {
		t.Errorf("target_ids = %s, want [] (never null, as in actions)", doc["target_ids"])
	}
}

func TestUnmarshalRefusesABrokenPiece(t *testing.T) {
	_, err := actioncodec.Unmarshal(uuid.New(), uuid.New(), []byte(`{"speed":{},"attack":"not an object"}`))
	if err == nil {
		t.Fatal("Unmarshal accepted an attack that is not an object")
	}
}
