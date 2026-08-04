package game

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	// maxMessageSize must fit the master's map_state_sync, whose size grows with the
	// board: every wall segment and every piece is serialized in one frame. A real
	// 35x35 map with 17 walls already exceeds 4 KB, and going over the limit makes
	// gorilla close the connection — the master then reconnects and re-sends forever
	// while players never receive a board, so their fog never lifts.
	// Guarded by TestE2E_LargeBoardSyncIsNotRejected.
	maxMessageSize = 1 << 20 // 1 MiB
)

type Client struct {
	userUUID uuid.UUID
	nickname string
	conn     *websocket.Conn
	send     chan []byte
	room     *Room
	done     chan struct{}
}

func NewClient(userUUID uuid.UUID, conn *websocket.Conn, nickname string) *Client {
	return &Client{
		userUUID: userUUID,
		nickname: nickname,
		conn:     conn,
		send:     make(chan []byte, 256),
		done:     make(chan struct{}),
	}
}

func (c *Client) GetUserUUID() uuid.UUID {
	return c.userUUID
}

func (c *Client) GetNickname() string {
	return c.nickname
}

func (c *Client) GetSendChan() <-chan []byte {
	return c.send
}

func (c *Client) SetRoom(room *Room) {
	c.room = room
}

// SendMessage queues a message for this client. It is safe to call concurrently with
// the client being torn down: shutdown closes `done`, never `send`, so a send here can
// never hit a closed channel (which would panic and take the whole server down).
func (c *Client) SendMessage(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("error marshaling message: %v", err)
		return
	}
	select {
	case <-c.done:
		return // shutting down; nothing is draining send anymore
	default:
	}
	select {
	case c.send <- data:
	case <-c.done:
	default:
		log.Printf("client %s send buffer full, dropping message", c.userUUID)
	}
}

func (c *Client) Close() {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
}

func (c *Client) ReadPump() {
	defer func() {
		if c.room != nil {
			c.room.unregister <- c
		}
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("websocket error for user %s: %v", c.userUUID, err)
			}
			break
		}
		if c.room != nil {
			c.room.handleClientMessage(c, rawMsg)
		}
	}
}

// flushSend writes every message already buffered, stopping at the first write error.
func (c *Client) flushSend() {
	for {
		select {
		case message := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		default:
			return
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			// Shutdown signals `done`, never closes `send`. Because select picks at
			// random among ready cases, anything already queued would otherwise be
			// dropped — including the final message that triggered the shutdown, such
			// as lobby_closed. Flush the backlog first, then close.
			c.flushSend()
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
	}
}
