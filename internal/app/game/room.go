package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/wire/actionwire"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
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
	// ErrMasterMovesByMasterAction and ErrPlayersMoveByAction are the two refusals a live
	// session gives handlePieceMoved/handlePieceRemoved (spec §4.3, "Quem move o quê", B14):
	// piece_moved/piece_removed are a LOBBY-ONLY pair of verbs. Once a match has a session,
	// the master places/moves/removes pieces through enqueue_master_action's move/remove
	// (Task 5), and a player only ever moves a piece by acting.
	ErrMasterMovesByMasterAction = errors.New(
		"during a match the master moves pieces with enqueue_master_action",
	)
	ErrPlayersMoveByAction = errors.New("players move by action")
	// ErrPieceNotOwnedByPlayer is the lobby-phase refusal for a player's piece_moved: the
	// piece has to already exist, and its CURRENT CharacterID has to belong to the sender
	// (spec §4.3, "Quem move o quê" — "só peça existente de personagem dele"). It also covers
	// the payload trying to relabel the piece under a different CharacterID: a player moves
	// their own piece, they do not reassign whose piece it is.
	ErrPieceNotOwnedByPlayer = errors.New("you can only move your own existing pieces")
	// ErrPlayersCannotRemovePieces is the lobby-phase refusal for a player's piece_removed:
	// the spec's table gives removal to the master only, in the lobby ("lobby | jogador |
	// piece_moved | ... não remove") — there is no ownership carve-out the way piece_moved has
	// one, a player never removes any piece, theirs or not.
	ErrPlayersCannotRemovePieces = errors.New("only the master can remove pieces")
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

type IOpenReaction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, callerUUID, reactionID uuid.UUID) (*appmatch.OpenReactionResult, error)
}

type ICloseTurn = appmatch.ICloseTurn

type IChangeScene interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, category enum.SceneCategory, briefDesc string) (*sceneentity.Scene, *roundentity.Round, error)
}

type IEnqueueMasterAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, ma *action.MasterAction) error
}

type IEditAction = appmatch.IEditAction

type IAddLiveNPC = appmatch.IAddLiveNPC

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
	// done is closed when Run returns — B7 (spec §4.6). Register and the unregister send in
	// ReadPump both select on it, so neither blocks forever against a Room whose goroutine
	// already exited: Register returns ErrRoomClosed instead of hanging on an unregister
	// channel nobody is draining anymore.
	done chan struct{}
	mu   sync.RWMutex
	// barsSeq stamps every bars_updated snapshot. Bumped under mu at the instant the snapshot
	// is taken, so the number orders the SNAPSHOTS, not the sends — broadcastBars hands the
	// channel off to a goroutine, and two rapid opens can reach it out of order.
	//
	// It starts at the clock (NewRoom), not at 0: the contract promises the counter never
	// restarts, and clients drop any seq lower than the highest they applied, across
	// reconnects. A Room that replaces another for the same match — the process restarted, or
	// the room emptied and closed itself in Run — would otherwise count from 0 again and lose
	// every snapshot to the old room's last one. Microseconds keep it below 2^53, exact as a
	// JavaScript number.
	barsSeq uint64
	// boardLoaded is true once loadBoard has installed a board at least once for this Room (a
	// load that found no map attached does not count). It is what
	// narrows the master-reconnect half of "quando carrega" (spec §4.3) to a SINGLE load once
	// the match has started: while session == nil (still a lobby) every master register
	// reloads regardless of this flag, but once playing, boardLoaded stops a reconnecting
	// master from silently resetting a live board out from under the session.
	boardLoaded bool
	// mapUUID is the map the room's board belongs to — set from Board.MapUUID in loadBoard.
	// uuid.Nil means "no board loaded yet" (or no map attached at all), and persistBoard is a
	// no-op in that state: there is nothing to write a match_boards row FOR (spec §4.3).
	mapUUID uuid.UUID
	// bg is the board's own background override, loaded from Board.Bg in loadBoard and
	// preserved on every persistBoard save. nil means "inherit the map's background"
	// (matchboard.Board.Bg's own doc comment) — nobody writes a non-nil value yet; the
	// in-match map editor that will is future work.
	bg *mapentity.BgImage
	// persistMu serializes every board snapshot-and-write pair across goroutines — persistBoard's
	// and persistClosedTurn's (whose PersistTurnClose writes the board in the turn's own
	// transaction) — so two saves racing from two read pumps land in the order their SNAPSHOTS
	// were taken, not the order their DB round trips happened to finish. Lock order: persistMu,
	// then r.mu. r.mu is only ever held for the snapshot half, never across the write.
	persistMu sync.Mutex
	// pendingTurns is what happened inside each open turn and is written only with that turn's
	// close (turnWrites, owner decision 2026-10-01), keyed by turn. Guarded by mu.
	pendingTurns map[uuid.UUID]*turnWrites
	// unwrittenSheets and unwrittenRoundEnds are what a failed write left for the next one to
	// carry (unwritten.go). Guarded by mu.
	unwrittenSheets    map[uuid.UUID]*csSheet.CharacterSheet
	unwrittenRoundEnds []appmatch.RoundEnd

	session *matchsession.MatchSession

	deps RoomDeps
}

