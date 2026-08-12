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
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
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
	pieces     map[string]PieceMovedPayload     // keyed by piece_id
	walls      map[string]mapentity.WallSegment // in-memory runtime wall state; keyed by wall ID
	grid       mapentity.GridShape              // full grid shape; used for movement blocking and fog coords
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
		grid:                  mapentity.DefaultGrid(), // default; overridden by map_state_sync
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
	r.session.SyncMapState(wallSlice, r.grid)
	r.session.SetPieceSource(r)
	// fogMode fixo em explored: PENDENTE de configurações de partida.
	// FogMode é real e usado (filter_map_state.go decide memória de parede por ele), mas
	// o valor persistido em maps.fog_mode nunca chega aqui porque não existe ainda o
	// mecanismo de configuração de campanha/partida no backend — fog_mode será uma opção
	// que o mestre escolhe ao criar/editar a partida. Quando esse mecanismo existir, ler o
	// modo da configuração da partida e passar aqui. NÃO remover FogMode achando que é
	// código morto: isso eliminaria o modo `live` do produto.
	// Ver: System_X_System_React/docs/superpowers/specs/2026-08-06-tactical-map-refactor-design.md §3
	r.session.SyncPlayerMemories(nil, fogentity.FogModeExplored)
	for _, pid := range r.session.PlayerIDs() {
		if _, err := r.session.RecomputeVisibility(pid); err != nil {
			log.Printf("rehydrate recompute visibility for %s: %v", pid, err)
		}
	}
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
				msg := r.buildMapFullState(client.userUUID, r.IsMaster(client.userUUID))
				client.SendMessage(*msg)
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
				// Signal shutdown via done, never by closing send: a concurrent
				// SendMessage on a closed channel panics and kills the process.
				client.Close()
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
				client.Close()
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
	r.session.SyncMapState(wallSlice, r.grid)
	r.session.SetPieceSource(r)
	// fogMode fixo em explored: PENDENTE de configurações de partida.
	// FogMode é real e usado (filter_map_state.go decide memória de parede por ele), mas
	// o valor persistido em maps.fog_mode nunca chega aqui porque não existe ainda o
	// mecanismo de configuração de campanha/partida no backend — fog_mode será uma opção
	// que o mestre escolhe ao criar/editar a partida. Quando esse mecanismo existir, ler o
	// modo da configuração da partida e passar aqui. NÃO remover FogMode achando que é
	// código morto: isso eliminaria o modo `live` do produto.
	// Ver: System_X_System_React/docs/superpowers/specs/2026-08-06-tactical-map-refactor-design.md §3
	r.session.SyncPlayerMemories(nil, fogentity.FogModeExplored)
	playerIDs := r.session.PlayerIDs()
	r.state = RoomStatePlaying
	r.mu.Unlock()

	for _, pid := range playerIDs {
		r.mu.Lock()
		_, err := r.session.RecomputeVisibility(pid)
		r.mu.Unlock()
		if err != nil {
			log.Printf("recompute visibility for %s: %v", pid, err)
		}
		// TODO(persistence): playerMemoryRepo.Upsert(session.GetPlayerMemory(pid))
	}

	// Send match_started directly per-client (in order) so it always precedes the
	// per-player map_full_state that follows. Using the broadcast channel here would
	// race against the direct sends below.
	startedMsg := NewServerMessage(MsgTypeMatchStarted, struct{}{})
	r.dispatchPerPlayer(func(_ uuid.UUID, _ bool) *Message {
		m := startedMsg
		return &m
	})

	// Push the filtered full board state to each client (master unfiltered).
	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		return r.buildMapFullState(pid, isMaster)
	})
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
					gridSize = r.grid.CellSize
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
		// Relay piece moves per-player with fog-of-war filtering.
		// No server-side piece ownership validation in Phase 6 — client restricts
		// drag to allowed pieces. TODO: validate piece ownership per user (Phase 7+)
		var payload PieceMovedPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid lobby_piece_moved payload"))
			return
		}
		r.handlePieceMoved(client, payload)

	case MsgTypePieceRemoved:
		var payload PieceRemovedPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid lobby_piece_removed payload"))
			return
		}
		r.handlePieceRemoved(client, payload)

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
		walls := make([]mapentity.WallSegment, len(payload.Walls))
		for i, w := range payload.Walls {
			walls[i] = toEntityWallSegment(w)
		}
		var grid *mapentity.GridShape
		if payload.Grid != nil {
			g := toEntityGridShape(*payload.Grid)
			grid = &g
		}
		r.mu.Lock()
		// A nil Pieces field means "no piece information in this sync" — keep the board.
		// Only an explicitly present array replaces it (empty array = board is empty).
		if payload.Pieces != nil {
			r.pieces = make(map[string]PieceMovedPayload, len(*payload.Pieces))
			for _, p := range *payload.Pieces {
				r.pieces[p.PieceID] = p
			}
		}
		r.walls = make(map[string]mapentity.WallSegment, len(walls))
		for _, w := range walls {
			r.walls[w.ID] = w
		}
		if grid != nil && grid.CellSize > 0 {
			r.grid = *grid
		}
		roomGrid := r.grid
		sess := r.session
		if sess != nil {
			wallSlice := append([]mapentity.WallSegment(nil), walls...)
			sess.SyncMapState(wallSlice, roomGrid)
			// The board just changed, so every player's cached LOS is stale — and
			// buildMapFullState serves the cache. Recompute here (still under the write
			// lock, as PlayerPiecePositions requires) so the refreshed state pushed below
			// reflects the board that was just seeded.
			for _, pid := range sess.PlayerIDs() {
				if _, err := sess.RecomputeVisibility(pid); err != nil {
					log.Printf("map_state_sync recompute visibility for %s: %v", pid, err)
				}
			}
		}
		r.mu.Unlock()

		// Re-push the full board to everyone. This is what makes the feature converge
		// regardless of connect order: whether the master syncs before or after a player
		// joins, and whether the server was restarted mid-match, each client ends up with
		// state built from the seeded board.
		if sess != nil {
			r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
				return r.buildMapFullState(pid, isMaster)
			})
		}

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
		// Reveal: master-only secret-door reveal. Marks the wall revealed in the session
		// and broadcasts the full real WallSegment to ALL clients.
		if ma.Interact != nil && ma.Interact.Kind == action.InteractReveal && len(ma.TargetID) > 0 {
			r.revealSecretDoors(ma.TargetID)
			return
		}
		// Wall interaction: handled in-memory + broadcast; does not go through the use case queue.
		if ma.Interact != nil && len(ma.TargetID) > 0 {
			for _, targetID := range ma.TargetID {
				newOpen, newLocked, ok := r.applyWallInteract(targetID.String(), ma.Interact)
				if !ok {
					// Wall not in in-memory state — skip silently.
					continue
				}
				r.broadcastWallStateChangedGated(targetID.String(), newOpen, newLocked)
			}
			// Wall geometry may have changed (open/close) → recompute and push LOS.
			r.pushVisibilityUpdates()
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
	// Hold the write lock across the session update too: MatchSession has no internal
	// lock, so r.mu is the only thing serializing access to its wall map.
	r.mu.Lock()
	defer r.mu.Unlock()

	w, exists := r.walls[wallID]
	if !exists {
		return false, false, false
	}
	updated, ok := domainservice.ApplyWallInteract(w, interact)
	if !ok {
		return false, false, false
	}
	r.walls[wallID] = updated
	if r.session != nil {
		r.session.UpdateWall(updated)
	}
	return updated.Open, updated.Locked, true
}

