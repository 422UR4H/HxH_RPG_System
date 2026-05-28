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
	enrollmentHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/enrollment"
	matchHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	scenarioHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/scenario"
	sheetHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	submissionHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/submission"
	uploadHandler "github.com/422UR4H/HxH_RPG_System/internal/app/api/upload"
	r2gw "github.com/422UR4H/HxH_RPG_System/internal/gateway/r2"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	authUC "github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/enrollment"
	ccEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/scenario"
	"github.com/422UR4H/HxH_RPG_System/internal/application/submission"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	scenarioPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/scenario"
	sessionPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/session"
	sheetPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	submissionPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/submission"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/user"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
	"github.com/ardanlabs/conf/v3"
	"github.com/joho/godotenv"
)

type config struct {
	ServerAddr           string        `conf:"env:SERVER_ADDR,default:localhost:5000"`
	ServerReadTimeout    time.Duration `conf:"default:30s"`
	ServerWriteTimeout   time.Duration `conf:"default:30s"`
	R2AccountID          string        `conf:"env:R2_ACCOUNT_ID"`
	R2AccessKeyID        string        `conf:"env:R2_ACCESS_KEY_ID"`
	R2SecretAccessKey    string        `conf:"env:R2_SECRET_ACCESS_KEY"`
	R2BucketName         string        `conf:"env:R2_BUCKET_NAME"`
	R2PublicURL          string        `conf:"env:R2_PUBLIC_URL"`
}

