package game

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
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

type IInitMatchSession interface {
	Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error)
}

type IOpenNextAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*appmatch.OpenNextActionResult, error)
}

type IPullAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, actionID uuid.UUID) (*appmatch.PullActionResult, error)
}

type IEnqueueAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, playerUUID uuid.UUID, a *action.Action) error
}

type IAttachReaction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, callerUUID uuid.UUID, r *action.Action) (*appmatch.AttachReactionResult, error)
}

type IChangeScene interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, category enum.SceneCategory, briefDesc string) (*sceneentity.Scene, *roundentity.Round, error)
}

type IEnqueueMasterAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, ma *action.MasterAction) error
}

type Room struct {
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	state      RoomState
	clients    map[uuid.UUID]*Client
	// pieces holds the authoritative in-memory board state. Updated on every
	// piece_moved / piece_removed. Sent to every new client on register so
	// late-joiners always see the current board.
	pieces   map[string]PieceMovedPayload      // keyed by piece_id
	walls    map[string]mapentity.WallSegment  // in-memory runtime wall state; keyed by wall ID
	gridSize float64                           // cell size in world coords; used for movement blocking
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	mu         sync.RWMutex

	session *matchsession.MatchSession

	startMatchUC          IStartMatch
	kickPlayerUC          IKickPlayer
	initSessionUC         IInitMatchSession
	openNextActionUC      IOpenNextAction
	pullActionUC          IPullAction
	enqueueActionUC       IEnqueueAction
	attachReactionUC      IAttachReaction
	changeSceneUC         IChangeScene
	roundRepo             appmatch.IRoundRepository
	enqueueMasterActionUC IEnqueueMasterAction
}

