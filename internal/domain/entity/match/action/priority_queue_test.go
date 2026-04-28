package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/google/uuid"
)

func makeAction(speed int) *action.Action {
	return action.NewAction(
		uuid.New(),
		nil,
		uuid.Nil,
		nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: speed}},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

func TestPriorityQueue_NewEmpty(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)
	if !pq.IsEmpty() {
		t.Error("new queue should be empty")
	}
	if pq.Len() != 0 {
		t.Errorf("Len() = %d, want 0", pq.Len())
	}
}

func TestPriorityQueue_InsertAndExtractMax(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)

	a1 := makeAction(10)
	a2 := makeAction(30)
	a3 := makeAction(20)

	pq.Insert(a1)
	pq.Insert(a2)
	pq.Insert(a3)

	if pq.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", pq.Len())
	}

	first := pq.ExtractMax()
	if first.Speed.Result != 30 {
		t.Errorf("first ExtractMax() speed = %d, want 30", first.Speed.Result)
	}

	second := pq.ExtractMax()
	if second.Speed.Result != 20 {
		t.Errorf("second ExtractMax() speed = %d, want 20", second.Speed.Result)
	}

	third := pq.ExtractMax()
	if third.Speed.Result != 10 {
		t.Errorf("third ExtractMax() speed = %d, want 10", third.Speed.Result)
	}

	if !pq.IsEmpty() {
		t.Error("queue should be empty after extracting all")
	}
}

func TestPriorityQueue_ExtractMax_EmptyReturnsNil(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)
	if pq.ExtractMax() != nil {
		t.Error("ExtractMax() on empty queue should return nil")
	}
}

func TestPriorityQueue_Peek(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)

	t.Run("empty queue", func(t *testing.T) {
		if pq.Peek() != nil {
			t.Error("Peek() on empty queue should return nil")
		}
	})

	a1 := makeAction(15)
	a2 := makeAction(25)
	pq.Insert(a1)
	pq.Insert(a2)

	t.Run("returns max without removing", func(t *testing.T) {
		peeked := pq.Peek()
		if peeked.Speed.Result != 25 {
			t.Errorf("Peek() speed = %d, want 25", peeked.Speed.Result)
		}
		if pq.Len() != 2 {
			t.Errorf("Len() after Peek() = %d, want 2", pq.Len())
		}
	})
}

func TestPriorityQueue_ExtractByID(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)

	a1 := makeAction(10)
	a2 := makeAction(20)
	a3 := makeAction(30)

	pq.Insert(a1)
	pq.Insert(a2)
	pq.Insert(a3)

	t.Run("extract existing by ID", func(t *testing.T) {
		targetID := a2.GetID()
		extracted := pq.ExtractByID(targetID)
		if extracted == nil {
			t.Fatal("ExtractByID returned nil")
		}
		if extracted.GetID() != targetID {
			t.Errorf("extracted ID = %v, want %v", extracted.GetID(), targetID)
		}
		if pq.Len() != 2 {
			t.Errorf("Len() after extract = %d, want 2", pq.Len())
		}
	})

	t.Run("extract non-existing ID returns nil", func(t *testing.T) {
		result := pq.ExtractByID(uuid.New())
		if result != nil {
			t.Error("ExtractByID with unknown ID should return nil")
		}
	})
}

func TestPriorityQueue_NewFromExisting(t *testing.T) {
	a1 := makeAction(5)
	a2 := makeAction(50)
	a3 := makeAction(25)
	actions := []*action.Action{a1, a2, a3}

	pq := action.NewActionPriorityQueue(&actions)

	if pq.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", pq.Len())
	}

	max := pq.ExtractMax()
	if max.Speed.Result != 50 {
		t.Errorf("ExtractMax() speed = %d, want 50", max.Speed.Result)
	}
}
