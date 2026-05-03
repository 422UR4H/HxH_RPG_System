# Listagem de Inscrições da Match — Spec de Design

## Problema

A página de match precisa exibir o roster de todas as inscrições de uma partida (`pending`, `accepted` e `rejected`). O `GET /matches/{uuid}` atual retorna apenas dados da match e é reutilizado em outras telas — embarcar inscrições acoplaria consumidores e inflaria payload de chamadas que não precisam dessa informação. É necessário um endpoint dedicado.

Além da listagem, dois controles de acesso devem ser aplicados:

1. **Visibilidade da match** — para matches privadas, apenas o mestre e jogadores que participam da campanha podem visualizar o roster.
2. **Escopo de dados por linha** — o mestre vê o summary completo do personagem (campos base + privados); demais visualizadores autorizados veem apenas os campos base.

A regra de privacidade de matches privadas hoje está inconsistente em `GetMatchUC` (apenas o mestre é liberado). Este spec retroporta a regra para alinhar os dois endpoints.

## Solução

Adicionar `GET /matches/{uuid}/enrollments` retornando o roster da match com visibilidade por linha derivada da relação do visualizador com a match. Manter `GET /matches/{uuid}` com o mesmo formato, mas harmonizar sua regra de acesso para matches privadas.

## Endpoint

```
GET /matches/{uuid}/enrollments
```

Auth: obrigatória (middleware existente).

### Autorização (nível da match)

O visualizador é autorizado quando **qualquer** das condições abaixo é verdadeira:

- A match é pública (`is_public = true`)
- O visualizador é o mestre da match (`userUUID == match.MasterUUID`)
- O visualizador possui ao menos uma ficha de personagem vinculada à campanha da match (`EXISTS(SELECT 1 FROM character_sheets WHERE player_uuid = $userUUID AND campaign_uuid = $match.CampaignUUID)`)

Caso contrário → `403 Forbidden`.

### Visibilidade (por linha)

Dois tiers, decididos por um único booleano computado uma vez no use case (`viewerIsMaster := userUUID == match.MasterUUID`):

| Visualizador | Payload do sheet por inscrição |
|---|---|
| Mestre da match | Campos base + objeto `private` aninhado preenchido para todas as linhas |
| Qualquer outro autorizado | Apenas campos base; `private` é `null` em todas as linhas |

Razões para não tratar de forma especial a linha do próprio jogador:
- Schema JSON estável simplifica o frontend.
- Robusto a sessões novas, hard refresh, deep link, multi-tab.
- O summary base é pequeno (~200 bytes); redundância negligível.
- Mantém a lógica do use case em um único booleano.

O frontend pode optar por descartar os dados redundantes da própria linha — é otimização de UI, não responsabilidade da API.

### Response

```json
{
  "enrollments": [
    {
      "uuid": "…",
      "status": "pending",
      "created_at": "Mon, 02 Jan 2006 15:04:05 GMT",
      "character_sheet": {
        "uuid": "…",
        "player_uuid": "…",
        "master_uuid": null,
        "campaign_uuid": "…",
        "nick_name": "Gon",
        "story_start_at": "2026-01-01",
        "story_current_at": "2026-01-15",
        "dead_at": null,
        "created_at": "…",
        "updated_at": "…",
        "private": {
          "full_name": "Gon Freecss",
          "alignment": "neutral_good",
          "character_class": "hunter",
          "birthday": "1987-05-05",
          "category_name": "reinforcement",
          "curr_hex_value": 80,
          "level": 5,
          "points": 12,
          "talent_lvl": 3,
          "physicals_lvl": 4,
          "mentals_lvl": 2,
          "spirituals_lvl": 3,
          "skills_lvl": 5,
          "stamina": { "min": 0, "current": 30, "max": 50 },
          "health":  { "min": 0, "current": 40, "max": 60 }
        }
      },
      "player": { "uuid": "…", "nick": "tiago" }
    }
  ]
}
```

`private` é sempre serializado como `null` para visualizadores não-mestres (sem `omitempty`) para que o shape do JSON seja estável entre os papéis.

