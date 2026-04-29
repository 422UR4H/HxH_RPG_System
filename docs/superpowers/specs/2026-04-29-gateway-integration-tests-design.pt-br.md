# Fase 5: Testes de Integração do Gateway — Spec de Design

**Data:** 29/04/2026  
**Escopo:** Testes de integração para os 8 pacotes de repositório PostgreSQL  
**Branch:** `feat/gateway-integration-tests`

## Problema

A camada gateway/pg tem 8 repositórios com 38 métodos no total, zero cobertura de testes. Esses repos executam SQL puro contra PostgreSQL via pgx/v5. Testes de integração verificam:
- Queries SQL sintaticamente corretas
- Mapeamento de colunas corresponde ao schema
- Constraints (FK, UNIQUE, CHECK) se comportam como esperado
- Gerenciamento de transações funciona corretamente

## Abordagem

### Estratégia de Banco

- **Banco de teste dedicado:** `hxh_rpg_test` no mesmo PostgreSQL do Docker Compose
- **Conexão:** variável `TEST_DATABASE_URL`, padrão `postgres://postgres:postgres@localhost:5432/hxh_rpg_test?sslmode=disable`
- **Schema:** Migrações Goose aplicadas no TestMain
- **Cleanup:** `TRUNCATE ... CASCADE` entre testes

### Isolamento com Build Tag

Todos os arquivos de teste de integração usam:
```go
//go:build integration
```

Isso garante que `go test ./...` roda apenas testes unitários. Para integração:
```bash
go test -tags=integration ./internal/gateway/pg/...
```

### Infraestrutura de Teste

Pacote `internal/gateway/pg/pgtest/` fornece:
- `SetupTestDB(t *testing.T) *pgxpool.Pool` — conecta, roda migrações, retorna pool
- `TruncateAll(t *testing.T, pool *pgxpool.Pool)` — limpa todas as tabelas entre testes
- `InsertTestUser(...)` — cria usuário pré-requisito
- `InsertTestScenario(...)` — cria cenário pré-requisito
- `InsertTestCampaign(...)` — cria campanha pré-requisito

## Repositórios & Casos de Teste

### user (4 métodos → 8 casos)

| Método | Casos |
|--------|-------|
| CreateUser | caminho feliz; erro email duplicado; erro nick duplicado |
| GetUserByEmail | encontrado; não encontrado |
| ExistsUserWithNick | true; false |
| ExistsUserWithEmail | true; false |

### session (3 métodos → 6 casos)

| Método | Casos |
|--------|-------|
| CreateSession | caminho feliz |
| GetSessionTokenByUserUUID | encontrado; não encontrado |
| ValidateSession | válido; token inválido; sem sessão |

### scenario (5 métodos → 9 casos)

| Método | Casos |
|--------|-------|
| CreateScenario | caminho feliz; nome duplicado |
| GetScenario | encontrado; não encontrado |
| ListScenariosByUserUUID | retorna lista; vazio |
| ExistsScenarioWithName | true; false |
| ExistsScenario | true; false |

### campaign (6 métodos → 10 casos)

| Método | Casos |
|--------|-------|
| CreateCampaign | caminho feliz |
| GetCampaign | encontrado; não encontrado |
| ListCampaignsByMasterUUID | retorna lista; vazio |
| GetCampaignMasterUUID | encontrado; não encontrado |
| GetCampaignStoryDates | encontrado; não encontrado |
| CountCampaignsByMasterUUID | count > 0; count = 0 |

### match (5 métodos → 9 casos)

| Método | Casos |
|--------|-------|
| CreateMatch | caminho feliz |
| GetMatch | encontrado; não encontrado |
| GetMatchCampaignUUID | encontrado; não encontrado |
| ListMatchesByMasterUUID | retorna lista; vazio |
| ListPublicUpcomingMatches | retorna lista; vazio; filtra por data |

### submission (5 métodos → 8 casos)

| Método | Casos |
|--------|-------|
| SubmitCharacterSheet | caminho feliz; duplicado |
| AcceptCharacterSheetSubmission | caminho feliz |
| RejectCharacterSheetSubmission | caminho feliz |
| GetSubmissionCampaignUUIDBySheetUUID | encontrado; não encontrado |
| ExistsSubmittedCharacterSheet | true; false |

### enrollment (2 métodos → 4 casos)

| Método | Casos |
|--------|-------|
| EnrollCharacterSheet | caminho feliz; duplicado |
| ExistsEnrolledCharacterSheet | true; false |

### sheet (8 métodos → 14 casos)

| Método | Casos |
|--------|-------|
| CreateCharacterSheet | caminho feliz (player); caminho feliz (master) |
| GetCharacterSheetByUUID | encontrado; não encontrado |
| GetCharacterSheetPlayerUUID | encontrado; não encontrado |
| GetCharacterSheetRelationshipUUIDs | encontrado; não encontrado |
| ExistsCharacterWithNick | true; false |
| CountCharactersByPlayerUUID | count > 0; count = 0 |
| ListCharacterSheetsByPlayerUUID | retorna lista; vazio |
| UpdateNenHexagonValue | caminho feliz |

**Total: ~68 casos de teste em 8 repositórios**

## Estrutura de Arquivos

```
internal/gateway/pg/
├── pgtest/
│   └── setup.go                  (infraestrutura de teste)
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

## Decisões-Chave

1. **Build tag `integration`** — separa de testes unitários
2. **PostgreSQL real** — sem SQLite ou substitutos em memória
3. **Migrações Goose no TestMain** — schema sempre igual à produção
4. **TRUNCATE CASCADE entre testes** — cleanup rápido e confiável
5. **Helpers para pré-requisitos de FK** — testes de `campaign` auto-criam `user` necessário
6. **Sem testes paralelos** — estado de DB compartilhado, execução sequencial
7. **Dependência nova:** `github.com/pressly/goose/v3` para rodar migrações programaticamente
