package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api"
	// charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	// "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
	"github.com/ardanlabs/conf/v3"
	"github.com/joho/godotenv"
)

type config struct {
	ServerAddr         string        `conf:"default:localhost:5000"`
	ServerReadTimeout  time.Duration `conf:"default:30s"`
	ServerWriteTimeout time.Duration `conf:"default:30s"`
}

func main() {
	loadEnvFile()

	cfg, err := loadConfig()
	if err != nil {
		panic(fmt.Errorf("error loading config: %w", err))
	}
	fmt.Println(cfg)

	// ctx, cancelCtx := context.WithCancel(context.Background())
	// defer cancelCtx()

	// pgPool, err := pgfs.New(ctx, "")
	// if err != nil {
	// 	panic(fmt.Errorf("error creating pg pool: %w", err))
	// }

	// initialize usecases
	// repo
	// createCharacterSheetUC := charactersheet.NewCreateCharacterSheetUC(
	// 	api.GetAllCharacterClasses(),
	// 	sheet.NewCharacterSheetFactory(),
	// )
	// other usecases
	chiServer := api.NewServer()
	// characterSheetsApi := api.Api{}

	a := api.Api{
		LivenessHandler:  api.LivenessHandler(),
		ReadinessHandler: api.ReadinessHandler(),
		// CharacterSheetHandler: characterSheetsApi,
		// Logger:                chiServer.Logger,
	}
	a.Routes(chiServer)

	server := http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      chiServer,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
	}

	// logger.Info("Starting server", zap.String("addr", cfg.ServerAddr))
	fmt.Println("Starting server")
	if err := server.ListenAndServe(); err != nil {
		// logger.Error("Server error", zap.Error(err))
		// TODO: remove this
		panic(err)
	}
}

func loadEnvFile() {
	_, err := os.Stat(".env")
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		return
	}
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
}

func loadConfig() (config, error) {
	var cfg config
	if _, err := conf.Parse("", &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}
