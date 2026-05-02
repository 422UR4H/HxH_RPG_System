---
applyTo: "internal/gateway/pg/**"
---

# Integration Test Conventions

## Structure

- File: `internal/gateway/pg/<package>/<package>_integration_test.go`
- Build tag: `//go:build integration` (first line)
- External test package: `package <pkg>_test`
- One file per gateway package; group related tests in same file

## Setup Pattern

```go
func TestFeatureName(t *testing.T) {
    pool := pgtest.SetupTestDB(t)
    repo := pkg.NewRepository(pool)
    ctx := context.Background()

    // Shared fixtures (users, campaigns, etc.)
    masterUUID := pgtest.InsertTestUser(t, pool, "gm1", "gm1@test.com", "pass")

    t.Run("happy path", func(t *testing.T) {
        pgtest.TruncateAll(t, pool) // isolation between sub-tests
        // ... test logic
    })
}
```

## Key Rules

1. Each sub-test calls `pgtest.TruncateAll(t, pool)` for DB isolation
2. Use `pgtest.InsertTest*` helpers to create prerequisite data
3. Assert domain errors with `errors.Is(err, domain.ErrXxx)`
4. Use `uuid.New().String()` for non-existent UUIDs in "not found" tests
5. Truncate time to microsecond (`time.Truncate(time.Microsecond)`) for PG comparison

## Available pgtest Helpers

| Helper | Returns | Purpose |
|--------|---------|---------|
| `SetupTestDB(t)` | `*pgxpool.Pool` | Connect, migrate, truncate |
| `TruncateAll(t, pool)` | — | Clean all tables (CASCADE) |
| `InsertTestUser(t, pool, nick, email, pass)` | UUID string | Auth user |
| `InsertTestScenario(t, pool, userUUID, name)` | UUID string | Scenario |
| `InsertTestCampaign(t, pool, masterUUID, name)` | UUID string | Campaign |
| `InsertTestMatch(t, pool, masterUUID, campaignUUID, title)` | UUID string | Match |
| `InsertTestCharacterSheet(t, pool, playerUUID*, masterUUID*, nick)` | UUID string | Sheet + profile |
| `InsertTestEnrollment(t, pool, matchUUID, sheetUUID, status)` | UUID string | Enrollment |

## Running

```bash
# All integration tests
go test -tags=integration ./internal/gateway/pg/...

# Specific package
go test -tags=integration ./internal/gateway/pg/match/...

# Vet (includes integration files)
go vet -tags=integration ./internal/gateway/pg/...
```

## Database

- Default: `postgres://postgres:postgres@localhost:5432/hxh_rpg_test?sslmode=disable`
- Override: `TEST_DATABASE_URL` env var
- Migrations: auto-applied via goose from `../../../../migrations` (relative to test pkg)

## Test Naming

- Function: `TestOperationName` (e.g., `TestCreateMatch`, `TestAcceptEnrollment`)
- Sub-tests: descriptive lowercase (e.g., `"happy path"`, `"not found"`, `"already started"`)