// ---------------------------------------------------------------------------
// Fog of war: per-player dispatch + filtering helpers
// ---------------------------------------------------------------------------

// dispatchPerPlayer sends a per-player-built message to each client. build returns nil to skip.
func (r *Room) dispatchPerPlayer(build func(playerID uuid.UUID, isMaster bool) *Message) {
	r.mu.RLock()
	type entry struct {
		c        *Client
		isMaster bool
	}
	entries := make(map[uuid.UUID]entry, len(r.clients))
	for id, c := range r.clients {
		entries[id] = entry{c: c, isMaster: id == r.masterUUID}
	}
	r.mu.RUnlock()

	for id, e := range entries {
		if msg := build(id, e.isMaster); msg != nil {
			e.c.SendMessage(*msg)
		}
	}
}

func (r *Room) sendToMaster(msg Message) {
	r.mu.RLock()
	c, ok := r.clients[r.masterUUID]
	r.mu.RUnlock()
	if ok {
		c.SendMessage(msg)
	}
}

func (r *Room) broadcastWallStateChanged(wallID string, open, locked bool) {
	msg := NewServerMessage(MsgTypeWallStateChanged, WallStateChangedPayload{
		WallID: wallID,
		Open:   open,
		Locked: locked,
	})
	data, _ := json.Marshal(msg)
	go func() { r.broadcast <- data }()
}