var dryCharacterClasses sync.Map
var characterClasses sync.Map
var characterSheets sync.Map
var sessions sync.Map

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

	r2Client := r2gw.NewClient(
		cfg.R2AccountID,
		cfg.R2AccessKeyID,
		cfg.R2SecretAccessKey,
		cfg.R2BucketName,
		cfg.R2PublicURL,
	)

	initDryCharacterClasses()

	authRepo := user.NewRepository(pgPool)
	sessionRepo := sessionPg.NewRepository(pgPool)
	characterSheetRepo := sheetPg.NewRepository(pgPool)
	scenarioRepo := scenarioPg.NewRepository(pgPool)
	campaignRepo := campaignPg.NewRepository(pgPool)
	matchRepo := matchPg.NewRepository(pgPool)
	submitRepo := submissionPg.NewRepository(pgPool)
	enrollmentRepo := enrollmentPg.NewRepository(pgPool)

	registerUC := authUC.NewRegisterUC(authRepo)
	loginUC := authUC.NewLoginUC(&sessions, authRepo, sessionRepo)
	authHandler := auth.NewAuthHandler(registerUC, loginUC)

	characterSheetFactory := sheet.NewCharacterSheetFactory()
	initCharacterClasses(characterSheetFactory)

	getCharacterSheetUC := cs.NewGetCharacterSheetUC(
		&characterSheets,
		characterSheetFactory,
		characterSheetRepo,
		campaignRepo,
		submitRepo,
	)
	listCharacterSheetsUC := cs.NewListCharacterSheetsUC(
		characterSheetRepo,
	)
	createCharacterSheetUC := cs.NewCreateCharacterSheetUC(
		&dryCharacterClasses,
		&characterSheets,
		characterSheetFactory,
		characterSheetRepo,
		campaignRepo,
	)
	listCharacterClassesUC := cs.NewListCharacterClassesUC(
		&dryCharacterClasses,
		&characterClasses,
	)
	getCharacterClassUC := cs.NewGetCharacterClassUC(
		&dryCharacterClasses,
		&characterClasses,
	)
	updateNenHexValUC := cs.NewUpdateNenHexagonValueUC(
		&characterSheets,
		characterSheetRepo,
	)
	deleteCharacterSheetUC := cs.NewDeleteCharacterSheetUC(characterSheetRepo, submitRepo)
	updateCharacterSheetUC := cs.NewUpdateCharacterSheetUC(
		&dryCharacterClasses,
		characterSheetFactory,
		characterSheetRepo,
		submitRepo,
	)
	characterSheetsApi := sheetHandler.Api{
		CreateCharacterSheetHandler:       sheetHandler.CreateCharacterSheetHandler(createCharacterSheetUC),
		GetCharacterSheetHandler:          sheetHandler.GetCharacterSheetHandler(getCharacterSheetUC, submitRepo),
		ListCharacterSheetsHandler:        sheetHandler.ListCharacterSheetsHandler(listCharacterSheetsUC),
		ListClassesHandler:                sheetHandler.ListClassesHandler(listCharacterClassesUC),
		GetClassHandler:                   sheetHandler.GetClassHandler(getCharacterClassUC),
		UpdateNenHexagonValueHandler:      sheetHandler.UpdateNenHexagonValueHandler(updateNenHexValUC, getCharacterSheetUC),
		PatchCharacterSheetProfileHandler: sheetHandler.PatchCharacterSheetProfileHandler(characterSheetRepo),
		DeleteCharacterSheetHandler:       sheetHandler.DeleteCharacterSheetHandler(deleteCharacterSheetUC),
		UpdateCharacterSheetHandler:       sheetHandler.UpdateCharacterSheetHandler(updateCharacterSheetUC, getCharacterSheetUC),
	}

	uploadApi := &uploadHandler.Api{
		PresignedURLHandler: uploadHandler.PresignedURLHandler(r2Client),
	}

	createScenarioUC := scenario.NewCreateScenarioUC(scenarioRepo)
	getScenarioUC := scenario.NewGetScenarioUC(scenarioRepo)
	listScenariosUC := scenario.NewListScenariosUC(scenarioRepo)

	scenariosApi := scenarioHandler.Api{
		CreateScenarioHandler: scenarioHandler.CreateScenarioHandler(createScenarioUC),
		GetScenarioHandler:    scenarioHandler.GetScenarioHandler(getScenarioUC),
		ListScenariosHandler:  scenarioHandler.ListScenariosHandler(listScenariosUC),
	}

	createCampaignUC := campaign.NewCreateCampaignUC(campaignRepo, scenarioRepo)
	getCampaignUC := campaign.NewGetCampaignUC(campaignRepo)
	listCampaignsUC := campaign.NewListCampaignsUC(campaignRepo)
	listPublicUpcomingCampaignsUC := campaign.NewListPublicUpcomingCampaignsUC(campaignRepo)
	deleteCampaignUC := campaign.NewDeleteCampaignUC(campaignRepo)
	updateCampaignUC := campaign.NewUpdateCampaignUC(campaignRepo)
	listPlayerEnrollmentsForCampaignUC := enrollment.NewListPlayerEnrollmentsForCampaignUC(enrollmentRepo)

	campaignsApi := campaignHandler.Api{
		CreateCampaignHandler:              campaignHandler.CreateCampaignHandler(createCampaignUC),
		GetCampaignHandler:                 campaignHandler.GetCampaignHandler(getCampaignUC, listPlayerEnrollmentsForCampaignUC),
		ListCampaignsHandler:               campaignHandler.ListCampaignsHandler(listCampaignsUC),
		ListPublicUpcomingCampaignsHandler: campaignHandler.ListPublicUpcomingCampaignsHandler(listPublicUpcomingCampaignsUC),
		DeleteCampaignHandler:              campaignHandler.DeleteCampaignHandler(deleteCampaignUC),
		UpdateCampaignHandler:              campaignHandler.UpdateCampaignHandler(updateCampaignUC),
	}

	createMatchUC := match.NewCreateMatchUC(matchRepo, campaignRepo)
	updateMatchUC := match.NewUpdateMatchUC(matchRepo, campaignRepo)
	deleteMatchUC := match.NewDeleteMatchUC(matchRepo)
	getMatchUC := match.NewGetMatchUC(matchRepo, characterSheetRepo)
	listMatchesUC := match.NewListMatchesUC(matchRepo)
	listPublicUpcomingMatchesUC := match.NewListPublicUpcomingMatchesUC(matchRepo)
	listMatchEnrollmentsUC := match.NewListMatchEnrollmentsUC(matchRepo, enrollmentRepo, characterSheetRepo)
	getMatchParticipantsUC := match.NewGetMatchParticipantsUC(matchRepo, characterSheetRepo)

	matchesApi := matchHandler.Api{
		CreateMatchHandler:               matchHandler.CreateMatchHandler(createMatchUC),
		UpdateMatchHandler:               matchHandler.UpdateMatchHandler(updateMatchUC),
		DeleteMatchHandler:               matchHandler.DeleteMatchHandler(deleteMatchUC),
		GetMatchHandler:                  matchHandler.GetMatchHandler(getMatchUC),
		ListMatchesHandler:               matchHandler.ListMatchesHandler(listMatchesUC),
		ListPublicUpcomingMatchesHandler: matchHandler.ListPublicUpcomingMatchesHandler(listPublicUpcomingMatchesUC),
		ListMatchEnrollmentsHandler:      matchHandler.ListMatchEnrollmentsHandler(listMatchEnrollmentsUC),
		GetMatchParticipantsHandler:      matchHandler.GetMatchParticipantsHandler(getMatchParticipantsUC),
	}

	submitCharacterSheetUC := submission.NewSubmitCharacterSheetUC(
		submitRepo,
		characterSheetRepo,
		campaignRepo,
	)
	acceptCharacterSheetSubmissionUC := submission.NewAcceptCharacterSheetSubmissionUC(
		submitRepo,
		campaignRepo,
		characterSheetRepo,
	)
	rejectCharacterSheetSubmissionUC := submission.NewRejectCharacterSheetSubmissionUC(
		submitRepo,
		campaignRepo,
	)
	submissionsApi := submissionHandler.Api{
		SubmitCharacterSheetHandler:  submissionHandler.SubmitCharacterSheetHandler(submitCharacterSheetUC),
		AcceptSheetSubmissionHandler: submissionHandler.AcceptSheetSubmissionHandler(acceptCharacterSheetSubmissionUC),
		RejectSheetSubmissionHandler: submissionHandler.RejectSheetSubmissionHandler(rejectCharacterSheetSubmissionUC),
	}

	enrollCharacterSheetUC := enrollment.NewEnrollCharacterInMatchUC(
		enrollmentRepo,
		matchRepo,
		characterSheetRepo,
	)
	acceptEnrollmentUC := enrollment.NewAcceptEnrollmentUC(
		enrollmentRepo,
		matchRepo,
		campaignRepo,
	)
	rejectEnrollmentUC := enrollment.NewRejectEnrollmentUC(
		enrollmentRepo,
		matchRepo,
		campaignRepo,
	)
	enrollmentApi := enrollmentHandler.Api{
		EnrollCharacterHandler:  enrollmentHandler.EnrollCharacterHandler(enrollCharacterSheetUC),
		AcceptEnrollmentHandler: enrollmentHandler.AcceptEnrollmentHandler(acceptEnrollmentUC),
		RejectEnrollmentHandler: enrollmentHandler.RejectEnrollmentHandler(rejectEnrollmentUC),
	}

	chiServer := api.NewServer()

	a := api.Api{
		LivenessHandler:       api.LivenessHandler(),
		ReadinessHandler:      api.ReadinessHandler(),
		CharacterSheetHandler: &characterSheetsApi,
		ScenarioHandler:       &scenariosApi,
		CampaignHandler:       &campaignsApi,
		MatchHandler:          &matchesApi,
		SubmissionHandler:     &submissionsApi,
		EnrollmentHandler:     &enrollmentApi,
		UploadHandler:         uploadApi,
		AuthHandler:           authHandler,
		// Logger:                chiServer.Logger,
	}
	authMiddleware := auth.AuthMiddlewareProvider(&sessions, sessionRepo)
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

func initDryCharacterClasses() {
	ccFactory := ccEntity.NewCharacterClassFactory()

	for name, class := range ccFactory.Build() {
		dryCharacterClasses.Store(name, class)
	}
}

func initCharacterClasses(sheetFactory *sheet.CharacterSheetFactory) {
	dryCharacterClasses.Range(func(key, value any) bool {
		name := key.(enum.CharacterClassName)
		class := value.(ccEntity.CharacterClass)

		profile := sheet.CharacterProfile{
			NickName:         name.String(),
			Alignment:        class.Profile.Alignment,
			Description:      class.Profile.Description,
			BriefDescription: class.Profile.BriefDescription,
		}
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

		classRef := &class
		charClass, err := sheetFactory.BuildHalfSheet(profile, classRef)
		if err != nil {
			panic(domain.NewDomainError(fmt.Errorf(
				"error building character class %s: %w", name, err),
			))
		}
		characterClasses.Store(name, charClass)
		// uncomment to print all character classes
		// fmt.Println(charClass.ToString())
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