Formatos de data herdados dos tipos summary existentes: `created_at` da inscrição usa `http.TimeFormat` (RFC1123 GMT, igual a `MatchResponse`); `created_at`/`updated_at` do sheet usam RFC3339; `story_start_at`/`story_current_at` usam `2006-01-02`; `dead_at` usa RFC3339 — tudo conforme os mapeamentos existentes em `toSummaryBaseResponse` e `ToPrivateSummaryResponse`.

Ordenação: `ORDER BY enrollments.created_at ASC`.

### Mapeamento de Erros

| Condição | HTTP |
|---|---|
| Não autenticado | 401 (middleware) |
| Match não encontrada | 404 (`ErrMatchNotFound`) |
| Match privada, visualizador não é mestre nem participante da campanha | 403 (`ErrInsufficientPermissions`) |
| Outro erro de repositório / servidor | 500 |

## Mudanças de Schema

### Migration: índice para `match_uuid + created_at`

```sql
-- migrations/<timestamp>_add_enrollments_match_uuid_index.sql
-- +goose Up
CREATE INDEX idx_enrollments_match_uuid_created_at
  ON enrollments(match_uuid, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_enrollments_match_uuid_created_at;
```

O índice composto cobre tanto o filtro quanto o `ORDER BY` da query, eliminando o sort step. O índice existente `idx_enrollments_sheet_match_uuid (character_sheet_uuid, match_uuid)` não ajuda — sua coluna líder é o sheet, não o match.

## Camada de Domínio

### Entidade Nova (`internal/domain/entity/enrollment/`)

Hoje não existe package de entidade para enrollment. Introduzir um seguindo a convenção de `entity/match/summary.go`:

```go
// internal/domain/entity/enrollment/enrollment.go
package enrollment

import (
    "time"

    sheetModel "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
    "github.com/google/uuid"
)

type PlayerRef struct {
    UUID uuid.UUID
    Nick string
}

type Enrollment struct {
    UUID           uuid.UUID
    Status         string
    CreatedAt      time.Time
    // TODO(architecture): CharacterSheetSummary fica em gateway/pg/model — entity não deveria
    // importar camadas externas. Tracked para cleanup: mover CharacterSheetSummary para
    // domain/entity/character_sheet/summary.go em task posterior e atualizar todos call sites
    // (use cases em domain/character_sheet/ já importam model.CharacterSheetSummary também,
    // então o cleanup é compartilhado, não específico de enrollment).
    CharacterSheet sheetModel.CharacterSheetSummary // inclui campos base + privados
    Player         PlayerRef
}
```

Razão: `CharacterSheetSummary` já carrega todos os campos que o `CharacterPrivateSummaryResponse` existente precisa. O use case nunca remove campos — essa decisão fica na camada de handler (visibilidade), mantendo o domínio livre de preocupações de apresentação. A violação arquitetural (entity importando model do gateway) é intencional e segue padrão existente na camada de use case; cleanup adiado para task separada.

### Use Case (`internal/domain/match/`)

Decisão: este UC fica em `domain/match`, não `domain/enrollment`. O endpoint é fundamentalmente uma leitura da página de match cujos inputs primários são o estado de privacidade da match e o relacionamento mestre/participante; inscrições são dados agregados, não o sujeito da orquestração. Operações que agem sobre uma única inscrição (accept/reject) permanecem em `domain/enrollment`.

Para evitar ciclo de dependência (`domain/enrollment` já importa `domain/match`), a dependência de listagem de enrollments é declarada como interface local em `domain/match` (idiom Go "interfaces são definidas onde são consumidas"). A mesma struct do gateway satisfaz ambas as interfaces via structural typing.