// broadcastWallStateChangedGated sends wall_state_changed to everyone, except that an
// unrevealed secret door's open/locked change goes to the master only. Players see such a
// door as a plain wall, and a plain wall has no open/locked state — broadcasting it would
// leak the door's identity. Mirrors the WallResultKindInteract gate in broadcastWallResults.
func (r *Room) broadcastWallStateChangedGated(wallID string, open, locked bool) {
	r.mu.RLock()
	w, ok := r.walls[wallID]
	r.mu.RUnlock()
	if ok && w.WallType == mapentity.WallTypeSecretDoor && !w.Revealed {
		r.sendToMaster(NewServerMessage(MsgTypeWallStateChanged, WallStateChangedPayload{
			WallID: wallID,
			Open:   open,
			Locked: locked,
		}))
		return
	}
	r.broadcastWallStateChanged(wallID, open, locked)
}

// broadcastWallHpChanged dispatches wall_hp_changed only to clients who can see the wall.
// The master always receives it. The payload carries no wall type, so this never leaks
// secret-door identity.
func (r *Room) broadcastWallHpChanged(w mapentity.WallSegment) {
	mid := domainservice.Point2D{
		X: (w.P1[0] + w.P2[0]) / 2,
		Y: (w.P1[1] + w.P2[1]) / 2,
	}
	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		if !isMaster && !domainservice.IsVisible(mid, r.visibilityFor(pid)) {
			return nil
		}
		m := NewServerMessage(MsgTypeWallHpChanged, WallHpChangedPayload{
			WallID:    w.ID,
			HP:        w.HP,
			MaxHP:     w.MaxHP,
			Destroyed: w.Destroyed,
		})
		return &m
	})
}

// revealSecretDoors marks each target wall revealed in the session and broadcasts the full
// real WallSegment to ALL clients. Master-only; the caller gates on master.
func (r *Room) revealSecretDoors(targetIDs []uuid.UUID) {
	r.mu.Lock()
	sess := r.session
	for _, targetID := range targetIDs {
		wallID := targetID.String()
		if sess != nil {
			sess.RevealSecretDoor(wallID)
			if w, ok := sess.GetWall(wallID); ok {
				r.walls[wallID] = w
			}
		} else if w, ok := r.walls[wallID]; ok {
			w.Revealed = true
			r.walls[wallID] = w
		}
	}
	r.mu.Unlock()

	for _, targetID := range targetIDs {
		wallID := targetID.String()
		r.mu.RLock()
		var w mapentity.WallSegment
		var ok bool
		if sess != nil {
			w, ok = sess.GetWall(wallID)
		} else {
			w, ok = r.walls[wallID]
		}
		r.mu.RUnlock()
		if !ok {
			continue
		}
		msg := NewServerMessage(MsgTypeWallRevealed, WallRevealedPayload{Wall: toWallSegmentPayload(w)})
		data, _ := json.Marshal(msg)
		go func(d []byte) { r.broadcast <- d }(data)
	}
	// Revealing changes nothing about geometry but the cache must reflect Revealed; push LOS.
	r.pushVisibilityUpdates()
}

// pushVisibilityUpdates recomputes each player's LOS and sends them a visibility_updated.
func (r *Room) pushVisibilityUpdates() {
	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		if isMaster {
			return nil
		}
		polys, err := func() ([]domainservice.VisibilityPolygon, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.session == nil {
				return nil, nil
			}
			return r.session.RecomputeVisibility(pid)
		}()
		if err != nil || polys == nil {
			return nil
		}
		payload := VisibilityUpdatedPayload{VisiblePolygons: polysToPayload(polys)}
		msg := NewServerMessage(MsgTypeVisibilityUpdated, payload)
		return &msg
	})
}

