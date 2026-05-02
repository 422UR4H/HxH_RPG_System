package game

import (
	"context"
	"log"
	"net/http"
	"strings"

	pkgAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MatchRepository interface {
	GetMatchMaster(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
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
	startMatchUC   IStartMatch
	kickPlayerUC   IKickPlayer
}

func NewHandler(
	hub *Hub,
	matchRepo MatchRepository,
	enrollmentRepo EnrollmentChecker,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
) *Handler {
	return &Handler{
		hub:            hub,
		matchRepo:      matchRepo,
		enrollmentRepo: enrollmentRepo,
		startMatchUC:   startMatchUC,
		kickPlayerUC:   kickPlayerUC,
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

	room := h.hub.GetOrCreateRoom(matchUUID, masterUUID, h.startMatchUC, h.kickPlayerUC)
	client := NewClient(userUUID, conn, nickname)

	room.Register(client)

	go client.WritePump()
	go client.ReadPump()
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