```go
// internal/domain/match/list_match_enrollments.go
package match

import (
    "context"
    "errors"

    "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
    enrollmentEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enrollment"
    matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
    "github.com/google/uuid"
)

type EnrollmentLister interface {
    ListByMatchUUID(
        ctx context.Context, matchUUID uuid.UUID,
    ) ([]*enrollmentEntity.Enrollment, error)
}

type CampaignParticipationChecker interface {
    ExistsSheetInCampaign(
        ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID,
    ) (bool, error)
}

type ListMatchEnrollmentsResult struct {
    Enrollments     []*enrollmentEntity.Enrollment
    ViewerIsMaster  bool
}

type IListMatchEnrollments interface {
    List(
        ctx context.Context, matchUUID uuid.UUID, userUUID uuid.UUID,
    ) (*ListMatchEnrollmentsResult, error)
}

type ListMatchEnrollmentsUC struct {
    matchRepo            IRepository
    enrollmentLister     EnrollmentLister
    participationChecker CampaignParticipationChecker
}

func NewListMatchEnrollmentsUC(
    matchRepo IRepository,
    enrollmentLister EnrollmentLister,
    participationChecker CampaignParticipationChecker,
) *ListMatchEnrollmentsUC { /* ... */ }

func (uc *ListMatchEnrollmentsUC) List(
    ctx context.Context, matchUUID uuid.UUID, userUUID uuid.UUID,
) (*ListMatchEnrollmentsResult, error) {
    match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
    if errors.Is(err, matchPg.ErrMatchNotFound) {
        return nil, ErrMatchNotFound
    }
    if err != nil {
        return nil, err
    }

    viewerIsMaster := match.MasterUUID == userUUID
    if !match.IsPublic && !viewerIsMaster {
        ok, err := uc.participationChecker.ExistsSheetInCampaign(
            ctx, userUUID, match.CampaignUUID,
        )
        if err != nil {
            return nil, err
        }
        if !ok {
            return nil, auth.ErrInsufficientPermissions
        }
    }

    enrollments, err := uc.enrollmentLister.ListByMatchUUID(ctx, matchUUID)
    if err != nil {
        return nil, err
    }
    return &ListMatchEnrollmentsResult{
        Enrollments:    enrollments,
        ViewerIsMaster: viewerIsMaster,
    }, nil
}
```

### Retrofit do `GetMatchUC` (mesmo PR, commit separado)

Atualizar a checagem atual de match privada para alinhar com a regra nova:

```go
// antes
if match.MasterUUID != userUUID && !match.IsPublic {
    return nil, auth.ErrInsufficientPermissions
}

// depois
if !match.IsPublic && match.MasterUUID != userUUID {
    ok, err := uc.participationChecker.ExistsSheetInCampaign(
        ctx, userUUID, match.CampaignUUID,
    )
    if err != nil { return nil, err }
    if !ok { return nil, auth.ErrInsufficientPermissions }
}
```

`GetMatchUC` ganha o parâmetro `participationChecker CampaignParticipationChecker` no construtor (mesma interface local). Wiring atualizado em `cmd/api/main.go`. Todos os testes e call sites de `GetMatchUC` atualizados.

## Camada de Gateway

### `internal/gateway/pg/enrollment/list_by_match_uuid.go`

```go
func (r *Repository) ListByMatchUUID(
    ctx context.Context, matchUUID uuid.UUID,
) ([]*enrollmentEntity.Enrollment, error) {
    const query = `
        SELECT
            e.uuid, e.status, e.created_at,
            cs.id, cs.uuid, cs.player_uuid, cs.master_uuid, cs.campaign_uuid,
            cs.category_name, cs.curr_hex_value,
            cs.level, cs.points, cs.talent_lvl, cs.skills_lvl,
            cs.health_min_pts, cs.health_curr_pts, cs.health_max_pts,
            cs.stamina_min_pts, cs.stamina_curr_pts, cs.stamina_max_pts,
            cs.physicals_lvl, cs.mentals_lvl, cs.spirituals_lvl,
            cs.aura_min_pts, cs.aura_curr_pts, cs.aura_max_pts,
            cs.created_at, cs.updated_at,
            cp.nickname, cp.fullname, cp.alignment, cp.character_class, cp.birthday,
            u.uuid, u.nick
        FROM enrollments e
        JOIN character_sheets cs   ON cs.uuid = e.character_sheet_uuid
        JOIN character_profiles cp ON cp.character_sheet_uuid = cs.uuid
        JOIN users u               ON u.uuid = cs.player_uuid
        WHERE e.match_uuid = $1
        ORDER BY e.created_at ASC
    `
    // scan em *enrollmentEntity.Enrollment, retorna slice
}
```

Notas:
- INNER JOIN em `users` é seguro: toda inscrição requer `cs.player_uuid != nil` (validado por `EnrollCharacterInMatchUC`).
- Retorna slice vazia (não erro) quando a match existe mas não tem inscrições.
- Reutiliza o conjunto de campos já selecionado por `ListCharacterSheetsByPlayerUUID` para consistência com o mapeamento existente do private summary.