func NewRoom(
	matchUUID, masterUUID uuid.UUID,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
	initSessionUC IInitMatchSession,
	openNextActionUC IOpenNextAction,
	pullActionUC IPullAction,
	enqueueActionUC IEnqueueAction,
	attachReactionUC IAttachReaction,
	changeSceneUC IChangeScene,
	roundRepo appmatch.IRoundRepository,
	enqueueMasterActionUC IEnqueueMasterAction,
) *Room {
	return &Room{
		matchUUID:             matchUUID,
		masterUUID:            masterUUID,
		state:                 RoomStateLobby,
		clients:               make(map[uuid.UUID]*Client),
		pieces:                make(map[string]PieceMovedPayload),
		walls:                 make(map[string]mapentity.WallSegment),
		gridSize:              64, // default; overridden by map_state_sync
		broadcast:             make(chan []byte, 256),
		register:              make(chan *Client),
		unregister:            make(chan *Client),
		stop:                  make(chan struct{}),
		startMatchUC:          startMatchUC,
		kickPlayerUC:          kickPlayerUC,
		initSessionUC:         initSessionUC,
		openNextActionUC:      openNextActionUC,
		pullActionUC:          pullActionUC,
		enqueueActionUC:       enqueueActionUC,
		attachReactionUC:      attachReactionUC,
		changeSceneUC:         changeSceneUC,
		roundRepo:             roundRepo,
		enqueueMasterActionUC: enqueueMasterActionUC,
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

func (r *Room) GetSession() *matchsession.MatchSession {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.session
}

// RehydrateSession restores session after a backend restart. Only called when
// the match was already started in DB but the in-memory Room has no session.
func (r *Room) RehydrateSession(session *matchsession.MatchSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.session != nil {
		return // another goroutine already rehydrated
	}
	r.session = session
	wallSlice := make([]mapentity.WallSegment, 0, len(r.walls))
	for _, w := range r.walls {
		wallSlice = append(wallSlice, w)
	}
	r.session.SyncMapState(wallSlice, r.gridSize)
	r.state = RoomStatePlaying
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
			r.mu.RLock()
			hasPieces := len(r.pieces) > 0
			r.mu.RUnlock()
			if hasPieces {
				r.sendMapFullState(client)
			}
			r.broadcastPlayerJoined(client)

		case client := <-r.unregister:
			// Guard: only remove if this exact client pointer is still registered.
			// A reconnecting user (e.g. React Strict Mode double-invoke) may have
			// already replaced the map entry before the old goroutine unregisters —
			// without this check the new connection would be evicted and the room
			// would close spuriously.
			r.mu.Lock()
			removed := false
			if current, ok := r.clients[client.userUUID]; ok && current == client {
				delete(r.clients, client.userUUID)
				close(client.send)
				removed = true
			}
			r.mu.Unlock()

			if !removed {
				continue
			}

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

	ctx := context.Background()
	if err := r.startMatchUC.Start(ctx, r.matchUUID, userUUID); err != nil {
		return err
	}

	session, err := r.initSessionUC.Init(ctx, r.matchUUID)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.session = session
	wallSlice := make([]mapentity.WallSegment, 0, len(r.walls))
	for _, w := range r.walls {
		wallSlice = append(wallSlice, w)
	}
	r.session.SyncMapState(wallSlice, r.gridSize)
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

func (r *Room) CloseLobby(masterUUID uuid.UUID) error {
	if !r.IsMaster(masterUUID) {
		return ErrNotMaster
	}

	r.mu.RLock()
	state := r.state
	r.mu.RUnlock()
	if state != RoomStateLobby {
		return ErrAlreadyPlaying // room is not in lobby state
	}

	msg := NewServerMessage(MsgTypeLobbyClosed, struct{}{})
	data, _ := json.Marshal(msg)

	r.mu.RLock()
	for _, c := range r.clients {
		select {
		case c.send <- data:
		default:
		}
	}
	r.mu.RUnlock()

	r.Stop()
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

	case MsgTypeOpenNextAction:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		r.mu.RLock()
		session := r.session
		r.mu.RUnlock()
		result, err := r.openNextActionUC.Execute(context.Background(), session, r.masterUUID, client.userUUID)
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}

		if result.ClosedTurn != nil {
			closedTurn := result.ClosedTurn
			closedAct := closedTurn.GetAction()
			r.mu.RLock()
			activeScene := session.GetActiveScene()
			activeRound := session.GetActiveRound()
			matchUUID := session.GetMatchUUID()
			r.mu.RUnlock()
			if err2 := r.roundRepo.PersistTurnClose(context.Background(), activeScene, activeRound, closedTurn, &closedAct, matchUUID); err2 != nil {
				log.Printf("PersistTurnClose error: %v", err2)
			} else {
				r.mu.Lock()
				session.MarkRoundPersisted()
				r.mu.Unlock()
			}
		}

		act := result.OpenedTurn.GetAction()
		out := NewServerMessage(MsgTypeTurnOpened, TurnOpenedPayload{
			TurnID:  result.OpenedTurn.GetID(),
			ActorID: act.GetActorID(),
		})
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()
		if result.Resolution != nil {
			r.broadcastWallResults(session, result.Resolution.WallResults)
		}

	case MsgTypePullAction:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload PullActionPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid pull_action payload"))
			return
		}
		r.mu.RLock()
		session := r.session
		r.mu.RUnlock()
		result, err := r.pullActionUC.Execute(context.Background(), session, r.masterUUID, client.userUUID, payload.ActionID)
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}

		if result.ClosedTurn != nil {
			closedTurn := result.ClosedTurn
			closedAct := closedTurn.GetAction()
			r.mu.RLock()
			activeScene := session.GetActiveScene()
			activeRound := session.GetActiveRound()
			matchUUID := session.GetMatchUUID()
			r.mu.RUnlock()
			if err2 := r.roundRepo.PersistTurnClose(context.Background(), activeScene, activeRound, closedTurn, &closedAct, matchUUID); err2 != nil {
				log.Printf("PersistTurnClose error: %v", err2)
			} else {
				r.mu.Lock()
				session.MarkRoundPersisted()
				r.mu.Unlock()
			}
		}

		act := result.OpenedTurn.GetAction()
		out := NewServerMessage(MsgTypeTurnOpened, TurnOpenedPayload{
			TurnID:  result.OpenedTurn.GetID(),
			ActorID: act.GetActorID(),
		})
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()
		if result.Resolution != nil {
			r.broadcastWallResults(session, result.Resolution.WallResults)
		}

	case MsgTypeEnqueueAction:
		var payload ActionPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid action payload"))
			return
		}
		if payload.Dodge != nil && payload.ReactToID == uuid.Nil {
			client.SendMessage(NewErrorMessage("invalid_action", "dodge must be a reaction — set react_to_id"))
			return
		}
		r.mu.RLock()
		session := r.session
		r.mu.RUnlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		// TODO: consider collapsing enqueue_action and attach_reaction into a single message type
		if payload.ReactToID != uuid.Nil {
			r.handleReaction(client, session, payload)
			return
		}
		a := buildAction(client.userUUID, payload)
		// Movement blocking: validate path against walls with move=true and !open.
		if a.Move != nil {
			from := a.Move.From
			to := a.Move.Position
			// Only validate when the client provided a non-zero From (zero means "not provided").
			if from != ([3]int{}) {
				r.mu.RLock()
				sess := r.session
				var gridSize float64
				var walls []mapentity.WallSegment
				if sess != nil {
					gridSize = sess.GetGridSize()
					walls = sess.GetWalls()
				} else {
					gridSize = r.gridSize
					walls = make([]mapentity.WallSegment, 0, len(r.walls))
					for _, w := range r.walls {
						walls = append(walls, w)
					}
				}
				r.mu.RUnlock()
				fromWorld := [2]float64{float64(from[0]) * gridSize, float64(from[1]) * gridSize}
				toWorld := [2]float64{float64(to[0]) * gridSize, float64(to[1]) * gridSize}
				if mapservice.IsPathBlocked(fromWorld, toWorld, walls) {
					client.SendMessage(NewErrorMessage("move_blocked", "movement blocked by a wall"))
					return
				}
			}
		}
		if err := r.enqueueActionUC.Execute(context.Background(), session, client.userUUID, a); err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		client.SendMessage(NewServerMessage(MsgTypeActionEnqueued, struct{}{}))

	case MsgTypeAttachReaction:
		var payload ActionPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid action payload"))
			return
		}
		if payload.ReactToID == uuid.Nil {
			client.SendMessage(NewErrorMessage("invalid_action", "reaction requires react_to_id"))
			return
		}
		r.mu.RLock()
		session := r.session
		r.mu.RUnlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		r.handleReaction(client, session, payload)

	case MsgTypeChangeScene:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload ChangeScenePayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid change_scene payload"))
			return
		}
		r.mu.RLock()
		session := r.session
		if session == nil {
			r.mu.RUnlock()
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		// Capture persisted flag BEFORE ChangeScene resets it
		sceneWasPersisted := session.IsScenePersisted()
		r.mu.RUnlock()

		oldScene, oldRound, err := r.changeSceneUC.Execute(
			context.Background(), session,
			r.masterUUID, client.userUUID,
			enum.SceneCategory(payload.Category), payload.BriefInitialDescription,
		)
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}

		if sceneWasPersisted && oldScene != nil && oldRound != nil && oldRound.GetFinishedAt() != nil {
			if dbErr := r.roundRepo.CloseSceneAndRound(
				context.Background(),
				oldScene.GetID(), oldRound.GetID(), *oldRound.GetFinishedAt(),
			); dbErr != nil {
				log.Printf("CloseSceneAndRound error: %v", dbErr)
			}
		}

		r.mu.RLock()
		activeScene := session.GetActiveScene()
		r.mu.RUnlock()

		out := NewServerMessage(MsgTypeSceneChanged, SceneChangedPayload{
			SceneID:                 activeScene.GetID(),
			Category:                string(activeScene.GetCategory()),
			BriefInitialDescription: activeScene.BriefInitialDescription,
		})
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()

	case MsgTypeCancelLobby:
		if err := r.CloseLobby(client.userUUID); err != nil {
			client.SendMessage(NewErrorMessage("forbidden", err.Error()))
		}

	case MsgTypePieceMoved:
		// Broadcast piece move to all OTHER participants in the lobby.
		// No server-side piece ownership validation in Phase 6 — client restricts
		// drag to allowed pieces. TODO: validate piece ownership per user (Phase 7+)
		var payload PieceMovedPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid lobby_piece_moved payload"))
			return
		}
		// Keep in-memory board state current so late-joiners get the right board.
		r.mu.Lock()
		r.pieces[payload.PieceID] = payload
		r.mu.Unlock()
		outMsg := NewClientMessage(MsgTypePieceMoved, client.userUUID, payload)
		data, _ := json.Marshal(outMsg)
		r.mu.RLock()
		for id, c := range r.clients {
			if id == client.userUUID {
				continue
			}
			select {
			case c.send <- data:
			default:
				log.Printf("dropping lobby_piece_moved for slow client %s", id)
			}
		}
		r.mu.RUnlock()

	case MsgTypePieceRemoved:
		var payload PieceRemovedPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid lobby_piece_removed payload"))
			return
		}
		r.mu.Lock()
		delete(r.pieces, payload.PieceID)
		r.mu.Unlock()
		outMsg := NewClientMessage(MsgTypePieceRemoved, client.userUUID, payload)
		data, _ := json.Marshal(outMsg)
		r.mu.RLock()
		for id, c := range r.clients {
			if id == client.userUUID {
				continue
			}
			select {
			case c.send <- data:
			default:
				log.Printf("dropping lobby_piece_removed for slow client %s", id)
			}
		}
		r.mu.RUnlock()

	case MsgTypeMapStateSync:
		// Only the master may seed the in-memory board (initial DB state on connect).
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload MapStateSyncPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid map_state_sync payload"))
			return
		}
		r.mu.Lock()
		r.pieces = make(map[string]PieceMovedPayload, len(payload.Pieces))
		for _, p := range payload.Pieces {
			r.pieces[p.PieceID] = p
		}
		r.walls = make(map[string]mapentity.WallSegment, len(payload.Walls))
		for _, w := range payload.Walls {
			r.walls[w.ID] = w
		}
		if payload.Grid != nil && payload.Grid.CellSize > 0 {
			r.gridSize = payload.Grid.CellSize
		}
		sess := r.session
		r.mu.Unlock()
		if sess != nil {
			wallSlice := make([]mapentity.WallSegment, 0, len(payload.Walls))
			for _, w := range payload.Walls {
				wallSlice = append(wallSlice, w)
			}
			sess.SyncMapState(wallSlice, r.gridSize)
		}
		// No relay — only seeds the server's in-memory state.

	case MsgTypeEnqueueMasterAction:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload MasterActionPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid enqueue_master_action payload"))
			return
		}
		r.mu.RLock()
		session := r.session
		r.mu.RUnlock()
		ma := buildMasterAction(client.userUUID, payload)
		// Wall interaction: handled in-memory + broadcast; does not go through the use case queue.
		if ma.Interact != nil && len(ma.TargetID) > 0 {
			for _, targetID := range ma.TargetID {
				newOpen, newLocked, ok := r.applyWallInteract(targetID.String(), ma.Interact)
				if !ok {
					// Wall not in in-memory state — skip silently.
					continue
				}
				evt := NewServerMessage(MsgTypeWallStateChanged, WallStateChangedPayload{
					WallID: targetID.String(),
					Open:   newOpen,
					Locked: newLocked,
				})
				data, _ := json.Marshal(evt)
				go func() { r.broadcast <- data }()
			}
			return
		}
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		if err := r.enqueueMasterActionUC.Execute(context.Background(), session, r.masterUUID, client.userUUID, ma); err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		out := NewServerMessage(MsgTypeMasterActionEnqueued, MasterActionEnqueuedPayload(payload))
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()

	default:
		client.SendMessage(NewErrorMessage("unknown_type", "unrecognized message type"))
	}
}