// visibilityFor returns the cached visibility polygons for a player (nil if no session).
func (r *Room) visibilityFor(pid uuid.UUID) []domainservice.VisibilityPolygon {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.session == nil {
		return nil
	}
	return r.session.GetVisibility(pid)
}

// gridShape returns the session's grid when a match is live, else the room's grid.
func (r *Room) gridShape() mapentity.GridShape {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.session != nil {
		return r.session.GetGrid()
	}
	return r.grid
}

func slotPayloadToWorld(s SlotPayload, g mapentity.GridShape) (float64, float64) {
	a, b := 0, 0
	if s.Kind == "hex" {
		if s.Q != nil {
			a = *s.Q
		}
		if s.R != nil {
			b = *s.R
		}
	} else {
		if s.Col != nil {
			a = *s.Col
		}
		if s.Row != nil {
			b = *s.Row
		}
	}
	return mapservice.SlotCenterToWorld(a, b, g)
}

// PlayerPiecePositions implements matchsession.PiecePositionSource. It returns the world
// positions of the pieces owned by playerID (resolved via the session's char→player map).
//
// LOCKING: the caller MUST already hold r.mu. This method deliberately takes no lock of
// its own because the session calls it back from inside RecomputeVisibility, which every
// caller invokes while holding r.mu for writing — taking r.mu.RLock() here would be an
// RLock inside a Lock on the same goroutine, which deadlocks Go's RWMutex permanently.
// This matches the room-owns-the-lock invariant documented in game-server.instructions.md.
// Guarded by TestRecomputeVisibilityUnderRoomWriteLock_DoesNotDeadlock.
func (r *Room) PlayerPiecePositions(playerID uuid.UUID) []domainservice.Point2D {
	grid := r.grid
	if r.session != nil {
		grid = r.session.GetGrid()
	}
	var charToPlayer map[string]uuid.UUID
	if r.session != nil {
		charToPlayer = r.session.GetCharToPlayer()
	}
	out := make([]domainservice.Point2D, 0)
	for _, p := range r.pieces {
		// Safe default: if the char→player map is missing, own nothing rather than
		// treating every piece as this player's LOS origin.
		if charToPlayer == nil || charToPlayer[p.CharacterID] != playerID {
			continue
		}
		x, y := slotPayloadToWorld(p.Slot, grid)
		out = append(out, domainservice.Point2D{X: x, Y: y})
	}
	return out
}

// buildMapFullState builds the filtered map_full_state for one viewer. Master gets the
// unfiltered board with no polygons; players get LOS-filtered walls/pieces plus their
// visible polygons and explored cells.
func (r *Room) buildMapFullState(playerID uuid.UUID, isMaster bool) *Message {
	r.mu.RLock()
	allWalls := make([]mapentity.WallSegment, 0, len(r.walls))
	for _, w := range r.walls {
		allWalls = append(allWalls, w)
	}
	allPieces := make([]PieceMovedPayload, 0, len(r.pieces))
	pieceProj := make([]domainservice.PieceVisibility, 0, len(r.pieces))
	grid := r.grid
	if r.session != nil {
		grid = r.session.GetGrid()
	}
	for _, p := range r.pieces {
		allPieces = append(allPieces, p)
		x, y := slotPayloadToWorld(p.Slot, grid)
		visible := true
		if p.Visible != nil {
			visible = *p.Visible
		}
		pieceProj = append(pieceProj, domainservice.PieceVisibility{
			ID:          p.PieceID,
			CharacterID: p.CharacterID,
			Pos:         domainservice.Point2D{X: x, Y: y},
			Visible:     visible,
		})
	}
	var polys []domainservice.VisibilityPolygon
	fogMode := fogentity.FogModeLive
	var memory *fogentity.PlayerMemory
	charToPlayer := map[string]uuid.UUID{}
	// LOS fog applies only once a match is live. In the lobby (no session) there is no LOS,
	// but secret doors are still masked and invisible pieces hidden from non-master players.
	isLobby := r.session == nil
	if r.session != nil {
		polys = r.session.GetVisibility(playerID)
		fogMode = r.session.GetFogMode()
		charToPlayer = r.session.GetCharToPlayer()
		if m, ok := r.session.GetPlayerMemory(playerID); ok {
			memory = m
		}
	}
	r.mu.RUnlock()

	var walls []mapentity.WallSegment
	var visIDs map[string]bool
	switch {
	case isMaster:
		// Master sees the true board, unmasked, with every piece.
		walls, visIDs = domainservice.FilterMapState(
			allWalls, pieceProj, polys, memory, fogMode, playerID, charToPlayer, true,
		)
	case isLobby:
		// Lobby: no LOS gating, but mask unrevealed secret doors and hide invisible pieces.
		walls, visIDs = computeLobbyMapState(allWalls, pieceProj)
	default:
		// In-match player: full per-player LOS filtering.
		walls, visIDs = domainservice.FilterMapState(
			allWalls, pieceProj, polys, memory, fogMode, playerID, charToPlayer, false,
		)
	}

	pieces := make([]PieceMovedPayload, 0, len(allPieces))
	for _, p := range allPieces {
		if isMaster || visIDs[p.PieceID] {
			pieces = append(pieces, p)
		}
	}
	wallPayloads := make([]WallSegmentPayload, len(walls))
	for i, w := range walls {
		wallPayloads[i] = toWallSegmentPayload(w)
	}
	payload := MapFullStatePayload{Pieces: pieces, Walls: wallPayloads, FogMode: string(fogMode)}
	if !isMaster && isLobby {
		// Disabled until the frontend consumes payload.Walls in lobby mode
		// (useLobbyWs.ts currently drops it). The masking computation above is intact —
		// flip this back to `payload.Walls = wallPayloads` to re-enable.
		payload.Walls = []WallSegmentPayload{}
	}
	if !isMaster && !isLobby {
		payload.VisiblePolygons = polysToPayload(polys)
	}
	msg := NewServerMessage(MsgTypeMapFullState, payload)
	return &msg
}

