# Roster da Match — Listagem de Inscrições

Documentação técnica do endpoint `GET /matches/{uuid}/enrollments`, suas
regras de autorização e a estratégia de visibilidade por linha aplicada ao
character sheet.

---

## 1. Endpoint

```
GET /matches/{uuid}/enrollments
```

Retorna o roster completo da partida (`pending` + `accepted` + `rejected`,
sem filtros, ordenado por `created_at ASC`).

## 2. Autorização

O visualizador autenticado é liberado quando **qualquer** das condições é
verdadeira:

1. A match é pública (`matches.is_public = true`).
2. O visualizador é o mestre da match (`userUUID == match.MasterUUID`).
3. O visualizador possui ao menos uma `character_sheets` cuja
   `campaign_uuid` é igual à `campaign_uuid` da match.

Caso nenhuma seja satisfeita → `403 Forbidden` (`ErrInsufficientPermissions`).

A mesma regra foi retroportada para `GET /matches/{uuid}` (`GetMatchUC`),
mantendo os dois endpoints consistentes — quem pode ver a match também pode
ver seu roster, e vice-versa.

## 3. Visibilidade por linha (master vs demais)

A camada de domínio devolve um booleano único `ViewerIsMaster` no resultado
do UC. O handler usa esse bool para decidir, **igual para todas as linhas**,
se anexa o sub-objeto `private` (campos sensíveis do character sheet).

| Visualizador | `character_sheet.private` em todas as linhas |
|---|---|
| Mestre da match | objeto populado |
| Demais (jogadores autorizados) | `null` (sempre serializado, sem `omitempty`) |

### Por que não tratar a linha do próprio jogador de forma especial?

Schema estável é mais valioso do que economizar ~200 bytes por linha. A
estratégia "incluir base sempre, anexar private só para o mestre" mantém:

- Shape único do JSON entre papéis.
- Robustez a sessões novas, hard refresh, deep link, multi-tab (não
  pressupõe estado client-side).
- Lógica do UC reduzida a um booleano (sem decisão por linha).

O frontend pode descartar a linha do próprio quando já tem a ficha local —
é otimização de UI, não responsabilidade da API.

## 4. Forma do response

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
        "campaign_uuid": "…",
        "nick_name": "Gon",
        "story_start_at": "2026-01-01",
        "created_at": "...",
        "updated_at": "...",
        "private": {
          "full_name": "Gon Freecss",
          "level": 5,
          "stamina": { "min": 0, "current": 30, "max": 50 }
        }
      },
      "player": { "uuid": "…", "nick": "tiago" }
    }
  ]
}
```

Formatos de data herdados dos summary types existentes
(`internal/app/api/sheet/character_sheet_sumary_response.go`):
`http.TimeFormat` para `enrollments[].created_at` (igual ao `MatchResponse`),
RFC3339 para `created_at`/`updated_at` do sheet, `2006-01-02` para
`story_start_at`/`story_current_at`, RFC3339 para `dead_at`.

## 5. Camadas

| Camada | Local | Responsabilidade |
|---|---|---|
| Entity | `internal/domain/entity/enrollment/enrollment.go` | Agregado `Enrollment` (uuid, status, created_at, character sheet summary, player ref) |
| Use case | `internal/domain/match/list_match_enrollments.go` | Orquestra match repo + enrollment lister + participation check; devolve `ViewerIsMaster` |
| Gateway (enrollment) | `internal/gateway/pg/enrollment/list_by_match_uuid.go` | SQL com JOIN em `character_sheets`, `character_profiles` e `users` |
| Gateway (sheet) | `internal/gateway/pg/sheet/exists_in_campaign.go` | Check de participação |
| Handler | `internal/app/api/match/list_match_enrollments.go` | Mapeia para o shape acima; popula `private` quando `ViewerIsMaster` |
| Routes | `internal/app/api/match/routes.go` | Registro em `/matches/{uuid}/enrollments` |

### Por que o use case fica em `domain/match` e não em `domain/enrollment`?

A leitura é semanticamente "roster da match" — entradas primárias são
estado de privacidade da match e relação mestre/participante. Inscrições
são dados agregados, não o sujeito da orquestração. Operações que **agem
sobre** uma inscrição (accept/reject) permanecem em `domain/enrollment`.

Para evitar ciclo (`domain/enrollment` já importa `domain/match`), as
dependências de listagem e participação são declaradas como interfaces
locais em `domain/match`:

- `EnrollmentLister.ListByMatchUUID(...)` — satisfeita pelo
  `*pg/enrollment.Repository` via structural typing.
- `CampaignParticipationChecker.ExistsSheetInCampaign(...)` — satisfeita
  pelo `*pg/sheet.Repository` via structural typing.

## 6. Índice

```sql
CREATE INDEX idx_enrollments_match_uuid_created_at
  ON enrollments(match_uuid, created_at);
```

Cobre o filtro por `match_uuid` e o `ORDER BY created_at` em uma única
estrutura — evita sort step. O índice existente
`idx_enrollments_sheet_match_uuid (character_sheet_uuid, match_uuid)` não
ajuda porque a coluna líder é o sheet.
