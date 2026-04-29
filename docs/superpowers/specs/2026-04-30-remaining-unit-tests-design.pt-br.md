# Fase 7 — Testes Unitários Restantes (Design)

**Data:** 30/04/2026
**Status:** Implementado
**Escopo:** Todos os pacotes restantes com lógica testável

## Objetivo

Completar a cobertura de testes unitários para todos os pacotes que contêm
lógica testável mas ainda não tinham testes. Esta é a fase final de testes
unitários — após ela, apenas a falha pré-existente em `turn/engine_test.go`
permanece (adiada: refatoração semântica de Turn/Round).

## Pacotes Testados

| Pacote | Testes | Descrição |
|--------|--------|-----------|
| `pkg/auth` | 6 | JWT GenerateToken + ValidateToken (segurança crítica) |
| `internal/domain` | 4 | Helpers de wrapping de erros (NewValidationError, NewDomainError, NewDBError) |
| `internal/domain/entity/campaign` | 2 | Construtor NewCampaign + campos opcionais |
| `internal/domain/entity/scenario` | 2 | Construtor NewScenario + unicidade de UUID |
| `internal/domain/entity/match/scene` | 7 | Ciclo de vida da Scene: criar, AddTurn, FinishScene, caminhos de erro |
| `internal/config` | 3 | Parsing de CORS com env vars — valores padrão e customizados |
| `pkg` | 3 | Config.ConnString() com diferentes modos de SSL |

**Total: 27 casos de teste**

## Pacotes Ignorados (com justificativa)

| Pacote | Motivo |
|--------|--------|
| `match/turn` | Teste pré-existente quebrado — adiado pelo usuário (refatoração semântica WIP) |
| `match/round` | Fortemente acoplado à refatoração de Turn — adiado |
| `match/battle` | Apenas struct, sem lógica testável |
| `entity/user` | Apenas struct + constantes de erro |
| `domain/session` | Apenas definição de interface |
| `domain/testutil` | Utilitários para testes |
| `gateway/pg/*` | Testes de integração existem (Fase 5, build tag `integration`) |
| `cmd/*` | Entry points (main.go) — não testados unitariamente |

## Padrões de Teste

- **Apenas `testing` da biblioteca padrão** — sem frameworks
- **Testes table-driven** com `t.Run()` onde múltiplos casos existem
- **Pacotes de teste externos** (`package X_test`)
- **`t.Setenv()`** para testes de variáveis de ambiente (limpeza automática)
- **Construção zero-value de struct** para `turn.Turn` em testes de Scene (evita lógica quebrada de Turn)

## Resumo de Cobertura

Após esta fase, todo pacote Go com lógica de produção testável tem ao menos
cobertura básica de testes, com a única exceção da área Turn/Round que está
sob refatoração semântica ativa.