### `internal/gateway/pg/sheet/exists_in_campaign.go`

```go
func (r *Repository) ExistsSheetInCampaign(
    ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID,
) (bool, error) {
    const query = `
        SELECT EXISTS (
            SELECT 1 FROM character_sheets
            WHERE player_uuid = $1 AND campaign_uuid = $2
        )
    `
    var exists bool
    err := r.q.QueryRow(ctx, query, playerUUID, campaignUUID).Scan(&exists)
    return exists, err
}
```

Adicionar a `domain/character_sheet/i_repository.go` (sheet repo também satisfaz `match.CampaignParticipationChecker` via este método).

## Camada App / Handler

### `internal/app/api/match/list_match_enrollments.go`

```go
type ListMatchEnrollmentsRequest struct {
    UUID uuid.UUID `path:"uuid" required:"true" doc:"UUID da match"`
}

type ListMatchEnrollmentsResponse struct {
    Body ListMatchEnrollmentsResponseBody `json:"body"`
}

type ListMatchEnrollmentsResponseBody struct {
    Enrollments []EnrollmentResponse `json:"enrollments"`
}

type EnrollmentResponse struct {
    UUID           uuid.UUID                              `json:"uuid"`
    Status         string                                 `json:"status"`
    CreatedAt      string                                 `json:"created_at"`
    CharacterSheet CharacterSheetWithVisibilityResponse   `json:"character_sheet"`
    Player         PlayerRefResponse                      `json:"player"`
}

type CharacterSheetWithVisibilityResponse struct {
    sheetHandler.CharacterBaseSummaryResponse
    Private *sheetHandler.CharacterPrivateOnlyResponse `json:"private"`
}

type PlayerRefResponse struct {
    UUID uuid.UUID `json:"uuid"`
    Nick string    `json:"nick"`
}
```

O `CharacterPrivateSummaryResponse` existente flat-aniza campos base + privados. Extrair os campos privados-only em `CharacterPrivateOnlyResponse` (struct sem o embedded base) para que o handler aninhe limpo. Manter `CharacterPrivateSummaryResponse` intacto para os callers existentes; a struct nova é um subset, definida ao lado.

Lógica do handler:
1. Decode UUID do path.
2. Lê `userUUID` do contexto.
3. Chama UC; mapeia erros de domínio → códigos HTTP.
4. Monta response: para cada inscrição, popula o summary base; se `result.ViewerIsMaster` for true, popula também o `private` aninhado.

### `internal/app/api/match/routes.go`

Adicionar campo `ListMatchEnrollmentsHandler` na struct `Api` e registrar:

```go
huma.Register(api, huma.Operation{
    Method:      http.MethodGet,
    Path:        "/matches/{uuid}/enrollments",
    Description: "List enrollments of a match (visibility per row depends on viewer)",
    Tags:        []string{"matches"},
    Errors: []int{
        http.StatusUnauthorized,
        http.StatusForbidden,
        http.StatusNotFound,
        http.StatusInternalServerError,
    },
}, a.ListMatchEnrollmentsHandler)
```

## Wiring (`cmd/api/main.go`)

```go
listMatchEnrollmentsUC := domainMatch.NewListMatchEnrollmentsUC(
    matchRepo,
    enrollmentRepo,        // satisfaz match.EnrollmentLister
    characterSheetRepo,    // satisfaz match.CampaignParticipationChecker
)

// passar characterSheetRepo também para o construtor (refatorado) de GetMatchUC:
getMatchUC := domainMatch.NewGetMatchUC(matchRepo, characterSheetRepo)

matchesApi := matchHandler.Api{
    // campos existentes...
    ListMatchEnrollmentsHandler: matchHandler.ListMatchEnrollmentsHandler(listMatchEnrollmentsUC),
}
```

## Testes

Conforme a estratégia TDD-por-camada do projeto:

