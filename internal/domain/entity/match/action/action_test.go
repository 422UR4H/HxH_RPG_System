package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/die"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/google/uuid"
)

func TestNewAction(t *testing.T) {
	actorID := uuid.New()
	targetIDs := []uuid.UUID{uuid.New(), uuid.New()}
	reactToID := uuid.New()

	a := action.NewAction(
		actorID, targetIDs, reactToID, nil,
		action.ActionSpeed{Bar: 5, RollCheck: action.RollCheck{Result: 42}},
		nil, nil, nil, nil, nil, nil,
	)

	if a.GetID() == uuid.Nil {
		t.Error("GetID() should not be nil UUID")
	}
	if a.GetActorID() != actorID {
		t.Errorf("GetActorID() = %v, want %v", a.GetActorID(), actorID)
	}
	if a.Speed.Result != 42 {
		t.Errorf("Speed.Result = %d, want 42", a.Speed.Result)
	}
	if a.Speed.Bar != 5 {
		t.Errorf("Speed.Bar = %d, want 5", a.Speed.Bar)
	}
}

func TestNewAction_UniqueIDs(t *testing.T) {
	a1 := makeAction(10)
	a2 := makeAction(20)

	if a1.GetID() == a2.GetID() {
		t.Error("two actions should have different UUIDs")
	}
}

func TestRollContext_GetDiceResult(t *testing.T) {
	d1 := die.NewDie(enum.D6)
	d2 := die.NewDie(enum.D8)

	r1 := d1.Roll()
	r2 := d2.Roll()

	rc := action.RollContext{
		Dice: []die.Die{*d1, *d2},
	}

	result := rc.GetDiceResult(*d1)
	expectedSum := r1 + r2
	if result != expectedSum {
		t.Errorf("GetDiceResult() = %d, want %d (sum of rolled dice)", result, expectedSum)
	}
}

func TestRollContext_GetDiceResult_Empty(t *testing.T) {
	rc := action.RollContext{
		Dice: []die.Die{},
	}
	result := rc.GetDiceResult(*die.NewDie(enum.D6))
	if result != 0 {
		t.Errorf("GetDiceResult() with empty dice = %d, want 0", result)
	}
}
