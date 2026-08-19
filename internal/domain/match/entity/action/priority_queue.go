package action

import "github.com/google/uuid"

// PriorityQueue holds the actions waiting for the master to open them.
//
// It USED to be a max-heap keyed on Action.Speed.Result. It is not any more, and the reason is
// structural rather than a matter of taste: the position of an action in the round is
//
//	carry + mean(acted ++ [speed]) − len(acted) × price
//
// which is state of the CHARACTER, not of the action, and which moves every time that
// character sends another action — the round average shifts under it. A heap cannot re-key an
// entry it already holds; it would keep answering with a stale order and never say so. So the
// key is computed at selection time, by service.RoundScheduler, and this becomes a plain list.
// With four to six characters at a table, scanning costs less than maintaining a heap, and
// ExtractByID scanned linearly all along.
//
// Insertion order is preserved and is meaningful: it is how ties between equal keys resolve —
// whoever sent first goes first.
type PriorityQueue []*Action

func NewActionPriorityQueue(actions *[]*Action) PriorityQueue {
	if actions == nil {
		return make(PriorityQueue, 0)
	}
	return PriorityQueue(*actions)
}

func (aq PriorityQueue) Len() int       { return len(aq) }
func (aq *PriorityQueue) IsEmpty() bool { return aq.Len() == 0 }

// Insert adds a new action to the back of the queue.
func (aq *PriorityQueue) Insert(newAction *Action) {
	*aq = append(*aq, newAction)
}

// All returns every pending action, in insertion order, as a copy. The scheduler iterates it
// to compute a key per entry; handing out the backing array would let a caller reshape the
// queue behind the session's back.
func (aq *PriorityQueue) All() []*Action {
	out := make([]*Action, len(*aq))
	copy(out, *aq)
	return out
}

// ExtractMax removes and returns the action with the highest Speed.Result.
//
// This is the FREE-round path, where there is no price, no average and no carry-over, so the
// rolled speed IS the order. A Race round goes through service.RoundScheduler instead, which
// knows about the bars.
func (aq *PriorityQueue) ExtractMax() *Action {
	idx := aq.indexOfMax()
	if idx < 0 {
		return nil
	}
	return aq.removeAt(idx)
}

// Peek returns the action with the highest Speed.Result without removing it.
func (aq *PriorityQueue) Peek() *Action {
	idx := aq.indexOfMax()
	if idx < 0 {
		return nil
	}
	return (*aq)[idx]
}

// ExtractByID searches and removes a specific action by UUID.
func (aq *PriorityQueue) ExtractByID(id uuid.UUID) *Action {
	for i, act := range *aq {
		if act.GetID() == id {
			return aq.removeAt(i)
		}
	}
	return nil
}

// indexOfMax returns the index of the highest Speed.Result, or -1 on an empty queue. Ties go
// to the earliest insertion.
func (aq *PriorityQueue) indexOfMax() int {
	best := -1
	for i, act := range *aq {
		if best == -1 || act.Speed.Result > (*aq)[best].Speed.Result {
			best = i
		}
	}
	return best
}

// removeAt takes an action out while preserving the order of the rest.
func (aq *PriorityQueue) removeAt(i int) *Action {
	old := *aq
	act := old[i]
	*aq = append(old[:i], old[i+1:]...)
	return act
}