func (r *Room) handleReaction(client *Client, session *matchsession.MatchSession, payload ActionPayload) {
	r.mu.RLock()
	masterClient, hasMaster := r.clients[r.masterUUID]
	r.mu.RUnlock()

	reaction := buildAction(client.userUUID, payload)
	result, err := r.attachReactionUC.Execute(context.Background(), session, client.userUUID, reaction)
	if err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	if hasMaster {
		out := NewServerMessage(MsgTypeResolutionUpdate, ResolutionUpdatedPayload{IsSettled: result.Resolution.IsSettled})
		masterClient.SendMessage(out)
	}
}

// applyWallInteract updates in-memory wall state for open/close/toggle.
// Returns (newOpen, newLocked, ok). ok=false means wall not found or interaction
// not applicable (e.g. lockpick/examine are player-only actions requiring rolls).
func (r *Room) applyWallInteract(wallID string, interact *action.Interact) (open, locked bool, ok bool) {
	r.mu.RLock()
	sess := r.session
	r.mu.RUnlock()

	r.mu.Lock()
	w, exists := r.walls[wallID]
	if !exists {
		r.mu.Unlock()
		return false, false, false
	}
	updated, ok := domainservice.ApplyWallInteract(w, interact)
	if !ok {
		r.mu.Unlock()
		return false, false, false
	}
	r.walls[wallID] = updated
	r.mu.Unlock()

	if sess != nil {
		sess.UpdateWall(updated)
	}
	return updated.Open, updated.Locked, true
}