func polysToPayload(polys []domainservice.VisibilityPolygon) [][]Point2DPayload {
	out := make([][]Point2DPayload, 0, len(polys))
	for _, poly := range polys {
		pts := make([]Point2DPayload, 0, len(poly.Vertices))
		for _, v := range poly.Vertices {
			pts = append(pts, Point2DPayload{X: v.X, Y: v.Y})
		}
		out = append(out, pts)
	}
	return out
}

// computeLobbyMapState computes masked walls and piece visibility for the lobby phase.
// It masks unrevealed secret doors (calling MaskSecretDoorForPlayer) and passes through
// normal walls. This wiring must remain testable independently since the computed walls
// are not sent to clients (see buildMapFullState override at ~line 1144).
func computeLobbyMapState(allWalls []mapentity.WallSegment, pieceProj []domainservice.PieceVisibility) ([]mapentity.WallSegment, map[string]bool) {
	walls := make([]mapentity.WallSegment, 0, len(allWalls))
	for _, w := range allWalls {
		if w.WallType == mapentity.WallTypeSecretDoor && !w.Revealed {
			walls = append(walls, domainservice.MaskSecretDoorForPlayer(w))
		} else {
			walls = append(walls, w)
		}
	}
	visIDs := make(map[string]bool, len(pieceProj))
	for _, p := range pieceProj {
		if p.Visible {
			visIDs[p.ID] = true
		}
	}
	return walls, visIDs
}