func NewRoom(
	matchUUID, masterUUID uuid.UUID,
	deps RoomDeps,
) *Room {
	return &Room{
		matchUUID:  matchUUID,
		masterUUID: masterUUID,
		state:      RoomStateLobby,
		clients:    make(map[uuid.UUID]*Client),
		pieces:     make(map[string]PieceMovedPayload),
		walls:      make(map[string]mapentity.WallSegment),
		grid:       mapentity.DefaultGrid(), // default; overridden by map_state_sync
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
		deps:       deps,
		barsSeq:    uint64(time.Now().UnixMicro()),
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
//
// It seeds NO persisted fog memory itself: this runs BEFORE Register, on purpose (see the
// comment at its one call site in handler.go), which is BEFORE loadBoard has ever run for
// this fresh Room — so the real mapUUID a memory needs (spec §4.3) is not known here yet.
// loadBoard's own session-branch, moments later on the SAME goroutine's path through Run's
// register case, is what loads the persisted rows and calls SyncPlayerMemories again with
// them, once the map is known. SyncPlayerMemories(nil, ...) below is a safe placeholder in
// the meantime — nothing observable can happen between the two calls, since no message is
// processed until Register completes.
func (r *Room) RehydrateSession(session *matchsession.MatchSession) {
	r.mu.Lock()
	if r.session != nil {
		r.mu.Unlock()
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
	r.mu.Unlock()

	// The active pair is usually rows already — FindActiveSession rebuilt the session from them.
	// It is not when the match had none to find (started before B15, or the start-time write
	// failed): the session then opened a fresh pair, which becomes rows now, at birth (§4.5).
	r.ensureActiveSceneAndRound("rehydrate")
}

func (r *Room) IsMaster(userUUID uuid.UUID) bool {
	return r.masterUUID == userUUID
}

func (r *Room) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Register hands the client to Run's own goroutine. It returns ErrRoomClosed, instead of
// blocking forever, when Run has already returned (B7, spec §4.6) — the caller (handler.go)
// retries against a fresh room (master) or refuses the connection (player), exactly as if
// the room had never existed.
func (r *Room) Register(client *Client) error {
	select {
	case r.register <- client:
		return nil
	case <-r.done:
		return ErrRoomClosed
	}
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
	// B7 (spec §4.6): done is closed exactly once Run has genuinely stopped servicing this
	// room's channels — every select alongside it (Register, ReadPump's unregister send) can
	// then stop waiting instead of blocking on a goroutine that is never coming back.
	defer close(r.done)
	for {
		select {
		case client := <-r.register:
			r.mu.Lock()
			old, hadOld := r.clients[client.userUUID]
			r.mu.Unlock()

			// B4 (spec §4.6): the same account connecting again — a second tab, a reload that
			// raced its own close — means the LAST connection wins. The old one is told and
			// closed BEFORE the new one is written into r.clients. Neither the send nor the
			// close runs with r.mu held: SendMessage/Close never touch the room's state, and
			// nothing that sends to a client runs under this lock (see the file's own rule).
			// Close() signals `done`, never closes `send` — the old client's own WritePump
			// flushes this error out before it tears the connection down, and the unregister
			// its ReadPump sends afterwards is a no-op: by then r.clients[userUUID] already
			// points at the NEW client, so the pointer guard below skips the removal instead
			// of evicting the connection that replaced it (review focus 5).
			if hadOld && old != client {
				old.SendMessage(NewErrorMessage(
					"connection_replaced", "this account connected again elsewhere",
				))
				old.Close()
			}

			r.mu.Lock()
			r.clients[client.userUUID] = client
			client.SetRoom(r)
			isMaster := r.IsMaster(client.userUUID)
			// The board loads at the room's birth and, while still a lobby, on every master
			// reconnect — exactly the moment map_state_sync used to arrive in production
			// (spec §4.3, "Quem carrega", B14). r.session == nil is "still a lobby" (or a
			// master rehydrating one that has not reached this branch yet); !r.boardLoaded
			// narrows the OTHER case — session already live, because handler.go rehydrated
			// it before Register — to a single load, at birth.
			shouldLoadBoard := isMaster && (r.session == nil || !r.boardLoaded)
			r.mu.Unlock()

			var boardCleared bool
			if shouldLoadBoard {
				boardCleared = r.loadBoard(context.Background())
				// r.pieces/r.walls just changed under whoever else was already at the table —
				// unlike the registering client, handled below, nobody re-reads the board for
				// them on its own. Re-push a fresh, fog-filtered map_full_state to every OTHER
				// connected client, the same repush map_state_sync's old arm used to do after
				// a seed (spec §4.3). dispatchPerPlayer releases r.mu before calling back into
				// buildMapFullState, so this is safe to call with no lock held here.
				r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
					if pid == client.userUUID {
						return nil // this client's own map_full_state is the hasPieces branch below
					}
					return r.buildMapFullState(pid, isMaster)
				})
			}

			r.sendRoomState(client)
			r.mu.RLock()
			// A board can now carry walls with no piece on it at all (spec §4.3), so a
			// wall-only board must still reach a connecting client instead of waiting for the
			// old pieces-only condition to go true by luck.
			hasPieces := len(r.pieces) > 0 || len(r.walls) > 0
			r.mu.RUnlock()
			// A lobby board just cleared (its map was detached) is news too: this client may
			// still be drawing the old one, and an empty board never passes hasPieces.
			if hasPieces || boardCleared {
				msg := r.buildMapFullState(client.userUUID, r.IsMaster(client.userUUID))
				client.SendMessage(*msg)
			}
			// map_full_state above covers the board and nothing else. A client that connects
			// or reconnects mid-combat still needs the bars, the round regime, the open turn
			// and — if they are the master — its resolution, or they sit blind until something
			// changes by luck. No r.mu is held here: buildMatchFullState takes it itself.
			if msg := r.buildMatchFullState(client.userUUID, r.IsMaster(client.userUUID)); msg != nil {
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
				dropped := r.describeDroppedTurnLocked()
				r.mu.Unlock()
				logDroppedTurn(r.matchUUID, dropped)
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
			dropped := r.describeDroppedTurnLocked()
			r.mu.Unlock()
			logDroppedTurn(r.matchUUID, dropped)
			return
		}
	}
}

// loadBoard reads the match's board from the database and replaces the room's in-memory
// board with it — this is what the master's map_state_sync used to seed by hand (spec §4.3,
// "Quem carrega", B14). Called from Run's register branch: once when the Room is born, and
// again on every master reconnect while the room is still a lobby (see boardLoaded's own doc
// comment on the Room struct). StartMatch also calls it, from the master's read pump, right
// before the start's own save: the map can change over REST while the lobby socket is open.
//
// r.deps.LoadBoardUC == nil means the room has no board capability — every test built before
// B14 leaves it unset, and this is a no-op for them, same as every other optional dependency.
//
// Runs on Run's own goroutine, OUTSIDE r.mu for the DB read (a network round trip must never
// hold the room's lock), then takes it once to replace the board — the same
// read-outside/write-inside shape StartMatch and RehydrateSession already use for the
// session's own board sync.
//
// Returns true when it CLEARED a lobby's board: the match has no map attached any more (it
// was detached over REST while the lobby was open), so the detached map's board must go — or
// the lobby keeps showing it, the session is built on its walls, and start_match writes a row
// for a map the match no longer has. The caller pushes the now-empty board to whoever needs it.
func (r *Room) loadBoard(ctx context.Context) bool {
	if r.deps.LoadBoardUC == nil {
		return false
	}
	board, err := r.deps.LoadBoardUC.Load(ctx, r.matchUUID)
	if err != nil {
		log.Printf("load match board for match %s: %v", r.matchUUID, err)
		return false
	}

	// Whether there is already a live session to seed memories for is read under its OWN
	// lock, before the (possible) memory read below — a DB round trip must never run inside
	// the critical section that applies the board, the same reason LoadBoardUC.Load itself
	// ran above, outside any lock. This is the ONLY place loadBoard's session-branch ever
	// really fires for a session that did not just come from THIS call (see boardLoaded's own
	// doc comment): a session set by RehydrateSession, moments earlier and with no mapUUID to
	// seed memories from yet (spec §4.3; RehydrateSession itself seeds none — this is where a
	// rehydrated session's persisted fog memory actually arrives).
	r.mu.RLock()
	hasSession := r.session != nil
	r.mu.RUnlock()

	var mems []fogentity.PlayerMemory
	if board != nil && hasSession && r.deps.MemoryLoader != nil {
		m, merr := r.deps.MemoryLoader.FindByMatchMap(ctx, r.matchUUID, board.MapUUID)
		if merr != nil {
			log.Printf("load match board: load player memories for match %s map %s: %v",
				r.matchUUID, board.MapUUID, merr)
		} else {
			mems = m
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if board == nil {
		// No map attached at all — nothing to load. A lobby that attaches a map later is
		// picked up by the NEXT master connect (or start_match), since boardLoaded only gates
		// the post-start case. And boardLoaded stays as it was: a load that installed nothing
		// is not the room's one load, or a playing room would never pick up a board that shows
		// up later.
		//
		// A LOBBY that still holds a board here had its map detached: the board is cleared,
		// mapUUID with it (so persistBoard is a no-op and start_match writes no row). A live
		// session's board is left alone — attach/detach are refused once the match started.
		if r.session == nil && (r.mapUUID != uuid.Nil || len(r.pieces) > 0 || len(r.walls) > 0) {
			r.pieces = map[string]PieceMovedPayload{}
			r.walls = map[string]mapentity.WallSegment{}
			r.grid = mapentity.GridShape{}
			r.bg = nil
			r.mapUUID = uuid.Nil
			return true
		}
		return false
	}
	r.boardLoaded = true

	pieces := make(map[string]PieceMovedPayload, len(board.Pieces))
	for _, p := range board.Pieces {
		payload, perr := pieceToPayload(p)
		if perr != nil {
			// A piece that will not convert is logged and skipped, not a reason to fail
			// loading the rest of the board.
			log.Printf("load match board for match %s: skipping piece %s: %v", r.matchUUID, p.ID, perr)
			continue
		}
		pieces[payload.PieceID] = payload
	}
	walls := make(map[string]mapentity.WallSegment, len(board.Walls))
	for _, w := range board.Walls {
		walls[w.ID] = w
	}
	r.pieces = pieces
	r.walls = walls
	r.grid = board.Grid
	r.mapUUID = board.MapUUID
	r.bg = board.Bg

	if r.session != nil {
		wallSlice := append([]mapentity.WallSegment(nil), board.Walls...)
		r.session.SyncMapState(wallSlice, board.Grid)
		r.session.SetMapUUID(r.mapUUID)
		// Replaces whatever RehydrateSession seeded (none, today — see its own doc comment)
		// with the persisted rows, now that the real mapUUID is known. A restart's fog memory
		// comes back HERE, not in RehydrateSession itself (spec §4.3, §5).
		r.session.SyncPlayerMemories(mems, fogentity.FogModeExplored)
		// The board just changed, so every player's cached LOS is stale — the same
		// recompute map_state_sync's arm used to do after seeding.
		for _, pid := range r.session.PlayerIDs() {
			if _, verr := r.session.RecomputeVisibility(pid); verr != nil {
				log.Printf("load match board recompute visibility for %s: %v", pid, verr)
			}
		}
	}
	return false
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
	if err := r.deps.StartMatchUC.Start(ctx, r.matchUUID, userUUID); err != nil {
		return err
	}

	// Reloaded first, while still a lobby: the map attached to the match can have changed over
	// REST (attach/inherit/detach) after the master's socket opened, and the room's in-memory
	// board would otherwise be written over the new map's row (or a row written for a map the
	// match no longer has) and start the match on the old map. Every lobby move already
	// persisted, so reloading costs the table nothing it had. This is also what sets r.mapUUID
	// for the memory read below.
	//
	// The lobby's board — whatever pieces and walls sit on it right now — is then written
	// BEFORE Init, so B11's NPC-enrollment (T9) reads what is actually on screen instead of
	// whatever the last save happened to catch (spec §4.3, "B11", "Quando persiste").
	//
	// persistMu is held from the reload's read through that save: a lobby move whose own save
	// is still in flight would otherwise be read back as the board from BEFORE it, installed,
	// and saved over the move. Lock order is persistMu, then r.mu (loadBoard and
	// persistBoardLocked take r.mu themselves).
	r.persistMu.Lock()
	r.loadBoard(ctx)
	r.persistBoardLocked("start_match", false)
	r.persistMu.Unlock()

	session, err := r.deps.InitSessionUC.Init(ctx, r.matchUUID)
	if err != nil {
		return err
	}

	// r.mapUUID is known by now: loadBoard just ran above, against the map attached NOW. Read
	// outside the lock, same shape every DB-round-trip-before-lock in this file uses.
	r.mu.RLock()
	mapUUID := r.mapUUID
	r.mu.RUnlock()
	var mems []fogentity.PlayerMemory
	if mapUUID != uuid.Nil && r.deps.MemoryLoader != nil {
		m, merr := r.deps.MemoryLoader.FindByMatchMap(ctx, r.matchUUID, mapUUID)
		if merr != nil {
			log.Printf("start_match: load player memories for match %s map %s: %v", r.matchUUID, mapUUID, merr)
		} else {
			mems = m
		}
	}

	r.mu.Lock()
	r.session = session
	wallSlice := make([]mapentity.WallSegment, 0, len(r.walls))
	for _, w := range r.walls {
		wallSlice = append(wallSlice, w)
	}
	r.session.SyncMapState(wallSlice, r.grid)
	r.session.SetPieceSource(r)
	r.session.SetMapUUID(mapUUID)
	// fogMode fixo em explored: PENDENTE de configurações de partida.
	// FogMode é real e usado (filter_map_state.go decide memória de parede por ele), mas
	// o valor persistido em maps.fog_mode nunca chega aqui porque não existe ainda o
	// mecanismo de configuração de campanha/partida no backend — fog_mode será uma opção
	// que o mestre escolhe ao criar/editar a partida. Quando esse mecanismo existir, ler o
	// modo da configuração da partida e passar aqui. NÃO remover FogMode achando que é
	// código morto: isso eliminaria o modo `live` do produto.
	// Ver: System_X_System_React/docs/superpowers/specs/2026-08-06-tactical-map-refactor-design.md §3
	r.session.SyncPlayerMemories(mems, fogentity.FogModeExplored)
	playerIDs := r.session.PlayerIDs()
	r.state = RoomStatePlaying
	r.mu.Unlock()

	// The scene and round the match starts on are rows from this moment, not from their first
	// closed turn (B15, spec §4.5) — a match whose first scene closes without a turn still has it.
	r.ensureActiveSceneAndRound("start_match")

	for _, pid := range playerIDs {
		r.mu.Lock()
		_, err := r.session.RecomputeVisibility(pid)
		r.mu.Unlock()
		if err != nil {
			log.Printf("recompute visibility for %s: %v", pid, err)
		}
		// Persistence of whatever memory this recompute just grew happens through
		// persistBoard() at the next definitive board change (a turn closing, a wall
		// interaction, a master action) — not here. The playerMemoryRepo.Upsert TODO that
		// used to sit on this line is resolved by that path, not by adding a call here.
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

	if err := r.deps.KickPlayerUC.Kick(context.Background(), r.matchUUID, playerUUID, masterUUID); err != nil {
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
		// The write lock is held ACROSS Execute, not just around reading the pointer.
		// MatchSession has no lock of its own — r.mu is the only thing serializing it — and
		// this path mutates a lot of it: it closes a turn, resolves it, applies damage to
		// the target sheets, pops the priority queue and resolves the newly opened turn.
		// A player enqueueing an action at the same moment writes to that same queue.
		r.mu.Lock()
		session := r.session
		var result *appmatch.OpenNextActionResult
		var err error
		if session != nil {
			result, err = r.deps.OpenNextActionUC.Execute(context.Background(), session, r.masterUUID, client.userUUID)
		}
		r.mu.Unlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}

		// The closed half of the transition is handled BEFORE the error is reported, even when
		// Execute also failed: OpenNextActionUC.Execute only returns a non-nil result alongside
		// a non-nil error when the previous turn already closed and its damage already applied
		// (tr.Closed != nil) before the next one failed to open. That closed turn still needs
		// PersistTurnClose and its settled resolution_updated — losing them here would silently
		// drop a real turn from the table and from the Action History. The master still learns
		// the open failed, just after the table has the turn that actually ended.
		if result != nil {
			r.broadcastBars(session)

			if result.ClosedTurn != nil {
				closedTurn := result.ClosedTurn
				// Opening the next action is ALSO a close, so the escapes whose step waited for
				// this moment are walked here too — an escape whose outcome depended on which
				// verb the master happened to use would be a bug, not a rule. Before
				// persistClosedTurn (a DB round trip) and before announceOpenedTurn, so the
				// table sees the turn that ended finish moving before the next one starts.
				r.applyClosedEscapes(closedTurn, result.ClosedResolution)
				// The board — the escape's piece included — is written by persistClosedTurn, in the
				// turn's own transaction (spec §4.3, "Quando persiste"; owner decision 2026-10-01):
				// the board on disk and the turns on disk never disagree. Before announceOpenedTurn
				// below, so the next turn's opened move is not in this turn's board.
				// result.ClosedRound is set when this same call also ended the round — no action
				// left that can pay its price: the session's active round is then already its
				// successor, the turn belongs to the round it closed in, and the round's end and
				// the successor's birth go in the turn's transaction.
				r.persistClosedTurn(session, closedTurn, result.ClosedResolution, result.ClosedRound, result.Damaged)
				// The HP the close applied, to the master and to each damaged sheet's owner.
				// After the write, so nobody is told a number the database does not hold yet,
				// and before turn_closed: both are on the direct per-client lane now
				// (dispatchPerPlayer, since B2) and sent by this same goroutine, so send
				// order IS arrival order — sending this one first is what keeps "the bar
				// moved" from landing after "the turn ended".
				r.broadcastHpChanges(result.Damaged)
				// The implicit close is announced exactly like the explicit one, in the same
				// place in the sequence close_turn puts it: after the escapes and the write,
				// before the settled resolution, and above all before the announceOpenedTurn
				// below — the table has to see this turn end before the next one begins.
				r.broadcastTurnClosed(closedTurn.GetID())
				// The settled resolution of the turn that just ended — this is the one whose
				// damage was actually applied.
				if result.ClosedResolution != nil {
					r.publishResolution(closedTurn.GetID(), result.ClosedResolution)
				}
			}
		}

		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}

		// The round ended: no action pending could still pay its price, so it closed instead of
		// opening anything. Everyone is told — the regime and the bars are table state.
		if result.ClosedRound != nil {
			// Written before the table is told. Its end and the round born in its place (a row
			// from birth, B15) go together: when a turn closed in this same command,
			// persistClosedTurn above already wrote both in that turn's transaction; when none
			// did, they get a transaction of their own. Never one without the other.
			if result.ClosedTurn == nil {
				r.persistRoundClose(session, result.ClosedRound)
			}
			out := NewServerMessage(MsgTypeRoundClosed, RoundClosedPayload{
				RoundMode: string(result.ClosedRound.GetMode()),
			})
			data, _ := json.Marshal(out)
			go func() { r.broadcast <- data }()
			return
		}

		// Belt and braces: a successful call that opened nothing has nothing to announce.
		if result.OpenedTurn == nil {
			return
		}

		r.announceOpenedTurn(session, result.OpenedTurn, result.Resolution)

	case MsgTypeChangeRoundMode:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload ChangeRoundModePayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid change_round_mode payload"))
			return
		}
		// Write lock across Execute — the regime decides how every later selection is scored.
		// The regime the table is told is read back from the session under the same lock, not
		// echoed from the client's request: what is public is the round's actual state.
		//
		// The regime it had before, and the scene/round it was applied to, are read under that
		// same lock too: they are what the roundModeChanged event records (B15, spec §4.5).
		r.mu.Lock()
		session := r.session
		var err error
		var oldMode, newMode enum.RoundMode
		var modeScene *sceneentity.Scene
		var modeRound *roundentity.Round
		if session != nil {
			modeScene, modeRound = session.GetActiveScene(), session.GetActiveRound()
			oldMode = modeRound.GetMode()
			err = r.deps.ChangeRoundModeUC.Execute(
				context.Background(), session, r.masterUUID, client.userUUID,
				enum.RoundMode(payload.Mode),
			)
			newMode = session.GetActiveRound().GetMode()
			// Copies, taken after the switch: the record below writes them after the unlock.
			modeScene, modeRound = snapshotSceneAndRound(modeScene, modeRound)
		}
		r.mu.Unlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		// Recorded before the table is told, so the history never lags what the table saw.
		r.recordRoundModeChanged(session, modeScene, modeRound, oldMode, newMode)
		out := NewServerMessage(MsgTypeRoundModeChanged, RoundModeChangedPayload{Mode: string(newMode)})
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()
		r.broadcastBars(session)

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
		// Write lock across Execute — same reason as open_next_action.
		r.mu.Lock()
		session := r.session
		var result *appmatch.PullActionResult
		var err error
		if session != nil {
			result, err = r.deps.PullActionUC.Execute(context.Background(), session, r.masterUUID, client.userUUID, payload.ActionID)
		}
		r.mu.Unlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}

		// The closed half of the transition is handled BEFORE the error is reported, even when
		// Execute also failed — same reason as open_next_action: PullActionUC.Execute only
		// returns a non-nil result alongside a non-nil error when the previous turn already
		// closed and its damage already applied (tr.Closed != nil) before the pull itself
		// failed. That closed turn still needs PersistTurnClose and its settled
		// resolution_updated — losing them here would silently drop a real turn from the table
		// and from the Action History. The master still learns the pull failed, just after the
		// table has the turn that actually ended.
		if result != nil {
			r.broadcastBars(session)

			if result.ClosedTurn != nil {
				closedTurn := result.ClosedTurn
				// Opening the next action is ALSO a close, so the escapes whose step waited for
				// this moment are walked here too — an escape whose outcome depended on which
				// verb the master happened to use would be a bug, not a rule. Before
				// persistClosedTurn (a DB round trip) and before announceOpenedTurn, so the
				// table sees the turn that ended finish moving before the next one starts.
				r.applyClosedEscapes(closedTurn, result.ClosedResolution)
				// The board — the escape's piece included — is written by persistClosedTurn, in the
				// turn's own transaction (spec §4.3, "Quando persiste"; owner decision 2026-10-01):
				// the board on disk and the turns on disk never disagree. Before announceOpenedTurn
				// below, so the next turn's opened move is not in this turn's board.
				r.persistClosedTurn(session, closedTurn, result.ClosedResolution, nil, result.Damaged)
				// The HP the close applied, to the master and to each damaged sheet's owner.
				// After the write, so nobody is told a number the database does not hold yet,
				// and before turn_closed: both are on the direct per-client lane now
				// (dispatchPerPlayer, since B2) and sent by this same goroutine, so send
				// order IS arrival order — sending this one first is what keeps "the bar
				// moved" from landing after "the turn ended".
				r.broadcastHpChanges(result.Damaged)
				// The implicit close is announced exactly like the explicit one, in the same
				// place in the sequence close_turn puts it: after the escapes and the write,
				// before the settled resolution, and above all before the announceOpenedTurn
				// below — the table has to see this turn end before the next one begins.
				r.broadcastTurnClosed(closedTurn.GetID())
				// The settled resolution of the turn that just ended — this is the one whose
				// damage was actually applied.
				if result.ClosedResolution != nil {
					r.publishResolution(closedTurn.GetID(), result.ClosedResolution)
				}
			}
		}

		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}

		// Belt and braces: a successful call that opened nothing has nothing to announce.
		if result.OpenedTurn == nil {
			return
		}

		r.announceOpenedTurn(session, result.OpenedTurn, result.Resolution)

	case MsgTypeEnqueueAction:
		var payload ActionPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid action payload"))
			return
		}
		// The two travel together or not at all. reactToId says "this is a reaction"; the kind
		// says what it costs. One without the other is a client that is going to be surprised.
		if (payload.ReactToID != uuid.Nil) != (payload.ReactionKind != "") {
			client.SendMessage(NewErrorMessage("invalid_action",
				"a reaction needs both reactToId and reactionKind; an action needs neither"))
			return
		}
		if payload.ActorID == uuid.Nil {
			client.SendMessage(NewErrorMessage("invalid_action", "actorId is required: the acting character's sheet UUID"))
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
		a, err := buildAction(payload.ActorID, payload)
		if err != nil {
			client.SendMessage(NewErrorMessage("invalid_action", err.Error()))
			return
		}
		// Movement blocking: validate path against walls with move=true and !open.
		//
		// B6: the origin is never the client's — the server owns the board, so `from` is the
		// ACTOR'S OWN PIECE POSITION, read off r.pieceSlotOf, not payload.Move.From (already
		// discarded by buildAction). nil (no piece on the board) means there is nothing to
		// check: the sentinel-based skip this used to do for [0,0,0] — which made a piece
		// genuinely standing at (0,0) unblockable — is gone along with the sentinel itself.
		//
		// B5/B10: the wall check reads cell CENTERS via mapservice.SlotCenterToWorld with the
		// session's own grid, not `col × cellSize` (yesterday's bug: that lands on a cell's
		// top-left corner, which is wrong on a square grid and meaningless on a hex one).
		if a.Move != nil {
			r.mu.RLock()
			from, hasPiece := r.pieceSlotOf(a.GetActorID().String())
			a.Move.From = from
			if !hasPiece {
				r.mu.RUnlock()
			} else {
				grid := r.grid
				var walls []mapentity.WallSegment
				if r.session != nil {
					grid = r.session.GetGrid()
					walls = r.session.GetWalls()
				} else {
					walls = make([]mapentity.WallSegment, 0, len(r.walls))
					for _, w := range r.walls {
						walls = append(walls, w)
					}
				}
				r.mu.RUnlock()

				to := a.Move.Position
				fx, fy := mapservice.SlotCenterToWorld(from[0], from[1], grid)
				tx, ty := mapservice.SlotCenterToWorld(to[0], to[1], grid)
				if mapservice.IsPathBlocked([2]float64{fx, fy}, [2]float64{tx, ty}, walls) {
					client.SendMessage(NewErrorMessage("move_blocked", "movement blocked by a wall"))
					return
				}
			}
		}
		// Write lock across Execute: enqueueing pushes onto the priority queue AND rolls the
		// action's dice into it, both of which the master's open_next_action reads.
		//
		// The master's action_queued payload is built in the SAME critical section: once
		// Execute returns, `a` sits in the session's queue and belongs to the session, and
		// newActionQueuedPayload walks its pointer fields — after the unlock that read would
		// race whatever the master does to the queue next. Sent after the unlock, like
		// everything that reaches a client.
		r.mu.Lock()
		errEnqueue := r.deps.EnqueueActionUC.Execute(context.Background(), session, client.userUUID, a)
		var queued ActionQueuedPayload
		actionID := a.GetID()
		if errEnqueue == nil {
			queued = newActionQueuedPayload(a)
		}
		r.mu.Unlock()
		if errEnqueue != nil {
			client.SendMessage(NewErrorMessage("game_error", errEnqueue.Error()))
			return
		}
		client.SendMessage(NewServerMessage(MsgTypeActionEnqueued, ActionEnqueuedPayload{ActionID: actionID}))
		// The sender's own ack now names the action too — it says "we got it, and here is what
		// you can refer to it by". This is different news, for a different recipient: the
		// master is the one who has to decide when it opens, and they need the ID to be able
		// to pull it.
		r.sendToMaster(NewServerMessage(MsgTypeActionQueued, queued))
		r.broadcastBars(session)

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
		// The two travel together or not at all. reactToId says "this is a reaction"; the kind
		// says what it costs. One without the other is a client that is going to be surprised.
		if (payload.ReactToID != uuid.Nil) != (payload.ReactionKind != "") {
			client.SendMessage(NewErrorMessage("invalid_action",
				"a reaction needs both reactToId and reactionKind; an action needs neither"))
			return
		}
		if payload.ActorID == uuid.Nil {
			client.SendMessage(NewErrorMessage("invalid_action", "actorId is required: the acting character's sheet UUID"))
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

	case MsgTypeOpenReaction:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload OpenReactionPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid open_reaction payload"))
			return
		}
		r.mu.RLock()
		session := r.session
		r.mu.RUnlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		r.mu.Lock()
		result, err := r.deps.OpenReactionUC.Execute(context.Background(), session, client.userUUID, payload.ReactionID)
		turnID := session.CurrentTurnID()
		var opened action.Action
		if err == nil {
			// A copy, under the lock: result.Opened aliases the turn's own reaction.
			opened = *result.Opened
		}
		r.mu.Unlock()
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		// No escape displaces here, not even the Shift one that rolls nothing: every escape can
		// FAIL — it clears its test only if the movement AND the dodge both beat the attacker's
		// hit — and whether it did is only known when the turn closes. Opening one shows the
		// intention; applyClosedEscapes puts the piece where the close decides
		// (front-combat-phases.md §6A.5, B13).
		//
		// What each session player sees of the escape's destination — the verdict the dispatch
		// below applies — recorded for the history, held with the turn until it closes. A
		// separate lock section from each recipient's projection, as in announceOpenedTurn, and
		// for the same reason nothing can change the verdict in between: every message that
		// could comes from the master, on the read pump that is running this open.
		r.recordOpenedReactionMoveViews(turnID, opened)
		// Who narrates next, and with what, is public — cut per recipient, so on the direct lane;
		// the master's resolution_updated below goes after it on the same lane, from this same
		// goroutine, which is what makes "reaction_opened, then the recomputed resolution" a
		// promise rather than luck.
		r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
			// viewerFor AND the projection both run inside this RLock: opened is a copy of the
			// reaction's struct, but its pointer fields still point at live session memory
			// (see announceOpenedTurn).
			r.mu.RLock()
			defer r.mu.RUnlock()
			msg := NewServerMessage(MsgTypeReactionOpened, ReactionOpenedPayload{
				TurnID: turnID, ReactionID: payload.ReactionID,
				Reaction: r.reactionWireLocked(opened, pid, r.viewerFor(pid, isMaster)),
			})
			return &msg
		})
		r.publishResolution(turnID, result.Resolution)

	case MsgTypeEditAction:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload EditActionPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid edit_action payload"))
			return
		}
		ma, err := buildEditAction(payload)
		if err != nil {
			client.SendMessage(NewErrorMessage("invalid_action", err.Error()))
			return
		}
		r.mu.RLock()
		session := r.session
		r.mu.RUnlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		// Write lock across Execute: the edit mutates the action itself and re-derives the
		// open turn's resolution — the same surface open_next_action and close_turn mutate.
		r.mu.Lock()
		result, err := r.deps.EditActionUC.Execute(
			context.Background(), session, r.masterUUID, client.userUUID, ma, escapeLandingEditOf(payload))
		r.mu.Unlock()
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		client.SendMessage(NewServerMessage(MsgTypeActionEdited, ActionEditedPayload{
			TurnID: result.TurnID, ActionID: ma.ActionID,
		}))
		r.publishResolution(result.TurnID, result.Resolution)

	case MsgTypeCloseTurn:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload CloseTurnPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid close_turn payload"))
			return
		}
		// Write lock across Execute — closing resolves the turn, applies damage to the target
		// sheets and advances every ledger. Exactly the same surface open_next_action mutates.
		r.mu.Lock()
		session := r.session
		var result *appmatch.CloseTurnResult
		var err error
		var turnID uuid.UUID
		if session != nil {
			turnID = session.CurrentTurnID()
			result, err = r.deps.CloseTurnUC.Execute(
				context.Background(), session, r.masterUUID, client.userUUID, payload.Confirm)
		}
		r.mu.Unlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		if len(result.Refused) > 0 {
			pending := make([]PendingReactionPayload, 0, len(result.Refused))
			for i := range result.Refused {
				pending = append(pending, PendingReactionPayload{
					ReactionID: result.Refused[i].GetID(),
					ActorID:    result.Refused[i].GetActorID(),
					Kind:       string(result.Refused[i].ReactionKind),
				})
			}
			client.SendMessage(NewServerMessage(MsgTypeCloseTurnRefused,
				CloseTurnRefusedPayload{TurnID: turnID, PendingReactions: pending}))
			return
		}

		closedTurn := result.ClosedTurn
		// result.Resolution here is the SETTLED one — CloseTurnUC resolves the turn it just
		// closed, not a next one — the same resolution published below via publishResolution.
		//
		// The escapes that were waiting on their own test settle with it: this is the third
		// way a turn closes (open_next_action and pull_action are the other two) and all three
		// have to walk the piece, or an escape's outcome would depend on which verb the master
		// used. Before turn_closed goes out: piece_moved and turn_closed are both on the
		// direct per-client lane (dispatchPerPlayer) since B2, sent by this same goroutine, so
		// applying the escape's move first is what keeps the table from seeing the turn end
		// with the piece still in its old slot.
		r.applyClosedEscapes(closedTurn, result.Resolution)
		// Same as the two implicit closes: the board — the escape's piece included — is written
		// by persistClosedTurn, in the turn's own transaction (spec §4.3).
		r.persistClosedTurn(session, closedTurn, result.Resolution, nil, result.Damaged)

		// Same place in the sequence the two implicit closes put it: after the write, before
		// turn_closed. See the comment there for why the order matters.
		r.broadcastHpChanges(result.Damaged)
		r.broadcastTurnClosed(closedTurn.GetID())
		r.publishResolution(closedTurn.GetID(), result.Resolution)
		r.broadcastBars(session)

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
		// Validated against the enum's exact values (same pattern as skill/weapon names),
		// not just cast and stored: an unrecognized category used to sail through as a
		// string matching neither "battle" nor "roleplay", and would come back out that
		// way in scene_changed and match_full_state for every client to trip on.
		//
		// Before the lock on purpose: it reads nothing of the session, and validating under
		// the write lock would hold the whole room for a pure string check.
		category, err := enum.SceneCategoryFrom(payload.Category)
		if err != nil {
			client.SendMessage(NewErrorMessage("invalid_action", err.Error()))
			return
		}

		// The write lock is held ACROSS Execute, the same way open_next_action holds it and
		// for the same reason: ChangeScene closes the active scene and round and installs new
		// ones, and MatchSession has no lock of its own. This arm used to release the lock
		// before Execute and mutate the session unguarded — a real race against every reader
		// (buildMatchFullState serving a client that connects at that instant is the one the
		// race detector catches). The scene_changed payload is built in the SAME critical
		// section, because reading the new scene's fields after the unlock is the same race
		// one step later.
		r.mu.Lock()
		session := r.session
		var sceneWasPersisted bool
		var oldScene *sceneentity.Scene
		var oldRound *roundentity.Round
		var scenePayload SceneChangedPayload
		var newScene *sceneentity.Scene
		var newRound *roundentity.Round
		var oldEnd *appmatch.RoundEnd
		if session != nil {
			// Captured BEFORE ChangeScene resets it.
			sceneWasPersisted = session.IsScenePersisted()
			oldScene, oldRound, err = r.deps.ChangeSceneUC.Execute(
				context.Background(), session,
				r.masterUUID, client.userUUID,
				category, payload.BriefInitialDescription,
			)
			if err == nil {
				// Copies: they are written after the unlock (ensureSceneAndRoundRows).
				newScene, newRound = snapshotSceneAndRound(session.GetActiveScene(), session.GetActiveRound())
				if oldScene != nil && oldRound != nil && oldRound.GetFinishedAt() != nil {
					sc, rd := snapshotSceneAndRound(oldScene, oldRound)
					oldEnd = &appmatch.RoundEnd{Scene: sc, Round: rd}
					// The old pair was never a row (its birth write failed): there is nothing for
					// CloseSceneAndRound to close, but the pair happened. It becomes an unwritten
					// round end, and the new pair's write below carries it, closed, in its own
					// transaction (unwritten.go).
					if !sceneWasPersisted {
						r.markRoundEndUnwrittenLocked(*oldEnd)
					}
				}
				if activeScene := session.GetActiveScene(); activeScene != nil {
					scenePayload = SceneChangedPayload{
						SceneID:                 activeScene.GetID(),
						Category:                string(activeScene.GetCategory()),
						BriefInitialDescription: activeScene.BriefInitialDescription,
					}
				}
			}
		}
		r.mu.Unlock()

		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}

		// Since B15 every scene and round is a row from birth, so sceneWasPersisted is true unless
		// the birth-time write itself failed (logged there). The check stays: closing a row that
		// was never written would be an UPDATE of nothing, and the flag is what says which case
		// this is. The old pair is closed BEFORE the new one is written, so the database never
		// holds two open scenes for the match — FindActiveSession reads the open one on a restart.
		if sceneWasPersisted && oldEnd != nil {
			if dbErr := r.deps.RoundRepo.CloseSceneAndRound(
				context.Background(),
				oldEnd.Scene.GetID(), oldEnd.Round.GetID(), *oldEnd.Round.GetFinishedAt(),
			); dbErr != nil {
				// Left open on disk, the old pair would sit next to the new one. Not just logged:
				// it becomes an unwritten round end, and the new pair's write right below closes it
				// in the same transaction — or, if that fails too, the next write does.
				log.Printf("CloseSceneAndRound FAILED — scene %s / round %s of match %s left open (kept for the next write): %v",
					oldEnd.Scene.GetID(), oldEnd.Round.GetID(), r.matchUUID, dbErr)
				r.mu.Lock()
				r.markRoundEndUnwrittenLocked(*oldEnd)
				r.mu.Unlock()
			}
		}
		// The new scene and its round are rows from this moment (B15, spec §4.5) — with any
		// round end still unwritten, the old pair's included, in the same transaction.
		r.ensureSceneAndRoundRows(session, newScene, newRound, "change_scene")

		out := NewServerMessage(MsgTypeSceneChanged, scenePayload)
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()

	case MsgTypeCancelLobby:
		if err := r.CloseLobby(client.userUUID); err != nil {
			client.SendMessage(NewErrorMessage("forbidden", err.Error()))
		}

	case MsgTypePieceMoved:
		// Relay piece moves per-player with fog-of-war filtering. Lobby-only, and
		// server-side piece ownership IS validated as of B14 (spec §4.3, "Quem move o
		// quê") — handlePieceMoved refuses a session outright and, in the lobby, a
		// player who does not own the piece's current CharacterID. See its own comment.
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
		// OBSOLETE since B14 (spec §4.3, "Quem carrega"): the server now owns the board —
		// it loads it itself, in Run's register branch, at the moment this used to arrive.
		// Still accepted from the master and answered with the current map_full_state, so the
		// front that has not yet dropped this send (F13) gets a reply instead of silence, but
		// it WRITES NOTHING. Remove this arm once F13 ships.
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload MapStateSyncPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid map_state_sync payload"))
			return
		}
		client.SendMessage(*r.buildMapFullState(client.userUUID, true))

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
		// MasterActionPayload carries no Attack field (B9, spec §2): a plain Unmarshal above
		// would just drop an "attack" key silently, which is exactly what must not happen — the
		// master has a way to attack, through an NPC with enqueue_action, and a client still
		// sending "attack" here needs to be told so, not ignored. Decoded separately, straight
		// off the raw JSON, so the refusal does not depend on MasterActionPayload ever having
		// had the field.
		var attackProbe struct {
			Attack json.RawMessage `json:"attack"`
		}
		_ = json.Unmarshal(incoming.Payload, &attackProbe)
		if attackProbe.Attack != nil {
			client.SendMessage(NewErrorMessage("invalid_action",
				"the master attacks through an NPC with enqueue_action"))
			return
		}
		r.mu.RLock()
		session := r.session
		r.mu.RUnlock()
		ma := buildMasterAction(client.userUUID, payload)
		// Every branch below that ACCEPTS the master action records it, the instant it is
		// applied, with what each player saw of it (spec §4.8, recordMasterAction). A refused one
		// is never recorded. edit_action has its own arm and never reaches here. With a turn
		// open, the record and the board save both wait for that turn's close and are written
		// with it (recordMasterAction, persistBoardOutsideTurn; owner decision, 2026-10-01).
		//
		// Piece: the master's drag, place or take-off (spec §4.3, "Master action de peça"). Before
		// the Interact branch: a piece action names a character, not a wall.
		if ma.Move != nil || payload.Remove != nil {
			r.applyMasterPieceAction(client, ma, payload.Remove != nil)
			return
		}
		// Reveal: master-only secret-door reveal. Marks the wall revealed in the session
		// and broadcasts the full real WallSegment to ALL clients.
		if ma.Interact != nil && ma.Interact.Kind == action.InteractReveal && len(ma.TargetID) > 0 {
			revealed, views := r.revealSecretDoors(client, ma.TargetID)
			r.persistBoardOutsideTurn("wall_interact")
			if len(revealed) > 0 {
				r.recordMasterAction(masteraction.KindRevealWall,
					wallActionContent{WallIDs: revealed, Interact: string(ma.Interact.Kind)}, views)
			}
			return
		}
		// Wall interaction: handled in-memory + broadcast; does not go through the use case queue.
		if ma.Interact != nil && len(ma.TargetID) > 0 {
			// One record per changed wall, each with ITS OWN views: a batch can mix an ordinary
			// door the table saw with an unrevealed secret door nobody but the master saw, and a
			// single record listing both under the union of their views would hand the secret
			// door's id to every player who saw the ordinary one, through GET /history.
			type wallRecord struct {
				wallID string
				views  map[uuid.UUID]masteraction.View
			}
			var changed []wallRecord
			for _, targetID := range ma.TargetID {
				wallID := targetID.String()
				r.mu.RLock()
				_, exists := r.walls[wallID]
				r.mu.RUnlock()
				if !exists {
					// Last defense (spec §4.3): a wall the server does not know about answers
					// unknown_wall to the master instead of the silent skip this used to be.
					client.SendMessage(NewErrorMessage("unknown_wall",
						fmt.Sprintf("wall %s is not on this match's board", wallID)))
					continue
				}
				newOpen, newLocked, ok := r.applyWallInteract(wallID, ma.Interact)
				if !ok {
					// The wall exists but the interact kind does not apply to it (lockpick,
					// examine — require a roll check, not yet handled). Not "unknown": the
					// server DOES know this wall, it just cannot act on this kind yet.
					continue
				}
				changed = append(changed, wallRecord{
					wallID: wallID,
					views:  r.broadcastWallStateChangedGated(wallID, newOpen, newLocked),
				})
			}
			// Wall geometry may have changed (open/close) → recompute and push LOS.
			r.pushVisibilityUpdates()
			r.persistBoardOutsideTurn("wall_interact")
			for _, c := range changed {
				r.recordMasterAction(masteraction.KindWallInteract,
					wallActionContent{WallIDs: []string{c.wallID}, Interact: string(ma.Interact.Kind)}, c.views)
			}
			return
		}
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		if err := r.deps.EnqueueMasterActionUC.Execute(context.Background(), session, r.masterUUID, client.userUUID, ma); err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		// A turn note: it hangs on the open turn and emits nothing to the table as an action, so
		// nobody but the master saw it — no views (spec §4.8). There is always a turn open here
		// (EnqueueMasterAction refuses otherwise), so it is written with that turn's close.
		r.recordMasterAction(masteraction.KindTurnNote, payload, map[uuid.UUID]masteraction.View{})
		out := NewServerMessage(MsgTypeMasterActionEnqueued, MasterActionEnqueuedPayload(payload))
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()

	case MsgTypeAddNPC:
		// Puts an NPC into the match mid-game. It lives ALONGSIDE the REST POST /npcs, it does
		// not replace it: REST builds the roster before there is a room, this verb is for the
		// middle of one. It does not call the REST either — it runs the very same AddMatchNPCUC
		// (same guards, same match_participants write) and then injects the sheet into the live
		// session, so the database and the memory cannot drift apart over a second step the
		// front might forget.
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload AddNPCPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil || payload.CharacterSheetUUID == uuid.Nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid add_npc payload"))
			return
		}
		r.enrollLiveNPC(client, payload.CharacterSheetUUID)

	default:
		client.SendMessage(NewErrorMessage("unknown_type", "unrecognized message type"))
	}
}

