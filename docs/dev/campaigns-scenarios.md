# Campanhas & Cenários

Documentação técnica dos fluxos de Campaign e Scenario — foco em decisões de design,
regras de validação e pontos não óbvios que o código-fonte sozinho não deixa claro.

---

## 1. Hierarquia de Entidades

A relação entre as entidades segue uma hierarquia de três níveis:

```
Scenario (template de mundo/universo)
  └── Campaign (história dentro de um cenário)
        └── Match (sessão de jogo dentro de uma campanha)
```

**Scenario → Campaign:** `Campaign` referencia o cenário via `ScenarioUUID *uuid.UUID`
(ponteiro — o vínculo é opcional hoje). Na direção inversa, `Scenario` carrega um slice
`Campaigns []*campaign.Summary` para listagem.

**Campaign → Match:** `Campaign` carrega `Matches []match.Summary`. Matches pertencem
a uma única campanha; a referência inversa é resolvida no repositório via
`GetMatchCampaignUUID`.

**Ponto de atenção:** O campo `ScenarioUUID` é `*uuid.UUID` na entidade `Campaign`,
mas no `Summary` é `uuid.UUID` (valor, não ponteiro). Isso significa que o Summary
assume que toda campanha listada já possui cenário associado — ou que um UUID zero
é aceitável na view de lista. Qualquer migração futura que torne o vínculo obrigatório
deve considerar essa divergência de tipos.

---

## 2. Regras de Validação

### CreateCampaign

| Campo                     | Regra                      | Erro                      |
|---------------------------|----------------------------|---------------------------|
| `Name`                    | `len >= 5`                 | `ErrMinNameLength`        |
| `Name`                    | `len <= 32`                | `ErrMaxNameLength`        |
| `StoryStartAt`            | não pode ser zero-value    | `ErrInvalidStartDate`     |
| `BriefInitialDescription` | `len <= 255`               | `ErrMaxBriefDescLength`   |
| Quantidade de campanhas   | `< 10` por master          | `ErrMaxCampaignsLimit`    |
| `ScenarioUUID` (opcional) | deve existir no repositório| `ErrScenarioNotFound`     |

> **⚠️ BUG DOCUMENTADO:** A mensagem de `ErrMaxBriefDescLength` em `error.go` diz
> *"cannot exceed 64 characters"*, porém o código em `create_campaign.go` valida
> `len(input.BriefInitialDescription) > 255`. A validação real é **255 caracteres**;
> a mensagem de erro está incorreta.

A validação de `ScenarioUUID` só ocorre quando o ponteiro não é `nil`. Um comentário
no código indica que *"currently campaigns do not belong to scenarios, but this will
change soon"* — ou seja, o vínculo deve se tornar obrigatório em versão futura.

### CreateScenario

| Campo              | Regra                              | Erro                            |
|--------------------|------------------------------------|---------------------------------|
| `Name`             | `len >= 5`                         | `ErrMinNameLength`              |
| `Name`             | `len <= 32`                        | `ErrMaxNameLength`              |
| `BriefDescription` | `len <= 64`                        | `ErrMaxBriefDescLength`         |
| `Name`             | único (via `ExistsScenarioWithName`) | `ErrScenarioNameAlreadyExists` |

Note que o limite de `BriefDescription` é **64** para Scenario e **255** para Campaign
(no código real). Isso é intencional — cenários têm descrições mais curtas na listagem.

---

## 3. Modelo de Visibilidade e Acesso

### Campaign

- **GetCampaign:** Campanhas **privadas** (`IsPublic == false`) só são visíveis para o
  master (`MasterUUID == userUUID`). Campanhas **públicas** são acessíveis por qualquer
  usuário autenticado. Caso contrário, retorna `auth.ErrInsufficientPermissions`.

- **ListCampaigns:** `ListCampaignsByMasterUUID` filtra exclusivamente pelo UUID do
  master — não há listagem pública de campanhas hoje.

### Scenario

- **GetScenario:** Apenas o dono (`UserUUID == userUUID`) pode visualizar o cenário.
  Qualquer outro usuário recebe `auth.ErrInsufficientPermissions`. Não existe conceito
  de cenário público no momento.

- **ListScenarios:** `ListScenariosByUserUUID` filtra pelo UUID do dono.

**Observação sobre erros de repositório:** Tanto `GetCampaign` quanto `GetScenario`
traduzem erros do gateway (`campaignPg.ErrCampaignNotFound`, `scenarioPg.ErrScenarioNotFound`)
para erros de domínio. Isso desacopla a camada de domínio do PostgreSQL e garante que
controllers recebam apenas erros do domínio.

---

## 4. Summary Pattern

Ambas as entidades possuem structs `Summary` para views de listagem, mais leves que
a entidade completa:

### Campaign Summary

Exclui os slices pesados que exigem JOINs adicionais:
- `CharacterSheets []model.CharacterSheetSummary` — removido
- `PendingSheets []model.CharacterSheetSummary` — removido
- `Matches []match.Summary` — removido
- `Description string` — mantida apenas `BriefInitialDescription`

Mantém todos os campos de metadados (datas, visibilidade, link).

### Scenario Summary

Exclui:
- `Campaigns []*campaign.Summary` — removido
- `Description string` — removido
- `UserUUID uuid.UUID` — removido (o contexto de listagem já filtra por dono)

Mantém: `UUID`, `Name`, `BriefDescription`, `CreatedAt`, `UpdatedAt`.

Esse padrão reduz a carga no banco (menos JOINs) e no payload HTTP. As queries de
listagem (`ListCampaignsByMasterUUID`, `ListScenariosByUserUUID`) retornam diretamente
slices de `Summary`.

---

## Referências de Código

| Arquivo                                | Responsabilidade                          |
|----------------------------------------|-------------------------------------------|
| `internal/domain/entity/campaign/campaign.go` | Entidade Campaign                   |
| `internal/domain/entity/campaign/summary.go`  | Summary de Campaign                 |
| `internal/domain/entity/scenario/scenario.go` | Entidade Scenario                   |
| `internal/domain/entity/scenario/summary.go`  | Summary de Scenario                 |
| `internal/domain/campaign/create_campaign.go`  | UC de criação de campanha           |
| `internal/domain/campaign/get_campaign.go`     | UC de leitura com controle de acesso|
| `internal/domain/campaign/error.go`            | Erros de domínio de Campaign        |
| `internal/domain/campaign/i_repository.go`     | Interface de repositório Campaign   |
| `internal/domain/scenario/create_scenario.go`  | UC de criação de cenário            |
| `internal/domain/scenario/get_scenario.go`     | UC de leitura com controle de acesso|
| `internal/domain/scenario/i_repository.go`     | Interface de repositório Scenario   |
