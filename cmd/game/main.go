package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/422UR4H/HxH_RPG_System/internal/application/enrollment"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	matchboarduc "github.com/422UR4H/HxH_RPG_System/internal/application/matchboard"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	fogPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/fog"
	mapPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
	masteractionPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/masteraction"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	matchboardPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchboard"
	matcheventPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchevent"
	matchmapPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
	roundPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/round"
	sheetPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
	"github.com/joho/godotenv"
)

func main() {
	// TODO: evaluate to action — consider config/env loading
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	addr := os.Getenv("GAME_SERVER_ADDR")
	if addr == "" {
		addr = ":8081"
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	pgPool, err := pgfs.New(ctx, "")
	if err != nil {
		panic(fmt.Errorf("error creating pg pool: %w", err))
	}
	defer pgPool.Close()

	matchRepository := matchPg.NewRepository(pgPool)
	enrollmentRepository := enrollmentPg.NewRepository(pgPool)
	sheetRepository := sheetPg.NewRepository(pgPool)
	roundRepository := roundPg.NewRepository(pgPool)

	startMatchUC := match.NewStartMatchUC(matchRepository)
	kickPlayerUC := enrollment.NewKickPlayerUC(matchRepository, enrollmentRepository)
	closeRoundUC := match.NewCloseRoundUC()
	openNextActionUC := match.NewOpenNextActionUC(closeRoundUC)
	pullActionUC := match.NewPullActionUC(closeRoundUC)
	enqueueActionUC := match.NewEnqueueActionUC()
	attachReactionUC := match.NewAttachReactionUC()
	openReactionUC := match.NewOpenReactionUC()
	closeTurnUC := match.NewCloseTurnUC()
	changeSceneUC := match.NewChangeSceneUC()
	enqueueMasterActionUC := match.NewEnqueueMasterActionUC()
	changeRoundModeUC := match.NewChangeRoundModeUC()
	editActionUC := match.NewEditActionUC()
	// Same assembly as cmd/api/main.go: the WS add_npc verb runs the very same AddMatchNPCUC
	// (same guards, same match_participants write) before injecting the sheet into the live session.
	addMatchNPCUC := match.NewAddMatchNPCUC(matchRepository, sheetRepository, matchRepository)
	addLiveNPCUC := match.NewAddLiveNPCUC(addMatchNPCUC, sheetRepository)

	// The board's own three repositories (spec §4.3, "Quem carrega", B14): what map is
	// attached, the match's own saved board line (if any), and the campaign map it started
	// as a fallback for one that never saved.
	mapRepository := mapPg.NewRepository(pgPool)
	matchMapRepository := matchmapPg.NewRepository(pgPool)
	matchBoardRepository := matchboardPg.NewRepository(pgPool)
	loadBoardUC := matchboarduc.NewLoadMatchBoardUC(matchBoardRepository, matchMapRepository, mapRepository)
	// initSessionUC is built AFTER addMatchNPCUC/loadBoardUC: B11 (spec §4.3) has Init scan the
	// board for a piece whose character is not yet a participant and enroll it as an NPC
	// through the SAME AddMatchNPCUC add_npc and POST /npcs already use — on start_match and on
	// rehydration alike, idempotently.
	initSessionUC := match.NewInitMatchSessionUC(matchRepository, sheetRepository, roundRepository, loadBoardUC, addMatchNPCUC)
	// The write side (spec §4.3, "Quando persiste", B3): the board row and every player's fog
	// memory, saved together by Room.persistBoard. playerMemoryRepository doubles as
	// RoomDeps.MemoryLoader — its FindByMatchMap is what seeds a rehydrated/started session.
	playerMemoryRepository := fogPg.NewPlayerMemoryRepository(pgPool)
	saveBoardUC := matchboarduc.NewSaveMatchBoardUC(matchBoardRepository, playerMemoryRepository)
	// Every accepted enqueue_master_action is recorded the instant it is applied, with what
	// each player saw of it live (spec §4.8) — Room.recordMasterAction is the one writer.
	masterActionRepository := masteractionPg.NewRepository(pgPool)
	// What happens inside a round that is not a turn — the regime change (spec §4.5, B15).
	matchEventRepository := matcheventPg.NewRepository(pgPool)

	hub := game.NewHub()
	// TODO: evaluate to a handler for package
	handler := game.NewHandler(
		hub, matchRepository, enrollmentRepository,
		game.RoomDeps{
			StartMatchUC:          startMatchUC,
			KickPlayerUC:          kickPlayerUC,
			InitSessionUC:         initSessionUC,
			OpenNextActionUC:      openNextActionUC,
			PullActionUC:          pullActionUC,
			EnqueueActionUC:       enqueueActionUC,
			AttachReactionUC:      attachReactionUC,
			OpenReactionUC:        openReactionUC,
			CloseTurnUC:           closeTurnUC,
			ChangeSceneUC:         changeSceneUC,
			RoundRepo:             roundRepository,
			EnqueueMasterActionUC: enqueueMasterActionUC,
			ChangeRoundModeUC:     changeRoundModeUC,
			EditActionUC:          editActionUC,
			AddLiveNPCUC:          addLiveNPCUC,
			LoadBoardUC:           loadBoardUC,
			SaveBoardUC:           saveBoardUC,
			MemoryLoader:          playerMemoryRepository,
			// Same sheetRepository already wired into addMatchNPCUC above — it satisfies
			// appmatch.ISheetOwnershipReader, which is all handlePieceMoved needs to check a
			// lobby player's ownership of an existing piece (spec §4.3, "Quem move o quê", B14).
			SheetOwnership:   sheetRepository,
			MasterActionRepo: masterActionRepository,
			EventRepo:        matchEventRepository,
		},
	)
	server := game.NewServer(addr, hub, handler)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.Start(); err != nil {
			log.Printf("game server error: %v", err)
		}
	}()

	// TODO: verify this before game testing with other players
	log.Printf("game server running on %s", addr)
	<-quit
	log.Println("shutting down game server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("game server shutdown error: %v", err)
	}
	log.Println("game server stopped")
}
