# Update Match (PATCH) — Design Spec

**Date:** 2026-05-23
**Status:** Approved
**Trigger:** Front-end precisa de uma página de edição de partida (`EditMatchPage`); o back-end não expõe nenhum endpoint de update para `Match` hoje. Esta spec cobre apenas o back-end + contrato; o front virá em sessão separada.

## Contexto

Hoje `internal/app/api/match/routes.go` registra apenas POST/GET/list/enrollments/participants. Não existe nem método `Update` no `IRepository`, nem use case, nem handler.

Também não existe `docs/dev/api/match.md` — toda a superfície HTTP de matches é hoje undocumented (apesar do `documentation-map.yaml` mapear `internal/app/api/match/` para um doc inexistente em `docs/dev/match/`). Esta spec aproveita o caminho para criar o contrato API completo.

**Regra de negócio central:** o mestre pode editar campos descritivos e de agendamento enquanto a partida ainda não começou (`game_start_at IS NULL`). Após `StartMatch`, qualquer PATCH falha com 422.

## Decisões

| Tópico | Decisão | Justificativa |
|---|---|---|
| Método HTTP | `PATCH /matches/{uuid}` | Semântica de update parcial; consistente com `PATCH /charactersheets/{uuid}` |
| Body | Todos os campos opcionais (ponteiros) | Permite envio só do que mudou; FE pode usar com `dirtyFields` |
| `campaign_uuid` no body | Não permitido | Mover partida entre campanhas é mudança de relação, não de atributo; mais complexidade que valor |
| Campos editáveis | `title`, `brief_initial_description`, `description`, `is_public`, `game_scheduled_at`, `story_start_at` | Mesmo conjunto de create |
| Pré-condição | `game_start_at IS NULL && story_end_at IS NULL` | Espelha checagens de `StartMatchUC` |
| Validações | Mesmas regras de tamanho/data de create, aplicadas só nos campos presentes | Não duplica regra; ainda permite parciais |
| Resposta | 200 com `MatchResponse` completo | Permite FE substituir a entidade no cache sem refetch |
| Repository SQL | `UPDATE ... WHERE uuid = $X AND game_start_at IS NULL` | Cinto de segurança para race com `StartMatch` concorrente |
| Update "vazio" | Retorna 200 com a partida atual, sem hit no banco | Idempotente; evita escrita supérflua |

## Arquitetura

Três camadas espelham `CreateMatch`:

```
HTTP                       Use Case                     Gateway
─────────────────────      ────────────────────────     ─────────────────────
PATCH /matches/{uuid}  →   UpdateMatchUC.Update    →   Repository.UpdateMatch
update_match.go            update_match.go             update_match.go
```

## Contrato HTTP

### Request

`PATCH /matches/{uuid}` — JWT obrigatório (mestre da partida).

Body — todos os campos opcionais:

```json
{
  "title": "Novo título",
  "brief_initial_description": "Descrição breve atualizada",
  "description": "Descrição completa",
  "is_public": false,
  "game_scheduled_at": "2026-07-20T19:30:00Z",
  "story_start_at": "2026-07-20"
}
```

### Response

200 OK — retorna a partida atualizada:

```json
{
  "match": {
    "uuid": "...",
    "master_uuid": "...",
    "campaign_uuid": "...",
    "title": "Novo título",
    "brief_initial_description": "...",
    "brief_final_description": null,
    "description": "...",
    "is_public": false,
    "game_scheduled_at": "2026-07-20T19:30:00Z",
    "game_start_at": null,
    "story_start_at": "2026-07-20",
    "story_end_at": null,
    "created_at": "...",
    "updated_at": "..."
  }
}
```

### Matriz de erros

| Status | Cenário |
|---|---|
| 200 | Atualizado (ou no-op se body vazio) |
| 400 | UUID inválido na rota ou JSON malformado |
| 401 | Sem JWT |
| 403 | Usuário autenticado não é o mestre |
| 404 | `matches.uuid` não existe ou campanha referenciada sumiu |
| 422 | Validação (tamanho, data fora de janela) ou partida já iniciada/encerrada |
| 500 | Erro interno |

## Camada HTTP

**Novo:** `internal/app/api/match/update_match.go`

