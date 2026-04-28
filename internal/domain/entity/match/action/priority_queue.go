package action

import (
	"container/heap"

	"github.com/google/uuid"
)

// PriorityQueue implements heap.Interface for a priority queue of actions
type PriorityQueue []*Action

func (aq PriorityQueue) Len() int { return len(aq) }

func (aq PriorityQueue) Less(i, j int) bool {
	// By default, heap is a min-heap. We invert the comparison to create a max-heap
	// so that actions with higher speed values are processed first
	return aq[i].Speed.Result > aq[j].Speed.Result
}

func (aq PriorityQueue) Swap(i, j int) {
	aq[i], aq[j] = aq[j], aq[i]
}

func (aq *PriorityQueue) Push(x any) {
	*aq = append(*aq, x.(*Action))
}

func (aq *PriorityQueue) Pop() any {
	old := *aq
	n := len(old)
	item := old[n-1]
	*aq = old[0 : n-1]
	return item
}

func NewActionPriorityQueue(actions *[]*Action) PriorityQueue {
	var aq PriorityQueue
	if actions != nil {
		aq = PriorityQueue(*actions)
		heap.Init(&aq)
	} else {
		aq = make(PriorityQueue, 0)
	}
	return aq
}

// Insert adds a new action to the queue while maintaining the priority
func (aq *PriorityQueue) Insert(newAction *Action) {
	heap.Push(aq, newAction)
}

// ExtractMax removes and returns the action with the highest speed
func (aq *PriorityQueue) ExtractMax() *Action {
	if aq.Len() == 0 {
		return nil
	}
	return heap.Pop(aq).(*Action)
}

// Peek returns the action with the highest speed without removing it
func (aq *PriorityQueue) Peek() *Action {
	if aq.Len() == 0 {
		return nil
	}
	return (*aq)[0]
}

// ExtractByID searches and removes a specific action by UUID
func (aq *PriorityQueue) ExtractByID(id uuid.UUID) *Action {
	for i, act := range *aq {
		if act.GetID() == id {
			heap.Remove(aq, i)
			return act
		}
	}
	return nil
}

func (aq *PriorityQueue) IsEmpty() bool {
	return aq.Len() == 0
}
