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
	"github.com/422UR4H/HxH_RPG_System/internal/domain/enrollment"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
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

	startMatchUC := domainMatch.NewStartMatchUC(matchRepository, enrollmentRepository)
	kickPlayerUC := enrollment.NewKickPlayerUC(matchRepository, enrollmentRepository)

	hub := game.NewHub()
	// TODO: evaluate to a handler for package
	handler := game.NewHandler(hub, matchRepository, enrollmentRepository, startMatchUC, kickPlayerUC)
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
