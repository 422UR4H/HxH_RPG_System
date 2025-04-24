package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	sheetHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	ccEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
	sheetPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/user"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
	jwtAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/ardanlabs/conf/v3"
	"github.com/danielgtaylor/huma/v2"
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

	authRepo := user.NewRepository(pgPool)
	authHandler := auth.NewAuthHandler(authRepo)

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
	characterSheetsApi := sheetHandler.Api{
		CreateCharacterSheetHandler:  sheetHandler.CreateCharacterSheetHandler(createCharacterSheetUC),
		GetCharacterSheetHandler:     sheetHandler.GetCharacterSheetHandler(getCharacterSheetUC),
		ListClassesHandler:           sheetHandler.ListClassesHandler(listCharacterClassesUC),
		GetClassHandler:              sheetHandler.GetClassHandler(getCharacterClassUC),
		UpdateNenHexagonValueHandler: sheetHandler.UpdateNenHexagonValueHandler(updateNenHexValUC, getCharacterSheetUC),
	}
	chiServer := api.NewServer()

	a := api.Api{
		LivenessHandler:       api.LivenessHandler(),
		ReadinessHandler:      api.ReadinessHandler(),
		CharacterSheetHandler: &characterSheetsApi,
		AuthHandler:           authHandler,
		// Logger:                chiServer.Logger,
	}
	a.Routes(chiServer, authMiddleware)

	server := http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      chiServer,
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
	}

	fmt.Println("Starting server")
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}

type contextKey string

const userIDKey contextKey = "userID"

// TODO: set up debug and run to see where the flow drops after the return
// and move this function there to handle the error
func authMiddleware(ctx huma.Context, next func(huma.Context)) {
	tokenStr := ctx.Header("Authorization")
	if tokenStr == "" {
		// huma.WriteErr(api, ctx, http.StatusUnauthorized,
		// 	"missing token", fmt.Errorf("error detail"),
		// )
		return
	}

	claims, err := jwtAuth.ValidateToken(tokenStr)
	if err != nil {
		// huma.WriteErr(api, ctx, http.StatusUnauthorized,
		// 	"invalid token", fmt.Errorf("error detail"),
		// )
		return
	}

	ctx.AppendHeader(string(userIDKey), claims.UserID.String())

	// fmt.Println("body reader", ctx.BodyReader())
	// fmt.Println("body writer", ctx.BodyWriter())
	// fmt.Println("header auth", ctx.Header("Authorization"))
	next(ctx)
}

func initCharacterClasses() {
	factory := sheet.NewCharacterSheetFactory()
	ccFactory := ccEntity.NewCharacterClassFactory()

	for name, class := range ccFactory.Build() {
		characterClasses.Store(name, class)
	}

	characterClasses.Range(func(key, value any) bool {
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
		newClass, err := factory.Build(profile, set.GetInitialHexValue(), nil, &class)
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
