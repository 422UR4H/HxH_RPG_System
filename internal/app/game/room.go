package game

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/google/uuid"
)

type RoomState string

const (
	RoomStateLobby   RoomState = "lobby"
	RoomStatePlaying RoomState = "playing"
	RoomStateClosed  RoomState = "closed"
)

var (
	ErrNotMaster      = errors.New("only the master can perform this action")
	ErrAlreadyPlaying = errors.New("match already started")
	ErrRoomClosed     = errors.New("room is closed")
)

type IStartMatch interface {
	Start(ctx context.Context, matchUUID uuid.UUID, masterUUID uuid.UUID) error
}

type IKickPlayer interface {
	Kick(ctx context.Context, matchUUID uuid.UUID, playerUUID uuid.UUID, masterUUID uuid.UUID) error
}

type Room struct {
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	state      RoomState
	clients    map[uuid.UUID]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	mu         sync.RWMutex

	startMatchUC IStartMatch
	kickPlayerUC IKickPlayer
}

func NewRoom(
	matchUUID, masterUUID uuid.UUID,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
) *Room {
	return &Room{
		matchUUID:    matchUUID,
		masterUUID:   masterUUID,
		state:        RoomStateLobby,
		clients:      make(map[uuid.UUID]*Client),
		broadcast:    make(chan []byte, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		stop:         make(chan struct{}),
		startMatchUC: startMatchUC,
		kickPlayerUC: kickPlayerUC,
	}
}

func (r *Room) GetMatchUUID() uuid.UUID {
	return r.matchUUID
}

func (r *Room) GetState() RoomState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

func (r *Room) IsMaster(userUUID uuid.UUID) bool {
	return r.masterUUID == userUUID
}

func (r *Room) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

func (r *Room) Register(client *Client) {
	r.register <- client
}

func (r *Room) Broadcast(data []byte) {
	r.broadcast <- data
}

func (r *Room) Stop() {
	select {
	case <-r.stop:
	default:
		close(r.stop)
	}
}

func (r *Room) Run() {
	for {
		select {
		case client := <-r.register:
			r.mu.Lock()
			r.clients[client.userUUID] = client
			client.SetRoom(r)
			r.mu.Unlock()

			r.sendRoomState(client)
			r.broadcastPlayerJoined(client)

		case client := <-r.unregister:
			r.mu.Lock()
			if _, ok := r.clients[client.userUUID]; ok {
				delete(r.clients, client.userUUID)
				close(client.send)
			}
			r.mu.Unlock()

			r.broadcastPlayerLeft(client)

			r.mu.RLock()
			empty := len(r.clients) == 0
			r.mu.RUnlock()
			if empty {
				r.mu.Lock()
				r.state = RoomStateClosed
				r.mu.Unlock()
				return
			}

		case message := <-r.broadcast:
			r.mu.RLock()
			for _, client := range r.clients {
				select {
				case client.send <- message:
				default:
					log.Printf("dropping message for slow client %s", client.userUUID)
				}
			}
			r.mu.RUnlock()

		case <-r.stop:
			r.mu.Lock()
			r.state = RoomStateClosed
			for _, client := range r.clients {
				close(client.send)
			}
			r.clients = make(map[uuid.UUID]*Client)
			r.mu.Unlock()
			return
		}
	}
}

func (r *Room) StartMatch(userUUID uuid.UUID) error {
	if !r.IsMaster(userUUID) {
		return ErrNotMaster
	}
	r.mu.RLock()
	if r.state != RoomStateLobby {
		r.mu.RUnlock()
		return ErrAlreadyPlaying
	}
	r.mu.RUnlock()

	if err := r.startMatchUC.Start(context.Background(), r.matchUUID, userUUID); err != nil {
		return err
	}

	r.mu.Lock()
	r.state = RoomStatePlaying
	r.mu.Unlock()

	msg := NewServerMessage(MsgTypeMatchStarted, struct{}{})
	data, _ := json.Marshal(msg)
	go func() { r.broadcast <- data }()
	return nil
}

func (r *Room) KickPlayer(masterUUID uuid.UUID, playerUUID uuid.UUID) error {
	if !r.IsMaster(masterUUID) {
		return ErrNotMaster
	}

	if err := r.kickPlayerUC.Kick(context.Background(), r.matchUUID, playerUUID, masterUUID); err != nil {
		return err
	}

	r.mu.Lock()
	client, ok := r.clients[playerUUID]
	if ok {
		delete(r.clients, playerUUID)
	}
	r.mu.Unlock()

	if ok {
		kickedMsg := NewServerMessage(MsgTypePlayerKicked, PlayerKickedPayload{
			UUID:     playerUUID,
			Nickname: client.nickname,
			Reason:   "kicked by master",
		})

		client.SendMessage(kickedMsg)
		close(client.send)

		data, _ := json.Marshal(kickedMsg)
		r.mu.RLock()
		for _, c := range r.clients {
			select {
			case c.send <- data:
			default:
			}
		}
		r.mu.RUnlock()
	}
	return nil
}

func (r *Room) handleClientMessage(client *Client, rawMsg []byte) {
	var incoming Message
	if err := json.Unmarshal(rawMsg, &incoming); err != nil {
		client.SendMessage(NewErrorMessage("invalid_message", "malformed JSON"))
		return
	}

	switch incoming.Type {
	case MsgTypeStartMatch:
		if err := r.StartMatch(client.userUUID); err != nil {
			client.SendMessage(NewErrorMessage("forbidden", err.Error()))
		}

	case MsgTypeKickPlayer:
		var kickPayload KickPlayerPayload
		if err := json.Unmarshal(incoming.Payload, &kickPayload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid kick payload"))
			return
		}
		if err := r.KickPlayer(client.userUUID, kickPayload.PlayerUUID); err != nil {
			client.SendMessage(NewErrorMessage("forbidden", err.Error()))
		}

	case MsgTypeChat:
		var chatPayload ChatPayload
		if err := json.Unmarshal(incoming.Payload, &chatPayload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid chat payload"))
			return
		}
		outMsg := NewClientMessage(MsgTypeChatMessage, client.userUUID, chatPayload)
		data, _ := json.Marshal(outMsg)
		r.broadcast <- data

	default:
		client.SendMessage(NewErrorMessage("unknown_type", "unrecognized message type"))
	}
}

func (r *Room) sendRoomState(client *Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]PlayerInfo, 0, len(r.clients))
	for _, c := range r.clients {
		players = append(players, PlayerInfo{
			UUID:     c.userUUID,
			Nickname: c.nickname,
			IsMaster: r.masterUUID == c.userUUID,
			IsOnline: true,
		})
	}

	msg := NewServerMessage(MsgTypeRoomState, RoomStatePayload{
		MatchUUID: r.matchUUID,
		State:     string(r.state),
		Players:   players,
	})
	client.SendMessage(msg)
}

func (r *Room) broadcastPlayerJoined(client *Client) {
	msg := NewServerMessage(MsgTypePlayerJoined, PlayerPayload{
		UUID:     client.userUUID,
		Nickname: client.nickname,
	})
	data, _ := json.Marshal(msg)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		if c.userUUID != client.userUUID {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

func (r *Room) broadcastPlayerLeft(client *Client) {
	msg := NewServerMessage(MsgTypePlayerLeft, PlayerPayload{
		UUID:     client.userUUID,
		Nickname: client.nickname,
	})
	data, _ := json.Marshal(msg)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		select {
		case c.send <- data:
		default:
		}
	}
}