```go
type UpdateMatchRequestBody struct {
    Title                   *string `json:"title,omitempty" minLength:"5" maxLength:"32"`
    BriefInitialDescription *string `json:"brief_initial_description,omitempty" maxLength:"64"`
    Description             *string `json:"description,omitempty"`
    IsPublic                *bool   `json:"is_public,omitempty"`
    GameScheduledAt         *string `json:"game_scheduled_at,omitempty" doc:"ISO 8601"`
    StoryStartAt            *string `json:"story_start_at,omitempty" doc:"YYYY-MM-DD"`
}

type UpdateMatchRequest struct {
    UUID uuid.UUID              `path:"uuid" required:"true"`
    Body UpdateMatchRequestBody `json:"body"`
}

type UpdateMatchResponseBody struct {
    Match MatchResponse `json:"match"`
}

type UpdateMatchResponse struct {
    Body UpdateMatchResponseBody `json:"body"`
}

func UpdateMatchHandler(uc domainMatch.IUpdateMatch) func(...) (...)
```

Responsabilidades:
- Parsing de `GameScheduledAt` (RFC3339) e `StoryStartAt` (YYYY-MM-DD) **só se não-nil**, retornando 422 em formato inválido.
- Monta `domainMatch.UpdateMatchInput` com ponteiros para os campos não-nil.
- Mapeia erros do UC para HTTP:

```go
switch {
case errors.Is(err, domainMatch.ErrMatchNotFound):     // 404
case errors.Is(err, domainCampaign.ErrCampaignNotFound): // 404
case errors.Is(err, domainMatch.ErrNotMatchMaster):    // 403
case errors.Is(err, domainMatch.ErrMatchAlreadyStarted): // 422
case errors.Is(err, domainMatch.ErrMatchAlreadyFinished): // 422
case errors.Is(err, domain.ErrValidation):             // 422
default: // 500
}
```

**Modificado:** `internal/app/api/match/routes.go` — adicionar `UpdateMatchHandler` em `Api` e registrar:

```go
huma.Register(api, huma.Operation{
    Method: http.MethodPatch,
    Path:   "/matches/{uuid}",
    Description: "Update a match (only by master, only before game starts)",
    Tags:   []string{"matches"},
    Errors: []int{
        http.StatusNotFound, http.StatusBadRequest, http.StatusForbidden,
        http.StatusUnauthorized, http.StatusUnprocessableEntity,
        http.StatusInternalServerError,
    },
}, a.UpdateMatchHandler)
```

## Camada Use Case

**Novo:** `internal/application/match/update_match.go`

```go
type IUpdateMatch interface {
    Update(ctx context.Context, input *UpdateMatchInput) (*match.Match, error)
}

type UpdateMatchInput struct {
    MatchUUID  uuid.UUID
    MasterUUID uuid.UUID

    Title                   *string
    BriefInitialDescription *string
    Description             *string
    IsPublic                *bool
    GameScheduledAt         *time.Time
    StoryStartAt            *time.Time
}

type UpdateMatchUC struct {
    matchRepo    IRepository
    campaignRepo domainCampaign.IRepository
}

func NewUpdateMatchUC(matchRepo IRepository, campaignRepo domainCampaign.IRepository) *UpdateMatchUC
```

### Fluxo de `Update`

1. **Load** — `matchRepo.GetMatch(ctx, input.MatchUUID)`; mapear `pgMatch.ErrMatchNotFound → domainMatch.ErrMatchNotFound`.
2. **Auth** — `match.MasterUUID != input.MasterUUID` → `ErrNotMatchMaster`.
3. **Estado** —
   - `match.GameStartAt != nil` → `ErrMatchAlreadyStarted`.
   - `match.StoryEndAt != nil` → `ErrMatchAlreadyFinished`.
4. **No-op short-circuit** — se todos os ponteiros são `nil`, retornar `match, nil` sem ir ao banco.
5. **Validações por campo (só não-nil):**
   - `Title`: 5 ≤ len ≤ 32 → `ErrMinTitleLength` / `ErrMaxTitleLength`.
   - `BriefInitialDescription`: len ≤ 64 → `ErrMaxBriefDescLength` (alinha com tag `maxLength:"64"` do create handler).
   - `GameScheduledAt`: `!Before(now)` e `!After(now+1y)` → `ErrMinOfGameScheduledAt` / `ErrMaxOfGameScheduledAt`.