// handlePieceMoved updates the board and relays the move per-player with fog filtering.
func (r *Room) handlePieceMoved(client *Client, payload PieceMovedPayload) {
	r.mu.Lock()
	old, hadOld := r.pieces[payload.PieceID]
	r.pieces[payload.PieceID] = payload
	r.mu.Unlock()

	grid := r.gridShape()
	newX, newY := slotPayloadToWorld(payload.Slot, grid)
	var oldX, oldY float64
	if hadOld {
		oldX, oldY = slotPayloadToWorld(old.Slot, grid)
	}
	hidden := payload.Visible != nil && !*payload.Visible

	moved := NewClientMessage(MsgTypePieceMoved, client.userUUID, payload)
	removed := NewClientMessage(MsgTypePieceRemoved, client.userUUID, PieceRemovedPayload{PieceID: payload.PieceID})

	r.mu.RLock()
	live := r.session != nil
	r.mu.RUnlock()

	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		if pid == client.userUUID {
			return nil // mover already applied the move locally
		}
		if isMaster {
			m := moved
			return &m
		}
		if !live {
			// Lobby phase: no fog, share the move with everyone.
			m := moved
			return &m
		}
		if hidden {
			return nil // hidden pieces never reach players
		}
		polys := r.visibilityFor(pid)
		seesNew := domainservice.IsVisible(domainservice.Point2D{X: newX, Y: newY}, polys)
		seesOld := hadOld && domainservice.IsVisible(domainservice.Point2D{X: oldX, Y: oldY}, polys)
		switch {
		case seesNew:
			m := moved
			return &m
		case seesOld:
			m := removed
			return &m
		default:
			return nil
		}
	})

	// When the mover moves their OWN piece, recompute their LOS and resend the full state.
	r.mu.RLock()
	sess := r.session
	var ownsPiece bool
	if sess != nil {
		ownsPiece = sess.GetCharToPlayer()[payload.CharacterID] == client.userUUID
	}
	r.mu.RUnlock()
	if sess != nil && ownsPiece {
		r.mu.Lock()
		_, err := r.session.RecomputeVisibility(client.userUUID)
		r.mu.Unlock()
		if err == nil {
			msg := r.buildMapFullState(client.userUUID, r.IsMaster(client.userUUID))
			client.SendMessage(*msg)
		}
	}
}

// handlePieceRemoved removes a piece and relays the removal per-player.
// A hidden piece (visible=false) is treated as master-only.
func (r *Room) handlePieceRemoved(client *Client, payload PieceRemovedPayload) {
	r.mu.Lock()
	old, hadOld := r.pieces[payload.PieceID]
	delete(r.pieces, payload.PieceID)
	r.mu.Unlock()

	hidden := hadOld && old.Visible != nil && !*old.Visible
	grid := r.gridShape()
	var oldX, oldY float64
	if hadOld {
		oldX, oldY = slotPayloadToWorld(old.Slot, grid)
	}

	removed := NewClientMessage(MsgTypePieceRemoved, client.userUUID, payload)

	r.mu.RLock()
	live := r.session != nil
	r.mu.RUnlock()

	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		if pid == client.userUUID {
			return nil
		}
		if isMaster {
			m := removed
			return &m
		}
		if !live {
			// Lobby phase: no fog, share the removal with everyone.
			m := removed
			return &m
		}
		if hidden {
			return nil
		}
		// Only notify players who could see the piece at its last known position.
		if hadOld && !domainservice.IsVisible(domainservice.Point2D{X: oldX, Y: oldY}, r.visibilityFor(pid)) {
			return nil
		}
		m := removed
		return &m
	})
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
	msgType := MsgTypePlayerJoined
	if r.IsMaster(client.userUUID) {
		msgType = MsgTypeMasterJoined
	}
	msg := NewServerMessage(msgType, PlayerPayload{
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
	msgType := MsgTypePlayerLeft
	if r.IsMaster(client.userUUID) {
		msgType = MsgTypeMasterLeft
	}
	msg := NewServerMessage(msgType, PlayerPayload{
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
	changed := false
	for _, wr := range results {
		session.UpdateWall(wr.UpdatedWall)
		r.mu.Lock()
		r.walls[wr.UpdatedWall.ID] = wr.UpdatedWall
		r.mu.Unlock()
		switch wr.Kind {
		case domainservice.WallResultKindAttack:
			// HP changes are gated per-player by line of sight to the wall midpoint.
			// The payload carries no wall type, so this is safe even for secret doors.
			r.broadcastWallHpChanged(wr.UpdatedWall)
			changed = true
		case domainservice.WallResultKindInteract:
			// Unrevealed secret doors must not leak their open/locked state to players.
			if wr.UpdatedWall.WallType == mapentity.WallTypeSecretDoor && !wr.UpdatedWall.Revealed {
				r.sendToMaster(NewServerMessage(MsgTypeWallStateChanged, WallStateChangedPayload{
					WallID: wr.UpdatedWall.ID,
					Open:   wr.UpdatedWall.Open,
					Locked: wr.UpdatedWall.Locked,
				}))
			} else {
				r.broadcastWallStateChanged(wr.UpdatedWall.ID, wr.UpdatedWall.Open, wr.UpdatedWall.Locked)
			}
			changed = true
		default:
			continue
		}
	}
	if changed {
		r.pushVisibilityUpdates()
	}
}
