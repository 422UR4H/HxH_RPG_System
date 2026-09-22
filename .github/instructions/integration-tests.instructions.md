---
applyTo: "internal/**"
---

# Testing Strategy

## TDD by Layer

| Layer | TDD with | Why |
|-------|----------|-----|
| `entity/` | Unit tests | Pure logic, no I/O |
| `domain/` services (engines → future refactor to domain services) | Unit tests | Pure domain logic integrating entities; no I/O, no mocks needed |
| `domain/` use cases | Unit tests (mocks) | Orchestrate I/O + domain; mock repos |
| `gateway/pg/` | **Integration tests** | Real SQL matters; mocks don't catch query bugs |
| `app/` (handlers) | Unit tests (mocks) | HTTP/WS contract |

When implementing a full slice: TDD each layer with its appropriate test type.
Integration tests for gateways are written **during gateway TDD**, not deferred.

## Integration Test Conventions

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

## Mandatory Verification (per task)

After every task that touches `internal/`, run vet before committing — no DB required, catches build errors in integration-tagged files:

```bash
go vet -tags=integration ./internal/gateway/pg/...
```

After modifying gateway code directly, also run the full suite — **with `-p 1`, see below**:

```bash
go test -tags=integration -p 1 ./internal/gateway/pg/...
```

Do **not** poll CI proactively — only check if a failure is reported.

## Running

```bash
# All integration tests — MUST run with -p 1 (see "Why -p 1" below)
go test -tags=integration -p 1 ./internal/gateway/pg/...

# Specific package — -p 1 not needed, only one package binary runs
go test -tags=integration ./internal/gateway/pg/match/...

# Vet (includes integration files) — no DB access, -p 1 not needed
go vet -tags=integration ./internal/gateway/pg/...
```

## Por que `-p 1`

**Sintoma:** rodar `go test -tags=integration ./internal/gateway/pg/...` sem `-p 1`
falha de forma intermitente — deadlock de conexões e violação de foreign key,
espalhados entre pacotes diferentes (campanha, usuário, sessão, mapa, submissão,
cenário). Os mesmos 241 testes, na mesma branch, passam em série.

**Causa:** todos os pacotes de `internal/gateway/pg/` apontam para o **mesmo banco**
(`pgtest.SetupTestDB`/`GetDatabaseURL` — ver acima), e cada um trunca e insere nas
**mesmas tabelas** via `pgtest.TruncateAll`. `go test` roda pacotes diferentes em
paralelo por padrão (o número de binários simultâneos é controlado por `-p`, que por
padrão é `GOMAXPROCS`) — então um `TRUNCATE ... CASCADE` de um pacote colide com um
`INSERT`/`SELECT` em andamento de outro, contra as mesmas tabelas, no mesmo banco.
Isso não é sobre paralelismo dentro de um pacote (sub-testes já se isolam com
`TruncateAll` entre si); é paralelismo **entre pacotes**, que a suíte não foi desenhada
para suportar porque o isolamento é por truncagem, não por schema/banco por pacote.

**`go test -tags=integration ./...` (ou `./internal/gateway/pg/...`) sem `-p 1` vai
falhar, de forma intermitente, e isso não é bug no código que você acabou de mexer** —
é a causa acima. Antes de sair depurando a mudança, rode de novo com `-p 1` primeiro:

```bash
go test -tags=integration -p 1 ./internal/gateway/pg/...
```

Se ainda falhar com `-p 1`, aí sim é um bug de verdade. A flag já está setada em
`.github/workflows/ci.yml` (passo "Integration tests") e no alvo `test-integration` do
`Makefile` — não remova, mesmo que pareça sobra numa limpeza de CI.

## Database

- Default: `postgres://postgres:postgres@localhost:5432/hxh_rpg_test?sslmode=disable`
- Override: `TEST_DATABASE_URL` env var
- Migrations: auto-applied via goose from `../../../../migrations` (relative to test pkg)

## Test Naming

- Function: `TestOperationName` (e.g., `TestCreateMatch`, `TestAcceptEnrollment`)
- Sub-tests: descriptive lowercase (e.g., `"happy path"`, `"not found"`, `"already started"`)
