# Phase 6 — HTTP Handler Unit Tests Design

## Problem Statement

The HTTP handler layer (`internal/app/api/`) contains 22 endpoints across 7 packages with no test coverage. These handlers are thin adapters (parse request → call UC → map errors → format response), making them ideal for isolated unit testing with mocked use cases.

## Approach

Unit-test every handler using `humatest` (Huma's built-in testing package) with mocked use case interfaces. Tests validate:

- Request parsing and struct tag validation (`required`, `maxLength`)
- Context extraction (`auth.UserIDKey`)
- Error-to-HTTP-status mapping
- Response formatting and structure
- Date/UUID parsing edge cases

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Test framework | `humatest` | Tests through Huma's validation/routing layer — matches production behavior |
| Mock organization | `mocks_test.go` per handler package | Go-idiomatic, no export needed, each package tests its own UCs |
| Auth handling | Context injection | Inject `auth.UserIDKey` directly — keeps tests focused on handler logic |
| Test style | Table-driven `t.Run()` | Standard Go, external test packages (`package X_test`) |
| Auth middleware | Separate test file (own tests) | Tested in isolation with mocked `session.IRepository` |

## Test Structure

```
internal/app/api/
├── health_test.go                          (2 tests)
├── auth/
│   ├── mocks_test.go                       (mock IRegister, ILogin)
│   ├── handler_test.go                     (Register: 5, Login: 5)
│   └── middleware_test.go                  (6 tests)
├── scenario/
│   ├── mocks_test.go                       (mock ICreateScenario, IGetScenario, IListScenarios)
│   ├── create_scenario_test.go             (4 tests)
│   ├── get_scenario_test.go                (4 tests)
│   └── list_scenarios_test.go              (2 tests)
├── campaign/
│   ├── mocks_test.go                       (mock ICreateCampaign, IGetCampaign, IListCampaigns)
│   ├── create_campaign_test.go             (5 tests)
│   ├── get_campaign_test.go                (4 tests)
│   └── list_campaigns_test.go              (2 tests)
├── match/
│   ├── mocks_test.go                       (mock ICreateMatch, IGetMatch, IListMatches, IListPublicUpcomingMatches)
│   ├── create_match_test.go                (6 tests)
│   ├── get_match_test.go                   (4 tests)
│   ├── list_matches_test.go                (2 tests)
│   └── list_public_upcoming_matches_test.go (2 tests)
├── sheet/
│   ├── mocks_test.go                       (mock ICreateCharacterSheet, IGetCharacterSheet, IListCharacterSheets, IListCharacterClasses, IGetCharacterClass, IUpdateNenHexagonValue)
│   ├── create_character_sheet_test.go      (5 tests)
│   ├── get_character_sheet_test.go         (4 tests)
│   ├── list_character_sheets_test.go       (2 tests)
│   ├── list_classes_test.go                (2 tests)
│   ├── get_class_test.go                   (3 tests)
│   └── update_nen_hexagonal_value_test.go  (5 tests)
├── submission/
│   ├── mocks_test.go                       (mock ISubmitCharacterSheet, IAcceptCharacterSheetSubmission, IRejectCharacterSheetSubmission)
│   ├── submit_character_sheet_test.go      (6 tests)
│   ├── accept_sheet_submission_test.go     (5 tests)
│   └── reject_sheet_submission_test.go     (5 tests)
└── enrollment/
    ├── mocks_test.go                       (mock IEnrollCharacterInMatch)
    └── enroll_character_sheet_test.go      (6 tests)
```

**Estimated total: ~96 test cases**

## Mock Pattern

Each `mocks_test.go` uses function-field mocks (Go-idiomatic, no frameworks):

```go
// scenario/mocks_test.go
package scenario_test

import (
    "context"
    domainScenario "github.com/422UR4H/HxH_RPG_System/internal/domain/scenario"
    "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
)

type mockCreateScenario struct {
    fn func(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenario.Scenario, error)
}

func (m *mockCreateScenario) CreateScenario(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenario.Scenario, error) {
    return m.fn(ctx, input)
}
```

## humatest Usage Pattern

```go
func TestCreateScenarioHandler_Success(t *testing.T) {
    _, api := humatest.New(t, huma.DefaultConfig("Test", "1.0.0"))

    mock := &mockCreateScenario{fn: func(ctx context.Context, input *domainScenario.CreateScenarioInput) (*scenario.Scenario, error) {
        return &scenario.Scenario{UUID: uuid.New(), Name: input.Name}, nil
    }}
    handler := scenario.CreateScenarioHandler(mock)

    huma.Register(api, huma.Operation{
        Method: http.MethodPost,
        Path:   "/scenarios",
    }, handler)

    // Make request with injected auth context
    resp := api.Post("/scenarios",
        strings.NewReader(`{"name":"test","brief_description":"desc"}`))

    assert resp.Code == http.StatusCreated
}
```

**Note:** For authenticated endpoints, we inject `auth.UserIDKey` into the context by wrapping the handler or using middleware in the test setup.

## Auth Context Injection

A test helper in each package (or inline) creates context with user UUID:

```go
func contextWithUser(userID uuid.UUID) context.Context {
    return context.WithValue(context.Background(), auth.UserIDKey, userID)
}
```

Since `humatest` controls the context, we'll register a test middleware that injects the user UUID before the handler runs.

## Test Scenarios per Handler Type

### Create endpoints (POST → 201)
1. ✅ Happy path — UC succeeds, returns 201 + body
2. ❌ Missing context user — returns 500
3. ❌ Domain conflict error — returns 409
4. ❌ Validation error — returns 422
5. ❌ Not found (dependency) — returns 404
6. ❌ Generic UC error — returns 500

### Get endpoints (GET → 200)
1. ✅ Happy path — UC succeeds, returns 200 + body
2. ❌ Not found — returns 404
3. ❌ Forbidden (permissions) — returns 403
4. ❌ Generic UC error — returns 500

### List endpoints (GET → 200)
1. ✅ Happy path — returns 200 + array
2. ❌ Generic UC error — returns 500

### Action endpoints (Accept/Reject/Enroll)
1. ✅ Happy path — returns 200/201/204
2. ❌ Invalid UUID parsing — returns 400
3. ❌ Not found — returns 404
4. ❌ Forbidden — returns 403
5. ❌ Conflict — returns 409
6. ❌ Generic UC error — returns 500

## Auth Package Tests

### Handler Tests (Register + Login)
- **Register:** 400 (missing fields), 409 (conflict), 422 (validation), 500 (generic), 201 (success)
- **Login:** 400 (missing fields), 401 (unauthorized), 422 (validation), 500 (generic), 200 (success)

### Middleware Tests (AuthMiddlewareProvider)
1. ❌ Missing Authorization header → 401
2. ❌ Invalid Bearer format → 401
3. ❌ Invalid JWT token → 401
4. ❌ Token not in cache or DB → 401
5. ❌ Token mismatch (cache vs request) → 401
6. ✅ Valid token in cache → passes through

## Implementation Plan

1. Create feature branch `feat/handler-unit-tests`
2. Write design spec (EN + PT-BR) — commit together
3. Create `mocks_test.go` for each handler package
4. Implement test files per handler (parallel agents per package)
5. Add health endpoint tests
6. Run full test suite, fix any issues
7. Commit all tests
8. Verify with `go test ./internal/app/api/...`

## Dependencies

- `github.com/danielgtaylor/huma/v2` (already in go.mod — includes `humatest` subpackage)
- No additional dependencies needed

## File Naming Convention

- `<handler_name>_test.go` — test file per handler
- `mocks_test.go` — mock definitions per package
- All use `package X_test` (external test package)
