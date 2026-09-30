package game

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	pkgAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MatchRepository interface {
	GetMatchMaster(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
	IsStarted(ctx context.Context, matchUUID uuid.UUID) (bool, error)
}

type EnrollmentChecker interface {
	IsPlayerEnrolledInMatch(ctx context.Context, playerUUID, matchUUID uuid.UUID) (bool, error)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: IN PRODUCTION, IMPLEMENT ORIGIN CHECKING
		return true
	},
}

type Handler struct {
	hub            *Hub
	matchRepo      MatchRepository
	enrollmentRepo EnrollmentChecker
	deps           RoomDeps
}

func NewHandler(
	hub *Hub,
	matchRepo MatchRepository,
	enrollmentRepo EnrollmentChecker,
	deps RoomDeps,
) *Handler {
	return &Handler{
		hub:            hub,
		matchRepo:      matchRepo,
		enrollmentRepo: enrollmentRepo,
		deps:           deps,
	}
}

func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userUUID, err := h.authenticateRequest(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	matchUUIDStr := r.URL.Query().Get("match_uuid")
	if matchUUIDStr == "" {
		http.Error(w, `{"error":"match_uuid query parameter required"}`, http.StatusBadRequest)
		return
	}
	matchUUID, err := uuid.Parse(matchUUIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid match_uuid format"}`, http.StatusBadRequest)
		return
	}

	masterUUID, err := h.matchRepo.GetMatchMaster(r.Context(), matchUUID)
	if err != nil {
		http.Error(w, `{"error":"match not found"}`, http.StatusNotFound)
		return
	}

	isMaster := masterUUID == userUUID
	if !isMaster {
		enrolled, err := h.enrollmentRepo.IsPlayerEnrolledInMatch(r.Context(), userUUID, matchUUID)
		if err != nil || !enrolled {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}

	nickname := r.URL.Query().Get("nickname")
	if nickname == "" {
		nickname = userUUID.String()[:8]
	}

	if !isMaster {
		room, ok := h.hub.GetRoom(matchUUID)
		if !ok {
			rejectLobbyNotOpen(conn)
			return
		}
		client := NewClient(userUUID, conn, nickname)
		// B7 (spec §4.6): the room can have closed in the gap between GetRoom above and this
		// Register — Run's goroutine already returned, so Register comes back with
		// ErrRoomClosed instead of hanging. A player gets the same refusal as "never opened":
		// there is nothing to retry into, a lobby only reopens when the master reconnects.
		if err := room.Register(client); err != nil {
			rejectLobbyNotOpen(conn)
			return
		}
		go client.WritePump()
		go client.ReadPump()
		return
	}

	client := NewClient(userUUID, conn, nickname)
	room := h.hub.GetOrCreateRoom(matchUUID, masterUUID, h.deps)

	// B7 (spec §4.6): at most one retry. GetOrCreateRoom can hand back a room whose Run
	// goroutine already returned (the same closing race Register itself guards against) —
	// unlike a player, the master always has somewhere to retry into: GetOrCreateRoom makes
	// a fresh room the moment the one it just got is seen as closed.
	for attempt := 0; ; attempt++ {
		// After a backend restart the Room is freshly created with nil session.
		// If the match was already started in DB, rehydrate the session so
		// players can take actions without a full match restart.
		//
		// This runs BEFORE Register below, on purpose: the board itself loads inside Room.Run's
		// register branch (spec §4.3, "Quem carrega", B14), and that branch tells "still a lobby"
		// from "already playing" by reading r.session — so the session has to already be set by
		// the time this master's register reaches Run, or a live match would be mistaken for a
		// lobby and reload its board on every reconnect instead of just at birth.
		if room.GetSession() == nil {
			if started, err := h.matchRepo.IsStarted(r.Context(), matchUUID); err == nil && started {
				if session, err := h.deps.InitSessionUC.Init(r.Context(), matchUUID); err == nil {
					room.RehydrateSession(session)
				} else {
					log.Printf("failed to rehydrate session for match %s: %v", matchUUID, err)
				}
			}
		}

		err := room.Register(client)
		if err == nil {
			break
		}
		if !errors.Is(err, ErrRoomClosed) || attempt > 0 {
			// Register's only error is ErrRoomClosed, and the retry above already gave this
			// one fresh shot — a second failure means something is genuinely wrong, not the
			// ordinary closing race this loop exists for.
			log.Printf("master register failed for match %s: %v", matchUUID, err)
			_ = conn.Close()
			return
		}
		room = h.hub.GetOrCreateRoom(matchUUID, masterUUID, h.deps)
	}

	go client.WritePump()
	go client.ReadPump()
}

// rejectLobbyNotOpen tells a connecting player there is no lobby to join — either it never
// opened, or it closed in the gap between the handler's own check and Register (B7, spec
// §4.6); both look the same from a player's side, since a closed lobby only reopens when the
// master reconnects.
func rejectLobbyNotOpen(conn *websocket.Conn) {
	msg := NewServerMessage(MsgTypeLobbyNotOpen, struct{}{})
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal lobby_not_open: %v", err)
	} else if wErr := conn.WriteMessage(websocket.TextMessage, data); wErr != nil {
		log.Printf("lobby_not_open write failed: %v", wErr)
	}
	_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(4001, "lobby not open"))
	// Drain in a separate goroutine so the HTTP handler returns immediately
	// while the close handshake completes. Without this, conn.Close() may
	// send a TCP RST before the browser receives the text frame above,
	// causing onclose to fire with code 1006 instead of 4001.
	go func() {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
		_ = conn.Close()
	}()
}

func (h *Handler) authenticateRequest(r *http.Request) (uuid.UUID, error) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		authHeader := r.Header.Get("Authorization")
		const bearerPrefix = "Bearer "
		if strings.HasPrefix(authHeader, bearerPrefix) {
			tokenStr = authHeader[len(bearerPrefix):]
		}
	}
	if tokenStr == "" {
		return uuid.Nil, http.ErrNoCookie
	}

	claims, err := pkgAuth.ValidateToken(tokenStr)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}
