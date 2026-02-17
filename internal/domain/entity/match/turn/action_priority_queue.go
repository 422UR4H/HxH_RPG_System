package turn

import (
	"container/heap"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/google/uuid"
)

// ActionPriorityQueue implements heap.Interface for a priority queue of actions
type ActionPriorityQueue []*action.Action

func (aq ActionPriorityQueue) Len() int { return len(aq) }

func (aq ActionPriorityQueue) Less(i, j int) bool {
	// By default, heap is a min-heap. We invert the comparison to create a max-heap
	// so that actions with higher speed values are processed first
	return aq[i].Speed.Result > aq[j].Speed.Result
}

func (aq ActionPriorityQueue) Swap(i, j int) {
	aq[i], aq[j] = aq[j], aq[i]
}

func (aq *ActionPriorityQueue) Push(x any) {
	*aq = append(*aq, x.(*action.Action))
}

func (aq *ActionPriorityQueue) Pop() any {
	old := *aq
	n := len(old)
	item := old[n-1]
	*aq = old[0 : n-1]
	return item
}

func NewActionPriorityQueue(actions *[]*action.Action) ActionPriorityQueue {
	var aq ActionPriorityQueue
	if actions != nil {
		aq = ActionPriorityQueue(*actions)
		heap.Init(&aq)
	} else {
		aq = make(ActionPriorityQueue, 0)
	}
	return aq
}

// Insert adds a new action to the queue while maintaining the priority
func (aq *ActionPriorityQueue) Insert(newAction *action.Action) {
	heap.Push(aq, newAction)
}

// ExtractMax removes and returns the action with the highest speed
func (aq *ActionPriorityQueue) ExtractMax() *action.Action {
	if aq.Len() == 0 {
		return nil
	}
	return heap.Pop(aq).(*action.Action)
}

// Peek returns the action with the highest speed without removing it
func (aq *ActionPriorityQueue) Peek() *action.Action {
	if aq.Len() == 0 {
		return nil
	}
	return (*aq)[0]
}

// ExtractByID searches and removes a specific action by UUID
func (aq *ActionPriorityQueue) ExtractByID(id uuid.UUID) *action.Action {
	for i, act := range *aq {
		if act.GetID() == id {
			heap.Remove(aq, i)
			return act
		}
	}
	return nil
}

func (aq *ActionPriorityQueue) IsEmpty() bool {
	return aq.Len() == 0
}