// broadcastTurnClosed tells the table a turn ended. Three verbs close one: close_turn says so
// outright, and open_next_action and pull_action close the open turn on their way through.
// All three reach here, because which verb the master happened to use must not change what
// the table is told — the implicit close used to be silent, and an event list that shows
// turns beginning and never ending is the front's problem to reconcile, not the back's to
// create.
//
// DIRECT lane (dispatchPerPlayer), not r.broadcast, since B2 (design spec §4.2): the very
// next thing every caller sends is announceOpenedTurn's turn_opened, which is now projected
// PER RECIPIENT and therefore has to live on this same lane. Sending both from here, in this
// order, on the SAME lane, from the SAME goroutine, is what makes "turn_closed arrives before
// turn_opened" a guarantee instead of a usual case: with one lane and one sender, send order
// IS arrival order — there is no channel hop in between for a later, faster send to overtake.
// Before B2 this queued onto r.broadcast synchronously, ahead of turn_opened's own detached
// `go func` — safe only because both eventually rode the SAME channel; the day turn_opened
// needed per-recipient projection, dispatchPerPlayer was the only way to build it, and leaving
// this one behind on the channel would have let a fast direct-lane turn_opened overtake a
// turn_closed still waiting on Run()'s goroutine to pick it up.
//
// The payload is IDENTICAL for every recipient — nothing here is projected — so the builder
// below ignores both of dispatchPerPlayer's arguments.
func (r *Room) broadcastTurnClosed(turnID uuid.UUID) {
	out := NewServerMessage(MsgTypeTurnClosed, TurnClosedPayload{TurnID: turnID})
	r.dispatchPerPlayer(func(_ uuid.UUID, _ bool) *Message {
		msg := out
		return &msg
	})
}