6. **Campanha (se `StoryStartAt != nil`):**
   - `campaignRepo.GetCampaignStoryDates(ctx, match.CampaignUUID)`.
   - `StoryStartAt < campaign.StoryStartAt` → `ErrMinOfStoryStartAt`.
   - `campaign.StoryEndAt != nil && StoryStartAt > *campaign.StoryEndAt` → `ErrMaxOfStoryStartAt`.
7. **Apply** — atribui os campos não-nil ao `match`, seta `match.UpdatedAt = time.Now()`.
8. **Persist** — `matchRepo.UpdateMatch(ctx, match)`; se retornar `pgMatch.ErrMatchNotFound`, mapear para `ErrMatchAlreadyStarted` (única causa de `RowsAffected == 0` num registro que acabamos de ler com `game_start_at IS NULL`).
9. Retorna `match` atualizado.

**Errors novos** em `internal/application/match/error.go`: nenhum — todos os erros já existem (`ErrMatchNotFound`, `ErrNotMatchMaster`, `ErrMatchAlreadyStarted`, `ErrMatchAlreadyFinished`, `ErrMin/MaxTitleLength`, `ErrMaxBriefDescLength`, `ErrMin/MaxOfGameScheduledAt`, `ErrMin/MaxOfStoryStartAt`).

## Camada Gateway

**Novo:** `internal/gateway/pg/match/update_match.go`

```go
func (r *Repository) UpdateMatch(ctx context.Context, m *match.Match) error {
    tx, err := r.q.Begin(ctx)
    // ... padrão de tx + defer rollback igual aos outros métodos
    const query = `
        UPDATE matches SET
            title = $1,
            brief_initial_description = $2,
            description = $3,
            is_public = $4,
            game_scheduled_at = $5,
            story_start_at = $6,
            updated_at = $7
        WHERE uuid = $8 AND game_start_at IS NULL
    `
    result, err := tx.Exec(ctx, query,
        m.Title, m.BriefInitialDescription, m.Description,
        m.IsPublic, m.GameScheduledAt, m.StoryStartAt,
        m.UpdatedAt, m.UUID,
    )
    if err != nil { return ... }
    if result.RowsAffected() == 0 { return ErrMatchNotFound }
    return tx.Commit(ctx)
}
```

O UC já carregou e aplicou os patches em memória; o gateway só escreve. `WHERE game_start_at IS NULL` é cinto de segurança contra `StartMatch` rodando em paralelo.

**Modificado:** `internal/application/match/i_repository.go` — adicionar à interface:

```go
type IRepository interface {
    // ... existentes
    UpdateMatch(ctx context.Context, match *match.Match) error
}
```

**Modificado:** `internal/application/testutil/mock_match_repo.go` — adicionar `UpdateMatchFn` e método `UpdateMatch`.

## Wiring

**Modificado:** `cmd/api/main.go` — após `createMatchUC`:

```go
updateMatchUC := domainMatch.NewUpdateMatchUC(matchRepo, campaignRepo)

matchesApi := matchHandler.Api{
    // ... existentes
    UpdateMatchHandler: matchHandler.UpdateMatchHandler(updateMatchUC),
}
```

## Testes

### Handler — `internal/app/api/match/update_match_test.go`

Casos:
- `success_full_patch` — todos os campos no body, status 200, response.match contém os novos valores.
- `success_partial_patch` — só `title`, retorna match com title novo e demais inalterados.
- `success_empty_body` — body `{}`, retorna match atual.
- `invalid_game_scheduled_at` — string mal formada → 422.
- `invalid_story_start_at` — string mal formada → 422.
- `match_not_found` → 404.
- `not_master` → 403.
- `already_started` → 422.
- `validation_error` (UC retorna `ErrMinTitleLength` por ex) → 422.
- `internal_server_error` → 500.

Mock em `mocks_test.go`: `mockUpdateMatch` que satisfaz `IUpdateMatch`.

### Use case — adicionar `TestUpdateMatch` em `internal/application/match/match_uc_test.go`

