# Testes de Use Cases do Domínio — Especificação de Design

## Objetivo

Fornecer cobertura completa de testes unitários para todos os 16 use cases de domínio em 6 pacotes: `scenario`, `campaign`, `match`, `auth`, `submission` e `enrollment`.

## Arquitetura

### Infraestrutura de Mocks

Todos os testes utilizam implementações de mock manuais localizadas em `internal/domain/testutil/`. Cada struct de mock possui campos de função configuráveis (ex: `CreateScenarioFn`) que permitem aos testes controlar valores de retorno e erros por caso de teste.

Arquivos de mock:
- `mock_scenario_repo.go` — implementa `scenario.IRepository`
- `mock_campaign_repo.go` — implementa `campaign.IRepository`
- `mock_match_repo.go` — implementa `match.IRepository`
- `mock_auth_repo.go` — implementa `auth.IRepository`
- `mock_session_repo.go` — implementa `session.IRepository`
- `mock_submission_repo.go` — implementa `submission.IRepository`
- `mock_enrollment_repo.go` — implementa `enrollment.IRepository`
- `mock_character_sheet_repo.go` — implementa `charactersheet.IRepository`

### Estratégia de Testes

- **Pacotes de teste externos** (`package scenario_test`) para testes caixa-preta
- **Testes table-driven** com sub-testes `t.Run()`
- **Comparação de erros** via string `.Error()` (porque os erros de domínio encapsulam via `domain.NewValidationError`)
- **Erros do gateway** importados diretamente para disparar os caminhos de tradução de erro nos use cases (ex: `scenarioPg.ErrScenarioNotFound` → `scenario.ErrScenarioNotFound`)

## Use Cases Cobertos

### Scenario (3 UCs, 14 casos de teste)
| Use Case | Casos de Teste |
|----------|---------------|
| CreateScenario | sucesso, nome curto demais, nome longo demais, descrição breve longa demais, nome já existe, erros de repo |
| GetScenario | sucesso como dono, não encontrado, permissões insuficientes, erro de repo |
| ListScenarios | sucesso com resultados, vazio, erro de repo |

### Campaign (3 UCs, 13 casos de teste)
| Use Case | Casos de Teste |
|----------|---------------|
| CreateCampaign | sucesso (com/sem cenário), tamanho do nome, data de início, desc breve, limite máximo, cenário não encontrado, erro de repo |
| GetCampaign | sucesso como dono, sucesso público outro usuário, privado negado, não encontrado, erro de repo |
| ListCampaigns | sucesso com resultados, vazio, erro de repo |

### Match (4 UCs, 14 casos de teste)
| Use Case | Casos de Teste |
|----------|---------------|
| CreateMatch | sucesso, tamanho do título, desc breve, início do jogo passado/futuro, campanha não encontrada, não é dono, limites de início da história |
| GetMatch | sucesso como dono, sucesso público, privado negado, não encontrado, erro de repo |
| ListMatches | sucesso, vazio, erro de repo |
| ListPublicUpcoming | sucesso, vazio, erro de repo |

### Auth (2 UCs, 23 casos de teste)
| Use Case | Casos de Teste |
|----------|---------------|
| Register | sucesso, nick ausente, tamanho do nick, email ausente, tamanho do email, senha ausente, confirmação ausente, tamanho da senha, senhas não coincidem, nick já existe, email já existe, erros de repo |
| Login | sucesso (verificação bcrypt), email ausente, tamanho do email, senha ausente, tamanho da senha, email não encontrado, senha incorreta |

### Submission (3 UCs, 14 casos de teste)
| Use Case | Casos de Teste |
|----------|---------------|
| Submit | sucesso, ficha não encontrada, não é dono, já submetida, campanha não encontrada, mestre auto-submissão, erro de repo |
| Accept | sucesso, submissão não encontrada, campanha não encontrada, não é mestre |
| Reject | sucesso, submissão não encontrada, não é mestre |

### Enrollment (1 UC, 9 casos de teste)
| Use Case | Casos de Teste |
|----------|---------------|
| Enroll | sucesso, ficha não encontrada, não é dono, UUID do jogador nulo, já inscrito, partida não encontrada, não está na campanha, UUID da campanha nulo, erro de repo |

## Decisões de Design

1. **Sem frameworks externos de mock** — mocks manuais com campos de função são mais simples, mais transparentes e não produzem dependências.

2. **Comparação por string de erro** — como os erros de domínio usam `domain.NewValidationError(errors.New(...))`, os erros resultantes implementam a interface `error`. Comparamos via método `.Error()` ao invés de igualdade de ponteiro.

3. **Importação de erros do gateway nos testes** — os testes importam erros da camada de gateway (ex: `pgUser.ErrEmailNotFound`) para disparar os caminhos de tradução de erro nos use cases. Isso é intencional e verifica se o UC traduz corretamente erros de infraestrutura para erros de domínio.

4. **Teste de Login usa bcrypt real** — o UC de Login chama `bcrypt.CompareHashAndPassword` diretamente, então os testes geram hashes bcrypt reais. Isso garante que o caminho criptográfico real é exercitado.

## Arquivos Criados

```
internal/domain/testutil/
├── doc.go
├── mock_auth_repo.go
├── mock_campaign_repo.go
├── mock_character_sheet_repo.go
├── mock_enrollment_repo.go
├── mock_match_repo.go
├── mock_scenario_repo.go
├── mock_session_repo.go
└── mock_submission_repo.go

internal/domain/scenario/scenario_test.go
internal/domain/campaign/campaign_test.go
internal/domain/match/match_uc_test.go
internal/domain/auth/auth_test.go
internal/domain/submission/submission_test.go
internal/domain/enrollment/enrollment_test.go
```

## Resultados dos Testes

- **21 pacotes passam** (todos novos + todos os testes de entidade pré-existentes)
- **1 falha pré-existente** (`turn/engine_test.go` — conhecido como quebrado pela refatoração Turn/Round, excluído do escopo)
- **87 novos casos de teste** no total, cobrindo 16 use cases
