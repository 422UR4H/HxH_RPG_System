package game

import (
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	rooms map[uuid.UUID]*Room
	mu    sync.RWMutex
	stop  chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[uuid.UUID]*Room),
		stop:  make(chan struct{}),
	}
}

func (h *Hub) Run() {
	<-h.stop
}

func (h *Hub) Stop() {
	select {
	case <-h.stop:
	default:
		close(h.stop)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for _, room := range h.rooms {
		room.Stop()
	}
}

func (h *Hub) GetOrCreateRoom(
	matchUUID, masterUUID uuid.UUID,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[matchUUID]; ok {
		return room
	}

	room := NewRoom(matchUUID, masterUUID, startMatchUC, kickPlayerUC)
	h.rooms[matchUUID] = room
	go room.Run()
	return room
}

func (h *Hub) GetRoom(matchUUID uuid.UUID) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, ok := h.rooms[matchUUID]
	return room, ok
}

func (h *Hub) RemoveRoom(matchUUID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[matchUUID]; ok {
		room.Stop()
		delete(h.rooms, matchUUID)
	}
}

func (h *Hub) RoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}