### Use case (`internal/domain/match/list_match_enrollments_test.go`)
Unit tests com mocks para `IRepository`, `EnrollmentLister`, `CampaignParticipationChecker`. Casos:
- Mestre em match privada → `ViewerIsMaster=true`, participation check não chamado
- Mestre em match pública → `ViewerIsMaster=true`
- Não-mestre em match pública → `ViewerIsMaster=false`, participation check não chamado
- Não-mestre em match privada, participa → `ViewerIsMaster=false`
- Não-mestre em match privada, não participa → `ErrInsufficientPermissions`
- Match não encontrada → `ErrMatchNotFound`
- `EnrollmentLister` retorna vazio → slice vazia, sem erro
- `EnrollmentLister` retorna erro → propagado
- `participationChecker` retorna erro → propagado

### `GetMatchUC` (arquivo de teste existente, expandir)
Adicionar casos para o caminho novo de participação:
- Não-mestre em match privada, participa → sucesso
- Não-mestre em match privada, não participa → `ErrInsufficientPermissions`

### Gateway — enrollment (`internal/gateway/pg/enrollment/enrollment_integration_test.go`)
Adicionar `TestListByMatchUUID` com sub-tests:
- Lista todos os status incluindo rejected
- Ordenação por `created_at` ASC
- Slice vazia quando match não tem inscrições
- JOIN materializa nick do jogador + base + campos privados do sheet
- Inscrições de outras matches não são incluídas

### Gateway — sheet (`internal/gateway/pg/sheet/sheet_integration_test.go`)
Adicionar `TestExistsSheetInCampaign` com:
- True quando jogador tem ao menos uma ficha na campanha
- False quando jogador não tem ficha na campanha
- False quando jogador tem fichas apenas em outras campanhas

### Handler (`internal/app/api/match/list_match_enrollments_test.go`)
Humatest, com mock do UC. Casos:
- 200 com `private` populado em todas linhas quando `ViewerIsMaster=true`
- 200 com `private` null/omitido em todas linhas quando `ViewerIsMaster=false`
- 200 com lista vazia
- 404 em `ErrMatchNotFound`
- 403 em `ErrInsufficientPermissions`
- 500 em erro genérico

Mocks adicionados ao `mocks_test.go` do package match handler.

## Documentação

Conforme `docs-workflow.instructions.md` (PT-BR para `docs/dev/`):

1. **Novo** `docs/dev/match/roster.md` — descreve o endpoint de listagem, regra de autorização (público / mestre / participante da campanha), tiers de visibilidade por linha, formato de response, justificativa de schema estável entre visualizadores.
2. **Atualizar** `docs/dev/enrollment.md` — adicionar §7 curta "Listagem por Match" referenciando `roster.md`, sumarizando a natureza cross-domain da leitura (entidade em `domain/entity/enrollment`, use case em `domain/match`).
3. **Atualizar** `docs/documentation-map.yaml` — adicionar mappings:
    - `internal/domain/match/list_match_enrollments.go` → `docs/dev/match/roster.md`
    - `internal/gateway/pg/enrollment/list_by_match_uuid.go` → `docs/dev/match/roster.md` + `docs/dev/enrollment.md`
    - `internal/domain/entity/enrollment/` → `docs/dev/enrollment.md`

## Fases de Implementação

**Fase 1 — Esqueleto do read path (sem retrofit de privacidade ainda)**
Migration + entidade + métodos do gateway + use case (sem checagem de privacidade) + handler com os dois tiers de visibilidade + testes + wiring + docs.

**Fase 2 — Retrofit de privacidade**
Adicionar a checagem de participação ao `ListMatchEnrollmentsUC` e ao `GetMatchUC` (commit separado). Atualizar testes dos dois UCs.

Separar o retrofit de privacidade permite que os reviewers foquem primeiro no trabalho de shape de dados e depois na mudança cross-cutting de autorização.

## Fora de Escopo

- Filtros / paginação por query params (ex: `?status=accepted`). Adiar até a página realmente precisar.
- Endpoint composto "página de match" agregando match + enrollments + estado da cena. Duas chamadas paralelas são suficientes na escala atual; revisitar apenas se latência medida virar problema.
- Permitir que o mestre inscreva ficha em nome de jogador (TODO existente em `EnrollCharacterInMatchUC`). Independente deste trabalho.
- Notificações ao jogador quando o status da inscrição muda. Fora de escopo para um endpoint de leitura.