Casos:
- `success_partial` — atualiza só `title`, demais permanecem.
- `success_full` — atualiza todos os campos.
- `no_op_empty_input` — todos os ponteiros nil → retorna match sem chamar `UpdateMatch` (verificar via `t.Fatal` no mock).
- `match_not_found` — repo retorna `pgMatch.ErrMatchNotFound`.
- `not_master` — match com outro `MasterUUID`.
- `already_started` — match com `GameStartAt != nil`.
- `already_finished` — match com `StoryEndAt != nil`.
- `title_too_short` / `title_too_long`.
- `brief_too_long`.
- `game_scheduled_in_past` / `game_scheduled_too_far`.
- `story_start_before_campaign` — exige `GetCampaignStoryDatesFn` no mock de campanha.
- `story_start_after_campaign_end` — campanha com `StoryEndAt` setado.
- `repo_update_returns_not_found_race` — `UpdateMatchFn` devolve `pgMatch.ErrMatchNotFound`; UC mapeia para `ErrMatchAlreadyStarted`.

### Integration — adicionar em `internal/gateway/pg/match/match_integration_test.go`

```go
func TestUpdateMatch(t *testing.T) {
    // happy path: cria, atualiza, lê de volta com novos valores e UpdatedAt > CreatedAt
    // race guard: cria, marca game_start_at, tenta UPDATE → ErrMatchNotFound
}
```

## Documentação

**Novo:** `docs/dev/api/match.md` — cobrir TODA a superfície atual de matches, não só o PATCH novo:

- `POST /matches` (campos, validações, erros)
- `GET /matches/{uuid}` (visibilidade pública/privada via participação)
- `GET /matches` (próprias do mestre)
- `GET /public/matches` (futuras públicas, exclui as próprias)
- `GET /matches/{uuid}/enrollments`
- `GET /matches/{uuid}/participants`
- **`PATCH /matches/{uuid}` (novo)** — body, response, matriz de erros, regra de pré-condição

**Modificado:** `docs/documentation-map.yaml` — adicionar mapeamento explícito para `internal/app/api/match/`:

```yaml
- code_path: internal/app/api/match/
  dev_docs:
    - path: docs/dev/api/match.md
      confidence: directly_affected
  notes: Match REST surface (CRUD + enrollment/participant listings)
```

## Out of scope

- **Front-end** (`EditMatchPage`, `useUpdateMatch`, `matchService.updateMatch`, botão "Editar" em `MatchPage`, rota `/campaigns/:campaignId/matches/:matchId/edit`) — sessão futura, consumindo `docs/dev/api/match.md`.
- **Docs ausentes em `docs/dev/match/`** — `scenes.md`, `turns-rounds.md`, `actions.md`, `roster.md` são referenciadas em `documentation-map.yaml` mas o diretório `docs/dev/match/` não existe. Cobrem mecânica de gameplay (cenas, turnos, ações, roster), fora do escopo desta task que toca só a superfície REST de metadados. Follow-up: criar essas docs ou remover as referências do map.
- **Edição de partidas em andamento ou encerradas** — explicitamente bloqueada por regra de negócio.
- **Mover partida entre campanhas** — `campaign_uuid` é imutável.

## Arquivos tocados

**Novos (5):**
- `internal/application/match/update_match.go`
- `internal/app/api/match/update_match.go`
- `internal/app/api/match/update_match_test.go`
- `internal/gateway/pg/match/update_match.go`
- `docs/dev/api/match.md`

**Modificados (8):**
- `internal/application/match/i_repository.go` — `+ UpdateMatch`
- `internal/application/testutil/mock_match_repo.go` — `+ UpdateMatchFn` + método
- `internal/application/match/match_uc_test.go` — `+ TestUpdateMatch`
- `internal/app/api/match/routes.go` — `+ UpdateMatchHandler` + `huma.Register`
- `internal/app/api/match/mocks_test.go` — `+ mockUpdateMatch`
- `internal/gateway/pg/match/match_integration_test.go` — `+ TestUpdateMatch`
- `cmd/api/main.go` — wire `updateMatchUC` + handler
- `docs/documentation-map.yaml` — mapeamento para `docs/dev/api/match.md`