func (r *Room) sendMapFullState(client *Client) {
	r.mu.RLock()
	pieces := make([]PieceMovedPayload, 0, len(r.pieces))
	for _, p := range r.pieces {
		pieces = append(pieces, p)
	}
	r.mu.RUnlock()
	msg := NewServerMessage(MsgTypeMapFullState, MapPiecesPayload{Pieces: pieces})
	client.SendMessage(msg)
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

func (r *Room) broadcastWallResults(session *matchsession.MatchSession, results []domainservice.WallResult) {
	for _, wr := range results {
		session.UpdateWall(wr.UpdatedWall)
		r.mu.Lock()
		r.walls[wr.UpdatedWall.ID] = wr.UpdatedWall
		r.mu.Unlock()
		var msg Message
		switch wr.Kind {
		case domainservice.WallResultKindAttack:
			msg = NewServerMessage(MsgTypeWallHpChanged, WallHpChangedPayload{
				WallID:    wr.UpdatedWall.ID,
				HP:        wr.UpdatedWall.HP,
				MaxHP:     wr.UpdatedWall.MaxHP,
				Destroyed: wr.UpdatedWall.Destroyed,
			})
		case domainservice.WallResultKindInteract:
			msg = NewServerMessage(MsgTypeWallStateChanged, WallStateChangedPayload{
				WallID: wr.UpdatedWall.ID,
				Open:   wr.UpdatedWall.Open,
				Locked: wr.UpdatedWall.Locked,
			})
		default:
			continue
		}
		data, _ := json.Marshal(msg)
		go func(d []byte) { r.broadcast <- d }(data)
	}
}
