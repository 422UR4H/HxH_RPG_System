# Phase 5: Gateway Integration Tests — Design Spec

**Date:** 2026-04-29  
**Scope:** Integration tests for all 8 PostgreSQL repository packages  
**Branch:** `feat/gateway-integration-tests`

## Problem

The gateway/pg layer has 8 repositories with 38 methods total, zero test coverage. These repos execute raw SQL against PostgreSQL via pgx/v5. Integration tests verify:
- SQL queries are syntactically correct
- Column mappings match the schema
- Constraints (FK, UNIQUE, CHECK) behave as expected
- Transaction handling works correctly

## Approach

### Database Strategy

- **Dedicated test database:** `hxh_rpg_test` on the same Docker Compose PostgreSQL
- **Connection:** `TEST_DATABASE_URL` env var, defaults to `postgres://postgres:postgres@localhost:5432/hxh_rpg_test?sslmode=disable`
- **Schema:** Goose migrations applied in TestMain
- **Cleanup:** `TRUNCATE ... CASCADE` between tests

### Build Tag Isolation

All integration test files use:
```go
//go:build integration
```

This ensures `go test ./...` only runs unit tests. Integration tests require:
```bash
go test -tags=integration ./internal/gateway/pg/...
```

### Test Infrastructure

Package `internal/gateway/pg/pgtest/` provides:
- `SetupTestDB(t *testing.T) *pgxpool.Pool` — connects, runs migrations, returns pool
- `TruncateAll(t *testing.T, pool *pgxpool.Pool)` — cleans all tables between tests
- `InsertTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID` — creates prerequisite user
- `InsertTestScenario(t *testing.T, pool *pgxpool.Pool, userUUID uuid.UUID) uuid.UUID` — creates scenario
- `InsertTestCampaign(t *testing.T, pool *pgxpool.Pool, masterUUID uuid.UUID) uuid.UUID` — creates campaign

### Makefile Integration

```makefile
test-integration:
	@docker compose up -d db
	@sleep 2
	@createdb -h localhost -U postgres hxh_rpg_test 2>/dev/null || true
	@go test -tags=integration ./internal/gateway/pg/...
```

## Repositories & Test Cases

### user (4 methods → 8 cases)

| Method | Cases |
|--------|-------|
| CreateUser | happy path; duplicate email error; duplicate nick error |
| GetUserByEmail | found; not found |
| ExistsUserWithNick | exists true; exists false |
| ExistsUserWithEmail | exists true; exists false (shared with above) |

### session (3 methods → 6 cases)

| Method | Cases |
|--------|-------|
| CreateSession | happy path |
| GetSessionTokenByUserUUID | found; not found |
| ValidateSession | valid; invalid token; no session |

### scenario (5 methods → 9 cases)

| Method | Cases |
|--------|-------|
| CreateScenario | happy path; duplicate name error |
| GetScenario | found; not found |
| ListScenariosByUserUUID | returns list; empty |
| ExistsScenarioWithName | true; false |
| ExistsScenario | true; false |

### campaign (6 methods → 10 cases)

| Method | Cases |
|--------|-------|
| CreateCampaign | happy path |
| GetCampaign | found; not found |
| ListCampaignsByMasterUUID | returns list; empty |
| GetCampaignMasterUUID | found; not found |
| GetCampaignStoryDates | found; not found |
| CountCampaignsByMasterUUID | count > 0; count = 0 |

### match (5 methods → 9 cases)

| Method | Cases |
|--------|-------|
| CreateMatch | happy path |
| GetMatch | found; not found |
| GetMatchCampaignUUID | found; not found |
| ListMatchesByMasterUUID | returns list; empty |
| ListPublicUpcomingMatches | returns list; empty; filters by date |

### submission (5 methods → 8 cases)

| Method | Cases |
|--------|-------|
| SubmitCharacterSheet | happy path; duplicate error |
| AcceptCharacterSheetSubmission | happy path |
| RejectCharacterSheetSubmission | happy path |
| GetSubmissionCampaignUUIDBySheetUUID | found; not found |
| ExistsSubmittedCharacterSheet | true; false |

### enrollment (2 methods → 4 cases)

| Method | Cases |
|--------|-------|
| EnrollCharacterSheet | happy path; duplicate error |
| ExistsEnrolledCharacterSheet | true; false |

### sheet (8 methods → 14 cases)

| Method | Cases |
|--------|-------|
| CreateCharacterSheet | happy path (player); happy path (master) |
| GetCharacterSheetByUUID | found; not found |
| GetCharacterSheetPlayerUUID | found; not found |
| GetCharacterSheetRelationshipUUIDs | found; not found |
| ExistsCharacterWithNick | true; false |
| CountCharactersByPlayerUUID | count > 0; count = 0 |
| ListCharacterSheetsByPlayerUUID | returns list; empty |
| UpdateNenHexagonValue | happy path |

**Total: ~68 test cases across 8 repositories**

## File Structure

```
internal/gateway/pg/
├── pgtest/
│   └── setup.go                  (test infrastructure)
├── user/
│   └── user_integration_test.go
├── session/
│   └── session_integration_test.go
├── scenario/
│   └── scenario_integration_test.go
├── campaign/
│   └── campaign_integration_test.go
├── match/
│   └── match_integration_test.go
├── submission/
│   └── submission_integration_test.go
├── enrollment/
│   └── enrollment_integration_test.go
└── sheet/
    └── sheet_integration_test.go
```

## Key Design Decisions

1. **Build tag `integration`** — separates from unit tests cleanly
2. **Real PostgreSQL** — no SQLite or in-memory substitutes
3. **Goose migrations in TestMain** — schema always matches production
4. **TRUNCATE CASCADE between tests** — fast, reliable cleanup
5. **Helper functions for FK prerequisites** — tests for `campaign` auto-create required `user`
6. **External test packages** — test files use the package they test (internal access for SQL)
7. **No parallel tests** — shared DB state, sequential execution per package

## Dependencies

- Existing: `pgx/v5`, `pgxpool`, Goose migrations
- New: `github.com/pressly/goose/v3` (programmatic migration runner in tests)
- Docker Compose PostgreSQL must be running
