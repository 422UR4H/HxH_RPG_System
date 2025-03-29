package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api"
	sheetHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	ccEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
	sheetPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
	"github.com/ardanlabs/conf/v3"
	"github.com/joho/godotenv"
)

type config struct {
	ServerAddr         string        `conf:"env:SERVER_ADDR,default:localhost:5000"`
	ServerReadTimeout  time.Duration `conf:"default:30s"`
	ServerWriteTimeout time.Duration `conf:"default:30s"`
}

var characterClasses sync.Map
var characterSheets sync.Map

// TODO: remove or handle after balancing
var charClassSheets map[enum.CharacterClassName]*sheet.CharacterSheet

func main() {
	loadEnvFile()

	cfg, err := loadConfig()
	if err != nil {
		panic(fmt.Errorf("error loading config: %w", err))
	}
	fmt.Println(cfg)

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	pgPool, err := pgfs.New(ctx, "")
	if err != nil {
		panic(fmt.Errorf("error creating pg pool: %w", err))
	}
	defer pgPool.Close()

	charClassSheets = make(map[enum.CharacterClassName]*sheet.CharacterSheet)
	initCharacterClasses()

	characterSheetFactory := sheet.NewCharacterSheetFactory()
	characterSheetRepo := sheetPg.NewRepository(pgPool)

	getCharacterSheetUC := cs.NewGetCharacterSheetUC(
		&characterSheets,
		characterSheetFactory,
		characterSheetRepo,
	)
	createCharacterSheetUC := cs.NewCreateCharacterSheetUC(
		&characterClasses,
		&characterSheets,
		characterSheetFactory,
		characterSheetRepo,
	)
	listCharacterClassesUC := cs.NewListCharacterClassesUC(
		&characterClasses,
	)
	getCharacterClassUC := cs.NewGetCharacterClassUC(
		&characterClasses,
	)
	updateNenHexValUC := cs.NewUpdateNenHexagonValueUC(
		&characterSheets,
		characterSheetRepo,
	)

	chiServer := api.NewServer()
	characterSheetsApi := sheetHandler.Api{
		CreateCharacterSheetHandler:  sheetHandler.CreateCharacterSheetHandler(createCharacterSheetUC),
		GetCharacterSheetHandler:     sheetHandler.GetCharacterSheetHandler(getCharacterSheetUC),
		ListClassesHandler:           sheetHandler.ListClassesHandler(listCharacterClassesUC),
		GetClassHandler:              sheetHandler.GetClassHandler(getCharacterClassUC),
		UpdateNenHexagonValueHandler: sheetHandler.UpdateNenHexagonValueHandler(updateNenHexValUC, getCharacterSheetUC),
	}

	a := api.Api{
		LivenessHandler:       api.LivenessHandler(),
		ReadinessHandler:      api.ReadinessHandler(),
		CharacterSheetHandler: &characterSheetsApi,
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

func initCharacterClasses() {
	factory := sheet.NewCharacterSheetFactory()
	ccFactory := ccEntity.NewCharacterClassFactory()

	for name, class := range ccFactory.Build() {
		characterClasses.Store(name, class)
	}

	characterClasses.Range(func(key, value interface{}) bool {
		name := key.(enum.CharacterClassName)
		class := value.(ccEntity.CharacterClass)
		profile := sheet.CharacterProfile{
			NickName:         name.String(),
			Alignment:        class.Profile.Alignment,
			Description:      class.Profile.Description,
			BriefDescription: class.Profile.BriefDescription,
		}
		set, err := sheet.NewTalentByCategorySet(
			map[enum.CategoryName]bool{
				enum.Reinforcement:   true,
				enum.Transmutation:   true,
				enum.Materialization: true,
				enum.Specialization:  true,
				enum.Manipulation:    true,
				enum.Emission:        true,
			},
			nil,
		)
		if err != nil {
			fmt.Println(err)
		}
		newClass, err := factory.Build(profile, set.GetInitialHexValue(), &class)
		if err != nil {
			fmt.Println(err)
		}
		charClassSheets[name] = newClass
		// uncomment to print all character classes
		// fmt.Println(newClass.ToString())
		return true
	})
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