// broadcastHpChanges tells the master and each damaged character's owner what the close just
// wrote to their sheet. The same three verbs that close a turn reach here, for the reason
// broadcastTurnClosed gives: which verb the master used must not change what anyone is told.
//
// It is called from the CLOSING path and not from turn_closed itself because damage is what
// the close APPLIED, and the applied numbers live in the *Result the arm already holds —
// turn_closed carries an id and nothing else. Healing and poison will move HP without any
// turn closing at all; when they do, they call this same helper with their own list.
//
// PROJECTED, not broadcast, via dispatchPerPlayer — the mechanism the fog and the settled
// resolution_updated already use. Do not grow a second one. The owner comes out of
// charToPlayer; an NPC has no entry there, so the master is the only recipient.
//
// Reading the maximum off the live sheet is session state, so it happens under r.mu — and
// ONLY the reading does. The payloads are finished before the lock is released and every
// send happens after it, because dispatchPerPlayer takes r.mu itself.
func (r *Room) broadcastHpChanges(damaged []matchsession.DamagedCharacter) {
	if len(damaged) == 0 {
		return
	}

	type projected struct {
		payload CharacterHpChangedPayload
		owner   uuid.UUID
	}

	r.mu.RLock()
	charToPlayer := map[string]uuid.UUID{}
	if r.session != nil {
		charToPlayer = r.session.GetCharToPlayer()
	}
	changes := make([]projected, 0, len(damaged))
	for _, d := range damaged {
		if d.Sheet == nil {
			continue
		}
		// The maximum comes off the SAME bar NewHP came off — a maximum read anywhere else
		// could disagree with the current value in the very payload that carries both.
		bar, ok := d.Sheet.GetAllStatusBar()[enum.Health]
		if !ok {
			continue
		}
		changes = append(changes, projected{
			payload: CharacterHpChangedPayload{
				CharacterID: d.CharacterID,
				HP:          d.NewHP,
				MaxHP:       bar.GetMax(),
				Damage:      d.Damage,
			},
			// An NPC IS in charToPlayer — it maps to the MASTER (indexParticipants, and
			// AddNPC for one that joined live), so owner is the master and the two arms
			// below name the same person. dispatchPerPlayer visits each client once, so the
			// master still gets exactly one copy. A character in no map at all reads as
			// uuid.Nil, which is nobody's player ID: the owner arm never matches it.
			owner: charToPlayer[d.CharacterID.String()],
		})
	}
	r.mu.RUnlock()

	for _, c := range changes {
		r.dispatchPerPlayer(func(playerID uuid.UUID, isMaster bool) *Message {
			if !isMaster && playerID != c.owner {
				return nil
			}
			msg := NewServerMessage(MsgTypeCharacterHpChanged, c.payload)
			return &msg
		})
	}
}

// turnActionWireLocked projects act to the wire cut this playerID/isMaster is entitled to AT
// TURN-OPEN TIME (isSettled=false, always — design spec §4.2, B2): the master keeps every
// number (actionwire.Full); everyone else — the actor's own owner included — gets the
// mechanics with the numbers cut (actionwire.Opened), only after service.ProjectAction's
// deny-list has already run (feint/trigger hidden from a third party while the turn stays
// open, closed reactions' label demoted). From carries no deny-list of its own — see its own
// doc — so ProjectAction has to run first, every time, for every non-master recipient.
//
// Then the move's WHERE, which no Level decides: move.from and move.position are the actor's
// piece leaving one cell and entering another, so a non-master who does not own the actor
// gets them only as far as the piece's own fog gate lets them (gatePieceMoveLocked, the
// decision the live relay makes) — the whole move, the origin alone, or neither. category
// stays: that the actor moves is public mechanics, where is not (owner decision 2026-10-01).
//
// origin is the cell the gate judges the piece as LEAVING. On the live turn_opened it is where
// the piece actually stood when the turn opened (applyOpenedMove) — the very origin the relay
// judged — because move.from is read at ENQUEUE (B6) and is stale once the piece moved in
// between (a master drag, an earlier queued move of the same character). On a reconnect there
// is no record of that slot, so it is move.from itself. And move.from only travels when it IS
// that origin: an "origin only" verdict on a cell the stale move.from does not name would
// otherwise hand out a slot the relay never showed this player.
//
// Shared by announceOpenedTurn (live turn_opened) and buildMatchFullState (OpenTurn on a
// reconnect's match_full_state), so a reconnecting client's snapshot can never disagree with
// the live event they may already have received — save for that one stale-origin window.
//
// The caller MUST hold r.mu (a read lock is enough): the gate reads r.pieces and the session's
// grid and visibility cache, and act's pointer fields point at live session memory (see
// announceOpenedTurn).
func (r *Room) turnActionWireLocked(act action.Action, pid uuid.UUID, v domainservice.Viewer, origin *[3]int) actionwire.Action {
	if v.IsMaster {
		return actionwire.From(act, actionwire.Full)
	}
	out := actionwire.From(domainservice.ProjectAction(act, v, false), actionwire.Opened)
	if out.Move == nil || v.SeesAllOf(out.ActorID) {
		return out
	}
	switch view, ok := r.openedMoveViewLocked(pid, out.ActorID, out.Move.From, out.Move.Position, origin); {
	case !ok:
		out.Move.From, out.Move.Position = nil, nil
	case view == masteraction.ViewLeft:
		out.Move.Position = nil
	}
	return out
}

// reactionWireLocked cuts an opened reaction for one recipient — the same rule as the turn's
// action (turnActionWireLocked), with the gate's origin at the reactor's piece NOW: a reaction
// never moves at the opening (an escape waits for the close, B13), so where the piece stands is
// both where it stood when the master opened it and what a reconnect can still read. A reaction
// carries no move.from (buildAction never derives one), so an "origin only" verdict leaves it
// with the category alone — position only reaches who sees the destination.
//
// Shared by the live reaction_opened and match_full_state's openTurn.reactions. The caller
// MUST hold r.mu (a read lock is enough).
func (r *Room) reactionWireLocked(react action.Action, pid uuid.UUID, v domainservice.Viewer) actionwire.Action {
	origin, _ := r.pieceSlotOf(react.GetActorID().String())
	return r.turnActionWireLocked(react, pid, v, origin)
}

// openedMoveViewLocked is what recipient pid may see of an opened move's WHERE: full (from and
// position), left (from only), or nothing (false). It is the ONE decision behind both the
// live turn_opened/openTurn (turnActionWireLocked) and the verdict the history records for
// the move (recordOpenedMoveViews), so GET /history can only ever show a reader what this
// told them live (owner decision, 2026-10-01).
//
// The piece's fog gate (gatePieceMoveLocked) judges the move from origin, and a "left" verdict
// carries move.from only when move.from IS that origin — otherwise it would hand out a slot the
// relay never showed this player, so it is nothing instead (see turnActionWireLocked).
//
// The caller MUST hold r.mu (a read lock is enough).
func (r *Room) openedMoveViewLocked(pid, actorID uuid.UUID, from, to, origin *[3]int) (masteraction.View, bool) {
	view, ok := r.gatePieceMoveLocked(pid, actorID, origin, to)
	if !ok {
		return "", false
	}
	if view == masteraction.ViewLeft && (from == nil || origin == nil || *from != *origin) {
		return "", false
	}
	return view, true
}

// ownerOfLocked is the player who owns characterID in the session (an NPC maps to the master),
// uuid.Nil when nobody does. The caller MUST hold r.mu and r.session must not be nil.
func (r *Room) ownerOfLocked(characterID uuid.UUID) uuid.UUID {
	return r.session.GetCharToPlayer()[characterID.String()]
}

// recordOpenedMoveViews records, for the turn that just opened, what each session player saw
// of its move — openedMoveViewLocked from the same origin the live turn_opened is judged on —
// and holds it with the turn (turnWrites.moveViews), to be written by PersistTurnClose with the
// action (actions.move_views). The history projects the move by it: nothing is recomputed at
// read time, when the fog is no longer that moment's.
//
// Only while that turn is still the open one: a turn already closed by a racing close has been
// drained, and an entry written now would never be. The caller must NOT hold r.mu.
func (r *Room) recordOpenedMoveViews(opened *turnentity.Turn, origin *[3]int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	turnID := opened.GetID()
	if r.openTurnIDLocked() != turnID {
		return
	}
	// Under the lock: the copy's Move points at live session memory (announceOpenedTurn).
	act := opened.GetAction()
	if act.Move == nil {
		return
	}
	actorID, from, to := act.GetActorID(), act.Move.From, act.Move.Position
	r.turnWritesLocked(turnID).moveViews = r.sessionPlayerViewsLocked(r.ownerOfLocked(actorID), func(pid uuid.UUID) (masteraction.View, bool) {
		return r.openedMoveViewLocked(pid, actorID, from, &to, origin)
	})
}

// recordOpenedReactionMoveViews records what each session player saw of an opened reaction's
// destination — the verdict reactionWireLocked's gate reached for them — and holds it with the
// turn (turnWrites.reactionMoveViews), to be written with the reaction's row at the close.
// Only while that turn is still the open one, like recordOpenedMoveViews. The caller must NOT
// hold r.mu.
func (r *Room) recordOpenedReactionMoveViews(turnID uuid.UUID, react action.Action) {
	if react.Move == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.openTurnIDLocked() != turnID {
		return
	}
	actorID, to := react.GetActorID(), react.Move.Position
	origin, _ := r.pieceSlotOf(actorID.String())
	views := r.sessionPlayerViewsLocked(r.ownerOfLocked(actorID), func(pid uuid.UUID) (masteraction.View, bool) {
		return r.openedMoveViewLocked(pid, actorID, nil, &to, origin)
	})
	w := r.turnWritesLocked(turnID)
	if w.reactionMoveViews == nil {
		w.reactionMoveViews = map[uuid.UUID]map[uuid.UUID]masteraction.View{}
	}
	w.reactionMoveViews[react.GetID()] = views
}

// moveFromOf is an action's enqueue-time move.from, nil when it has no Move or no origin.
func moveFromOf(a action.Action) *[3]int {
	if a.Move == nil {
		return nil
	}
	return a.Move.From
}

