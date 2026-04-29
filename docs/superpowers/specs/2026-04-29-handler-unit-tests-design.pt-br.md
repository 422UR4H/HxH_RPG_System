# Fase 6 — Testes Unitários de Handlers HTTP

## Declaração do Problema

A camada de handlers HTTP (`internal/app/api/`) contém 22 endpoints em 7 packages sem cobertura de testes. Os handlers são adaptadores finos (parsear request → chamar UC → mapear erros → formatar response), tornando-os ideais para testes unitários isolados com use cases mockados.

## Abordagem

Testar unitariamente cada handler usando `humatest` (pacote de testes built-in do Huma) com interfaces de use cases mockadas. Os testes validam:

- Parsing de request e validação de struct tags (`required`, `maxLength`)
- Extração de contexto (`auth.UserIDKey`)
- Mapeamento de erros para status HTTP
- Formatação e estrutura da response
- Edge cases de parsing de datas/UUIDs

## Decisões de Arquitetura

| Decisão | Escolha | Racional |
|---------|---------|----------|
| Framework de teste | `humatest` | Testa através da camada de validação/roteamento do Huma — corresponde ao comportamento em produção |
| Organização de mocks | `mocks_test.go` por package de handler | Go-idiomático, sem export necessário, cada package testa seus próprios UCs |
| Tratamento de auth | Injeção no context | Injetar `auth.UserIDKey` diretamente — mantém testes focados na lógica do handler |
| Estilo de teste | Table-driven `t.Run()` | Go padrão, packages de teste externos (`package X_test`) |
| Auth middleware | Arquivo de teste separado | Testado em isolamento com `session.IRepository` mockado |

## Estrutura dos Testes

```
internal/app/api/
├── health_test.go                          (2 testes)
├── auth/
│   ├── mocks_test.go                       (mock IRegister, ILogin)
│   ├── handler_test.go                     (Register: 5, Login: 5)
│   └── middleware_test.go                  (6 testes)
├── scenario/
│   ├── mocks_test.go                       (mock ICreateScenario, IGetScenario, IListScenarios)
│   ├── create_scenario_test.go             (4 testes)
│   ├── get_scenario_test.go                (4 testes)
│   └── list_scenarios_test.go              (2 testes)
├── campaign/
│   ├── mocks_test.go                       (mock ICreateCampaign, IGetCampaign, IListCampaigns)
│   ├── create_campaign_test.go             (5 testes)
│   ├── get_campaign_test.go                (4 testes)
│   └── list_campaigns_test.go              (2 testes)
├── match/
│   ├── mocks_test.go                       (mock ICreateMatch, IGetMatch, IListMatches, IListPublicUpcomingMatches)
│   ├── create_match_test.go                (6 testes)
│   ├── get_match_test.go                   (4 testes)
│   ├── list_matches_test.go                (2 testes)
│   └── list_public_upcoming_matches_test.go (2 testes)
├── sheet/
│   ├── mocks_test.go                       (mock ICreateCharacterSheet, IGetCharacterSheet, IListCharacterSheets, IListCharacterClasses, IGetCharacterClass, IUpdateNenHexagonValue)
│   ├── create_character_sheet_test.go      (5 testes)
│   ├── get_character_sheet_test.go         (4 testes)
│   ├── list_character_sheets_test.go       (2 testes)
│   ├── list_classes_test.go                (2 testes)
│   ├── get_class_test.go                   (3 testes)
│   └── update_nen_hexagonal_value_test.go  (5 testes)
├── submission/
│   ├── mocks_test.go                       (mock ISubmitCharacterSheet, IAcceptCharacterSheetSubmission, IRejectCharacterSheetSubmission)
│   ├── submit_character_sheet_test.go      (6 testes)
│   ├── accept_sheet_submission_test.go     (5 testes)
│   └── reject_sheet_submission_test.go     (5 testes)
└── enrollment/
    ├── mocks_test.go                       (mock IEnrollCharacterInMatch)
    └── enroll_character_sheet_test.go      (6 testes)
```

**Total estimado: ~96 casos de teste**

## Padrão de Mock

Cada `mocks_test.go` usa mocks com campos de função (Go-idiomático, sem frameworks):

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

## Padrão de Uso do humatest

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

    resp := api.Post("/scenarios",
        strings.NewReader(`{"name":"test","brief_description":"desc"}`))

    assert resp.Code == http.StatusCreated
}
```

**Nota:** Para endpoints autenticados, injetamos `auth.UserIDKey` no context usando middleware de teste.

## Injeção de Contexto Auth

Um helper de teste em cada package cria context com user UUID:

```go
func contextWithUser(userID uuid.UUID) context.Context {
    return context.WithValue(context.Background(), auth.UserIDKey, userID)
}
```

Como `humatest` controla o context, registramos um middleware de teste que injeta o user UUID antes do handler executar.

## Cenários de Teste por Tipo de Handler

### Endpoints Create (POST → 201)
1. ✅ Happy path — UC sucede, retorna 201 + body
2. ❌ Contexto sem user — retorna 500
3. ❌ Erro de conflito no domínio — retorna 409
4. ❌ Erro de validação — retorna 422
5. ❌ Não encontrado (dependência) — retorna 404
6. ❌ Erro genérico do UC — retorna 500

### Endpoints Get (GET → 200)
1. ✅ Happy path — UC sucede, retorna 200 + body
2. ❌ Não encontrado — retorna 404
3. ❌ Proibido (permissões) — retorna 403
4. ❌ Erro genérico do UC — retorna 500

### Endpoints List (GET → 200)
1. ✅ Happy path — retorna 200 + array
2. ❌ Erro genérico do UC — retorna 500

### Endpoints de Ação (Accept/Reject/Enroll)
1. ✅ Happy path — retorna 200/201/204
2. ❌ UUID inválido — retorna 400
3. ❌ Não encontrado — retorna 404
4. ❌ Proibido — retorna 403
5. ❌ Conflito — retorna 409
6. ❌ Erro genérico do UC — retorna 500

## Testes do Package Auth

### Testes de Handler (Register + Login)
- **Register:** 400 (campos faltando), 409 (conflito), 422 (validação), 500 (genérico), 201 (sucesso)
- **Login:** 400 (campos faltando), 401 (não autorizado), 422 (validação), 500 (genérico), 200 (sucesso)

### Testes de Middleware (AuthMiddlewareProvider)
1. ❌ Header Authorization ausente → 401
2. ❌ Formato Bearer inválido → 401
3. ❌ Token JWT inválido → 401
4. ❌ Token não encontrado no cache nem no DB → 401
5. ❌ Token incompatível (cache vs request) → 401
6. ✅ Token válido no cache → passa adiante

## Plano de Implementação

1. Criar feature branch `feat/handler-unit-tests`
2. Escrever design spec (EN + PT-BR) — commit juntos
3. Criar `mocks_test.go` para cada package de handler
4. Implementar arquivos de teste por handler (agentes paralelos por package)
5. Adicionar testes de health endpoint
6. Rodar suite completa de testes, corrigir problemas
7. Commitar todos os testes
8. Verificar com `go test ./internal/app/api/...`

## Dependências

- `github.com/danielgtaylor/huma/v2` (já no go.mod — inclui subpackage `humatest`)
- Nenhuma dependência adicional necessária

## Convenção de Nomes de Arquivo

- `<nome_handler>_test.go` — arquivo de teste por handler
- `mocks_test.go` — definições de mock por package
- Todos usam `package X_test` (package de teste externo)
