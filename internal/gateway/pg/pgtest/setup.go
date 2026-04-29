//go:build integration

package pgtest

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultTestDatabaseURL = "postgres://postgres:postgres@localhost:5432/hxh_rpg_test?sslmode=disable"

func GetDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}
	return defaultTestDatabaseURL
}

func SetupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	dbURL := GetDatabaseURL()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping test database: %v (is PostgreSQL running? try: docker compose up -d)", err)
	}

	runMigrations(t, dbURL)
	TruncateAll(t, pool)

	return pool
}

func runMigrations(t *testing.T, dbURL string) {
	t.Helper()

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatalf("failed to open DB for migrations: %v", err)
	}
	defer db.Close()

	migrationsDir := findMigrationsDir()
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("failed to set goose dialect: %v", err)
	}
	if err := goose.Up(db, migrationsDir); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
}

func findMigrationsDir() string {
	candidates := []string{
		"../../../../migrations",
		"../../../../../migrations",
		"migrations",
	}
	for _, dir := range candidates {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	return "../../../../migrations"
}

func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE enrollments, submissions, sessions,
			matches, campaigns, scenarios,
			joint_proficiencies, proficiencies, character_profiles,
			character_sheets, users
		CASCADE
	`)
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
}

func InsertTestUser(t *testing.T, pool *pgxpool.Pool, nick, email, password string) string {
	t.Helper()
	ctx := context.Background()

	var userUUID string
	err := pool.QueryRow(ctx,
		`INSERT INTO users (nick, email, password) VALUES ($1, $2, $3) RETURNING uuid`,
		nick, email, password,
	).Scan(&userUUID)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}
	return userUUID
}

func InsertTestScenario(t *testing.T, pool *pgxpool.Pool, userUUID, name string) string {
	t.Helper()
	ctx := context.Background()

	var scenarioUUID string
	err := pool.QueryRow(ctx,
		`INSERT INTO scenarios (user_uuid, name, description) VALUES ($1, $2, $3) RETURNING uuid`,
		userUUID, name, "Test scenario description",
	).Scan(&scenarioUUID)
	if err != nil {
		t.Fatalf("failed to insert test scenario: %v", err)
	}
	return scenarioUUID
}

func InsertTestCampaign(t *testing.T, pool *pgxpool.Pool, masterUUID, name string) string {
	t.Helper()
	ctx := context.Background()

	var campaignUUID string
	err := pool.QueryRow(ctx,
		`INSERT INTO campaigns (master_uuid, name, story_start_at)
		 VALUES ($1, $2, CURRENT_DATE) RETURNING uuid`,
		masterUUID, name,
	).Scan(&campaignUUID)
	if err != nil {
		t.Fatalf("failed to insert test campaign: %v", err)
	}
	return campaignUUID
}

func InsertTestMatch(t *testing.T, pool *pgxpool.Pool, masterUUID, campaignUUID, title string) string {
	t.Helper()
	ctx := context.Background()

	var matchUUID string
	err := pool.QueryRow(ctx,
		`INSERT INTO matches (master_uuid, campaign_uuid, title, game_start_at, story_start_at)
		 VALUES ($1, $2, $3, NOW() + INTERVAL '1 day', CURRENT_DATE) RETURNING uuid`,
		masterUUID, campaignUUID, title,
	).Scan(&matchUUID)
	if err != nil {
		t.Fatalf("failed to insert test match: %v", err)
	}
	return matchUUID
}

func InsertTestCharacterSheet(t *testing.T, pool *pgxpool.Pool, playerUUID *string, masterUUID *string, nick string) string {
	t.Helper()
	ctx := context.Background()

	var sheetUUID string
	err := pool.QueryRow(ctx,
		`INSERT INTO character_sheets (player_uuid, master_uuid, category_name)
		 VALUES ($1, $2, 'Reinforcement') RETURNING uuid`,
		playerUUID, masterUUID,
	).Scan(&sheetUUID)
	if err != nil {
		t.Fatalf("failed to insert test character sheet: %v", err)
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO character_profiles (character_sheet_uuid, nickname, fullname, character_class)
		 VALUES ($1, $2, $3, 'Swordsman')`,
		sheetUUID, nick, nick+" FullName",
	)
	if err != nil {
		t.Fatalf("failed to insert test character profile: %v", err)
	}
	return sheetUUID
}