// gatePieceMoveLocked is pieceMoveView — the live relay's fog gate — for one character's piece
// going from `from` to `to`, as recipient pid sees it RIGHT NOW: their cached visibility
// polygons, the session's grid (cell centres, as the wall check reads them), and the piece's
// own visible flag. It is the one place the wire's move.from/position (turn_opened, openTurn)
// and a settled escape's landing ask "may this player know where the piece went", so those
// surfaces cannot drift from what piece_moved/piece_removed told the same player.
//
// nil from means no known origin (only the destination can be seen); nil to means no
// destination (only the origin can) — never cell (0,0).
//
// The piece is hidden — nothing for any player — when it is visible:false, and also when the
// character has NO piece on the board: applyMove then moves nothing and the relay tells
// nobody anything, so neither may this.
//
// No session (the lobby) means no fog: everything is visible, as relayPieceMove has it.
//
// The caller MUST hold r.mu (a read lock is enough): it reads r.pieces and session state.
func (r *Room) gatePieceMoveLocked(pid, characterID uuid.UUID, from, to *[3]int) (masteraction.View, bool) {
	if r.session == nil {
		return masteraction.ViewFull, true
	}
	pieceID := r.pieceOfLocked(characterID.String())
	if pieceID == "" {
		return "", false
	}
	if vis := r.pieces[pieceID].Visible; vis != nil && !*vis {
		return "", false
	}
	grid := r.session.GetGrid()
	polys := r.session.GetVisibility(pid)
	var oldPt domainservice.Point2D
	if from != nil {
		ox, oy := mapservice.SlotCenterToWorld(from[0], from[1], grid)
		oldPt = domainservice.Point2D{X: ox, Y: oy}
	}
	if to == nil {
		// No destination to see: only the origin can decide, and only as "left".
		if from != nil && domainservice.IsVisible(oldPt, polys) {
			return masteraction.ViewLeft, true
		}
		return "", false
	}
	nx, ny := mapservice.SlotCenterToWorld(to[0], to[1], grid)
	return pieceMoveView(polys, domainservice.Point2D{X: nx, Y: ny}, oldPt, from != nil, false)
}

// gateEscapeLandingsLocked withholds a settled escape's landing from a recipient who could not
// see the piece land there — the same news as a move's destination, so the same gate
// (gatePieceMoveLocked, with no origin: the escape payload names none). The master and the
// escaping character's owner keep it. Only the landing goes: the verdict (escaped, movePassed,
// dodgePassed, awaitsMaster) is public once settled.
//
// The caller MUST hold r.mu (a read lock is enough).
func (r *Room) gateEscapeLandingsLocked(p *ResolutionUpdatedPayload, pid uuid.UUID, v domainservice.Viewer) {
	for i := range p.Targets {
		tr := &p.Targets[i]
		if tr.Escape == nil || tr.Escape.Landing == nil || v.SeesAllOf(tr.TargetID) {
			continue
		}
		if _, ok := r.escapeLandingViewLocked(pid, tr.TargetID, tr.Escape.Landing); !ok {
			esc := *tr.Escape
			esc.Landing = nil
			tr.Escape = &esc
		}
	}
}

// escapeLandingViewLocked is whether recipient pid may see where an escaping character's piece
// ended — the ONE decision behind both the live settled resolution_updated
// (gateEscapeLandingsLocked) and the verdict the history records (escapeLandingViewsLocked).
// With no origin it is "full iff the destination is visible" — exactly when the relay of that
// move sent this player a piece_moved.
//
// The caller MUST hold r.mu (a read lock is enough).
func (r *Room) escapeLandingViewLocked(pid, characterID uuid.UUID, landing *[3]int) (masteraction.View, bool) {
	return r.gatePieceMoveLocked(pid, characterID, nil, landing)
}

// escapeLandingViewsLocked records, per escape of the closed turn that moved its piece, what
// each session player saw of WHERE it ended — keyed by the escaping character (TargetID). The
// destination is closedEscapeMove's, the one applyClosedEscapes applied: a failed escape's
// landing (the settled escape.landing) or an escaped one's own destination (its reaction's
// move.position, which the history shows only by this). Run by persistClosedTurn after
// applyClosedEscapes put the pieces down and before the settled resolution_updated is sent, on
// that same board, so it is the verdict the live gate and the relay reached. nil when no escape
// moved.
//
// The caller MUST hold r.mu (a read lock is enough). closed's reactions are frozen (the turn is
// finished), as applyClosedEscapes already relies on.
func (r *Room) escapeLandingViewsLocked(closed *turnentity.Turn, res *domainservice.TurnResolution) map[uuid.UUID]map[uuid.UUID]masteraction.View {
	if closed == nil || res == nil || r.session == nil {
		return nil
	}
	reactions := closed.GetReactions()
	byID := make(map[uuid.UUID]*action.Action, len(reactions))
	for i := range reactions {
		byID[reactions[i].GetID()] = &reactions[i]
	}
	var out map[uuid.UUID]map[uuid.UUID]masteraction.View
	for _, cr := range res.CharacterResults {
		reaction, ok := byID[cr.ReactionID]
		if !ok {
			continue
		}
		mv := closedEscapeMove(reaction, cr)
		if mv == nil {
			continue
		}
		target, dest := cr.TargetID, mv.Position
		if out == nil {
			out = map[uuid.UUID]map[uuid.UUID]masteraction.View{}
		}
		out[target] = r.sessionPlayerViewsLocked(r.ownerOfLocked(target), func(pid uuid.UUID) (masteraction.View, bool) {
			return r.escapeLandingViewLocked(pid, target, &dest)
		})
	}
	return out
}

// announceOpenedTurn is the tail both open_next_action and pull_action end with, byte for
// byte: once the baton has moved, "the next one opened" and "this one was pulled out of
// order" are the same news, and the two arms had drifted apart only by accident so far.
//
// The move goes out BEFORE turn_opened, with no r.mu held: Execute released it in the caller,
// and applyMove takes it itself. The table must never see the turn open with the piece still
// in the old slot, and piece_moved goes straight into each client's queue while turn_opened
// is now ALSO on that same direct lane — so applying the move first is still what fixes the
// order, exactly as before B2; only the reason the two shared a fate (both eventually landing
// in the same client queue) changed from "same broadcast channel" to "same direct lane".
//
// turn_opened is PROJECTED per recipient since B2 (design spec §4.2) — the master sees Full,
// everyone else sees Opened after the deny-list — so it travels through dispatchPerPlayer, one
// payload built per recipient under r.mu.RLock (session state has no lock of its own), handed
// back outside the lock as dispatchPerPlayer requires. act is fixed ONCE, before the loop: it
// does not change per recipient, only its projection (turnActionWireLocked) does.
//
// res is nil-safe: a turn can open with nothing to resolve.
func (r *Room) announceOpenedTurn(
	session *matchsession.MatchSession, opened *turnentity.Turn, res *domainservice.TurnResolution,
) {
	// origin is where the piece stood when the turn opened — what the relay just judged the
	// move on, and so what turn_opened's move gate judges it on too (turnActionWireLocked).
	origin := r.applyOpenedMove(opened)
	// What each session player sees of the move — the verdict the dispatch below applies —
	// recorded for the history, held with the turn until it closes. A separate lock section
	// from each recipient's projection below, but nothing can move a piece or change anyone's
	// sight in between: every message that can (master drags, wall changes, a close) comes from
	// the master, on the same read pump that is running this open.
	r.recordOpenedMoveViews(opened, origin)

	// GetAction returns a COPY, so it goes into a variable before any getter with a pointer
	// receiver is called on it — the same shape persistClosedTurn already uses.
	act := opened.GetAction()
	turnID, actorID, actionID := opened.GetID(), act.GetActorID(), act.GetID()

	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		// viewerFor AND turnActionWireLocked both run inside this SAME RLock section — not just
		// viewerFor. act is a copy of the Turn's Action struct (GetAction's own doc), but its
		// pointer/slice fields (Move, Attack, Skills, TargetID, ...) still point AT the live
		// session's memory: a concurrent edit_action/attach_reaction on this same match
		// mutates through them under r.mu, with no lock of MatchSession's own to stop a
		// reader outside r.mu from racing it. turnActionWireLocked (via actionwire.From) walks every
		// one of those fields, so projecting it has to happen under the lock too, not just
		// building the Viewer.
		r.mu.RLock()
		v := r.viewerFor(pid, isMaster)
		wireAction := r.turnActionWireLocked(act, pid, v, origin)
		r.mu.RUnlock()
		msg := NewServerMessage(MsgTypeTurnOpened, TurnOpenedPayload{
			TurnID:  turnID,
			ActorID: actorID,
			// The id the enqueuer got back in action_enqueued and the master got in
			// action_queued. Without it, two queued actions of the same character produce two
			// turn_openeds a client cannot tell apart — actorId is the same in both.
			ActionID: actionID,
			Action:   wireAction,
		})
		return &msg
	})

	if res != nil {
		r.broadcastWallResults(session, res.WallResults)
		// The projection for the turn just opened. publishResolution keeps this master-only on
		// its own: the mechanics are public when a turn opens, but the calculation stays with
		// the master until it closes (IsSettled is false here).
		r.publishResolution(opened.GetID(), res)
	}
}

// persistClosedTurn writes one turn that just closed to the round repository, together with
// its settled resolution, and marks the session's round persisted on success. It is the one
// place that does this — open_next_action, pull_action and close_turn all reach a closed
// turn by different paths, but from here on they all needed the same read-persist-flip
// sequence, and it had been copy-pasted into three ~15-line blocks.
//
// It takes r.mu itself, exactly as the three call sites used to: a Lock to snapshot the
// scene/round/matchUUID session needs, the board as the close left it (boardSnapshotLocked) and
// drain the turn's captured overrides and the master actions held for it (turnWrites) — both
// drains mutate, so this is a Lock now, not the RLock it started as — then, only on success, a
// second Lock to flip MarkRoundPersisted. Callers must NOT hold r.mu when calling this:
// sync.RWMutex is not reentrant, and a nested acquire would deadlock the room permanently.
// Every call site below already releases r.mu before reaching here.
//
// Everything the turn changed is written by ONE PersistTurnClose, in one transaction: the turn,
// its action and reactions, the overrides, the HP the close applied (damaged, copied here), the
// master actions applied inside it, and the board with every player's fog memory (owner
// decisions, 2026-10-01 and 2026-10-02: one master command, one transaction). It also carries
// what an earlier failed write left behind (unwritten.go): sheets whose HP a failed close never
// wrote, and round ends that were never written — so a failure heals on the next success. The board is the
// board AFTER applyClosedEscapes and BEFORE announceOpenedTurn — every caller walks the escapes
// first and announces the next turn after this returns — so an open_next_action that already
// opened the next turn in the same Execute never writes that turn's opened move under this one.
//
// persistMu is held from the snapshot through the write, the same rule persistBoard follows
// (lock order persistMu, then r.mu): a between-turns save racing this close lands after it,
// never under it with an older picture.
//
// A PersistTurnClose failure is logged and swallowed, not returned: the turn already closed
// in memory and the match goes on regardless. But say WHAT was lost — this ran silently for
// two phases while an FK mismatch dropped every single turn on the floor.
//
// closedIn is the round the turn closed in when that is no longer the session's active round —
// an open_next_action that closes the turn AND, finding no action that can still pay its price,
// the round with it (CloseRound installs the successor before this runs). nil means the active
// round. The turn is written under closedIn, and closedIn's finish and the successor's birth go
// in the same transaction (TurnCloseData.NextRound): one master command, one transaction (owner
// decision, 2026-10-02).
func (r *Room) persistClosedTurn(
	session *matchsession.MatchSession, t *turnentity.Turn, res *domainservice.TurnResolution,
	closedIn *roundentity.Round, damaged []matchsession.DamagedCharacter,
) {
	act := t.GetAction()
	r.persistMu.Lock()
	defer r.persistMu.Unlock()
	// Write lock, not read: TakeOverridesFor DRAINS two maps on the session, it does not just
	// read them, so it needs the same lock a mutation would. Reading Scene/Round/MatchUUID
	// here too costs nothing extra — they were already read under a lock, just a lesser one —
	// and doing the drain in the same critical section avoids a second acquire.
	//
	// The lock is released before PersistTurnClose: that call is a DB round trip, and every
	// caller of persistClosedTurn has already released r.mu before calling in, so there is no
	// nested acquire on this path either way — but holding the mutex across network I/O would
	// still block the whole room's message loop for no reason.
	r.mu.Lock()
	activeScene := session.GetActiveScene()
	activeRound := session.GetActiveRound()
	var nextRound *roundentity.Round
	if closedIn != nil {
		// This same command also ended the round: CloseRound already made its successor the
		// active round, and that successor is born a row in this turn's transaction.
		if activeRound != nil && activeRound.GetID() != closedIn.GetID() {
			nextRound = activeRound
		}
		activeRound = closedIn
	}
	// Copies: PersistTurnClose reads them after the unlock, and the live objects are the
	// session's (a regime switch mutates the active round under this lock).
	activeScene, activeRound = snapshotSceneAndRound(activeScene, activeRound)
	_, nextRound = snapshotSceneAndRound(nil, nextRound)
	matchUUID := session.GetMatchUUID()
	overrides := session.TakeOverridesFor(t)
	// What the master did inside this turn, held until now (turnWrites): written in the turn's
	// own transaction, or lost with it.
	inTurn := r.takeTurnWritesLocked(t.GetID())
	// The board as this close left it, in the same critical section: what the turn's write
	// says the board is, exactly.
	board, memories := r.boardSnapshotLocked()
	// Who saw where each escape's piece ended, on that same board — the verdict the settled
	// resolution_updated sent after this applies (escapeLandingViewLocked), recorded for the
	// history. A separate lock section from that send, but nothing can move a piece or change
	// anyone's sight in between: every message that can (master drags, wall changes, the next
	// open) comes from the master, on the same read pump that is running this close.
	landingViews := r.escapeLandingViewsLocked(t, res)
	// The bars of every sheet this close damaged — and of any an earlier failed close left
	// unwritten — as the session holds them now: copies, read in this same critical section,
	// since the next message may move the live ones.
	sheets := r.sheetsToWriteLocked(damaged)
	statusBars := statusBarsLocked(sheets)
	// Round ends an earlier command could not write: closed in this same transaction, before the
	// round this turn belongs to (or its successor) can be born next to them still open.
	roundEnds := r.unwrittenRoundEndsLocked()
	r.mu.Unlock()

	err := r.deps.RoundRepo.PersistTurnClose(context.Background(), appmatch.TurnCloseData{
		Scene: activeScene, Round: activeRound, Turn: t, Action: &act,
		MatchUUID: matchUUID, Resolution: res, Overrides: overrides,
		MasterActions: inTurn.masterActions, Board: board, Memories: memories,
		MoveViews: inTurn.moveViews, LandingViews: landingViews, StatusBars: statusBars,
		NextRound: nextRound, UnwrittenRoundEnds: roundEnds,
	})
	if err != nil {
		log.Printf("PersistTurnClose FAILED — turn %s of match %s was NOT persisted, nor the %d master action(s) applied inside it, "+
			"nor the HP it applied to %d sheet(s) (character_sheets keep the last close), "+
			"nor the board it left (match_boards and player_memories keep the last close): %v",
			t.GetID(), matchUUID, len(inTurn.masterActions), len(statusBars), err)
		// The overrides and the turn's master actions were already drained above and are lost
		// with the turn. That is correct, not a leak to plug: overridden_action_values.action_uuid
		// references actions(uuid), so the override rows could not have been inserted without the
		// action row; and the master actions belong to the turn by decision — durable with it or
		// not at all. The board in memory is untouched and still live: the next save (a close,
		// or a change between turns) writes it whole.
		//
		// The HP is not lost either: the sheets keep it in memory, and they are remembered as
		// unwritten (the ones of earlier failed closes stay remembered), so the NEXT close writes
		// them with its own, whichever character it damages (unwritten.go).
		r.mu.Lock()
		r.markSheetsUnwrittenLocked(damaged)
		r.mu.Unlock()
		// The round's end is salvaged at once. The round DID end at the table, and leaving
		// its row open while the successor is not one would have the successor's first write (a
		// master action, the next close) leave the scene with two open rounds. So the end and the
		// birth still go together, in a transaction of their own — the same one a round that ends
		// with no turn closing writes. If that fails too, writeRoundClose remembers the end, and
		// the successor's next write carries it.
		if nextRound != nil {
			r.writeRoundClose(session, activeScene, activeRound, nextRound, "round end after its last turn's failed close")
		}
		return
	}
	// The round the session is in is a row now — the one this wrote, or, when this close also
	// ended it, the successor this same transaction gave birth to. Only if it is still the active
	// one: a later command may already have moved on.
	written := activeRound
	if nextRound != nil {
		written = nextRound
	}
	r.mu.Lock()
	r.forgetSheetsWrittenLocked(sheets)
	r.forgetRoundEndsWrittenLocked(roundEnds)
	markRoundPersistedIfActiveLocked(session, written.GetID())
	r.mu.Unlock()
}

