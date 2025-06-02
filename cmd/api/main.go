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
	campaignHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/campaign"
	matchHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	scenarioHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/scenario"
	sheetHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	ccEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	domainScenario "github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	scenarioPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/scenario"
	sheetPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/user"
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
var sessions sync.Map

// TODO: remove or handle after balancing
// var charClassSheets map[enum.CharacterClassName]*sheet.CharacterSheet

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

	// TODO: remove or handle after balancing
	// charClassSheets = make(map[enum.CharacterClassName]*sheet.CharacterSheet)
	initCharacterClasses()

	authRepo := user.NewRepository(pgPool)
	characterSheetRepo := sheetPg.NewRepository(pgPool)
	scenarioRepo := scenarioPg.NewRepository(pgPool)
	campaignRepo := campaignPg.NewRepository(pgPool)
	matchRepo := matchPg.NewRepository(pgPool)

	registerUC := domainAuth.NewRegisterUC(authRepo)
	loginUC := domainAuth.NewLoginUC(&sessions, authRepo)
	authHandler := auth.NewAuthHandler(registerUC, loginUC)

	characterSheetFactory := sheet.NewCharacterSheetFactory()

	getCharacterSheetUC := cs.NewGetCharacterSheetUC(
		&characterSheets,
		characterSheetFactory,
		characterSheetRepo,
	)
	listCharacterSheetsUC := cs.NewListCharacterSheetsUC(
		characterSheetRepo,
	)
	createCharacterSheetUC := cs.NewCreateCharacterSheetUC(
		&characterClasses,
		&characterSheets,
		characterSheetFactory,
		characterSheetRepo,
	)
	submitCharacterSheetUC := cs.NewSubmitCharacterSheetUC(
		characterSheetRepo,
		campaignRepo,
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
		SubmitCharacterSheetHandler:  sheetHandler.SubmitCharacterSheetHandler(submitCharacterSheetUC),
		GetCharacterSheetHandler:     sheetHandler.GetCharacterSheetHandler(getCharacterSheetUC),
		ListCharacterSheetsHandler:   sheetHandler.ListCharacterSheetsHandler(listCharacterSheetsUC),
		ListClassesHandler:           sheetHandler.ListClassesHandler(listCharacterClassesUC),
		GetClassHandler:              sheetHandler.GetClassHandler(getCharacterClassUC),
		UpdateNenHexagonValueHandler: sheetHandler.UpdateNenHexagonValueHandler(updateNenHexValUC, getCharacterSheetUC),
	}

	createScenarioUC := domainScenario.NewCreateScenarioUC(scenarioRepo)
	getScenarioUC := domainScenario.NewGetScenarioUC(scenarioRepo)
	listScenariosUC := domainScenario.NewListScenariosUC(scenarioRepo)

	scenariosApi := scenarioHandler.Api{
		CreateScenarioHandler: scenarioHandler.CreateScenarioHandler(createScenarioUC),
		GetScenarioHandler:    scenarioHandler.GetScenarioHandler(getScenarioUC),
		ListScenariosHandler:  scenarioHandler.ListScenariosHandler(listScenariosUC),
	}

	createCampaignUC := domainCampaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)
	getCampaignUC := domainCampaign.NewGetCampaignUC(campaignRepo)
	listCampaignsUC := domainCampaign.NewListCampaignsUC(campaignRepo)

	campaignsApi := campaignHandler.Api{
		CreateCampaignHandler: campaignHandler.CreateCampaignHandler(createCampaignUC),
		GetCampaignHandler:    campaignHandler.GetCampaignHandler(getCampaignUC),
		ListCampaignsHandler:  campaignHandler.ListCampaignsHandler(listCampaignsUC),
	}

	createMatchUC := domainMatch.NewCreateMatchUC(matchRepo, campaignRepo)
	getMatchUC := domainMatch.NewGetMatchUC(matchRepo)
	listMatchesUC := domainMatch.NewListMatchesUC(matchRepo)

	matchesApi := matchHandler.Api{
		CreateMatchHandler: matchHandler.CreateMatchHandler(createMatchUC),
		GetMatchHandler:    matchHandler.GetMatchHandler(getMatchUC),
		ListMatchesHandler: matchHandler.ListMatchesHandler(listMatchesUC),
	}

	chiServer := api.NewServer()

	a := api.Api{
		LivenessHandler:       api.LivenessHandler(),
		ReadinessHandler:      api.ReadinessHandler(),
		CharacterSheetHandler: &characterSheetsApi,
		ScenarioHandler:       &scenariosApi,
		CampaignHandler:       &campaignsApi,
		MatchHandler:          &matchesApi,
		AuthHandler:           authHandler,
		// Logger:                chiServer.Logger,
	}
	authMiddleware := auth.AuthMiddlewareProvider(&sessions)
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

func initCharacterClasses() {
	ccFactory := ccEntity.NewCharacterClassFactory()

	for name, class := range ccFactory.Build() {
		characterClasses.Store(name, class)
	}

	// uncomment to print all character classes
	// factory := sheet.NewCharacterSheetFactory()
	// characterClasses.Range(func(key, value any) bool {
	// 	name := key.(enum.CharacterClassName)
	// 	class := value.(ccEntity.CharacterClass)
	// 	profile := sheet.CharacterProfile{
	// 		NickName:         name.String(),
	// 		Alignment:        class.Profile.Alignment,
	// 		Description:      class.Profile.Description,
	// 		BriefDescription: class.Profile.BriefDescription,
	// 	}
	// 	set, err := sheet.NewTalentByCategorySet(
	// 		map[enum.CategoryName]bool{
	// 			enum.Reinforcement:   true,
	// 			enum.Transmutation:   true,
	// 			enum.Materialization: true,
	// 			enum.Specialization:  true,
	// 			enum.Manipulation:    true,
	// 			enum.Emission:        true,
	// 		},
	// 		nil,
	// 	)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// 	newClass, err := factory.Build(profile, set.GetInitialHexValue(), nil, &class)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// 	charClassSheets[name] = newClass
	// 	fmt.Println(newClass.ToString())
	// 	return true
	// })
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