func (r *Room) handleReaction(client *Client, session *matchsession.MatchSession, payload ActionPayload) {
	reaction, err := buildAction(payload.ActorID, payload)
	if err != nil {
		client.SendMessage(NewErrorMessage("invalid_action", err.Error()))
		return
	}
	// Write lock across Execute: attaching a reaction rolls its dice and re-resolves the
	// open turn, and the master may be closing that same turn from another goroutine.
	r.mu.Lock()
	result, err := r.deps.AttachReactionUC.Execute(context.Background(), session, client.userUUID, reaction)
	var turnID uuid.UUID
	var attached ReactionAttachedPayload
	if err == nil {
		turnID = session.CurrentTurnID()
		// Read under the lock: `reaction` is ours, but Execute just wrote ConsumedActionIDs into
		// it, and the session's copy is the one later readers use — both are written before the
		// unlock, never after.
		attached = ReactionAttachedPayload{
			TurnID: turnID, ReactionID: reaction.GetID(), ActorID: reaction.GetActorID(),
			ConsumedActionIDs: append([]uuid.UUID{}, reaction.ConsumedActionIDs...),
		}
	}
	r.mu.Unlock()
	if err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	// Before the master's resolution_updated, on the same lane and goroutine: the reactor and
	// the master learn the reaction's ID before anything else names it.
	ack := NewServerMessage(MsgTypeReactionAttached, attached)
	client.SendMessage(ack)
	if !r.IsMaster(client.userUUID) {
		// The master reacting through an NPC is reactor AND master: one copy, not two.
		r.sendToMaster(ack)
	}
	// publishResolution is the single place that decides master-only vs projected, by reading
	// IsSettled. An attach lands on an open, unsettled turn — and ONLY that, because
	// MatchSession.AttachReaction now refuses (ErrTurnAlreadyClosed) before rolling a single
	// die or charging a single bar when the turn's finishedAt is already set, so `err` above
	// is non-nil and we never reach this line for a closed turn. That guard did not exist
	// before Phase 5's close_turn made "closed and nothing open" a state the master could sit
	// in on purpose; until then a stale attach was merely caught downstream by a ReactToID
	// mismatch. This is routed through the same door as every other resolution anyway, so a
	// future change to attach timing cannot silently reintroduce a master-only leak here — but
	// the guard, not the routing, is what keeps this comment true.
	r.publishResolution(turnID, result.Resolution)
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

// newBarsUpdatedPayload snapshots both clocks for the whole table.
//
// The caller holds r.mu — it reads session state that every open and every enqueue mutates —
// and stamps Seq from the room counter in that same critical section.
func newBarsUpdatedPayload(session *matchsession.MatchSession) BarsUpdatedPayload {
	prices := map[string]int{}
	for bar, price := range session.RoundPrices() {
		prices[string(bar)] = price
	}

	out := BarsUpdatedPayload{Prices: prices}
	for _, charID := range session.CharacterIDs() {
		actionCarry, actionSpeeds := session.BarState(charID, action.BarAction)
		moveCarry, moveSpeeds := session.BarState(charID, action.BarMove)
		out.Characters = append(out.Characters, CharacterBarsPayload{
			CharacterID:   charID,
			ActionBalance: actionCarry,
			MoveBalance:   moveCarry,
			ActionSpeeds:  append([]int(nil), actionSpeeds...),
			MoveSpeeds:    append([]int(nil), moveSpeeds...),
		})
	}

	for _, slot := range session.ProjectedOrder() {
		bars := make([]string, 0, len(slot.Bars))
		for _, b := range slot.Bars {
			bars = append(bars, string(b))
		}
		out.Order = append(out.Order, BarSlotPayload{
			ActorID: slot.ActorID,
			Bars:    bars,
			Key:     slot.Key,
		})
	}
	return out
}

// viewerFor builds the domainservice.Viewer this playerID/isMaster pair is entitled to, from
// the session's live charToPlayer. The caller MUST hold r.mu (a read lock is enough) —
// MatchSession has no lock of its own, and GetCharToPlayer reads live session state.
//
// Pulled out of buildMatchFullState and publishResolution, which had each grown their own copy
// of this exact loop (uuid.Parse of the char string, matched against playerID) for the same
// reason: both need a per-recipient Viewer, one to project a resolution, the other to project
// an action. Now announceOpenedTurn's per-recipient turn_opened uses it too.
func (r *Room) viewerFor(playerID uuid.UUID, isMaster bool) domainservice.Viewer {
	owns := map[uuid.UUID]bool{}
	if r.session != nil {
		for charStr, pid := range r.session.GetCharToPlayer() {
			if pid != playerID {
				continue
			}
			if charID, err := uuid.Parse(charStr); err == nil {
				owns[charID] = true
			}
		}
	}
	return domainservice.Viewer{IsMaster: isMaster, Owns: owns}
}

// publishResolution sends one turn's resolution to everyone entitled to a version of it.
//
// TWO axes, not one:
//
//   - TIME: while the turn is open the calculation belongs to the master alone
//     (combat-engine.md § Visibilidade). Only a SETTLED resolution reaches the table.
//   - CLASS: master / owner / everyone else, applied by service.ProjectResolution.
//
// It reuses dispatchPerPlayer, which is the mechanism the fog of war already uses for exactly
// this. Do not grow a second one.
func (r *Room) publishResolution(turnID uuid.UUID, res *domainservice.TurnResolution) {
	if res == nil {
		return
	}
	if !res.IsSettled {
		r.sendToMaster(NewServerMessage(
			MsgTypeResolutionUpdate, newResolutionUpdatedPayload(turnID, res)))
		return
	}

	r.dispatchPerPlayer(func(playerID uuid.UUID, isMaster bool) *Message {
		// The landing gate reads the board and the visibility cache, so the payload is built
		// under the same RLock as the Viewer; it is sent, as always, after the unlock.
		r.mu.RLock()
		v := r.viewerFor(playerID, isMaster)
		p := newResolutionUpdatedPayload(turnID, domainservice.ProjectResolution(res, v))
		r.gateEscapeLandingsLocked(&p, playerID, v)
		r.mu.RUnlock()
		msg := NewServerMessage(MsgTypeResolutionUpdate, p)
		return &msg
	})
}

// broadcastBars publishes the clocks to everyone. Called after anything that moves them:
// an action enqueued, a turn opened, a round closed, a regime switched.
func (r *Room) broadcastBars(session *matchsession.MatchSession) {
	if session == nil {
		return
	}
	// Write-locked, not read-locked: the sequence counter is bumped in the same critical
	// section that reads the state, so the number and the snapshot it stamps cannot disagree.
	r.mu.Lock()
	r.barsSeq++
	payload := newBarsUpdatedPayload(session)
	payload.Seq = r.barsSeq
	r.mu.Unlock()
	data, _ := json.Marshal(NewServerMessage(MsgTypeBarsUpdated, payload))
	go func() { r.broadcast <- data }()
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
//
// It returns what each player of the session saw of it (spec §4.8), by that same criterion:
// everyone for a wall that went to everyone, nobody for one that went to the master alone.
// nil in the lobby.
func (r *Room) broadcastWallStateChangedGated(wallID string, open, locked bool) map[uuid.UUID]masteraction.View {
	r.mu.RLock()
	w, ok := r.walls[wallID]
	r.mu.RUnlock()
	if ok && w.WallType == mapentity.WallTypeSecretDoor && !w.Revealed {
		r.sendToMaster(NewServerMessage(MsgTypeWallStateChanged, WallStateChangedPayload{
			WallID: wallID,
			Open:   open,
			Locked: locked,
		}))
		return r.sessionViews(seenByNone)
	}
	r.broadcastWallStateChanged(wallID, open, locked)
	return r.sessionViews(seenByAll)
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
//
// A target wall the server does not know answers unknown_wall to client — the master, always,
// since the caller already gated on master — instead of the silent skip this used to be (spec
// §4.3, "Última defesa"). The rest of the batch still runs.
//
// It returns the walls it actually revealed and what each player of the session saw: a reveal
// goes to everyone, so everyone saw it in full (spec §4.8). Both are empty when nothing was
// revealed — then there is no master action to record.
func (r *Room) revealSecretDoors(client *Client, targetIDs []uuid.UUID) ([]string, map[uuid.UUID]masteraction.View) {
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

	var revealed []string
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
			client.SendMessage(NewErrorMessage("unknown_wall",
				fmt.Sprintf("wall %s is not on this match's board", wallID)))
			continue
		}
		msg := NewServerMessage(MsgTypeWallRevealed, WallRevealedPayload{Wall: toWallSegmentPayload(w)})
		data, _ := json.Marshal(msg)
		go func(d []byte) { r.broadcast <- d }(data)
		revealed = append(revealed, wallID)
	}
	// Revealing changes nothing about geometry but the cache must reflect Revealed; push LOS.
	r.pushVisibilityUpdates()
	if len(revealed) == 0 {
		return nil, nil
	}
	return revealed, r.sessionViews(seenByAll)
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

// slotToAB reads a piece's grid position [a, b] out of its SlotPayload — (a, b) is (q, r)
// axial on a hex slot, (col, row) on a square one. Shared by slotPayloadToWorld (which turns
// it into a world point) and pieceSlotOf (which reports it as the Move.From a/b pair
// verbatim), so the two never drift into disagreeing about which field means what.
func slotToAB(s SlotPayload) (a, b int) {
	if s.Kind == "hex" {
		if s.Q != nil {
			a = *s.Q
		}
		if s.R != nil {
			b = *s.R
		}
		return a, b
	}
	if s.Col != nil {
		a = *s.Col
	}
	if s.Row != nil {
		b = *s.Row
	}
	return a, b
}

func slotPayloadToWorld(s SlotPayload, g mapentity.GridShape) (float64, float64) {
	a, b := slotToAB(s)
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

// buildMatchFullState snapshots the combat for one recipient. Returns nil when there is no
// session: in the lobby there is no combat to sync.
//
// The assembly is its OWN function, on purpose — the same shape buildMapFullState already
// has, and it is what makes match_full_state testable without standing up a connection.
//
// The caller must NOT hold r.mu — this takes it, held for the whole read (session, barsSeq,
// scene, round, turn and the ResolveTurn recompute), exactly as buildMapFullState does with
// its own single RLock/RUnlock pair. There is no reason to split it into two critical
// sections here: nothing in between needs the lock released, and ResolveTurn is a pure
// recompute (no I/O, no sheet writes), so it is safe to run under RLock.
func (r *Room) buildMatchFullState(playerID uuid.UUID, isMaster bool) *Message {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session := r.session
	if session == nil {
		return nil
	}

	payload := MatchFullStatePayload{Bars: newBarsUpdatedPayload(session)}
	// The CURRENT counter, not a new one — see MatchFullStatePayload.Bars' own doc comment.
	// broadcastBars is the only place that bumps r.barsSeq; this just reads it.
	payload.Bars.Seq = r.barsSeq

	if scene := session.GetActiveScene(); scene != nil {
		// The same struct scene_changed sends, not a second flattened copy of it — see
		// MatchFullStatePayload.Scene's own doc comment.
		payload.Scene = &SceneChangedPayload{
			SceneID:  scene.GetID(),
			Category: string(scene.GetCategory()),
			// BriefInitialDescription is a public field on scene.Scene, not a getter — there is
			// no GetBriefInitialDescription method (the brief for this task assumed one;
			// checked against the real type before writing this).
			BriefInitialDescription: scene.BriefInitialDescription,
		}
	}
	if round := session.GetActiveRound(); round != nil {
		payload.RoundMode = string(round.GetMode())
		// HasOpenTurn, not a bare "CurrentTurn() != nil" check: CurrentTurn returns the round's
		// LAST turn regardless of whether it already closed, and a round sits with its last
		// turn closed for the whole window between close_turn and the next open_next_action.
		// A bare nil check would hand a late joiner a stale OpenTurn for a turn that already
		// settled — and would hand the master a Resolution for it too, which is table state
		// they were already sent when it closed, not something this snapshot owes them again.
		if round.HasOpenTurn() {
			t := round.CurrentTurn()
			// GetAction returns a COPY (turn.go is explicit about this — ActionRef is the one
			// that hands out a pointer, deliberately narrower, for the one caller that mutates
			// it). Assigning it to a variable first, then calling GetActorID on the variable,
			// is required: Go cannot take the address of a bare method-call result to satisfy
			// GetActorID's pointer receiver.
			act := t.GetAction()
			// v is this recipient's Viewer — built once and reused for BOTH the action
			// projection below and, on the master branch, the resolution projection: the two
			// used to build their own copy of the exact same owns-lookup loop (viewerFor's own
			// doc).
			v := r.viewerFor(playerID, isMaster)
			payload.OpenTurn = &OpenTurnPayload{
				TurnID:  t.GetID(),
				ActorID: act.GetActorID(),
				// ActionID and Action are B2 (design spec §4.2): the same two fields the live
				// turn_opened carries, projected by the exact same rule (turnActionWireLocked) — a
				// reconnecting client's snapshot can never disagree with the live event they
				// may already have received.
				ActionID: act.GetID(),
				// The reconnect has no record of where the piece stood at the opening, so the
				// gate judges the move from its enqueue-time move.from (turnActionWireLocked).
				Action: r.turnActionWireLocked(act, playerID, v, moveFromOf(act)),
			}
			if isMaster {
				// ResolveTurn is a pure recompute, never a re-roll: the dice fell when the
				// action arrived. attach_reaction and edit_action already call it the same way.
				if res := session.ResolveTurn(t); res != nil {
					// ProjectResolution is called even though this branch is master-only and the
					// projection is the IDENTITY for a master (Viewer.SeesAllOf is true for every
					// character, and PendingReactions/Errors are exactly what a master keeps).
					// The call is here so the PORT is already the right shape: without it this is
					// the only emitter of a resolution that never goes through the projection,
					// and the day reaction visibility stops being master-only it would start
					// leaking with no test able to catch it.
					p := newResolutionUpdatedPayload(t.GetID(), domainservice.ProjectResolution(res, v))
					payload.Resolution = &p
				}
			}
		}
	}

	if isMaster {
		// MASTER-ONLY, the same axis Resolution above is gated on, and for the reason
		// MatchFullStatePayload.Queue documents: the queue is secret. Deliberately OUTSIDE the
		// round/HasOpenTurn block — the queue exists whether or not a turn is open, and the
		// state a reconnecting master is most likely to land in ("nothing opened yet, three
		// things waiting") is exactly the one where round.HasOpenTurn() is false.
		//
		// PendingActions reads s.activeQueue with no lock of its own; r.mu (held for this whole
		// function) is what serializes it against a concurrent enqueue_action.
		for _, a := range session.PendingActions() {
			payload.Queue = append(payload.Queue, newActionQueuedPayload(a))
		}
	} else {
		// OwnQueue is B12 (design spec §4.2) — see MatchFullStatePayload.OwnQueue's own doc for
		// the shape and the reconciliation rule. Deliberately OUTSIDE the round/HasOpenTurn
		// block too, mirroring Queue above: a recipient's own pending actions exist whether or
		// not a turn is open.
		//
		// charToPlayer is read directly rather than through r.viewerFor: viewerFor builds an
		// Owns SET keyed by character for a visibility deny-list, which is not what this filter
		// needs — this is the same actor-owns-the-queue-entry check EnqueueAction itself made
		// at insertion time (matchsession.EnqueueAction's own doc), just re-applied per pending
		// action instead of once at insert.
		//
		// The slice is built non-nil even when no entry matches, and the field is a taken
		// address of it — never left as the zero `*[]OwnQueuedActionPayload` (nil) — so a
		// non-master ALWAYS gets `"ownQueue": []` at minimum. See the field's own doc for why
		// that distinction (present-and-empty vs absent) matters to a reconnecting client.
		charToPlayer := session.GetCharToPlayer()
		pending := session.PendingActions()
		own := make([]OwnQueuedActionPayload, 0, len(pending))
		for _, a := range pending {
			if charToPlayer[a.GetActorID().String()] != playerID {
				continue
			}
			own = append(own, OwnQueuedActionPayload{
				ActionID: a.GetID(),
				Action:   actionwire.From(*a, actionwire.Declaration),
			})
		}
		payload.OwnQueue = &own
	}

	msg := NewServerMessage(MsgTypeMatchFullState, payload)
	return &msg
}

// newActionQueuedPayload is one pending action as the MASTER reads it: the ID pull_action
// needs, who queued it, which clocks it will charge, and — B1, design spec §4.2 — the whole
// declaration, at actionwire.Full. No projection: both surfaces this feeds are already
// master-only, so Full is safe here the same way it is safe on the REST Action History (see
// ActionQueuedPayload's own doc).
//
// Shared by the action_queued emitted at enqueue time and by match_full_state's Queue, so the
// live event and the snapshot can never describe the same action differently.
func newActionQueuedPayload(a *action.Action) ActionQueuedPayload {
	bars := make([]string, 0, 2)
	for _, b := range a.Bars() {
		bars = append(bars, string(b))
	}
	return ActionQueuedPayload{
		ActionID: a.GetID(), ActorID: a.GetActorID(), Bars: bars,
		Action: actionwire.From(*a, actionwire.Full),
	}
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

// applyAndRelayPieceMove puts a piece on the board and tells everyone entitled to know.
//
// The fog gate is the point, and it is a PAIR: whoever can see the destination gets
// piece_moved, whoever could only see the origin gets piece_removed (the piece walked out of
// sight), whoever sees neither gets nothing. That is why the server-applied move reuses this
// instead of growing a second path — and why it stays on piece_moved rather than a new type,
// which would have to duplicate the pair to keep the "walked out of sight" case.
//
// origin is the player whose own browser already applied this move locally and must therefore
// not be echoed back to. It is uuid.Nil when the SERVER is the mover — nobody predicted that
// one, so nobody is skipped and the envelope goes out as a server message (senderId zero),
// which is how a client tells the two apart if it ever needs to.
//
// The owner is not a parameter: it comes from payload.CharacterID through GetCharToPlayer().
// The PLAYER who owns the moved character gets a fresh map_full_state, because their line of
// sight just changed — and that holds even when the mover is someone else (the master dragging
// a player's piece, or the engine applying a resolved move). An NPC's owner is the master,
// whose view has no fog, so an NPC's move refreshes nobody (see relayPieceMove).
//
// The caller must NOT hold r.mu — this takes it, and so do gridShape, visibilityFor,
// dispatchPerPlayer and buildMapFullState. The short lock/unlock blocks here and in
// relayPieceMove are deliberate: nothing that sends to a client runs inside a critical section.
func (r *Room) applyAndRelayPieceMove(payload PieceMovedPayload, origin uuid.UUID) {
	r.mu.Lock()
	old, hadOld := r.pieces[payload.PieceID]
	r.pieces[payload.PieceID] = payload
	r.mu.Unlock()

	r.relayPieceMove(payload, old, hadOld, origin)
}

// applyAndRelayPieceMoveIfOwned is applyAndRelayPieceMove, guarded: it only writes when the
// piece STILL exists with the exact CharacterID requireCharacterID — checked in the SAME
// critical section as the write, not before it.
//
// This is the fix for a TOCTOU handlePieceMoved's player path would otherwise hit:
// playerOwnsExistingPiece's ownership read is I/O and runs UNLOCKED, so a concurrent
// handlePieceRemoved (master-only) can delete the exact piece being approved while that read
// is in flight. Re-checking existence and CharacterID here, atomically with the write, is what
// stops a stale approval from resurrecting a piece the master just removed — relayed to the
// table and persisted, as if the removal had never happened (review round 1, Important 1).
//
// Returns false — writing, relaying and persisting NOTHING — when the piece is gone or its
// CharacterID no longer matches; the caller must refuse the same way it would have refused
// had playerOwnsExistingPiece failed outright.
//
// The caller must NOT hold r.mu.
func (r *Room) applyAndRelayPieceMoveIfOwned(
	payload PieceMovedPayload, origin uuid.UUID, requireCharacterID string,
) bool {
	r.mu.Lock()
	old, hadOld := r.pieces[payload.PieceID]
	if !hadOld || old.CharacterID != requireCharacterID {
		r.mu.Unlock()
		return false
	}
	r.pieces[payload.PieceID] = payload
	r.mu.Unlock()

	r.relayPieceMove(payload, old, hadOld, origin)
	return true
}

// relayPieceMove is everything applyAndRelayPieceMove does once the board is already written:
// the fog-gated per-player dispatch and the owner's refreshed map_full_state. It is split out
// so a caller that has to CHOOSE the piece before writing it — applyOpenedMove does — can do
// the read and the write in one critical section and still reuse the relay.
//
// old/hadOld are the piece as it stood BEFORE the write; they are what the piece_removed half
// of the pair is decided on.
//
// It returns what each player of the session saw of the move (spec §4.8) — the SAME fog gate
// the dispatch uses (pieceMoveView), run over every player of the session, connected or not.
// Only a master action records it; every other caller ignores it. nil in the lobby.
//
// The caller must NOT hold r.mu.
func (r *Room) relayPieceMove(payload, old PieceMovedPayload, hadOld bool, origin uuid.UUID) map[uuid.UUID]masteraction.View {
	grid := r.gridShape()
	newX, newY := slotPayloadToWorld(payload.Slot, grid)
	var oldX, oldY float64
	if hadOld {
		oldX, oldY = slotPayloadToWorld(old.Slot, grid)
	}
	hidden := payload.Visible != nil && !*payload.Visible

	// A server-authored move carries no sender. NewClientMessage with uuid.Nil would encode
	// the same bytes, but saying NewServerMessage keeps the intent readable at the call site.
	moved := NewServerMessage(MsgTypePieceMoved, payload)
	removed := NewServerMessage(MsgTypePieceRemoved, PieceRemovedPayload{PieceID: payload.PieceID})
	if origin != uuid.Nil {
		moved = NewClientMessage(MsgTypePieceMoved, origin, payload)
		removed = NewClientMessage(MsgTypePieceRemoved, origin, PieceRemovedPayload{PieceID: payload.PieceID})
	}

	// The owner of the moved character is resolved from the CHARACTER, not from the sender: a
	// move the owner did not send (master drag, engine-applied move) changes their sight just
	// the same, and resolving it here is what lets the server-authored path reuse this helper.
	r.mu.RLock()
	sess := r.session
	live := sess != nil
	var owner uuid.UUID
	if sess != nil {
		owner = sess.GetCharToPlayer()[payload.CharacterID]
	}
	r.mu.RUnlock()
	// An NPC is "owned" by the master in charToPlayer, but the master's view has no fog: they
	// see the board unfiltered, and buildMapFullState drops the polygons for isMaster anyway.
	// Treating the master as an owner cost a recompute of (NPCs × walls) under the WRITE lock,
	// a PlayerMemory nobody ever reads, and the whole board resent to the master on every drag.
	// So the master owns nothing here: the recompute, the memory and the extra map_full_state
	// are all skipped. The master still hears about the move through the isMaster branch of
	// the dispatch below — or gets no echo, when the master is the one who dragged it. This
	// holds for all three paths through here: a client drag, a turn's move, a reaction escape.
	if owner == r.masterUUID {
		owner = uuid.Nil
	}

	// The owner's line of sight is recomputed BEFORE the dispatch below, not after it.
	//
	// The fog gate in the dispatch reads the CACHE (r.visibilityFor), which is pure. Refreshing
	// the cache afterwards gated the owner on the polygon of the slot they had just LEFT: a
	// piece stepping out of its own former field of view scored seesOld=true, seesNew=false,
	// and its own owner was sent piece_removed for it — the corrective map_full_state only
	// arriving later. A client that treats piece_removed as authoritative and map_full_state as
	// a merge loses the token for good.
	//
	// Skipping the owner in the dispatch would hide the symptom too, but it would make the
	// map_full_state below their ONLY notice of the move — and that one is not sent when the
	// recompute fails. Fixing the staleness keeps the gate honest for every recipient instead
	// of carving out an exception, and still leaves the owner a piece_moved on the error path.
	var recomputeErr error
	if live && owner != uuid.Nil {
		r.mu.Lock()
		_, recomputeErr = r.session.RecomputeVisibility(owner)
		r.mu.Unlock()
		if recomputeErr != nil {
			// Logged, not swallowed: every other RecomputeVisibility call site in this file
			// logs, and a failure here means somebody is being gated on a stale polygon.
			log.Printf("piece move recompute visibility for %s: %v", owner, recomputeErr)
		}
	}

	// The fog gate, one player at a time: seeing the destination is the move (full); seeing only
	// the origin is the piece leaving (left, the piece_removed half); neither is nothing. Hidden
	// pieces never reach players. It is used TWICE — for the connected clients below, and for
	// every player of the session in views — so what is recorded is what was sent.
	newPt := domainservice.Point2D{X: newX, Y: newY}
	oldPt := domainservice.Point2D{X: oldX, Y: oldY}
	gate := func(polys []domainservice.VisibilityPolygon) (masteraction.View, bool) {
		return pieceMoveView(polys, newPt, oldPt, hadOld, hidden)
	}
	views := r.sessionViews(gate)

	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		if origin != uuid.Nil && pid == origin {
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
		switch v, ok := gate(r.visibilityFor(pid)); {
		case !ok:
			return nil
		case v == masteraction.ViewFull:
			m := moved
			return &m
		default:
			m := removed
			return &m
		}
	})

	// The owner's line of sight already changed, so resend them the full state. This stays
	// AFTER the dispatch on purpose: the negative assertions in the move tests use the owner's
	// second map_full_state as the ordering barrier that proves the relay was already queued.
	if !live || owner == uuid.Nil || recomputeErr != nil {
		return views
	}
	// The cache was refreshed even for an owner who is not connected — they would otherwise
	// reconnect onto a stale polygon. Only the push needs somebody on the other end.
	r.mu.RLock()
	ownerClient, online := r.clients[owner]
	r.mu.RUnlock()
	if online {
		msg := r.buildMapFullState(owner, r.IsMaster(owner))
		ownerClient.SendMessage(*msg)
	}
	return views
}

// refuseIfInMatch is handlePieceMoved's and handlePieceRemoved's shared mid-match refusal
// (spec §4.3, "Quem move o quê", B14): once r.session != nil, BOTH verbs are refused
// unconditionally for master AND player alike — the master moves/places/removes pieces
// through enqueue_master_action's move/remove (applyMasterPieceAction), and a player only ever
// moves by acting. This verb specifically is never the master's, in a match.
//
// Returns true when it sent a refusal — the caller must return immediately, without touching
// r.pieces or calling persistBoard.
func (r *Room) refuseIfInMatch(client *Client) bool {
	r.mu.RLock()
	inMatch := r.session != nil
	isMaster := r.masterUUID == client.userUUID
	r.mu.RUnlock()

	if !inMatch {
		return false
	}
	if isMaster {
		client.SendMessage(NewErrorMessage("forbidden", ErrMasterMovesByMasterAction.Error()))
	} else {
		client.SendMessage(NewErrorMessage("forbidden", ErrPlayersMoveByAction.Error()))
	}
	return true
}

// handlePieceMoved relays a move a client made in its own browser — but only in the LOBBY,
// and only a piece the sender is actually allowed to touch (spec §4.3, "Quem move o quê",
// B14). See refuseIfInMatch for the mid-match refusal.
//
// In the lobby the master can drag anything — the lobby has no notion of ownership beyond
// "the master runs the table". A player can only move a piece that ALREADY exists in
// r.pieces, whose CURRENT CharacterID's sheet is theirs (per SheetOwnership — the lobby has
// no charToPlayer to ask instead), and whose payload does not try to relabel that piece under
// a different CharacterID: moving is not how a player reassigns whose piece something is.
//
// The ownership read is I/O and runs outside r.mu — which opens a TOCTOU a concurrent
// handlePieceRemoved (master-only) can hit: the piece this check just approved can be gone by
// the time the write would happen. applyAndRelayPieceMoveIfOwned is what closes that window —
// see its own comment (review round 1, Important 1).
func (r *Room) handlePieceMoved(client *Client, payload PieceMovedPayload) {
	if r.refuseIfInMatch(client) {
		return
	}

	r.mu.RLock()
	isMaster := r.masterUUID == client.userUUID
	existing, hadExisting := r.pieces[payload.PieceID]
	r.mu.RUnlock()

	if isMaster {
		r.applyAndRelayPieceMove(payload, client.userUUID)
	} else {
		if !r.playerOwnsExistingPiece(client.userUUID, payload, existing, hadExisting) {
			client.SendMessage(NewErrorMessage("forbidden", ErrPieceNotOwnedByPlayer.Error()))
			return
		}
		if !r.applyAndRelayPieceMoveIfOwned(payload, client.userUUID, existing.CharacterID) {
			// The piece playerOwnsExistingPiece just approved is gone now, or its
			// CharacterID changed under us (e.g. the master's handlePieceRemoved ran while
			// the ownership I/O above was in flight, unlocked) — refuse instead of
			// resurrecting/relaying/persisting a piece that no longer exists as approved.
			client.SendMessage(NewErrorMessage("forbidden", ErrPieceNotOwnedByPlayer.Error()))
			return
		}
	}

	// "lobby": persistBoard itself only writes once a map is attached (r.mapUUID != uuid.Nil);
	// restricting this call to the LOBBY phase specifically is T4's job (spec §4.3, "Quando
	// persiste" lists "movimento e remoção no lobby" as its own line, distinct from the
	// master's in-match piece actions).
	r.persistBoard("lobby")
}

// playerOwnsExistingPiece answers handlePieceMoved's lobby-phase question for a NON-master
// sender: does this piece already exist, does the sheet behind its CURRENT CharacterID belong
// to this player, and does the payload agree on which character it is (a player cannot use a
// move to relabel a piece under a different CharacterID — that would be reassigning ownership,
// not moving).
//
// r.deps.SheetOwnership == nil is NOT "no capability, skip the check" the way most of
// RoomDeps' fields read: nobody has wired a way to verify ownership, so the only safe answer
// is "no" — this is the one field in RoomDeps whose absence fails closed, not open.
//
// Must NOT be called with r.mu held: the ownership read is I/O.
func (r *Room) playerOwnsExistingPiece(
	playerUUID uuid.UUID, payload PieceMovedPayload, existing PieceMovedPayload, hadExisting bool,
) bool {
	if !hadExisting || r.deps.SheetOwnership == nil {
		return false
	}
	if payload.CharacterID != existing.CharacterID {
		return false
	}
	charUUID, err := uuid.Parse(existing.CharacterID)
	if err != nil {
		return false
	}
	rel, err := r.deps.SheetOwnership.GetCharacterSheetRelationshipUUIDs(context.Background(), charUUID)
	if err != nil {
		return false
	}
	return rel.PlayerUUID != nil && *rel.PlayerUUID == playerUUID
}

// applyOpenedMove walks the opened action's Move onto the board.
//
// The position cannot wait for the close the way damage does: damage may still be edited
// while the turn is open, but the reactions that follow depend on where the piece IS.
//
// It returns applyMove's origin: where the piece stood when the turn opened, nil when nothing
// moved.
//
// The caller must NOT hold r.mu — applyMove takes it.
func (r *Room) applyOpenedMove(opened *turnentity.Turn) *[3]int {
	a := opened.GetAction()
	return r.applyMove(a.GetActorID(), a.Move)
}

// applyClosedEscapes puts on the board what the close decided about every escape of the turn
// — the one moment an escape's piece moves, since none moves when it opens (any of them can
// fail; see the open_reaction arm). The rule (front-combat-phases.md §6A.5, B13):
//
//   - it ESCAPED (movement and dodge both beat the attacker's hit) → its own destination;
//   - it failed and the master chose where the piece ended up (edit_action's escapeLanding) →
//     there;
//   - it failed with no choice → the piece stays where it stood.
//
// "Escaped" is read off cr.Escape, the verdict the resolver already reached — never
// recomputed here, so the board and the resolution the table reads can never disagree about
// it. cr.Escape is nil for every reaction that does not displace, which is also what keeps a
// Move that reached a non-escape by some other path from walking anyone.
//
// A reaction whose actor produced no CharacterResult at all — the engine could not classify
// the target, or their sheet never reached the resolver — displaces nothing. That is the safe
// side of the two: a piece that stays put is a visible non-event the master can narrate
// around, while moving it on a test that was never computed would be a rule applied out of
// nothing.
//
// It reads the closed turn without r.mu. That turn is finished — ApplyMasterAction,
// SetEscapeLanding and OpenReaction all refuse a turn with a finishedAt — so its reactions are
// frozen; it is the same window persistClosedTurn already reads GetAction() in. The caller
// must NOT hold r.mu: applyMove takes it.
func (r *Room) applyClosedEscapes(closed *turnentity.Turn, res *domainservice.TurnResolution) {
	if closed == nil || res == nil {
		return
	}
	reactions := closed.GetReactions()
	byID := make(map[uuid.UUID]*action.Action, len(reactions))
	for i := range reactions {
		byID[reactions[i].GetID()] = &reactions[i]
	}
	for _, cr := range res.CharacterResults {
		reaction, ok := byID[cr.ReactionID]
		if !ok {
			// The zero ReactionID of a target that answered with the passive defaults lands
			// here too: no reaction, nothing to displace.
			continue
		}
		if mv := closedEscapeMove(reaction, cr); mv != nil {
			r.applyMove(reaction.GetActorID(), mv)
		}
	}
}

// closedEscapeMove is the Move the close walks an escape's piece by — nil when it stays put.
// Shared by applyClosedEscapes (which applies it) and escapeLandingViewsLocked (which records
// who saw where it put the piece), so the destination recorded is exactly the one applied.
// Pure.
func closedEscapeMove(reaction *action.Action, cr domainservice.CharacterResult) *action.Move {
	switch {
	case cr.Escape != nil && cr.Escape.Escaped:
		return reaction.Move
	case cr.Escape != nil && cr.Escape.Landing != nil:
		// The escape failed and the master decided where the piece ended up, as part of
		// resolving this turn — not a drag (front-combat-phases.md §6A.5, B13). This is the
		// branch the definitive collision design will replace: where a failed escape lands
		// is a rule that does not exist yet.
		var landing action.Move
		if reaction.Move != nil {
			landing = *reaction.Move
		}
		landing.Position = *cr.Escape.Landing
		return &landing
	default:
		// Failed with no choice made: the piece stays where it stood. Same pointer as above
		// for the definitive design.
		return nil
	}
}

// escapeLandingEditOf lifts edit_action's escapeLanding into the use case's shape. nil when
// the payload carries none. ActionID names the escape; the session decides whether it is one.
func escapeLandingEditOf(p EditActionPayload) *appmatch.EscapeLandingEdit {
	if p.EscapeLanding == nil {
		return nil
	}
	return &appmatch.EscapeLandingEdit{ReactionID: p.ActionID, Position: p.EscapeLanding.Position}
}

// applyMove walks ONE Move onto the board, on behalf of the character that owns it.
//
// It is the ONE displacement path, shared by both callers: the action of a turn
// (applyOpenedMove) and an escape REACTION (applyClosedEscapes). It does not decide WHETHER
// or WHERE the piece moves — each caller has already decided that — it only puts it where the
// Move says.
//
// WHEN each caller fires is the rule, and the two halves of it are different questions:
//
//   - An ACTION displaces at the OPENING, whatever the category. Nothing is coming at it, so
//     there is nothing to clear, and the position cannot wait the way damage can: the
//     reactions that follow depend on where the piece stands.
//   - An escape REACTION displaces only at the CLOSE. It has a test to clear — movement and
//     dodge against the attacker's hit — and any escape can fail it, so where its piece ends
//     up is only known once the turn settles. See applyClosedEscapes.
//
// A move that DOES test for any OTHER reason (a leap, a squeeze past, a landing on an occupied
// slot) still has no case that can reach this code — moveSpeedSkill refuses Back, Roll, Slide,
// Jump and FlatJump at the WS boundary — so that branch is still not written here. Inventing
// one to fill the table would be guessing at a rule nobody has decided.
//
// A nil Move is a no-op: most actions and most reactions do not displace at all.
//
// A character with no piece on the board is NOT an error: there is simply nothing to move, so
// no error message goes out for it.
//
// It returns where the piece stood BEFORE the write, as a grid position [a, b, 0] (the same
// shape pieceSlotOf reports) — the origin the relay just judged the move on. nil when nothing
// moved (no Move, no piece). Only announceOpenedTurn reads it: turn_opened's move.from/position
// have to be gated from this same origin, not from the enqueue-time move.from, which is stale
// once the piece moved in between (turnActionWireLocked).
//
// The caller must NOT hold r.mu — this takes it, and applyAndRelayPieceMove's relay takes it
// again afterwards.
func (r *Room) applyMove(actor uuid.UUID, move *action.Move) *[3]int {
	if move == nil {
		return nil
	}
	actorID := actor.String()
	pos := move.Position

	// Finding the piece and writing it back happen in ONE write-locked section. Reading it
	// under RLock, releasing, and then storing the edited copy would leave a window in which
	// a concurrent piece_removed or map_state_sync lands: the stale copy would resurrect a
	// piece that was just taken off the board, or overwrite Visible/CharacterID with values
	// that are no longer true. The race detector can never see this — every single access is
	// correctly locked; it is the gap between two of them that is wrong.
	r.mu.Lock()
	// TODO: a character with more than one piece on the board is not a decided situation.
	// Nothing creates it today; whoever makes it possible has to say which piece an action
	// moves. Until then the lowest piece ID wins — an arbitrary choice, but a STABLE one, so
	// the day it happens it reproduces instead of flickering with map iteration order.
	pieceID := r.pieceOfLocked(actorID)
	if pieceID == "" {
		r.mu.Unlock()
		return nil
	}
	old := r.pieces[pieceID]
	oa, ob := slotToAB(old.Slot)
	origin := [3]int{oa, ob, 0}
	moved := old
	moved.Slot = slotKeepingKind(old.Slot, pos)
	// Z is deliberately NOT touched. It is the piece's virtual height in METRES, while
	// Move.Position[2] is a grid index — nobody has checked that the two are the same number,
	// and the front draws whatever position arrives without recomputing it. Writing pos[2]
	// here would drop an elevated piece to the ground on any horizontal step whose z is 0.
	// Preserving it is always right for a horizontal move; the vertical case belongs to
	// whoever writes the contract.
	r.pieces[pieceID] = moved
	r.mu.Unlock()

	// origin is uuid.Nil: the server moved this one and nobody's browser predicted it, so
	// nobody is skipped and the message goes out as a server message.
	r.relayPieceMove(moved, old, true, uuid.Nil)
	return &origin
}

// pieceOfLocked is the piece a character moves by: its lowest piece ID — see the TODO in
// applyMove for why that choice. "" when the character has no piece on the board.
//
// The caller MUST hold r.mu (for writing, when it goes on to write the piece back: the read and
// the write have to be one critical section — see applyMove).
func (r *Room) pieceOfLocked(characterID string) string {
	pieceID := ""
	for id, p := range r.pieces {
		if p.CharacterID == characterID && (pieceID == "" || id < pieceID) {
			pieceID = id
		}
	}
	return pieceID
}

// pieceSlotOf is the origin B6 reads for a Move: the ACTOR'S OWN piece position on the
// server's board, never the client's declared "from" (spec §4.3 "B5, B6 e B10"). It reuses
// pieceOfLocked for the same "lowest piece ID wins" lookup applyMove already does, so the two
// never disagree about which piece a multi-piece character moves by.
//
// Returns (nil, false) when the character has no piece on the board — there is nothing to
// check and nothing to report. Otherwise [a, b, 0]: (a, b) is (col, row) on a square slot or
// (q, r) axial on a hex one (slotToAB), and z is always 0 — a piece's z is its virtual height
// in metres (see applyMove's own doc), not a grid index, so it has no place in a grid position.
//
// The caller MUST hold r.mu (a read lock is enough: this only reads r.pieces).
func (r *Room) pieceSlotOf(characterID string) (pos *[3]int, ok bool) {
	pieceID := r.pieceOfLocked(characterID)
	if pieceID == "" {
		return nil, false
	}
	a, b := slotToAB(r.pieces[pieceID].Slot)
	out := [3]int{a, b, 0}
	return &out, true
}

// slotKeepingKind puts a piece at grid position pos in the slot SHAPE it already had. The
// board can be hexagonal, and forcing "square" here would put a hex piece at the world position
// of a square cell. An empty Kind is left empty on purpose: slotPayloadToWorld already reads
// anything that is not "hex" as square, and rewriting it would change what the client seeded.
func slotKeepingKind(old SlotPayload, pos [3]int) SlotPayload {
	if old.Kind == "hex" {
		qAxis, rAxis := pos[0], pos[1]
		return SlotPayload{Kind: "hex", Q: &qAxis, R: &rAxis}
	}
	col, row := pos[0], pos[1]
	return SlotPayload{Kind: old.Kind, Col: &col, Row: &row}
}

// handlePieceRemoved removes a piece and relays the removal per-player — lobby-only, and
// master-only, the same rule handlePieceMoved's own doc comment explains (spec §4.3, "Quem
// move o quê", B14): a session forbids it outright for master AND player alike (the master
// removes through enqueue_master_action's remove, Task 5), and even in the lobby a player
// never removes any piece — only the master does. See refuseIfInMatch for the mid-match half.
// A hidden piece (visible=false) is treated as master-only.
func (r *Room) handlePieceRemoved(client *Client, payload PieceRemovedPayload) {
	if r.refuseIfInMatch(client) {
		return
	}

	r.mu.RLock()
	isMaster := r.masterUUID == client.userUUID
	r.mu.RUnlock()

	if !isMaster {
		client.SendMessage(NewErrorMessage("forbidden", ErrPlayersCannotRemovePieces.Error()))
		return
	}

	r.mu.Lock()
	old, hadOld := r.pieces[payload.PieceID]
	delete(r.pieces, payload.PieceID)
	r.mu.Unlock()
	if !hadOld {
		// Nothing was there: the relay still goes out under the id the client named, as it
		// always has — there is just no last position to gate it on.
		old = PieceMovedPayload{PieceID: payload.PieceID}
	}

	r.relayPieceRemoved(old, hadOld, client.userUUID)
	// "lobby" — see handlePieceMoved's own comment; the same rule applies here.
	r.persistBoard("lobby")
}

// relayPieceRemoved is everything handlePieceRemoved does once the piece is already gone from
// r.pieces: the fog-gated per-player dispatch — only whoever could see the piece at its last
// position is told; a hidden piece reaches the master alone — split out so the master's
// mid-match "remove" (applyMasterPieceAction) can share it.
//
// old is the piece as it stood before the delete (hadOld false: only its PieceID is known).
// origin is the sender, skipped by the dispatch because their screen already removed it;
// uuid.Nil for a server-applied removal, which then goes out as a server message to everyone
// entitled, the master included.
//
// With a live session, the removed piece's OWNER then has their line of sight recomputed and
// resent: the piece was one of the points they see from, and a stale cache would keep showing
// them the board through a piece that is gone. AFTER the dispatch, unlike relayPieceMove: here
// the owner is supposed to be told the piece left, and the old polygon is the one that saw it.
// The master owns nothing, for the reason relayPieceMove gives.
//
// It returns what each player of the session saw (spec §4.8), from the same gate as the
// dispatch (pieceRemovedView). nil in the lobby.
//
// The caller must NOT hold r.mu.
func (r *Room) relayPieceRemoved(old PieceMovedPayload, hadOld bool, origin uuid.UUID) map[uuid.UUID]masteraction.View {
	hidden := hadOld && old.Visible != nil && !*old.Visible
	grid := r.gridShape()
	var oldPt domainservice.Point2D
	if hadOld {
		oldPt.X, oldPt.Y = slotPayloadToWorld(old.Slot, grid)
	}

	payload := PieceRemovedPayload{PieceID: old.PieceID}
	removed := NewServerMessage(MsgTypePieceRemoved, payload)
	if origin != uuid.Nil {
		removed = NewClientMessage(MsgTypePieceRemoved, origin, payload)
	}

	r.mu.RLock()
	live := r.session != nil
	var owner uuid.UUID
	if live && hadOld {
		owner = r.session.GetCharToPlayer()[old.CharacterID]
	}
	r.mu.RUnlock()
	if owner == r.masterUUID {
		owner = uuid.Nil
	}

	gate := func(polys []domainservice.VisibilityPolygon) (masteraction.View, bool) {
		return pieceRemovedView(polys, oldPt, hadOld, hidden)
	}
	views := r.sessionViews(gate)

	r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
		if origin != uuid.Nil && pid == origin {
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
		if _, ok := gate(r.visibilityFor(pid)); !ok {
			return nil
		}
		m := removed
		return &m
	})

	if !live || owner == uuid.Nil {
		return views
	}
	r.mu.Lock()
	_, err := r.session.RecomputeVisibility(owner)
	r.mu.Unlock()
	if err != nil {
		log.Printf("piece removal recompute visibility for %s: %v", owner, err)
		return views
	}
	r.mu.RLock()
	ownerClient, online := r.clients[owner]
	r.mu.RUnlock()
	if online {
		ownerClient.SendMessage(*r.buildMapFullState(owner, false))
	}
	return views
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
